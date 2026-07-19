// Example: asset transfers and wallet operations (requires API credentials).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/phemex-go"
	"github.com/tigusigalpa/phemex-go/assets"
)

func main() {
	ctx := context.Background()
	client := assets.NewClient(phemex.NewClient(phemex.Config{
		APIKey:    os.Getenv("PHEMEX_API_KEY"),
		APISecret: os.Getenv("PHEMEX_API_SECRET"),
		BaseURI:   "https://api.phemex.com",
	}))

	transfer, err := client.Transfer(ctx, map[string]any{
		"currency": "BTC",
		"amount":   "0.01",
		"from":     "spot",
		"to":       "future",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Transfer:", transfer)

	deposit, err := client.DepositAddress(ctx, "BTC")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deposit address:", deposit)
}
