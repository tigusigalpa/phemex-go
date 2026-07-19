// Package market provides public market-data endpoints for the Phemex API.
package market

import (
	"context"

	"github.com/tigusigalpa/phemex-go"
)

// Client exposes public market-data methods.
type Client struct {
	raw *phemex.Client
}

// NewClient creates a market-data client.
func NewClient(c *phemex.Client) *Client {
	return &Client{raw: c}
}

// Products queries available products and their metadata.
//
// See https://phemex-docs.github.io/#query-product-information
func (c *Client) Products(ctx context.Context) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/public/products", nil)
}

// Time queries the Phemex server time.
//
// See https://phemex-docs.github.io/#query-server-time
func (c *Client) Time(ctx context.Context) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/public/time", nil)
}

// OrderBook queries the order book for a symbol.
//
// See https://phemex-docs.github.io/#query-order-book
func (c *Client) OrderBook(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/md/orderbook", map[string]any{"symbol": symbol})
}

// FullBook queries the full order book for a symbol.
//
// See https://phemex-docs.github.io/#query-full-order-book
func (c *Client) FullBook(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/md/fullbook", map[string]any{"symbol": symbol})
}

// KlineRequest groups the parameters for querying kline/candlestick data.
type KlineRequest struct {
	Symbol     string `json:"symbol"`
	Resolution string `json:"resolution"`
	From       *int64 `json:"from,omitempty"`
	To         *int64 `json:"to,omitempty"`
	Limit      *int64 `json:"limit,omitempty"`
}

// Kline queries kline/candlestick data for a symbol and resolution.
//
// See https://phemex-docs.github.io/#query-kline
func (c *Client) Kline(ctx context.Context, req KlineRequest) (*phemex.Response, error) {
	if req.Resolution == "" {
		req.Resolution = "1h"
	}
	params := map[string]any{
		"symbol":     req.Symbol,
		"resolution": req.Resolution,
	}
	if req.From != nil {
		params["from"] = *req.From
	}
	if req.To != nil {
		params["to"] = *req.To
	}
	if req.Limit != nil {
		params["limit"] = *req.Limit
	}
	return c.raw.Do(ctx, "GET", "/exchange/public/md/v2/kline", params)
}

// Trades queries recent trades for a symbol.
//
// See https://phemex-docs.github.io/#query-recent-trades
func (c *Client) Trades(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/md/trade", map[string]any{"symbol": symbol})
}

// Ticker24h queries the 24-hour ticker for a symbol.
//
// See https://phemex-docs.github.io/#query-24-hours-ticker
func (c *Client) Ticker24h(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/md/v3/ticker/24hr", map[string]any{"symbol": symbol})
}

// Ticker24hAll queries the 24-hour ticker for all symbols.
//
// See https://phemex-docs.github.io/#query-24-hours-ticker-for-all-symbols
func (c *Client) Ticker24hAll(ctx context.Context) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/md/v3/ticker/24hr/all", nil)
}

// FundingRateHistoryRequest groups the parameters for funding rate history.
type FundingRateHistoryRequest struct {
	Symbol string `json:"symbol"`
	Start  *int64 `json:"start,omitempty"`
	End    *int64 `json:"end,omitempty"`
	Limit  *int64 `json:"limit,omitempty"`
	Offset *int64 `json:"offset,omitempty"`
}

// FundingRateHistory queries funding rate history for a symbol.
//
// See https://phemex-docs.github.io/#query-funding-rate-history
func (c *Client) FundingRateHistory(ctx context.Context, req FundingRateHistoryRequest) (*phemex.Response, error) {
	params := map[string]any{"symbol": req.Symbol}
	if req.Start != nil {
		params["start"] = *req.Start
	}
	if req.End != nil {
		params["end"] = *req.End
	}
	if req.Limit != nil {
		params["limit"] = *req.Limit
	}
	if req.Offset != nil {
		params["offset"] = *req.Offset
	}
	return c.raw.Do(ctx, "GET", "/api-data/public/data/funding-rate-history", params)
}
