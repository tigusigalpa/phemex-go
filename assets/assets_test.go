package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigusigalpa/phemex-go"
)

func TestClientEndpoints(t *testing.T) {
	type received struct{ method, path, query string }
	requests := make(chan received, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- received{r.Method, r.URL.Path, r.URL.RawQuery}
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()
	client := NewClient(phemex.NewClient(phemex.Config{BaseURI: server.URL}))
	params := map[string]any{"marker": "value"}
	tests := []struct {
		name, method, path, query string
		call                      func() error
	}{
		{"transfer", "POST", "/assets/transfer", "", func() error { _, err := client.Transfer(context.Background(), params); return err }},
		{"universal transfer", "POST", "/wallets/account/transfer", "", func() error { _, err := client.UniversalTransfer(context.Background(), params); return err }},
		{"deposit address", "GET", "/exchange/wallets/v2/depositAddress", "currency=BTC", func() error { _, err := client.DepositAddress(context.Background(), "BTC"); return err }},
		{"deposit history", "GET", "/exchange/wallets/depositList", "marker=value", func() error { _, err := client.DepositHistory(context.Background(), params); return err }},
		{"withdraw history", "GET", "/exchange/wallets/withdrawList", "marker=value", func() error { _, err := client.WithdrawHistory(context.Background(), params); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatal(err)
			}
			got := <-requests
			if got.method != tt.method || got.path != tt.path || got.query != tt.query {
				t.Fatalf("request = %+v, want %s %s?%s", got, tt.method, tt.path, tt.query)
			}
		})
	}
}
