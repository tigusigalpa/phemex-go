// Package assets provides wallet operations and transfer endpoints for the Phemex API.
package assets

import (
	"context"

	"github.com/tigusigalpa/phemex-go"
)

// Client exposes asset and transfer methods.
type Client struct {
	raw *phemex.Client
}

// NewClient creates an assets client.
func NewClient(c *phemex.Client) *Client {
	return &Client{raw: c}
}

// Transfer transfers funds between spot and futures wallets.
//
// See https://phemex-docs.github.io/#transfer-between-spot-and-futures
func (c *Client) Transfer(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "POST", "/assets/transfer", params)
}

// UniversalTransfer performs a universal transfer between wallets within an account.
//
// See https://phemex-docs.github.io/#transfer-between-wallets-within-an-account
func (c *Client) UniversalTransfer(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "POST", "/wallets/account/transfer", params)
}

// DepositAddress queries the deposit address for a currency.
//
// See https://phemex-docs.github.io/#query-deposit-address-information
func (c *Client) DepositAddress(ctx context.Context, currency string) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/exchange/wallets/v2/depositAddress", map[string]any{"currency": currency})
}

// DepositHistory queries deposit history.
//
// See https://phemex-docs.github.io/#query-deposit-history-records
func (c *Client) DepositHistory(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/exchange/wallets/depositList", params)
}

// WithdrawHistory queries withdraw history.
//
// See https://phemex-docs.github.io/#query-withdraw-history-records
func (c *Client) WithdrawHistory(ctx context.Context, params map[string]any) (*phemex.Response, error) {
	return c.raw.Do(ctx, "GET", "/exchange/wallets/withdrawList", params)
}
