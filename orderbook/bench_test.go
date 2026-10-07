package orderbook

import (
	"math/rand"
	"testing"
)

// BenchmarkLimitResting adds orders on both sides of a 200-tick-wide spread without crossing.
func BenchmarkLimitResting(b *testing.B) {
	book := NewBook()
	r := rand.New(rand.NewSource(1))
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			book.Limit(Buy, 1000-int64(r.Intn(100)), 1)
		} else {
			book.Limit(Sell, 1001+int64(r.Intn(100)), 1)
		}
	}
}

// BenchmarkMatch rests one sell and takes it with a market buy, over 50 price levels.
func BenchmarkMatch(b *testing.B) {
	book := NewBook()
	for i := 0; i < b.N; i++ {
		book.Limit(Sell, 1000+int64(i%50), 10)
		book.Market(Buy, 10)
	}
}

// BenchmarkCancel cancels and re-adds orders in a book that holds 10,000 resting orders.
func BenchmarkCancel(b *testing.B) {
	book := NewBook()
	ids := make([]int, 10000)
	for j := range ids {
		ids[j], _, _ = book.Limit(Buy, 1000-int64(j%100), 1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := i % len(ids)
		book.Cancel(ids[k])
		ids[k], _, _ = book.Limit(Buy, 1000-int64(k%100), 1)
	}
}
