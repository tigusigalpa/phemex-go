// Example: margin trading endpoints (requires API credentials).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/phemex-go"
	"github.com/tigusigalpa/phemex-go/margin"
)

func main() {
	ctx := context.Background()
	client := margin.NewClient(phemex.NewClient(phemex.Config{
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
	fmt.Println("Create margin order:", order)

	history, err := client.BorrowHistory(ctx, map[string]any{
		"currency": "USDT",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Borrow history:", history)
}
