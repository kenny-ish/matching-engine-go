// Command matching-engine is an interactive shell around the orderbook package.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kenny-ish/matching-engine-go/orderbook"
)

func printTrades(ts []orderbook.Trade) {
	for _, t := range ts {
		fmt.Printf("TRADE %d @ %d (maker %d, taker %d)\n", t.Qty, t.Price, t.Maker, t.Taker)
	}
}

func parseSide(s string) (orderbook.Side, bool) {
	switch s {
	case "buy":
		return orderbook.Buy, true
	case "sell":
		return orderbook.Sell, true
	}
	return 0, false
}

func main() {
	b := orderbook.NewBook()
	sc := bufio.NewScanner(os.Stdin)
	fmt.Println("commands: buy P Q | sell P Q | market buy|sell Q | cancel ID | book | quit")
	for fmt.Print("> "); sc.Scan(); fmt.Print("> ") {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		switch {
		case f[0] == "quit":
			return
		case f[0] == "book":
			asks := b.Depth(orderbook.Sell)
			for i := len(asks) - 1; i >= 0; i-- {
				fmt.Printf("  ask %7d x %d\n", asks[i][0], asks[i][1])
			}
			for _, l := range b.Depth(orderbook.Buy) {
				fmt.Printf("  bid %7d x %d\n", l[0], l[1])
			}
		case f[0] == "cancel" && len(f) == 2:
			id, _ := strconv.Atoi(f[1])
			fmt.Println("cancelled:", b.Cancel(id))
		case f[0] == "market" && len(f) == 3:
			side, ok := parseSide(f[1])
			q, err := strconv.ParseInt(f[2], 10, 64)
			if !ok || err != nil {
				fmt.Println("usage: market buy|sell QTY")
				continue
			}
			_, ts, err := b.Market(side, q)
			if err != nil {
				fmt.Println("rejected:", err)
				continue
			}
			printTrades(ts)
		case len(f) == 3:
			side, ok := parseSide(f[0])
			p, err1 := strconv.ParseInt(f[1], 10, 64)
			q, err2 := strconv.ParseInt(f[2], 10, 64)
			if !ok || err1 != nil || err2 != nil {
				fmt.Println("usage: buy|sell PRICE QTY")
				continue
			}
			id, ts, err := b.Limit(side, p, q)
			if err != nil {
				fmt.Println("rejected:", err)
				continue
			}
			printTrades(ts)
			fmt.Println("order id", id)
		default:
			fmt.Println("unknown command")
		}
	}
}
