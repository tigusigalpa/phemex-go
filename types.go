package phemex

import (
	"encoding/json"
	"fmt"
)

// Response is the raw JSON-decoded response returned by the REST client. Most
// endpoints reply with `{ "code": <int>, "msg": <string>, "data": <mixed> }`,
// while market-data endpoints under `/md` use `{ "error": null, "id": 0,
// "result": <mixed> }`.
type Response struct {
	Raw map[string]json.RawMessage
}

// Code returns the API result code. Most endpoints return 0 for success.
func (r *Response) Code() int {
	var code int
	if r.Raw == nil {
		return 0
	}
	_ = json.Unmarshal(r.Raw["code"], &code)
	return code
}

// Msg returns the human-readable message from the API.
func (r *Response) Msg() string {
	var msg string
	if r.Raw == nil {
		return ""
	}
	_ = json.Unmarshal(r.Raw["msg"], &msg)
	return msg
}

// Data unmarshals the generic `data` field into v.
func (r *Response) Data(v any) error {
	if r.Raw == nil {
		return nil
	}
	raw, ok := r.Raw["data"]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// ErrorMsg returns the error field from market-data responses.
func (r *Response) ErrorMsg() string {
	if r.Raw == nil {
		return ""
	}
	var errMsg string
	_ = json.Unmarshal(r.Raw["error"], &errMsg)
	return errMsg
}

// ID returns the request correlation id from market-data responses.
func (r *Response) ID() int {
	var id int
	if r.Raw == nil {
		return 0
	}
	_ = json.Unmarshal(r.Raw["id"], &id)
	return id
}

// Result unmarshals the market-data `result` field into v.
func (r *Response) Result(v any) error {
	if r.Raw == nil {
		return nil
	}
	raw, ok := r.Raw["result"]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// IsSuccess returns true when the response does not contain an error.
func (r *Response) IsSuccess() bool {
	if r.Raw == nil {
		return true
	}
	if raw, ok := r.Raw["code"]; ok {
		var code int
		_ = json.Unmarshal(raw, &code)
		return code == 0
	}
	if raw, ok := r.Raw["error"]; ok {
		var errMsg string
		_ = json.Unmarshal(raw, &errMsg)
		return errMsg == ""
	}
	return true
}

// String returns a compact JSON representation of the raw response.
func (r *Response) String() string {
	if r.Raw == nil {
		return "{}"
	}
	b, _ := json.Marshal(r.Raw)
	return string(b)
}

// TimeResponse is the response envelope for /public/time.
type TimeResponse struct {
	Timestamp int64 `json:"timestamp"`
}

// FundingRateHistory represents a single funding rate history entry.
type FundingRateHistory struct {
	Symbol      string `json:"symbol"`
	Rate        string `json:"rate"`
	FundingTime int64  `json:"fundingTime"`
}

// OrderBookEntry represents a price/quantity level in an order book.
type OrderBookEntry struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBook represents a Phemex order book response.
type OrderBook struct {
	Asks []OrderBookEntry `json:"asks"`
	Bids []OrderBookEntry `json:"bids"`
}

// Trade represents a single public trade.
type Trade struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

// Ticker24h represents a 24-hour ticker entry.
type Ticker24h struct {
	Symbol    string `json:"symbol"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	Turnover  string `json:"turnover"`
	Timestamp int64  `json:"timestamp"`
}

// Kline represents a single candlestick entry.
type Kline struct {
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	Turnover  string `json:"turnover"`
	Timestamp int64  `json:"timestamp"`
}

// StringJSON is a helper for typed debug output.
func StringJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%%!json(%v)", err)
	}
	return string(b)
}
