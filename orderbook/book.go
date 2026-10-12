// Package orderbook implements a limit order book with price-time priority matching.
//
// Prices are integer ticks and quantities integer lots, so there is no floating point anywhere.
// A Book is not safe for concurrent use; keep it owned by one goroutine.
package orderbook

import (
	"errors"
	"sort"
)

// Side is the side of an order.
type Side int

// Order sides.
const (
	Buy Side = iota
	Sell
)

func (s Side) String() string {
	switch s {
	case Buy:
		return "buy"
	case Sell:
		return "sell"
	}
	return "invalid"
}

// Errors returned for orders the book refuses. A rejected order does not consume an id.
var (
	ErrInvalidSide  = errors.New("orderbook: side must be Buy or Sell")
	ErrInvalidPrice = errors.New("orderbook: price must be positive")
	ErrInvalidQty   = errors.New("orderbook: quantity must be positive")
)

// Trade is one fill between a resting (maker) order and an incoming (taker) order.
type Trade struct {
	Maker, Taker int
	Price, Qty   int64
}

type order struct {
	id    int
	side  Side
	price int64 // limit price; unused for market orders
	qty   int64 // remaining quantity; 0 marks a cancelled order still sitting in its queue
}

type level struct {
	price  int64
	qty    int64    // live quantity at this price
	dead   int      // cancelled orders not yet dropped from the queue
	orders []*order // FIFO: time priority within the price level
}

type ref struct {
	o   *order
	lvl *level
}

// Book holds the resting limit orders of both sides.
type Book struct {
	bids   []*level // best (highest) first
	asks   []*level // best (lowest) first
	nextID int
	index  map[int]ref // resting order id -> the order and its level
}

// NewBook returns an empty book. Order ids start at 1.
func NewBook() *Book {
	return &Book{nextID: 1, index: map[int]ref{}}
}

func validate(side Side, price, qty int64, market bool) error {
	switch {
	case side != Buy && side != Sell:
		return ErrInvalidSide
	case !market && price <= 0:
		return ErrInvalidPrice
	case qty <= 0:
		return ErrInvalidQty
	}
	return nil
}

func (b *Book) newID() int {
	id := b.nextID
	b.nextID++
	return id
}

// levels returns one side of the book and the "better price" ordering for it.
func (b *Book) levels(side Side) (*[]*level, func(a, c int64) bool) {
	if side == Sell {
		return &b.asks, func(a, c int64) bool { return a < c }
	}
	return &b.bids, func(a, c int64) bool { return a > c }
}

// crosses reports whether an incoming order with the given limit can trade at price.
func crosses(side Side, limit, price int64, market bool) bool {
	if market {
		return true
	}
	if side == Buy {
		return price <= limit
	}
	return price >= limit
}

func (b *Book) match(o *order, market bool) []Trade {
	opposite := Sell
	if o.side == Sell {
		opposite = Buy
	}
	opp, _ := b.levels(opposite)
	var trades []Trade
	for o.qty > 0 && len(*opp) > 0 {
		lvl := (*opp)[0]
		if !crosses(o.side, o.price, lvl.price, market) {
			break
		}
		for o.qty > 0 && len(lvl.orders) > 0 {
			maker := lvl.orders[0]
			if maker.qty == 0 { // cancelled earlier, dropped now
				lvl.orders = lvl.orders[1:]
				lvl.dead--
				continue
			}
			q := min(o.qty, maker.qty)
			trades = append(trades, Trade{Maker: maker.id, Taker: o.id, Price: lvl.price, Qty: q})
			o.qty -= q
			maker.qty -= q
			lvl.qty -= q
			if maker.qty == 0 {
				lvl.orders = lvl.orders[1:]
				delete(b.index, maker.id)
			}
		}
		if lvl.qty == 0 {
			*opp = (*opp)[1:]
		}
	}
	return trades
}

func (b *Book) rest(o *order) {
	levels, better := b.levels(o.side)
	i := sort.Search(len(*levels), func(i int) bool { return !better((*levels)[i].price, o.price) })
	var lvl *level
	if i < len(*levels) && (*levels)[i].price == o.price {
		lvl = (*levels)[i]
	} else {
		lvl = &level{price: o.price}
		*levels = append(*levels, nil)
		copy((*levels)[i+1:], (*levels)[i:])
		(*levels)[i] = lvl
	}
	lvl.orders = append(lvl.orders, o)
	lvl.qty += o.qty
	b.index[o.id] = ref{o, lvl}
}

func (b *Book) removeLevel(side Side, lvl *level) {
	levels, better := b.levels(side)
	i := sort.Search(len(*levels), func(i int) bool { return !better((*levels)[i].price, lvl.price) })
	if i < len(*levels) && (*levels)[i] == lvl {
		*levels = append((*levels)[:i], (*levels)[i+1:]...)
	}
}

// compact drops cancelled orders once they make up more than half of the queue, so a level that
// sees many cancels without trading does not grow without bound.
func (l *level) compact() {
	live := l.orders[:0]
	for _, o := range l.orders {
		if o.qty > 0 {
			live = append(live, o)
		}
	}
	clear(l.orders[len(live):])
	l.orders = live
	l.dead = 0
}

// Limit submits a limit order and returns its id and the trades it caused. Whatever is not
// filled immediately rests on the book at the limit price.
func (b *Book) Limit(side Side, price, qty int64) (int, []Trade, error) {
	if err := validate(side, price, qty, false); err != nil {
		return 0, nil, err
	}
	o := &order{id: b.newID(), side: side, price: price, qty: qty}
	trades := b.match(o, false)
	if o.qty > 0 {
		b.rest(o)
	}
	return o.id, trades, nil
}

// Market executes immediately against the book, best price first, and discards whatever cannot
// be filled (immediate-or-cancel). It never rests.
func (b *Book) Market(side Side, qty int64) (int, []Trade, error) {
	if err := validate(side, 0, qty, true); err != nil {
		return 0, nil, err
	}
	o := &order{id: b.newID(), side: side, qty: qty}
	return o.id, b.match(o, true), nil
}

// Cancel removes a resting order in O(1): the index points straight at the order, which is
// dropped from its queue lazily. It reports false if the id is unknown, already filled or
// already cancelled.
func (b *Book) Cancel(id int) bool {
	r, ok := b.index[id]
	if !ok {
		return false
	}
	delete(b.index, id)
	lvl := r.lvl
	lvl.qty -= r.o.qty
	r.o.qty = 0
	lvl.dead++
	if lvl.qty == 0 {
		b.removeLevel(r.o.side, lvl)
	} else if lvl.dead > len(lvl.orders)/2 {
		lvl.compact()
	}
	return true
}

// Depth returns [price, total quantity] for each level of one side, best level first.
func (b *Book) Depth(side Side) [][2]int64 {
	levels, _ := b.levels(side)
	out := make([][2]int64, 0, len(*levels))
	for _, l := range *levels {
		out = append(out, [2]int64{l.price, l.qty})
	}
	return out
}
