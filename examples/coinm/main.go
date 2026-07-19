// Example: Coin-M perpetual contract endpoints (requires API credentials).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/phemex-go"
	"github.com/tigusigalpa/phemex-go/coinm"
)

func main() {
	ctx := context.Background()
	client := coinm.NewClient(phemex.NewClient(phemex.Config{
		APIKey:    os.Getenv("PHEMEX_API_KEY"),
		APISecret: os.Getenv("PHEMEX_API_SECRET"),
		BaseURI:   "https://api.phemex.com",
	}))

	order, err := client.CreateOrder(ctx, map[string]any{
		"symbol":   "BTCUSD",
		"side":     "Buy",
		"ordType":  "Limit",
		"price":    "65000",
		"orderQty": "1",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Create order:", order)

	positions, err := client.AccountPositions(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Positions:", positions)

	lev, err := client.SetLeverage(ctx, map[string]any{
		"symbol":   "BTCUSD",
		"leverage": 10,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Leverage set:", lev)
}
