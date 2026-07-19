// Example: spot trading endpoints (requires API credentials).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/phemex-go"
	"github.com/tigusigalpa/phemex-go/spot"
)

func main() {
	ctx := context.Background()
	client := spot.NewClient(phemex.NewClient(phemex.Config{
		APIKey:    os.Getenv("PHEMEX_API_KEY"),
		APISecret: os.Getenv("PHEMEX_API_SECRET"),
		BaseURI:   "https://api.phemex.com",
	}))

	order, err := client.CreateOrder(ctx, map[string]any{
		"symbol":   "BTCUSDT",
		"side":     "Buy",
		"ordType":  "Limit",
		"price":    "65000",
		"orderQty": "0.001",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Create order:", order)

	openOrders, err := client.QueryOpenOrders(ctx, "BTCUSDT")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Open orders:", openOrders)

	wallets, err := client.Wallets(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Wallets:", wallets)
}
