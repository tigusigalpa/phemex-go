package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestClientMultipleSubscriptionsAndUnsubscribe(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	requests := make(chan map[string]any, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var request map[string]any
			if json.Unmarshal(message, &request) == nil {
				requests <- request
			}
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: "ws" + strings.TrimPrefix(server.URL, "http")})
	if err := client.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Subscribe(context.Background(), "trade.subscribe", "BTCUSD", "ETHUSD"); err != nil {
		t.Fatal(err)
	}
	if err := client.Unsubscribe(context.Background(), "trade", "BTCUSD"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"trade.subscribe", "trade.subscribe", "trade.unsubscribe"} {
		select {
		case request := <-requests:
			if request["method"] != want {
				t.Fatalf("method = %v, want %s", request["method"], want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s", want)
		}
	}
	client.subMu.Lock()
	defer client.subMu.Unlock()
	if len(client.subs) != 1 {
		t.Fatalf("tracked subscriptions = %#v", client.subs)
	}
}

func TestClientValidationAndLifecycleBranches(t *testing.T) {
	client := NewClient(Config{BaseURL: "://bad"})
	if NewClient(Config{}).config.baseURL() != defaultBaseURL || NewClient(Config{BaseURL: "ws://example.test"}).config.baseURL() != "ws://example.test" {
		t.Fatal("base URL configuration")
	}
	if err := client.Subscribe(context.Background(), "trade"); err == nil {
		t.Fatal("expected missing symbol error")
	}
	if err := client.Unsubscribe(context.Background(), "trade"); err == nil {
		t.Fatal("expected missing symbol error")
	}
	if err := client.writeJSON(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected not connected error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.writeJSONTo(ctx, &websocket.Conn{}, map[string]any{}); err != context.Canceled {
		t.Fatalf("cancelled write = %v", err)
	}
	if err := client.Connect(context.Background()); err == nil {
		t.Fatal("expected invalid default connection error")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Connect(context.Background()); err == nil {
		t.Fatal("closed client should not connect")
	}

	reconnecting := NewClient(Config{})
	reconnecting.startReconnect()
	if reconnecting.currentConn() != nil {
		t.Fatal("unexpected connection")
	}
	if err := reconnecting.Close(); err != nil {
		t.Fatal(err)
	}
}
