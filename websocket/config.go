package capitalws

import (
	"errors"
)

// Config is a set of WebSocket client options.
type Config struct {
	// APIKey is the API key used to authenticate against the server.
	APIKey string

	// MaxRetries is the maximum number of retry attempts that will occur. If the maximum
	// is reached, the client will close the connection. Omitting this will cause the
	// client to reconnect indefinitely until the user closes it.
	MaxRetries *uint64

	// RawData is a flag indicating whether data should be returned as a raw JSON or raw bytes. If BypassRawDataRouting is unset
	// then the data will be returned as raw JSON, otherwise it will be raw bytes.
	RawData bool

	// BypassRawDataRouting is a flag that interacts with the RawData flag. If RawData flag is unset then this flag is ignored.
	// If RawData is `true`, then this flag indicates whether the raw data should be parsed as json.RawMessage
	// and routed via the client's internal logic (`BypassRawDataRouting=false`), or returned to the application code as []byte (`BypassRawDataRouting=true`).
	// If this flag is `true`, it's up to the caller to handle all message types including auth and subscription responses.
	BypassRawDataRouting bool

	// Log is an optional logger. Any logger implementation can be used as long as it
	// implements the basic Logger interface. Omitting this will disable client logging.
	Log Logger
}

func (c *Config) validate() error {
	if c.APIKey == "" {
		return errors.New("API key is required")
	}

	if c.Log == nil {
		c.Log = &nopLogger{}
	}

	return nil
}

type Topic uint8

const (
	MarketSubscribe   Topic = 10
	MarketUnsubscribe Topic = 11
	OHLCSubscribe     Topic = 12
	OHLCUnsubscribe   Topic = 13
	PING              Topic = 14
)

func (t Topic) Destination() string {
	switch t {
	case MarketSubscribe:
		return "marketData.subscribe"
	case MarketUnsubscribe:
		return "marketData.unsubscribe"
	case OHLCSubscribe:
		return "OHLCMarketData.subscribe"
	case OHLCUnsubscribe:
		return "OHLCMarketData.unsubscribe"
	case PING:
		return "ping"
	}
	return ""
}

func (t Topic) CorrelationID() string {
	switch t {
	case MarketSubscribe:
		return "1"
	case MarketUnsubscribe:
		return "1"
	case OHLCSubscribe:
		return "2"
	case OHLCUnsubscribe:
		return "2"
	case PING:
		return "3"
	}
	return ""
}

// Logger is a basic logger interface used for logging within the client.
type Logger interface {
	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
}

type nopLogger struct{}

func (l *nopLogger) Debugf(template string, args ...any) {}
func (l *nopLogger) Infof(template string, args ...any)  {}
func (l *nopLogger) Errorf(template string, args ...any) {}
