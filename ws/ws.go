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
	"strings"
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

	conn    *websocket.Conn
	connMu  sync.RWMutex
	writeMu sync.Mutex
	events  chan Event
	done    chan struct{}
	wg      sync.WaitGroup

	subMu sync.Mutex
	subs  []subscription

	stateMu      sync.Mutex
	started      bool
	closed       bool
	reconnecting bool
	closeOnce    sync.Once
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
	c.stateMu.Lock()
	if c.closed {
		c.stateMu.Unlock()
		return errors.New("ws: client is closed")
	}
	if c.started {
		c.stateMu.Unlock()
		return errors.New("ws: client is already connected")
	}
	c.started = true
	c.stateMu.Unlock()

	if err := c.dial(ctx); err != nil {
		c.stateMu.Lock()
		c.started = false
		c.stateMu.Unlock()
		return err
	}
	c.stateMu.Lock()
	if c.closed {
		c.stateMu.Unlock()
		return errors.New("ws: client is closed")
	}
	c.wg.Add(2)
	c.stateMu.Unlock()
	go c.readLoop()
	go c.pingLoop()
	return nil
}

// Close terminates the connection and background goroutines.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.stateMu.Lock()
		c.closed = true
		close(c.done)
		c.stateMu.Unlock()

		c.connMu.Lock()
		conn := c.conn
		c.conn = nil
		c.connMu.Unlock()
		if conn != nil {
			c.writeMu.Lock()
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(defaultWriteTimeout))
			c.writeMu.Unlock()
			_ = conn.Close()
		}
		c.wg.Wait()
		close(c.events)
	})
	return nil
}

// Subscribe sends a subscription request for one or more symbols on a channel.
func (c *Client) Subscribe(ctx context.Context, channel string, symbols ...string) error {
	if len(symbols) == 0 {
		return errors.New("ws: at least one symbol is required")
	}
	for _, symbol := range symbols {
		req := subscription{
			ID:     int(time.Now().UnixNano()),
			Method: subscriptionMethod(channel, "subscribe"),
			Params: []string{symbol},
		}
		if err := c.writeJSON(ctx, req); err != nil {
			return err
		}
		c.trackSubscription(req)
	}
	return nil
}

// Unsubscribe removes a previously created subscription.
func (c *Client) Unsubscribe(ctx context.Context, channel string, symbols ...string) error {
	if len(symbols) == 0 {
		return errors.New("ws: at least one symbol is required")
	}
	for _, symbol := range symbols {
		req := subscription{
			ID:     int(time.Now().UnixNano()),
			Method: subscriptionMethod(channel, "unsubscribe"),
			Params: []string{symbol},
		}
		if err := c.writeJSON(ctx, req); err != nil {
			return err
		}
		c.untrackSubscription(subscriptionMethod(channel, "subscribe"), symbol)
	}
	return nil
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
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(defaultReadTimeout))
	})

	if c.config.APIKey != "" && c.config.APISecret != "" {
		if err := c.authenticate(ctx, conn); err != nil {
			_ = conn.Close()
			return err
		}
	}

	c.stateMu.Lock()
	if c.closed {
		c.stateMu.Unlock()
		_ = conn.Close()
		return errors.New("ws: client is closed")
	}
	c.connMu.Lock()
	previous := c.conn
	c.conn = conn
	c.connMu.Unlock()
	c.stateMu.Unlock()
	if previous != nil && previous != conn {
		_ = previous.Close()
	}

	c.resubscribe(ctx)
	return nil
}

func (c *Client) authenticate(ctx context.Context, conn *websocket.Conn) error {
	expiry := time.Now().Unix() + 60
	message := fmt.Sprintf("%s%d", c.config.APIKey, expiry)
	mac := hmac.New(sha256.New, []byte(c.config.APISecret))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	return c.writeJSONTo(ctx, conn, map[string]any{
		"method": "user.auth",
		"params": []any{
			"API",
			c.config.APIKey,
			signature,
			expiry,
		},
		"id": int(time.Now().UnixNano()),
	})
}

func (c *Client) trackSubscription(req subscription) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.subs = append(c.subs, req)
}

func (c *Client) untrackSubscription(method, symbol string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for i := len(c.subs) - 1; i >= 0; i-- {
		params, ok := c.subs[i].Params.([]string)
		if c.subs[i].Method == method && ok && len(params) == 1 && params[0] == symbol {
			c.subs = append(c.subs[:i], c.subs[i+1:]...)
		}
	}
}

func (c *Client) resubscribe(ctx context.Context) {
	c.subMu.Lock()
	subs := append([]subscription(nil), c.subs...)
	c.subMu.Unlock()
	for _, sub := range subs {
		_ = c.writeJSON(ctx, sub)
	}
}

func (c *Client) writeJSON(ctx context.Context, v any) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return errors.New("ws: not connected")
	}
	return c.writeJSONTo(ctx, conn, v)
}

func (c *Client) writeJSONTo(ctx context.Context, conn *websocket.Conn, v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	_ = conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	return conn.WriteJSON(v)
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
			if c.isClosed() {
				return
			}
			if c.currentConn() == conn {
				c.startReconnect()
			}
			select {
			case <-c.done:
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}

		var ev Event
		ev.Raw = msg
		if err := json.Unmarshal(msg, &ev); err != nil {
			continue
		}
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
	defer func() {
		c.stateMu.Lock()
		c.reconnecting = false
		c.stateMu.Unlock()
	}()

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

func (c *Client) startReconnect() {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.closed || c.reconnecting {
		return
	}
	c.reconnecting = true
	go c.reconnect()
}

func (c *Client) isClosed() bool {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.closed
}

func (c *Client) currentConn() *websocket.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

func subscriptionMethod(channel, action string) string {
	channel = strings.TrimSpace(channel)
	channel = strings.TrimSuffix(channel, ".subscribe")
	channel = strings.TrimSuffix(channel, ".unsubscribe")
	return channel + "." + action
}
