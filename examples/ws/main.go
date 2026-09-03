// Example: WebSocket market feed.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tigusigalpa/phemex-go/ws"
)

func main() {
	client := ws.NewClient(ws.Config{
		APIKey:    os.Getenv("PHEMEX_API_KEY"),
		APISecret: os.Getenv("PHEMEX_API_SECRET"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Subscribe(ctx, "orderbook", "BTCUSDT"); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case ev, ok := <-client.Events():
			if !ok {
				return
			}
			fmt.Printf("event: %+v\n", ev)
		case <-ctx.Done():
			return
		}
	}
}
