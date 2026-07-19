// Package coinm provides Coin-M perpetual contract endpoints for the Phemex API.
package coinm

import (
	"context"

	"github.com/tigusigalpa/phemex-go"
)

// Client exposes Coin-M perpetual contract methods.
type Client struct {
	raw *phemex.Client
}

// NewClient creates a Coin-M client.
func NewClient(c *phemex.Client) *Client {
	return &Client{raw: c}
}

// CreateOrder places a new Coin-M order.
//
// See https://phemex-docs.github.io/#place-order-http-put-prefered
func (c *Client) CreateOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/orders/create", params)
}

// AmendOrder amends an existing Coin-M order.
//
// See https://phemex-docs.github.io/#amend-order-by-order-id
func (c *Client) AmendOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/orders/replace", params)
}

// CancelOrder cancels a Coin-M order by order ID or client order ID.
//
// See https://phemex-docs.github.io/#cancel-order-by-order-id-or-client-order-id
func (c *Client) CancelOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/orders/cancel", params)
}

// CancelAllOrders cancels all Coin-M orders for a symbol.
//
// See https://phemex-docs.github.io/#cancel-all-orders
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/orders/all", map[string]any{"symbol": symbol})
}

// QueryOpenOrders queries open Coin-M orders by symbol.
//
// See https://phemex-docs.github.io/#query-open-orders-by-symbol
func (c *Client) QueryOpenOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/orders/activeList", map[string]any{"symbol": symbol})
}

// AccountPositions queries trading account and positions for Coin-M contracts.
//
// See https://phemex-docs.github.io/#query-trading-account-and-positions
func (c *Client) AccountPositions(ctx context.Context) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/accounts/accountPositions", nil)
}

// SetLeverage sets leverage for a Coin-M symbol.
//
// See https://phemex-docs.github.io/#set-leverage
func (c *Client) SetLeverage(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/positions/leverage", params)
}

// AssignPositionBalance assigns position balance in isolated margin mode.
//
// See https://phemex-docs.github.io/#assign-position-balance-in-isolated-marign-mode
func (c *Client) AssignPositionBalance(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "POST", "/positions/assign", params)
}
