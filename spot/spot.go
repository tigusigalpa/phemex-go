// Package spot provides spot trading endpoints for the Phemex API.
package spot

import (
	"context"

	"github.com/tigusigalpa/phemex-go"
)

// Client exposes spot trading methods.
type Client struct {
	raw *phemex.Client
}

// NewClient creates a spot trading client.
func NewClient(c *phemex.Client) *Client {
	return &Client{raw: c}
}

// CreateOrder places a new spot order.
//
// See https://phemex-docs.github.io/#place-order-http-put-prefered-3
func (c *Client) CreateOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/spot/orders/create", params)
}

// AmendOrder amends an existing spot order.
//
// See https://phemex-docs.github.io/#amend-order
func (c *Client) AmendOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/spot/orders", params)
}

// CancelOrder cancels a spot order by order ID or client order ID.
//
// See https://phemex-docs.github.io/#cancel-order
func (c *Client) CancelOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/spot/orders", params)
}

// CancelAllOrders cancels all spot orders for a symbol.
//
// See https://phemex-docs.github.io/#cancel-all-order-by-symbol
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/spot/orders/all", map[string]any{"symbol": symbol})
}

// QueryOpenOrder queries a single open spot order by order ID or client order ID.
//
// See https://phemex-docs.github.io/#query-open-order-by-order-id-or-client-order-id
func (c *Client) QueryOpenOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/spot/orders/active", params)
}

// QueryOpenOrders queries all open spot orders for a symbol.
//
// See https://phemex-docs.github.io/#query-all-open-orders-by-symbol
func (c *Client) QueryOpenOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/spot/orders", map[string]any{"symbol": symbol})
}

// Wallets queries spot wallets.
//
// See https://phemex-docs.github.io/#query-wallets
func (c *Client) Wallets(ctx context.Context) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/spot/wallets", nil)
}

// OrderHistory queries spot order history.
//
// See https://phemex-docs.github.io/#query-order-history
func (c *Client) OrderHistory(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/api-data/spots/orders", params)
}
