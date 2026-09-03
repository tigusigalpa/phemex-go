package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigusigalpa/phemex-go"
)

func TestClientEndpoints(t *testing.T) {
	type received struct {
		method string
		path   string
		query  map[string][]string
	}
	requests := make(chan received, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- received{method: r.Method, path: r.URL.Path, query: r.URL.Query()}
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()

	client := NewClient(phemex.NewClient(phemex.Config{BaseURI: server.URL}))
	from, to, limit, offset := int64(10), int64(20), int64(30), int64(40)
	tests := []struct {
		name, method, path, queryKey, queryValue string
		call                                     func() error
	}{
		{"products", "GET", "/public/products", "", "", func() error { _, err := client.Products(context.Background()); return err }},
		{"time", "GET", "/public/time", "", "", func() error { _, err := client.Time(context.Background()); return err }},
		{"order book", "GET", "/md/orderbook", "symbol", "BTCUSD", func() error { _, err := client.OrderBook(context.Background(), "BTCUSD"); return err }},
		{"full book", "GET", "/md/fullbook", "symbol", "BTCUSD", func() error { _, err := client.FullBook(context.Background(), "BTCUSD"); return err }},
		{"kline", "GET", "/exchange/public/md/v2/kline", "resolution", "1h", func() error {
			_, err := client.Kline(context.Background(), KlineRequest{Symbol: "BTCUSD", From: &from, To: &to, Limit: &limit})
			return err
		}},
		{"trades", "GET", "/md/trade", "symbol", "BTCUSD", func() error { _, err := client.Trades(context.Background(), "BTCUSD"); return err }},
		{"ticker", "GET", "/md/v3/ticker/24hr", "symbol", "BTCUSD", func() error { _, err := client.Ticker24h(context.Background(), "BTCUSD"); return err }},
		{"all tickers", "GET", "/md/v3/ticker/24hr/all", "", "", func() error { _, err := client.Ticker24hAll(context.Background()); return err }},
		{"funding history", "GET", "/api-data/public/data/funding-rate-history", "offset", "40", func() error {
			_, err := client.FundingRateHistory(context.Background(), FundingRateHistoryRequest{Symbol: "BTCUSD", Start: &from, End: &to, Limit: &limit, Offset: &offset})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatal(err)
			}
			got := <-requests
			if got.method != tt.method || got.path != tt.path {
				t.Fatalf("request = %s %s, want %s %s", got.method, got.path, tt.method, tt.path)
			}
			if tt.queryKey != "" && (len(got.query[tt.queryKey]) == 0 || got.query[tt.queryKey][0] != tt.queryValue) {
				t.Fatalf("query %s = %q, want %q", tt.queryKey, got.query[tt.queryKey], tt.queryValue)
			}
		})
	}
}
