package main

import "testing"

func TestPriceTimePriority(t *testing.T) {
	b := NewBook()
	a1, _ := b.Limit(Sell, 101, 5)
	a2, _ := b.Limit(Sell, 100, 5)
	a3, _ := b.Limit(Sell, 100, 5)
	_, trades := b.Limit(Buy, 101, 12)
	if len(trades) != 3 {
		t.Fatalf("want 3 trades, got %v", trades)
	}
	if trades[0].Maker != a2 || trades[1].Maker != a3 || trades[2].Maker != a1 {
		t.Fatalf("wrong priority: %v", trades)
	}
	if trades[0].Price != 100 || trades[2].Price != 101 || trades[2].Qty != 2 {
		t.Fatalf("wrong prices/qty: %v", trades)
	}
	if d := b.Depth(Sell); len(d) != 1 || d[0] != [2]int64{101, 3} {
		t.Fatalf("remaining asks: %v", d)
	}
}

func TestLimitRestsWhenNotCrossing(t *testing.T) {
	b := NewBook()
	b.Limit(Buy, 99, 1)
	b.Limit(Buy, 98, 1)
	_, trades := b.Limit(Sell, 100, 1)
	if len(trades) != 0 {
		t.Fatal("should not cross")
	}
	if d := b.Depth(Buy); d[0][0] != 99 {
		t.Fatalf("best bid should be 99, got %v", d)
	}
}

func TestMarketDropsRemainder(t *testing.T) {
	b := NewBook()
	b.Limit(Sell, 100, 2)
	_, trades := b.Market(Buy, 5)
	if len(trades) != 1 || trades[0].Qty != 2 {
		t.Fatalf("trades %v", trades)
	}
	if len(b.Depth(Buy)) != 0 {
		t.Fatal("market remainder must not rest")
	}
}

func TestCancel(t *testing.T) {
	b := NewBook()
	id, _ := b.Limit(Buy, 50, 3)
	if !b.Cancel(id) || b.Cancel(id) {
		t.Fatal("cancel should succeed once")
	}
	if len(b.Depth(Buy)) != 0 {
		t.Fatal("level should be removed")
	}
}
