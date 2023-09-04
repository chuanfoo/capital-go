package capital

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chuanfoo/capital-go/websocket/models"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	API_URL = "https://api-capital.backend-capital.com"
	// 获取服务器时间
	SERVER_TIME = "/api/v1/time"
	// Ping维持活跃10分钟
	PING = "/api/v1/ping"
	//创建新的session
	CREATE_NEW_SESSION = "/api/v1/session"
	// 所有市场
	MARKET_NAVIGATION = "/api/v1/marketnavigation"
	// 子市场 %s 为nodeId
	SUB_MARKETS = "/api/v1/marketnavigation/%s?limit=500"
	// 多个市场详情epics为交易对，多个交易对使用英文逗号分隔
	MARKETS_DETAILS        = "/api/v1/markets?epics=%s"
	MARKETS_DETAILS_SEARCH = "/api/v1/markets?searchTerm=%s"
	// 单一市场行情，%s为交易对
	SINGLE_MARKET_DETAILS = "/api/v1/markets/%s"
	// 历史价格 https://open-api.capital.com/#tag/Markets-Info-greater-Prices
	// /api/v1/prices/{{epic}}?resolution=MINUTE&max=10&from=2022-02-24T00:00:00&to=2022-02-24T01:00:00
	HISTORICAL_PRICES = "/api/v1/prices/%s?resolution=%s&max=%s&from=%s&to=%s"
)

type PriceNotFound error

//------------------------------------------------------

type Node struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Nodes struct {
	Nodes []Node
}

//------------------------------------------------------

type Market struct {
	DelayTime                int     `json:"delayTime"`
	Epic                     string  `json:"epic"`
	NetChange                float64 `json:"netChange"`
	LotSize                  int     `json:"lotSize"`
	Expiry                   string  `json:"expiry"`
	InstrumentType           string  `json:"instrumentType"`
	InstrumentName           string  `json:"instrumentName"`
	High                     float64 `json:"high"`
	Low                      float64 `json:"low"`
	PercentageChange         float64 `json:"percentageChange"`
	UpdateTime               string  `json:"updateTime"`
	UpdateTimeUTC            string  `json:"updateTimeUTC"`
	Bid                      float64 `json:"bid"`
	Offer                    float64 `json:"offer"`
	StreamingPricesAvailable bool    `json:"streamingPricesAvailable"`
	MarketStatus             string  `json:"marketStatus"`
	ScalingFactor            int     `json:"scalingFactor"`
}

type Markets struct {
	Markets []Market
}

// ------------------------------------------------------
type Price struct {
	SnapshotTime     string     `json:"snapshotTime"`
	SnapshotTimeUTC  string     `json:"snapshotTimeUTC"`
	OpenPrice        OpenPrice  `json:"openPrice"`
	ClosePrice       ClosePrice `json:"closePrice"`
	HighPrice        HighPrice  `json:"highPrice"`
	LowPrice         LowPrice   `json:"lowPrice"`
	LastTradedVolume int        `json:"lastTradedVolume"`
}
type OpenPrice struct {
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}
type ClosePrice struct {
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}
type HighPrice struct {
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}
type LowPrice struct {
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}

type Prices struct {
	Prices []Price
}

// ------------------------------------------------------
type Client struct {
	ApiKey        string
	Identifier    string
	Password      string
	CstToken      string
	SecurityToken string
	client        *resty.Client
	headers       map[string]string
	log           Logger
}

func New(apiKey, identifier, password string, log Logger) *Client {
	return &Client{
		ApiKey:     apiKey,
		Identifier: identifier,
		Password:   password,
		client:     resty.New(),
		headers:    map[string]string{"X-CAP-API-KEY": apiKey},
		log:        log,
	}
}

func (c *Client) ping() (err error) {
	resp, err := c.client.R().
		SetHeaders(c.headers).
		Get(API_URL + PING)
	if err != nil {
		return
	}
	if resp.StatusCode() != 200 {
		c.CstToken = ""
		err = c.CreateNewSession()
	}
	return
}

// 只需要运行一次
func (c *Client) CreateNewSession() (err error) {
	url := API_URL + CREATE_NEW_SESSION
	if c.CstToken != "" {
		return
	}
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetBody(map[string]interface{}{"identifier": c.Identifier, "password": c.Password}).
		Post(url)
	if err != nil {
		return
	}
	if resp.StatusCode() == 200 {
		c.CstToken = resp.Header().Get("CST")
		c.SecurityToken = resp.Header().Get("X-SECURITY-TOKEN")
		c.headers["CST"] = c.CstToken
		c.headers["X-SECURITY-TOKEN"] = c.SecurityToken
		// 每隔9分钟刷新一次
		go func() {
			time.Sleep(9 * time.Minute)
			_ = c.ping()
		}()
	} else {
		c.log.Errorf("URL: %s", url)
		err = errors.New(resp.Status())
	}
	return
}

// 顶级市场分类
func (c *Client) MarketNavigation() ([]Node, error) {
	var nodes Nodes
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&nodes).
		Get(API_URL + MARKET_NAVIGATION)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		return nodes.Nodes, nil
	} else {
		return nil, errors.New(resp.Status())
	}
}

// 子市场
func (c *Client) SubMarkets(nodeId string) ([]Node, error) {
	var nodes Nodes
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&nodes).
		Get(API_URL + fmt.Sprintf(SUB_MARKETS, nodeId))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		return nodes.Nodes, nil
	} else {
		return nil, errors.New(resp.Status())
	}
}

// 市场详情
func (c *Client) MarketsDetails(epics string) ([]Market, error) {
	var markets Markets
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&markets).
		Get(API_URL + fmt.Sprintf(MARKETS_DETAILS, epics))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		return markets.Markets, nil
	} else {
		return nil, errors.New(resp.Status())
	}
}

// 市场详情
func (c *Client) MarketsDetailsSearch(searchTerm string) ([]Market, error) {
	var markets Markets
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&markets).
		Get(API_URL + fmt.Sprintf(MARKETS_DETAILS_SEARCH, searchTerm))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		return markets.Markets, nil
	} else {
		return nil, errors.New(resp.Status())
	}
}

// 交易对查询
func (c *Client) EpicsSearch(searchTerm string) string {
	url := API_URL + fmt.Sprintf(MARKETS_DETAILS_SEARCH, searchTerm)
	epics := ""
	var markets Markets
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&markets).
		Get(url)
	if err != nil {
		fmt.Println(err)
	}
	if resp.StatusCode() == 200 {
		for _, m := range markets.Markets {
			epics += fmt.Sprintf("%s,%s,%s\n", m.Epic, m.InstrumentName, m.InstrumentType)
		}
	} else {
		c.log.Errorf("URL: %s", url)
		fmt.Println(resp.Status(), string(resp.Body()))
	}
	return epics
}

// 历史价格
// 1分钟线最多支持跨度1000分钟
// epic 交易对
// resolution 时间间隔 MINUTE, MINUTE_5, MINUTE_15, MINUTE_30, HOUR, HOUR_4, DAY, WEEK
// max 返回结果数量 默认 = 10, 最大 = 1000
// from 开始时间 格式 YYYY-MM-DDTHH:MM:SS (e.g. 2022-04-01T01:01:00)
// to 结束时间  格式 YYYY-MM-DDTHH:MM:SS (e.g. 2022-04-01T01:01:00)
func (c *Client) HistoricalPrices(epic, resolution, max, from, to string) ([]Price, error) {
	url := API_URL + fmt.Sprintf(HISTORICAL_PRICES, epic, resolution, max, from, to)
	var prices Prices
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&prices).
		Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		return prices.Prices, nil
	} else {
		if strings.Contains(string(resp.Body()), "error.prices.not-found") {
			return nil, nil
		}
		c.log.Errorf("URL: %s", url)
		return nil, errors.New(resp.Status() + ":" + string(resp.Body()))
	}
}

// 单一市场详情
func (c *Client) SingleMarketDetails(epic string) (*models.SingleMarketDetails, error) {
	url := API_URL + fmt.Sprintf(SINGLE_MARKET_DETAILS, epic)
	var result interface{}
	resp, err := c.client.R().
		SetHeaders(c.headers).
		SetResult(&result).
		Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 200 {
		bz, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		var detail models.SingleMarketDetails
		err = json.Unmarshal(bz, &detail)
		if err != nil {
			return nil, err
		}
		return &detail, nil
	} else {
		c.log.Errorf("URL: %s", url)
		return nil, errors.New(resp.Status() + ":" + string(resp.Body()))
	}
}

type Logger interface {
	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
}

type nopLogger struct{}

func (l *nopLogger) Debugf(template string, args ...any) {}
func (l *nopLogger) Infof(template string, args ...any)  {}
func (l *nopLogger) Errorf(template string, args ...any) {}
