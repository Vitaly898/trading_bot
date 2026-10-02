package binance

import "encoding/json"

// ------------------
// ------ HTTP ------
// ------------------

type RequestParams struct {
	Method      string
	Path        string
	QueryParams map[string]string
	BodyParams  map[string]string
}

// ------ /fapi/v1/exchangeInfo ------

type ExchangeInfoResp struct {
	ExchangeFilters []any            `json:"exchangeFilters"`
	RateLimits      []RateLimitsInfo `json:"rateLimits"`
	ServerTime      int64            `json:"serverTime"`
	Assets          []AssetsInfo     `json:"assets"`
	Symbols         []SymbolInfo     `json:"symbols"`
	Timezone        string           `json:"timezone"`
}

type RateLimitsInfo struct {
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
	RateLimitType string `json:"rateLimitType"`
}

type AssetsInfo struct {
	Asset             string  `json:"asset"`
	MarginAvailable   bool    `json:"marginAvailable"`
	AutoAssetExchange *string `json:"autoAssetExchange"`
}

type SymbolInfo struct {
	Symbol                string        `json:"symbol"`
	Pair                  string        `json:"pair"`
	ContractType          string        `json:"contractType"`
	DeliveryDate          int64         `json:"deliveryDate"`
	OnboardDate           int64         `json:"onboardDate"`
	Status                string        `json:"status"`
	MaintMarginPercent    string        `json:"maintMarginPercent"`
	RequiredMarginPercent string        `json:"requiredMarginPercent"`
	BaseAsset             string        `json:"baseAsset"`
	QuoteAsset            string        `json:"quoteAsset"`
	MarginAsset           string        `json:"marginAsset"`
	PricePrecision        int           `json:"pricePrecision"`
	QuantityPrecision     int           `json:"quantityPrecision"`
	BaseAssetPrecision    int           `json:"baseAssetPrecision"`
	QuotePrecision        int           `json:"quotePrecision"`
	UnderlyingType        string        `json:"underlyingType"`
	UnderlyingSubType     []string      `json:"underlyingSubType"`
	SettlePlan            int           `json:"settlePlan"`
	TriggerProtect        string        `json:"triggerProtect"`
	Filters               []FiltersInfo `json:"filters"`
	OrderTypes            []string      `json:"orderTypes"`
	TimeInForce           []string      `json:"timeInForce"`
	LiquidationFee        string        `json:"liquidationFee"`
	MarketTakeBound       string        `json:"marketTakeBound"`
}

type FiltersInfo struct {
	FilterType        string `json:"filterType"`
	MaxPrice          string `json:"maxPrice,omitempty"`
	MinPrice          string `json:"minPrice,omitempty"`
	TickSize          string `json:"tickSize,omitempty"`
	MaxQty            string `json:"maxQty,omitempty"`
	MinQty            string `json:"minQty,omitempty"`
	StepSize          string `json:"stepSize,omitempty"`
	Limit             int    `json:"limit,omitempty"`
	Notional          string `json:"notional,omitempty"`
	MultiplierUp      string `json:"multiplierUp,omitempty"`
	MultiplierDown    string `json:"multiplierDown,omitempty"`
	MultiplierDecimal string `json:"multiplierDecimal,omitempty"`
}

// ------ /fapi/v1/ticker/24hr ------

type DayTickerResp []DayTicker

type DayTicker struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstID            int    `json:"firstId"`
	LastID             int    `json:"lastId"`
	Count              int    `json:"count"`
}

type KlinesResp [][]any

// ------------------
// ------- WS -------
// ------------------

type wsMessage struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type wsResultMsg struct {
	Result json.RawMessage `json:"result"`
	ID     int             `json:"id"`
}

type wsSubscribeParams struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     uint     `json:"id"`
}

type wsKlineMessage struct {
	EventType string  `json:"e"`
	EventTime int64   `json:"E"`
	Symbol    string  `json:"s"`
	KlineData wsKline `json:"k"`
}
type wsKline struct {
	TimeStart    int64  `json:"t"`
	TimeClose    int64  `json:"T"`
	Symbol       string `json:"s"`
	Interval     string `json:"i"`
	FirstTradeID int    `json:"f"`
	LastTradeID  int    `json:"L"`
	Open         string `json:"o"`
	Close        string `json:"c"`
	High         string `json:"h"`
	Low          string `json:"l"`
	Volume       string `json:"v"`
	TradesNumber int    `json:"n"`
	IsClosed     bool   `json:"x"`
	QuoteVolume  string `json:"q"`
	// V            string `json:"V"` // Taker buy base asset volume
	// Q            string `json:"Q"` // Taker buy quote asset volume
	// B            string `json:"B"` // Ignore
}
