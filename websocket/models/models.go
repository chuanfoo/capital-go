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

//----------------------------------------- marketData.subscribe  -----------------------------------------

type MarketDataSubscribe struct {
	Status        string                     `json:"status"`
	Destination   string                     `json:"destination"`
	CorrelationID string                     `json:"correlationId"`
	Payload       MarketDataSubscribePayload `json:"payload"`
}
type Subscriptions struct {
	OILCRUDE string `json:"OIL_CRUDE"`
}
type MarketDataSubscribePayload struct {
	Subscriptions Subscriptions `json:"subscriptions"`
}

//----------------------------------------- MarketData -----------------------------------------

type MarketData struct {
	Status      string          `json:"status"`
	Destination string          `json:"destination"`
	Payload     PayloadOfMarket `json:"payload"`
}
type PayloadOfMarket struct {
	Epic      string  `json:"epic"`
	Product   string  `json:"product"`
	Bid       float64 `json:"bid"`
	BidQty    float64 `json:"bidQty"`
	Ofr       float64 `json:"ofr"`
	OfrQty    float64 `json:"ofrQty"`
	Timestamp int64   `json:"timestamp"`
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

type SingleMarketDetails struct {
	Instrument struct {
		Epic                     string `json:"epic"`
		Expiry                   string `json:"expiry"`
		Name                     string `json:"name"`
		LotSize                  int    `json:"lotSize"`
		Type                     string `json:"type"`
		ControlledRiskAllowed    bool   `json:"controlledRiskAllowed"`
		StreamingPricesAvailable bool   `json:"streamingPricesAvailable"`
		Currency                 string `json:"currency"`
		MarginFactor             int    `json:"marginFactor"`
		MarginFactorUnit         string `json:"marginFactorUnit"`
		OpeningHours             struct {
			Mon  []string      `json:"mon"`
			Tue  []string      `json:"tue"`
			Wed  []string      `json:"wed"`
			Thu  []string      `json:"thu"`
			Fri  []string      `json:"fri"`
			Sat  []interface{} `json:"sat"`
			Sun  []string      `json:"sun"`
			Zone string        `json:"zone"`
		} `json:"openingHours"`
		Country string `json:"country"`
	} `json:"instrument"`
	DealingRules struct {
		MinStepDistance struct {
			Unit  string  `json:"unit"`
			Value float64 `json:"value"`
		} `json:"minStepDistance"`
		MinDealSize struct {
			Unit  string  `json:"unit"`
			Value float64 `json:"value"`
		} `json:"minDealSize"`
		MinControlledRiskStopDistance struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
		} `json:"minControlledRiskStopDistance"`
		MinNormalStopOrLimitDistance struct {
			Unit  string  `json:"unit"`
			Value float64 `json:"value"`
		} `json:"minNormalStopOrLimitDistance"`
		MaxStopOrLimitDistance struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
		} `json:"maxStopOrLimitDistance"`
		MarketOrderPreference   string `json:"marketOrderPreference"`
		TrailingStopsPreference string `json:"trailingStopsPreference"`
	} `json:"dealingRules"`
	Snapshot struct {
		MarketStatus        string  `json:"marketStatus"`
		NetChange           float64 `json:"netChange"`
		PercentageChange    float64 `json:"percentageChange"`
		UpdateTime          string  `json:"updateTime"`
		DelayTime           int     `json:"delayTime"`
		Bid                 float64 `json:"bid"`
		Offer               float64 `json:"offer"`
		High                float64 `json:"high"`
		Low                 float64 `json:"low"`
		DecimalPlacesFactor int     `json:"decimalPlacesFactor"`
		ScalingFactor       int     `json:"scalingFactor"`
	} `json:"snapshot"`
}