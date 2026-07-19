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

func TestClient_SubscribeAndReceiveEvent(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	var captured []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			captured = append(captured, msg...)

			var req map[string]any
			_ = json.Unmarshal(msg, &req)

			if m, ok := req["method"].(string); ok && m == "subscribe" {
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
	defer client.Close()

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

	if !strings.Contains(string(captured), "orderbook") {
		t.Fatalf("expected subscription message to contain orderbook, got %s", captured)
	}
}

func TestClient_AuthSendsSignature(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	var captured []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			captured = append(captured, msg...)
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
	defer client.Close()

	// Wait briefly for auth message to be sent.
	time.Sleep(200 * time.Millisecond)

	if !strings.Contains(string(captured), "user.auth") {
		t.Fatalf("expected auth message, got %s", captured)
	}
	if !strings.Contains(string(captured), "api-key") {
		t.Fatalf("expected auth message to contain api-key, got %s", captured)
	}
}
