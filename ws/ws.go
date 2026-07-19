// Package ws provides a WebSocket client for the Phemex real-time API.
package ws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultBaseURL      = "wss://ws.phemex.com"
	defaultPingInterval = 30 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultReadTimeout  = 60 * time.Second
	defaultDialTimeout  = 10 * time.Second
	defaultReconnectMin = 1 * time.Second
	defaultReconnectMax = 30 * time.Second
)

// Config holds WebSocket client configuration.
type Config struct {
	APIKey     string
	APISecret  string
	BaseURL    string
	HTTPClient *http.Client
	Dialer     *websocket.Dialer
}

func (c Config) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultBaseURL
}

// Event is a message received from the Phemex WebSocket feed.
type Event struct {
	Method string          `json:"method"`
	ID     int             `json:"id"`
	Params json.RawMessage `json:"params"`
	Error  json.RawMessage `json:"error"`
	Result json.RawMessage `json:"result"`
	Raw    []byte
}

// Client manages a WebSocket connection to Phemex.
type Client struct {
	config Config

	conn   *websocket.Conn
	connMu sync.RWMutex
	events chan Event
	done   chan struct{}
	wg     sync.WaitGroup

	subMu sync.Mutex
	subs  []subscription
}

type subscription struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

// NewClient creates a WebSocket client from the provided configuration.
func NewClient(cfg Config) *Client {
	return &Client{
		config: cfg,
		events: make(chan Event, 256),
		done:   make(chan struct{}),
	}
}

// Events returns the channel of received WebSocket events.
func (c *Client) Events() <-chan Event {
	return c.events
}

// Connect establishes the WebSocket connection and starts background loops.
// It attempts to reconnect automatically on unexpected disconnects.
func (c *Client) Connect(ctx context.Context) error {
	if err := c.dial(ctx); err != nil {
		return err
	}
	c.wg.Add(2)
	go c.readLoop()
	go c.pingLoop()
	return nil
}

// Close terminates the connection and background goroutines.
func (c *Client) Close() error {
	close(c.done)
	c.connMu.Lock()
	if c.conn != nil {
		_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = c.conn.Close()
	}
	c.connMu.Unlock()
	c.wg.Wait()
	close(c.events)
	return nil
}

// Subscribe sends a subscription request for one or more symbols on a channel.
func (c *Client) Subscribe(ctx context.Context, channel string, symbols ...string) error {
	if len(symbols) == 0 {
		return errors.New("ws: at least one symbol is required")
	}
	req := subscription{
		ID:     int(time.Now().UnixNano()),
		Method: "subscribe",
		Params: []string{channel, symbols[0]},
	}
	c.trackSubscription(req)
	return c.writeJSON(ctx, req)
}

// Unsubscribe removes a previously created subscription.
func (c *Client) Unsubscribe(ctx context.Context, channel string, symbols ...string) error {
	if len(symbols) == 0 {
		return errors.New("ws: at least one symbol is required")
	}
	return c.writeJSON(ctx, subscription{
		ID:     int(time.Now().UnixNano()),
		Method: "unsubscribe",
		Params: []string{channel, symbols[0]},
	})
}

func (c *Client) dial(ctx context.Context) error {
	dialer := c.config.Dialer
	if dialer == nil {
		dialer = &websocket.Dialer{
			HandshakeTimeout: defaultDialTimeout,
			Proxy:            http.ProxyFromEnvironment,
		}
	}
	// A custom Dialer can be provided via Config.Dialer if proxy or TLS
	// settings need to be adjusted beyond the defaults.

	conn, _, err := dialer.DialContext(ctx, c.config.baseURL(), nil)
	if err != nil {
		return fmt.Errorf("ws: dial %s: %w", c.config.baseURL(), err)
	}
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	if c.config.APIKey != "" && c.config.APISecret != "" {
		if err := c.authenticate(ctx); err != nil {
			_ = conn.Close()
			return err
		}
	}

	c.resubscribe(ctx)
	return nil
}

func (c *Client) authenticate(ctx context.Context) error {
	timestamp := time.Now().UnixMilli()
	message := fmt.Sprintf("%s%s%d", c.config.APIKey, c.config.APISecret, timestamp)
	mac := hmac.New(sha256.New, []byte(c.config.APISecret))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	return c.writeJSON(ctx, map[string]any{
		"method": "user.auth",
		"params": []any{
			"API",
			c.config.APIKey,
			strconv.FormatInt(timestamp, 10),
			signature,
		},
		"id": int(timestamp),
	})
}

func (c *Client) trackSubscription(req subscription) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.subs = append(c.subs, req)
}

func (c *Client) resubscribe(ctx context.Context) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for _, sub := range c.subs {
		_ = c.writeJSON(ctx, sub)
	}
}

func (c *Client) writeJSON(ctx context.Context, v any) error {
	deadline := time.Now().Add(defaultWriteTimeout)
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return errors.New("ws: not connected")
	}
	_ = conn.SetWriteDeadline(deadline)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return conn.WriteJSON(v)
	}
}

func (c *Client) readLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()
		if conn == nil {
			time.Sleep(defaultReconnectMin)
			continue
		}

		_ = conn.SetReadDeadline(time.Now().Add(defaultReadTimeout))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				go c.reconnect()
			}
			select {
			case <-c.done:
				return
			case <-time.After(defaultReconnectMin):
			}
			continue
		}

		var ev Event
		ev.Raw = msg
		_ = json.Unmarshal(msg, &ev)
		select {
		case c.events <- ev:
		case <-c.done:
			return
		}
	}
}

func (c *Client) pingLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(defaultPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()
			if conn == nil {
				continue
			}
			_ = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(defaultWriteTimeout))
		}
	}
}

func (c *Client) reconnect() {
	backoff := defaultReconnectMin
	for {
		select {
		case <-c.done:
			return
		case <-time.After(backoff):
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
		err := c.dial(ctx)
		cancel()
		if err == nil {
			return
		}
		backoff *= 2
		if backoff > defaultReconnectMax {
			backoff = defaultReconnectMax
		}
	}
}
