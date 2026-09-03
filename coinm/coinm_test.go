package coinm

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
		{"create", "PUT", "/orders/create", "", func() error { _, err := client.CreateOrder(context.Background(), params); return err }},
		{"amend", "PUT", "/orders/replace", "", func() error { _, err := client.AmendOrder(context.Background(), params); return err }},
		{"cancel", "DELETE", "/orders/cancel", "marker=value", func() error { _, err := client.CancelOrder(context.Background(), params); return err }},
		{"cancel all", "DELETE", "/orders/all", "symbol=BTCUSD", func() error { _, err := client.CancelAllOrders(context.Background(), "BTCUSD"); return err }},
		{"open orders", "GET", "/orders/activeList", "symbol=BTCUSD", func() error { _, err := client.QueryOpenOrders(context.Background(), "BTCUSD"); return err }},
		{"positions", "GET", "/accounts/accountPositions", "", func() error { _, err := client.AccountPositions(context.Background()); return err }},
		{"leverage", "PUT", "/positions/leverage", "", func() error { _, err := client.SetLeverage(context.Background(), params); return err }},
		{"balance", "POST", "/positions/assign", "", func() error { _, err := client.AssignPositionBalance(context.Background(), params); return err }},
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
