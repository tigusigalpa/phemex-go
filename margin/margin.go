// Package margin provides margin trading endpoints for the Phemex API.
package margin

import (
	"context"

	"github.com/tigusigalpa/phemex-go"
)

// Client exposes margin trading methods.
type Client struct {
	raw *phemex.Client
}

// NewClient creates a margin trading client.
func NewClient(c *phemex.Client) *Client {
	return &Client{raw: c}
}

// CreateOrder places a new margin order.
//
// See https://phemex-docs.github.io/#place-order-http-put-prefered-4
func (c *Client) CreateOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/margin-trade/orders/create", params)
}

// CancelOrder cancels a margin order.
//
// See https://phemex-docs.github.io/#cancel-order-2
func (c *Client) CancelOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/margin-trade/orders", params)
}

// CancelAllOrders cancels all margin orders for a symbol.
//
// See https://phemex-docs.github.io/#cancel-all-order-by-symbol-2
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/margin-trade/orders/all", map[string]any{"symbol": symbol})
}

// QueryOpenOrder queries a single open margin order.
//
// See https://phemex-docs.github.io/#query-open-order-by-order-id-or-client-order-id-2
func (c *Client) QueryOpenOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/margin-trade/orders/active", params)
}

// BorrowHistory queries margin borrow history.
//
// See https://phemex-docs.github.io/#query-margin-borrow-history-records
func (c *Client) BorrowHistory(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/margin/borrow", params)
}

// Borrow posts a margin borrow request.
//
// See https://phemex-docs.github.io/#post-margin-borrow-request
func (c *Client) Borrow(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "POST", "/margin/borrow", params)
}

// Payback posts a margin payback request.
//
// See https://phemex-docs.github.io/#post-margin-payback-history
func (c *Client) Payback(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "POST", "/margin/payback", params)
}
