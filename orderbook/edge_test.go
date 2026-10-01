package orderbook

import (
	"reflect"
	"testing"
)

func mustLimit(t *testing.T, b *Book, side Side, price, qty int64) (int, []Trade) {
	t.Helper()
	id, trades, err := b.Limit(side, price, qty)
	if err != nil {
		t.Fatalf("Limit(%v, %d, %d): %v", side, price, qty, err)
	}
	return id, trades
}

func TestPartialFillKeepsMakerPriority(t *testing.T) {
	b := NewBook()
	first, _ := mustLimit(t, b, Sell, 100, 10)
	second, _ := mustLimit(t, b, Sell, 100, 10)
	mustLimit(t, b, Buy, 100, 4) // first has 6 left and stays at the front
	taker, trades := mustLimit(t, b, Buy, 100, 8)
	want := []Trade{
		{Maker: first, Taker: taker, Price: 100, Qty: 6},
		{Maker: second, Taker: taker, Price: 100, Qty: 2},
	}
	if !reflect.DeepEqual(trades, want) {
		t.Fatalf("got %v, want %v", trades, want)
	}
	if d := b.Depth(Sell); len(d) != 1 || d[0] != [2]int64{100, 8} {
		t.Fatalf("asks %v", d)
	}
}

func TestTakerSweepsLevelsAndRestsTheRemainder(t *testing.T) {
	b := NewBook()
	mustLimit(t, b, Sell, 101, 3)
	mustLimit(t, b, Sell, 102, 3)
	mustLimit(t, b, Sell, 105, 3)
	id, trades := mustLimit(t, b, Buy, 102, 10)
	if len(trades) != 2 || trades[0].Price != 101 || trades[1].Price != 102 {
		t.Fatalf("trades %v", trades)
	}
	if d := b.Depth(Buy); len(d) != 1 || d[0] != [2]int64{102, 4} {
		t.Fatalf("the unfilled 4 should rest at 102, bids %v", d)
	}
	if d := b.Depth(Sell); len(d) != 1 || d[0] != [2]int64{105, 3} {
		t.Fatalf("asks %v", d)
	}
	if !b.Cancel(id) {
		t.Fatal("the resting remainder should be cancellable")
	}
}

func TestCancelAfterPartialFill(t *testing.T) {
	b := NewBook()
	maker, _ := mustLimit(t, b, Buy, 50, 10)
	if _, trades, _ := b.Market(Sell, 7); len(trades) != 1 || trades[0].Qty != 7 {
		t.Fatalf("trades %v", trades)
	}
	if !b.Cancel(maker) {
		t.Fatal("partially filled order should be cancellable")
	}
	if len(b.Depth(Buy)) != 0 {
		t.Fatal("level should be gone")
	}
	if _, trades, _ := b.Market(Sell, 1); len(trades) != 0 {
		t.Fatal("a cancelled order must not trade")
	}
}

func TestCancelInTheMiddleKeepsFIFO(t *testing.T) {
	b := NewBook()
	a, _ := mustLimit(t, b, Sell, 100, 1)
	mid, _ := mustLimit(t, b, Sell, 100, 1)
	c, _ := mustLimit(t, b, Sell, 100, 1)
	b.Cancel(mid)
	_, trades := mustLimit(t, b, Buy, 100, 2)
	if len(trades) != 2 || trades[0].Maker != a || trades[1].Maker != c {
		t.Fatalf("trades %v", trades)
	}
}

func TestCancelUnknownOrFilledOrder(t *testing.T) {
	b := NewBook()
	if b.Cancel(42) {
		t.Fatal("unknown id")
	}
	id, _ := mustLimit(t, b, Sell, 100, 1)
	mustLimit(t, b, Buy, 100, 1)
	if b.Cancel(id) {
		t.Fatal("a filled order cannot be cancelled")
	}
}

func TestCrossesAtExactlyTheBestPrice(t *testing.T) {
	b := NewBook()
	mustLimit(t, b, Sell, 100, 1)
	if _, trades := mustLimit(t, b, Buy, 99, 1); len(trades) != 0 {
		t.Fatal("99 does not reach an ask at 100")
	}
	if _, trades := mustLimit(t, b, Buy, 100, 1); len(trades) != 1 {
		t.Fatal("a bid at 100 crosses an ask at 100")
	}
}

func TestMarketOnAnEmptyBook(t *testing.T) {
	b := NewBook()
	_, trades, err := b.Market(Buy, 5)
	if err != nil || len(trades) != 0 {
		t.Fatalf("trades %v, err %v", trades, err)
	}
}
