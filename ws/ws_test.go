package ws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestClient_SubscribeAndReceiveEvent(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req map[string]any
			_ = json.Unmarshal(msg, &req)
			requests <- req

			if m, ok := req["method"].(string); ok && strings.HasSuffix(m, ".subscribe") {
				_ = conn.WriteMessage(mt, []byte(`{"method":"subscribe","result":{"status":"success"}}`))
			}
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewClient(Config{BaseURL: url})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Subscribe(ctx, "orderbook", "BTCUSDT"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	select {
	case ev := <-client.Events():
		if ev.Method != "subscribe" {
			t.Fatalf("expected subscribe event, got %q", ev.Method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	select {
	case req := <-requests:
		if req["method"] != "orderbook.subscribe" {
			t.Fatalf("method = %v, want orderbook.subscribe", req["method"])
		}
		params, ok := req["params"].([]any)
		if !ok || len(params) != 1 || params[0] != "BTCUSDT" {
			t.Fatalf("unexpected subscription params: %#v", req["params"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscription request")
	}
}

func TestClient_AuthSendsSignature(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req map[string]any
			_ = json.Unmarshal(msg, &req)
			requests <- req
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewClient(Config{
		BaseURL:   url,
		APIKey:    "api-key",
		APISecret: "api-secret",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = client.Close() }()

	select {
	case req := <-requests:
		if req["method"] != "user.auth" {
			t.Fatalf("method = %v, want user.auth", req["method"])
		}
		params, ok := req["params"].([]any)
		if !ok || len(params) != 4 || params[0] != "API" || params[1] != "api-key" {
			t.Fatalf("unexpected auth params: %#v", req["params"])
		}
		expiry, ok := params[3].(float64)
		if !ok || int64(expiry) <= time.Now().Unix() {
			t.Fatalf("invalid auth expiry: %#v", params[3])
		}
		mac := hmac.New(sha256.New, []byte("api-secret"))
		mac.Write([]byte("api-key" + strconv.FormatInt(int64(expiry), 10)))
		if params[2] != hex.EncodeToString(mac.Sum(nil)) {
			t.Fatalf("invalid auth signature: %v", params[2])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for authentication request")
	}
}

func TestClient_CloseIsIdempotent(t *testing.T) {
	client := NewClient(Config{})
	if err := client.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}
