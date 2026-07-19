// Example: public market data endpoints.
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
	client := market.NewClient(phemex.NewClient(phemex.Config{
		BaseURI: "https://api.phemex.com",
	}))

	products, err := client.Products(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Products:", products)

	timeResp, err := client.Time(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var timeData phemex.TimeResponse
	if err := timeResp.Data(&timeData); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server time:", timeData.Timestamp)

	book, err := client.OrderBook(ctx, "BTCUSD")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Order book:", book)

	limit := int64(10)
	klines, err := client.Kline(ctx, market.KlineRequest{
		Symbol:     "BTCUSDT",
		Resolution: "1h",
		Limit:      &limit,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Klines:", klines)

	ticker, err := client.Ticker24hAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("24h ticker:", ticker)
}
