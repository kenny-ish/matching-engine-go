package main

import "sort"

type Side int

const (
	Buy Side = iota
	Sell
)

type Order struct {
	ID    int
	Side  Side
	Price int64 // ignored for market orders
	Qty   int64
}

type Trade struct {
	Maker, Taker int
	Price, Qty   int64
}

type level struct {
	price  int64
	orders []*Order
}

type Book struct {
	bids   []*level // best (highest) first
	asks   []*level // best (lowest) first
	nextID int
	index  map[int]Side
}

func NewBook() *Book {
	return &Book{nextID: 1, index: map[int]Side{}}
}

func (b *Book) id() int {
	id := b.nextID
	b.nextID++
	return id
}

func crosses(side Side, limit, price int64, market bool) bool {
	if market {
		return true
	}
	if side == Buy {
		return price <= limit
	}
	return price >= limit
}

func (b *Book) match(o *Order, market bool) []Trade {
	opp := &b.asks
	if o.Side == Sell {
		opp = &b.bids
	}
	var trades []Trade
	for o.Qty > 0 && len(*opp) > 0 {
		lvl := (*opp)[0]
		if !crosses(o.Side, o.Price, lvl.price, market) {
			break
		}
		for o.Qty > 0 && len(lvl.orders) > 0 {
			maker := lvl.orders[0]
			q := min(o.Qty, maker.Qty)
			trades = append(trades, Trade{Maker: maker.ID, Taker: o.ID, Price: lvl.price, Qty: q})
			o.Qty -= q
			maker.Qty -= q
			if maker.Qty == 0 {
				lvl.orders = lvl.orders[1:]
				delete(b.index, maker.ID)
			}
		}
		if len(lvl.orders) == 0 {
			*opp = (*opp)[1:]
		}
	}
	return trades
}

func (b *Book) rest(o *Order) {
	levels := &b.bids
	better := func(a, c int64) bool { return a > c }
	if o.Side == Sell {
		levels = &b.asks
		better = func(a, c int64) bool { return a < c }
	}
	i := sort.Search(len(*levels), func(i int) bool { return !better((*levels)[i].price, o.Price) })
	if i < len(*levels) && (*levels)[i].price == o.Price {
		(*levels)[i].orders = append((*levels)[i].orders, o)
	} else {
		*levels = append(*levels, nil)
		copy((*levels)[i+1:], (*levels)[i:])
		(*levels)[i] = &level{price: o.Price, orders: []*Order{o}}
	}
	b.index[o.ID] = o.Side
}

// Limit submits a limit order and returns its id and any trades.
func (b *Book) Limit(side Side, price, qty int64) (int, []Trade) {
	o := &Order{ID: b.id(), Side: side, Price: price, Qty: qty}
	trades := b.match(o, false)
	if o.Qty > 0 {
		b.rest(o)
	}
	return o.ID, trades
}

// Market executes against the book; any unfilled remainder is dropped.
func (b *Book) Market(side Side, qty int64) (int, []Trade) {
	o := &Order{ID: b.id(), Side: side, Qty: qty}
	return o.ID, b.match(o, true)
}

func (b *Book) Cancel(id int) bool {
	side, ok := b.index[id]
	if !ok {
		return false
	}
	levels := &b.bids
	if side == Sell {
		levels = &b.asks
	}
	for li, lvl := range *levels {
		for oi, o := range lvl.orders {
			if o.ID == id {
				lvl.orders = append(lvl.orders[:oi], lvl.orders[oi+1:]...)
				if len(lvl.orders) == 0 {
					*levels = append((*levels)[:li], (*levels)[li+1:]...)
				}
				delete(b.index, id)
				return true
			}
		}
	}
	return false
}

// Depth returns total quantity per price level, best first.
func (b *Book) Depth(side Side) [][2]int64 {
	levels := b.bids
	if side == Sell {
		levels = b.asks
	}
	out := make([][2]int64, 0, len(levels))
	for _, l := range levels {
		var q int64
		for _, o := range l.orders {
			q += o.Qty
		}
		out = append(out, [2]int64{l.price, q})
	}
	return out
}
