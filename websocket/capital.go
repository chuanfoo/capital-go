package capitalws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/chuanfoo/capital-go/websocket/models"
	rest "github.com/chuanfoo/capital-go/rest"
	"github.com/gorilla/websocket"
	_ "golang.org/x/exp/maps"
	_ "golang.org/x/exp/slices"
	"gopkg.in/tomb.v2"
)

const (
	writeWait      = 5 * time.Second
	pingPeriod     = 9 * time.Minute
	maxMessageSize = 1000000 // 1MB
	apiUrl         = "wss://api-streaming-capital.backend-capital.com/connect"
)

// Client defines a client to the Polygon WebSocket API.
type Client struct {
	cstToken      string
	securityToken string
	apiKey        string
	identifier    string
	password      string
	url           string

	shouldClose bool
	backoff     backoff.BackOff

	mtx    sync.Mutex
	rwtomb tomb.Tomb
	ptomb  tomb.Tomb

	conn   *websocket.Conn
	rQueue chan json.RawMessage
	wQueue chan json.RawMessage
	subs   subscriptions
	epics  []string

	rawData              bool
	bypassRawDataRouting bool
	output               chan any
	err                  chan error

	log Logger
}

// New creates a client for the Polygon WebSocket API.
func New(config Config, cstToken, securityToken, identifier, password string) (*Client, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid client options: %w", err)
	}

	c := &Client{
		cstToken:             cstToken,
		securityToken:        securityToken,
		identifier:           identifier,
		password:             password,
		apiKey:               config.APIKey,
		backoff:              backoff.NewExponentialBackOff(),
		rQueue:               make(chan json.RawMessage, 10000),
		wQueue:               make(chan json.RawMessage, 1000),
		subs:                 make(subscriptions),
		rawData:              config.RawData,
		bypassRawDataRouting: config.BypassRawDataRouting,
		output:               make(chan any, 100000),
		err:                  make(chan error),
		log:                  config.Log,
	}

	uri, err := url.Parse(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid data feed format: %v", err)
	}
	c.url = uri.String()

	if config.MaxRetries != nil {
		c.backoff = backoff.WithMaxRetries(c.backoff, *config.MaxRetries)
	}

	return c, nil
}

// Connect dials the WebSocket server and starts the read/write and process threads.
// If any subscription messages are pushed before connecting, it will also send those
// to the server.
func (c *Client) Connect() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.conn != nil {
		return nil
	}

	notify := func(err error, _ time.Duration) {
		c.log.Errorf(err.Error())
	}
	if err := backoff.RetryNotify(c.connect(false), c.backoff, notify); err != nil {
		return err
	}

	return nil
}

// Subscribe sends a subscription message for a topic and set of tickers. If no
// tickers are passed, it will subscribe to all tickers for a given topic.
func (c *Client) Subscribe(topic Topic, epics []string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if len(epics) == 0 {
		return errors.New("交易对不能为空")
	}

	subscribe, err := getSub(topic, c.cstToken, c.securityToken, epics...)
	if err != nil {
		return err
	}

	c.subs.add(topic, epics...)

	c.wQueue <- subscribe

	return nil
}

// Unsubscribe sends a message to unsubscribe from a topic and set of tickers. If no
// tickers are passed, it will unsubscribe from all tickers for a given topic.
func (c *Client) Unsubscribe(topic Topic, tickers []string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if len(tickers) == 0 {
		return errors.New("交易对不能为空")
	}

	unsubscribe, err := getSub(topic, c.cstToken, c.securityToken, tickers...)
	if err != nil {
		return err
	}

	c.wQueue <- unsubscribe

	return nil
}

// Output returns the output queue.
func (c *Client) Output() <-chan any {
	return c.output
}

// Error returns an error channel. If the client hits a fatal error (e.g. auth failed),
// it will push an error to this channel and close the connection.
func (c *Client) Error() <-chan error {
	return c.err
}

// Close attempts to gracefully close the connection to the server.
func (c *Client) Close() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.close(false)
}

func newConn(uri string) (*websocket.Conn, error) {
	conn, res, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial server: %w", err)
	} else if res.StatusCode != 101 {
		return nil, errors.New("server failed to switch protocols")
	}

	conn.SetReadLimit(maxMessageSize)
	//if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
	//	return nil, fmt.Errorf("failed to set read deadline: %w", err)
	//}
	//conn.SetPongHandler(func(string) error {
	//	return conn.SetReadDeadline(time.Now().Add(pongWait))
	//})

	return conn, nil
}

func (c *Client) connect(reconnect bool) func() error {
	return func() error {
		// dial the server
		conn, err := newConn(c.url)
		if err != nil {
			return err
		}
		c.conn = conn

		// reset write queue and push auth message
		c.wQueue = make(chan json.RawMessage, 1000)

		// push subscription messages
		c.log.Debugf("connect: CST %s,X-SECURITY-TOKEN %s", c.cstToken, c.securityToken)
		subs := c.subs.get(c.cstToken, c.securityToken)
		for _, msg := range subs {
			c.wQueue <- msg
		}

		// start the threads
		c.rwtomb = tomb.Tomb{}
		c.rwtomb.Go(c.read)
		c.rwtomb.Go(c.write)
		if !reconnect {
			c.ptomb = tomb.Tomb{}
			c.ptomb.Go(c.process)
		}

		return nil
	}
}

func (c *Client) reconnect() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.shouldClose {
		return
	}

	c.log.Debugf("unexpected disconnect: reconnecting")
	c.close(true)

	notify := func(err error, _ time.Duration) {
		c.log.Errorf(err.Error())
	}
	err := backoff.RetryNotify(c.connect(true), c.backoff, notify)
	if err != nil {
		err = fmt.Errorf("error reconnecting: %w: closing connection", err)
		c.log.Errorf(err.Error())
		c.close(false)
		c.err <- err
	}
}

func (c *Client) closeOutput() {
	close(c.output)
	c.log.Debugf("output channel closed")
}

func (c *Client) close(reconnect bool) {
	if c.conn == nil {
		return
	}

	c.rwtomb.Kill(nil)
	if err := c.rwtomb.Wait(); err != nil {
		c.log.Errorf("r/w threads closed: %v", err)
	}

	if !reconnect {
		c.ptomb.Kill(nil)
		if err := c.ptomb.Wait(); err != nil {
			c.log.Errorf("process thread closed: %v", err)
		}
		c.shouldClose = true
		c.closeOutput()
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) read() error {
	defer func() {
		c.log.Debugf("read thread closed")
	}()

	for {
		select {
		case <-c.rwtomb.Dying():
			return nil
		default:
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return nil
				} else if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					return fmt.Errorf("connection closed unexpectedly: %w", err)
				}
				return fmt.Errorf("failed to read message: %w", err)
			}
			c.rQueue <- msg
		}
	}
}

func (c *Client) write() error {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.log.Debugf("write thread closed")
		ticker.Stop()
		go c.reconnect()
	}()

	for {
		select {
		case <-c.rwtomb.Dying():
			if err := c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(writeWait)); err != nil {
				return fmt.Errorf("failed to gracefully close: %w", err)
			}
			return nil
		case <-ticker.C:
			msg, _ := json.Marshal(&models.PING{
				Destination:   "ping",
				CorrelationID: "3",
				Cst:           c.cstToken,
				SecurityToken: c.securityToken,
			})
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return fmt.Errorf("failed to set write deadline: %w", err)
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
		case msg := <-c.wQueue:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return fmt.Errorf("failed to set write deadline: %w", err)
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
		}
	}
}

func (c *Client) process() (err error) {
	defer func() {
		// this client should close if it hits a fatal error (e.g. auth failed)
		c.log.Debugf("process thread closed")
		if err != nil {
			go c.Close()
			c.err <- err
		}
	}()

	for {
		select {
		case <-c.ptomb.Dying():
			return nil
		case data := <-c.rQueue:
			if c.rawData && c.bypassRawDataRouting {
				c.output <- data // push raw bytes to output channel
				continue
			}
			var msgs []json.RawMessage
			msgs = append(msgs, data)
			if err := c.route(msgs); err != nil {
				return err
			}
		}
	}
}

func (c *Client) route(msgs []json.RawMessage) error {
	for _, msg := range msgs {
		var status models.Status
		err := json.Unmarshal(msg, &status)
		if err != nil {
			c.log.Errorf("failed to process message: %v", err)
			continue
		}
		if status.Status != "OK" {
			if strings.Contains(string(msg), "invalid.session.token") {
				// create new session
				client := rest.New(c.apiKey, c.identifier, c.password)
				c.cstToken = client.CstToken
				c.securityToken = client.SecurityToken
				c.log.Debugf("Create New Session: CST %s, X-SECURITY-TOKEN %s", c.cstToken, c.securityToken)
			}
			c.log.Errorf("error message:%s", string(msg))
			continue
		}

		var ev models.Destination
		err = json.Unmarshal(msg, &ev)
		if err != nil {
			c.log.Errorf("failed to process message: %v", err)
			continue
		}

		switch ev.Destination {
		case "OHLCMarketData.subscribe":
			if err := c.handleStatus(msg); err != nil {
				return err
			}
		case "ping":
			c.log.Infof("ping success")
		default:
			c.handleData(ev.Destination, msg)
		}
	}

	return nil
}

func (c *Client) handleStatus(msg json.RawMessage) error {
	var cm models.Status
	if err := json.Unmarshal(msg, &cm); err != nil {
		c.log.Errorf("failed to unmarshal message: %v", err)
		return nil
	}

	switch cm.Status {
	case "OK":
		c.log.Debugf("subscribe successful")
	default:
		c.log.Infof("unknown status message '%v': %v", sanitize(cm.Status))
	}

	return nil
}

func (c *Client) handleData(eventType string, msg json.RawMessage) {
	if c.rawData {
		c.output <- msg // push raw JSON to output channel
		return
	}

	switch eventType {
	case "ohlc.event":
		var out models.OHLC
		if err := json.Unmarshal(msg, &out); err != nil {
			c.log.Errorf("failed to unmarshal message: %v", err)
			return
		}
		c.output <- out
	default:
		c.log.Infof("unknown message type '%s'", sanitize(eventType))
	}
}

func sanitize(s string) string {
	return strings.Replace(s, "\n", "", -1)
}
