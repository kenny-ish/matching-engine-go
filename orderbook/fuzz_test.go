package orderbook

import "testing"

type modelOrder struct{ price, qty int64 }

// FuzzBook drives the book with random operations and checks its invariants after every step.
// Each operation takes three bytes: kind, price offset, quantity.
func FuzzBook(f *testing.F) {
	f.Add([]byte{0, 5, 3, 1, 5, 2, 2, 0, 4, 3, 1, 1, 4, 0, 0})
	f.Add([]byte{1, 9, 9, 0, 1, 9, 2, 3, 3, 3, 0, 0, 0, 7, 7, 4, 1, 0})
	f.Fuzz(func(t *testing.T, ops []byte) {
		b := NewBook()
		live := map[int]modelOrder{} // what should be resting, by id
		var ids []int
		for i := 0; i+2 < len(ops); i += 3 {
			side := Side(ops[i] % 2)
			price := int64(ops[i+1]%16) + 90
			qty := int64(ops[i+2]%8) + 1
			switch ops[i] % 5 {
			case 0, 1:
				id, trades, err := b.Limit(side, price, qty)
				if err != nil {
					t.Fatal(err)
				}
				filled := checkTrades(t, trades, id, side, price, false, live)
				if filled > qty {
					t.Fatalf("limit for %d filled %d", qty, filled)
				}
				if filled < qty {
					live[id] = modelOrder{price, qty - filled}
				}
				ids = append(ids, id)
			case 2, 3:
				id, trades, err := b.Market(side, qty)
				if err != nil {
					t.Fatal(err)
				}
				if filled := checkTrades(t, trades, id, side, 0, true, live); filled > qty {
					t.Fatalf("market for %d filled %d", qty, filled)
				}
			case 4:
				if len(ids) == 0 {
					continue
				}
				id := ids[int(ops[i+1])%len(ids)]
				_, resting := live[id]
				if got := b.Cancel(id); got != resting {
					t.Fatalf("Cancel(%d) = %v, order resting: %v", id, got, resting)
				}
				delete(live, id)
			}
			checkBook(t, b, live)
		}
	})
}

// checkTrades verifies the trades of one incoming order against the model and applies them.
func checkTrades(t *testing.T, trades []Trade, taker int, side Side, limit int64, market bool, live map[int]modelOrder) int64 {
	t.Helper()
	var filled int64
	for _, tr := range trades {
		maker, ok := live[tr.Maker]
		switch {
		case tr.Taker != taker:
			t.Fatalf("trade %+v: taker should be %d", tr, taker)
		case !ok:
			t.Fatalf("trade %+v: maker is not resting", tr)
		case tr.Qty <= 0 || tr.Qty > maker.qty:
			t.Fatalf("trade %+v: maker had %d", tr, maker.qty)
		case tr.Price != maker.price:
			t.Fatalf("trade %+v: must print at the maker's price %d", tr, maker.price)
		case !market && side == Buy && tr.Price > limit, !market && side == Sell && tr.Price < limit:
			t.Fatalf("trade %+v: outside the taker's limit %d", tr, limit)
		}
		if maker.qty == tr.Qty {
			delete(live, tr.Maker)
		} else {
			live[tr.Maker] = modelOrder{maker.price, maker.qty - tr.Qty}
		}
		filled += tr.Qty
	}
	return filled
}

func checkBook(t *testing.T, b *Book, live map[int]modelOrder) {
	t.Helper()
	bids, asks := b.Depth(Buy), b.Depth(Sell)
	want := map[int64]int64{}
	for _, o := range live {
		want[o.price] += o.qty
	}
	got := map[int64]int64{}
	for i, l := range bids {
		if l[1] <= 0 || (i > 0 && l[0] >= bids[i-1][0]) {
			t.Fatalf("bids must be non-empty and strictly descending: %v", bids)
		}
		got[l[0]] += l[1]
	}
	for i, l := range asks {
		if l[1] <= 0 || (i > 0 && l[0] <= asks[i-1][0]) {
			t.Fatalf("asks must be non-empty and strictly ascending: %v", asks)
		}
		got[l[0]] += l[1]
	}
	if len(bids) > 0 && len(asks) > 0 && bids[0][0] >= asks[0][0] {
		t.Fatalf("crossed book: best bid %d, best ask %d", bids[0][0], asks[0][0])
	}
	if len(got) != len(want) {
		t.Fatalf("levels %v, model %v", got, want)
	}
	for price, qty := range want {
		if got[price] != qty {
			t.Fatalf("level %d holds %d, model says %d", price, got[price], qty)
		}
	}
}
