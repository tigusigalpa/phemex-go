# Phemex crypto-exchange Golang SDK

![Phemex Golang SDK](https://i.postimg.cc/Dy6hdCt3/phemex-golang-banner.jpg)

**A friendly, batteries-included Golang SDK for the [Phemex](https://phemex.com/) crypto exchange.**

Trade spot, perpetuals, and margin, stream live market data, and manage your wallets — all with clean, idiomatic Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/phemex-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/phemex-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tigusigalpa/phemex-go)](https://goreportcard.com/report/github.com/tigusigalpa/phemex-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tigusigalpa/phemex-go)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/tigusigalpa/phemex-go/pulls)

## Related Projects and Documentation

- **[phemex-go Wiki](https://github.com/tigusigalpa/phemex-go/wiki)** - Complete guides, API notes, and practical usage examples.
- **[phemex-php](https://github.com/tigusigalpa/phemex-php)** - The companion Phemex SDK for PHP and Laravel projects.

---

## Why phemex-go?

Talking to a crypto exchange should feel boring — in the best possible way. No surprises, no leaked goroutines, no
mystery around request signing. That's exactly what this library is built for.

Under the hood, `phemex-go` takes care of the tedious, error-prone parts so you can focus on your strategy:

- **Contexts everywhere.** Every REST and WebSocket call takes a `context.Context`, so cancellation, deadlines, and
  graceful shutdown just work.
- **Signing that stays out of your way.** Private requests are signed with HMAC SHA256 automatically — you never touch a
  timestamp or a header.
- **A WebSocket client that survives the real world.** Networks drop. Servers hiccup. The client reconnects on its own,
  keeps the connection alive with heartbeats, and hands you events over plain Go channels.
- **Typed by domain.** Market data, spot, USDⓈ-M, Coin-M, margin, and assets each live in their own small, focused
  package.
- **Featherweight.** The only third-party dependency is `gorilla/websocket`. Everything else is the standard library.
- **Actually tested.** Signature generation and the REST layer are covered by unit tests and `httptest` mocks, so you
  can trust it without hitting live endpoints.

## Requirements

- Go **1.21** or newer (the library uses the `any` alias and modern `context` semantics)
- A Phemex account with API credentials for any private (trading/account) endpoints

> Public market-data endpoints work without credentials — great for prototyping and read-only tools.

## Installation

```bash
go get github.com/tigusigalpa/phemex-go
```

Then import the core package plus whichever domain packages you need:

```go
import (
    "github.com/tigusigalpa/phemex-go"
    "github.com/tigusigalpa/phemex-go/market"
)
```

## Quick Start

Here's the shortest path to your first response — fetching the list of tradable products without any credentials:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tigusigalpa/phemex-go"
    "github.com/tigusigalpa/phemex-go/market"
)

func main() {
    ctx := context.Background()

    // The core client handles transport, signing, and retries.
    // Domain clients (like market) wrap it with typed helpers.
    c := market.NewClient(phemex.NewClient(phemex.Config{
        BaseURI: "https://api.phemex.com",
    }))

    products, err := c.Products(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(products)
}
```

## Core Concepts

A little mental model goes a long way. There are two layers you'll interact with:

1. **The core client** (`phemex.NewClient`) — owns the HTTP transport, credentials, signing, retries, and error mapping.
2. **Domain clients** (`market.NewClient`, `spot.NewClient`, …) — thin, typed wrappers around the core client, one per
   API area.

You create the core client once and share it across as many domain clients as you like — it's safe for concurrent use.

```go
core := phemex.NewClient(phemex.Config{
    APIKey:    os.Getenv("PHEMEX_API_KEY"),
    APISecret: os.Getenv("PHEMEX_API_SECRET"),
})

spotClient := spot.NewClient(core)
usdtmClient := usdtm.NewClient(core)
assetsClient := assets.NewClient(core)
```

## Configuration

`phemex.Config` is a plain struct — set only what you need and rely on sensible defaults for the rest.

| Field            | Type            | Default                  | Description                                                                  |
|------------------|-----------------|--------------------------|------------------------------------------------------------------------------|
| `APIKey`         | `string`        | `""`                     | Your Phemex API key. Optional for public endpoints.                          |
| `APISecret`      | `string`        | `""`                     | Your Phemex API secret. Required to sign private requests.                   |
| `BaseURI`        | `string`        | `https://api.phemex.com` | REST base URL. Point this at the testnet when experimenting.                 |
| `HTTPClient`     | `*http.Client`  | internal client          | Bring your own client to control proxies, TLS, or transport pooling.         |
| `Timeout`        | `time.Duration` | `30s`                    | Per-request timeout (used only when you don't supply your own `HTTPClient`). |
| `Retries`        | `int`           | `3`                      | How many times to retry on rate limits and 5xx responses.                    |
| `RetryDelay`     | `time.Duration` | `1s`                     | Base delay for exponential backoff between retries.                          |
| `RequestTracing` | `string`        | `""`                     | Optional trace token attached to signed requests for debugging.              |

```go
client := phemex.NewClient(phemex.Config{
    APIKey:     os.Getenv("PHEMEX_API_KEY"),
    APISecret:  os.Getenv("PHEMEX_API_SECRET"),
    BaseURI:    "https://api.phemex.com",
    Timeout:    15 * time.Second,
    Retries:    5,
    RetryDelay: 500 * time.Millisecond,
})
```

## Authentication

For anything that touches your account — placing orders, reading balances, transferring funds — provide an API key and
secret. Signing happens automatically on every private call, so you never build a signature by hand.

```go
client := phemex.NewClient(phemex.Config{
    APIKey:    os.Getenv("PHEMEX_API_KEY"),
    APISecret: os.Getenv("PHEMEX_API_SECRET"),
    BaseURI:   "https://api.phemex.com",
})
```

> **Keep secrets out of source code.** Prefer environment variables or a secrets manager. Never commit real keys to
> version control.

Behind the scenes, each private request gets three headers:

- `x-phemex-access-token` — your API key
- `x-phemex-request-expiry` — a short-lived expiry timestamp
- `x-phemex-request-signature` — an HMAC SHA256 signature over the path, query, expiry, and body

## REST API

The library mirrors Phemex's structure with one small package per domain. Each exposes typed helpers that return a
`*phemex.Response`.

| Package            | What it covers                                                                                                            |
|--------------------|---------------------------------------------------------------------------------------------------------------------------|
| [`market`](market) | Public market data: products, server time, order book, full book, klines, recent trades, 24h ticker, funding-rate history |
| [`spot`](spot)     | Spot trading: place, amend, cancel, and query orders; wallets; order history                                              |
| [`usdtm`](usdtm)   | USDⓈ-M perpetuals: orders, account positions, leverage, position mode, balance assignment                                 |
| [`coinm`](coinm)   | Coin-M perpetuals: orders, trading account & positions, leverage, balance assignment                                      |
| [`margin`](margin) | Margin trading: orders plus borrow and payback history                                                                    |
| [`assets`](assets) | Wallets & transfers: spot↔futures transfers, universal transfers, deposit addresses, deposit/withdraw history             |

### Working with responses

Every call returns a `*phemex.Response` that wraps the raw JSON envelope. You can check success and unmarshal the
payload into your own structs:

```go
resp, err := marketClient.Time(ctx)
if err != nil {
    log.Fatal(err)
}

if !resp.IsSuccess() {
    log.Fatalf("phemex returned an error: %s", resp.Msg())
}

var t phemex.TimeResponse
if err := resp.Data(&t); err != nil {
    log.Fatal(err)
}

fmt.Println("Server time:", t.Timestamp)
```

For public market-data endpoints under `/md`, use `Result(...)`, `ErrorMsg()`, and `ID()` instead of `Data(...)`,
`Msg()`, and `Code()` — the envelope shape differs slightly, and the response helpers cover both.

### Market data example

```go
import (
    "github.com/tigusigalpa/phemex-go"
    "github.com/tigusigalpa/phemex-go/market"
)

c := market.NewClient(phemex.NewClient(phemex.Config{}))

// Order book snapshot.
book, _ := c.OrderBook(ctx, "BTCUSD")

// Candlesticks with a typed request.
limit := int64(100)
klines, _ := c.Kline(ctx, market.KlineRequest{
    Symbol:     "BTCUSDT",
    Resolution: "1h",
    Limit:      &limit,
})
```

### Spot trading example

```go
import (
    "github.com/tigusigalpa/phemex-go"
    "github.com/tigusigalpa/phemex-go/spot"
)

client := spot.NewClient(phemex.NewClient(phemex.Config{
    APIKey:    os.Getenv("PHEMEX_API_KEY"),
    APISecret: os.Getenv("PHEMEX_API_SECRET"),
}))

// Place a limit buy order.
resp, err := client.CreateOrder(ctx, map[string]any{
    "symbol":   "BTCUSDT",
    "side":     "Buy",
    "ordType":  "Limit",
    "price":    "65000",
    "orderQty": "0.001",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("Order placed:", resp)

// Review what's still open.
open, _ := client.QueryOpenOrders(ctx, "BTCUSDT")
fmt.Println("Open orders:", open)
```

## WebSocket API

The `ws` package gives you a resilient, real-time feed. It reconnects automatically after drops, sends periodic
heartbeats to keep the socket healthy, replays your subscriptions on reconnect, and delivers every message over a Go
channel.

```go
import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/tigusigalpa/phemex-go/ws"
)

client := ws.NewClient(ws.Config{
    APIKey:    os.Getenv("PHEMEX_API_KEY"),    // optional — only needed for private channels
    APISecret: os.Getenv("PHEMEX_API_SECRET"),
})

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer client.Close()

// Subscribe to the order book for a symbol.
if err := client.Subscribe(ctx, "orderbook", "BTCUSDT"); err != nil {
    log.Fatal(err)
}

// Consume events as they arrive.
for ev := range client.Events() {
    fmt.Printf("%+v\n", ev)
}
```

### Supported channels

| Channel     | Description                                                  |
|-------------|--------------------------------------------------------------|
| `orderbook` | Level 2 order book updates                                   |
| `trade`     | Real-time public trades                                      |
| `kline`     | Streaming candlesticks                                       |
| `aop`       | Account / Order / Position updates (requires authentication) |

When you provide credentials, the client authenticates with a `user.auth` message before your subscriptions go out, so
private channels like `aop` are ready to use.

## Error Handling

Errors are typed, so you can react to specific conditions with a simple type switch instead of parsing strings:

```go
resp, err := client.CreateOrder(ctx, params)
if err != nil {
    switch e := err.(type) {
    case *phemex.RateLimitError:
        log.Printf("slow down — retry after %v", e.RetryAfter)
    case *phemex.AuthenticationError:
        log.Fatal("check your API key and secret")
    case *phemex.ValidationError:
        log.Printf("the request was rejected: %s", e.Message)
    case *phemex.NotFoundError:
        log.Printf("resource not found: %s", e.Message)
    case *phemex.APIError:
        log.Printf("phemex error %d: %s", e.StatusCode, e.Message)
    default:
        log.Printf("unexpected error: %v", err)
    }
}
```

Rate limits (`429`) and server errors (`5xx`) are retried automatically with exponential backoff, honoring the
`Retry-After` header when present. Only after retries are exhausted does the error reach you.

## Examples

Every API group ships with a runnable example under [`examples/`](examples):

```bash
go run ./examples/market   # public market data (no credentials needed)
go run ./examples/spot     # spot trading
go run ./examples/usdtm    # USDⓈ-M perpetuals
go run ./examples/coinm    # Coin-M perpetuals
go run ./examples/margin   # margin trading
go run ./examples/assets   # transfers & wallets
go run ./examples/ws       # live WebSocket feed
```

The trading examples read `PHEMEX_API_KEY` and `PHEMEX_API_SECRET` from the environment, so export them first:

```bash
export PHEMEX_API_KEY="your-key"
export PHEMEX_API_SECRET="your-secret"
```

## Testing

The whole suite runs offline — no live network calls, no real credentials:

```bash
go test ./...
```

Coverage includes:

- HMAC SHA256 signature correctness (including method case-insensitivity and body signing)
- Signature-header injection for private endpoints, and their absence on public ones
- Query and body serialization edge cases (repeated keys, boolean `true`/`false`)
- HTTP error classification and retry behavior (404, 429, 5xx)
- WebSocket subscribe/receive and `user.auth` handshake, exercised with a local `httptest` server

## Contributing

Contributions are genuinely welcome — bug reports, docs fixes, new endpoints, and ideas all help. A good flow:

1. Open an issue to discuss anything non-trivial before diving in.
2. Fork the repo and create a feature branch.
3. Run `go test ./...`, `go vet ./...`, and `gofmt -w .` before pushing.
4. Send a pull request with a clear description of the change.

## Documentation

- [Phemex API Docs](https://phemex-docs.github.io) — the authoritative source for endpoints and parameters
- [Package reference on pkg.go.dev](https://pkg.go.dev/github.com/tigusigalpa/phemex-go)
- [phemex-go Wiki](https://github.com/tigusigalpa/phemex-go/wiki) - guides, API notes, and practical examples
- [phemex-php](https://github.com/tigusigalpa/phemex-php) - the companion SDK for PHP and Laravel projects

## Disclaimer

Trading cryptocurrencies carries significant risk. This library is provided as-is, without any warranty. You are solely
responsible for your API keys, your orders, and your funds. Test thoroughly — ideally against the Phemex testnet —
before running anything with real money.

## Author

**Igor Sazonov**

- GitHub: [@tigusigalpa](https://github.com/tigusigalpa)
- Email: [sovletig@gmail.com](mailto:sovletig@gmail.com)

## License

Released under the MIT License — see [LICENSE](LICENSE) for the full text. Use it freely, build something great, and a
star on GitHub is always appreciated.
