package orderbook_test

import (
	"fmt"

	"github.com/kenny-ish/matching-engine-go/orderbook"
)

func ExampleBook_Limit() {
	b := orderbook.NewBook()
	b.Limit(orderbook.Sell, 101, 5)
	b.Limit(orderbook.Sell, 100, 5)
	id, trades, _ := b.Limit(orderbook.Buy, 101, 7)
	for _, t := range trades {
		fmt.Printf("order %d bought %d @ %d from order %d\n", id, t.Qty, t.Price, t.Maker)
	}
	// Output:
	// order 3 bought 5 @ 100 from order 2
	// order 3 bought 2 @ 101 from order 1
}

func ExampleBook_Market() {
	b := orderbook.NewBook()
	b.Limit(orderbook.Buy, 99, 3)
	_, trades, _ := b.Market(orderbook.Sell, 10) // only 3 are bid; the other 7 are discarded
	fmt.Println(trades)
	fmt.Println(b.Depth(orderbook.Buy))
	// Output:
	// [{1 2 99 3}]
	// []
}

func ExampleBook_Cancel() {
	b := orderbook.NewBook()
	id, _, _ := b.Limit(orderbook.Buy, 50, 10)
	fmt.Println(b.Cancel(id), b.Cancel(id))
	// Output: true false
}
