// Package usdtm provides USDⓈ-M perpetual contract endpoints for the Phemex API.
package usdtm

import (
	"context"

	"github.com/tigusigalpa/phemex-go"
)

// Client exposes USDⓈ-M perpetual contract methods.
type Client struct {
	raw *phemex.Client
}

// NewClient creates a USDⓈ-M client.
func NewClient(c *phemex.Client) *Client {
	return &Client{raw: c}
}

// CreateOrder places a new USDⓈ-M order.
//
// See https://phemex-docs.github.io/#place-order-http-put-prefered-2
func (c *Client) CreateOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/g-orders/create", params)
}

// AmendOrder amends an existing USDⓈ-M order.
//
// See https://phemex-docs.github.io/#amend-order-by-orderid
func (c *Client) AmendOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/g-orders/replace", params)
}

// CancelOrder cancels a single USDⓈ-M order.
//
// See https://phemex-docs.github.io/#cancel-single-order-by-orderid
func (c *Client) CancelOrder(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/g-orders/cancel", params)
}

// CancelAllOrders cancels all USDⓈ-M orders for a symbol.
//
// See https://phemex-docs.github.io/#cancel-all-orders-2
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "DELETE", "/g-orders/all", map[string]any{"symbol": symbol})
}

// QueryOpenOrders queries open USDⓈ-M orders by symbol.
//
// See https://phemex-docs.github.io/#query-open-orders-by-symbol-2
func (c *Client) QueryOpenOrders(ctx context.Context, symbol string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/g-orders/activeList", map[string]any{"symbol": symbol})
}

// AccountPositions queries account positions for USDⓈ-M contracts.
//
// See https://phemex-docs.github.io/#query-account-positions
func (c *Client) AccountPositions(ctx context.Context) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/g-accounts/accountPositions", nil)
}

// SwitchPosMode switches position mode (One-way / Hedge) for a symbol.
//
// See https://phemex-docs.github.io/#switch-position-mode-synchronously
func (c *Client) SwitchPosMode(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/g-positions/switch-pos-mode-sync", params)
}

// SetLeverage sets leverage for a symbol.
//
// See https://phemex-docs.github.io/#set-leverage-2
func (c *Client) SetLeverage(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "PUT", "/g-positions/leverage", params)
}

// AssignPositionBalance assigns position balance in isolated margin mode.
//
// See https://phemex-docs.github.io/#assign-position-balance
func (c *Client) AssignPositionBalance(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "POST", "/g-positions/assign", params)
}
