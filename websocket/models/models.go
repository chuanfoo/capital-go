package models

//----------------------------------------- OHLCMarketData.subscribe -----------------------------------------
type OHLCMarketDataSubscribe struct {
	Destination   string  `json:"destination"`
	CorrelationID string  `json:"correlationId"`
	Cst           string  `json:"cst"`
	SecurityToken string  `json:"securityToken"`
	Payload       Payload `json:"payload"`
}
type Payload struct {
	Epics       []string `json:"epics"`
	Resolutions []string `json:"resolutions"`
	Type        string   `json:"type"`
}

type Status struct {
	Status string `json:"status"`
}

type Destination struct {
	Destination string `json:"destination"`
}

//----------------------------------------- OHLCMarketData -----------------------------------------

type OHLC struct {
	Status      string      `json:"status"`
	Destination string      `json:"destination"`
	Payload     OHLCPayload `json:"payload"`
}
type OHLCPayload struct {
	Resolution string  `json:"resolution"`
	Epic       string  `json:"epic"`
	Type       string  `json:"type"`
	PriceType  string  `json:"priceType"`
	T          int64   `json:"t"`
	H          float64 `json:"h"`
	L          float64 `json:"l"`
	O          float64 `json:"o"`
	C          float64 `json:"c"`
}

//----------------------------------------- OHLCMarketData -----------------------------------------
type PING struct {
	Destination   string `json:"destination"`
	CorrelationID string `json:"correlationId"`
	Cst           string `json:"cst"`
	SecurityToken string `json:"securityToken"`
}