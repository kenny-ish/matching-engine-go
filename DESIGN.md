# Design notes

The aim was an order book short enough to read in one go that still follows the rules of a real
continuous-trading venue. These notes explain the choices.

## Data structures

- Each side is a slice of price levels sorted best first: bids descending, asks ascending.
- A level is a FIFO queue (a slice) of the resting orders at one price, plus the live quantity
  at that price and a count of cancelled orders still in the queue.
- `index` maps a resting order id to the order and its level.

A sorted slice keeps the best price at index 0, which is all matching needs. Inserting a new
price level costs a binary search plus a copy of the slice tail. For books with tens to a few
hundred active levels that's cheap, and friendlier to the cache than a tree.

## Matching rules

1. Price priority. An incoming order matches the best opposite level first and walks outward
   while its limit still crosses.
2. Time priority. Inside a level, orders fill in arrival order.
3. Trades print at the maker's price. The resting order's price was public when the taker
   arrived, so a taker willing to pay more still gets the better price. This is how continuous
   limit-order markets work.
4. Whatever is left of a limit order rests at its limit price after matching.
5. Market orders are immediate-or-cancel. They sweep the book until filled or until the
   opposite side is empty, and the unfilled rest is discarded. Resting a market order would mean
   inventing a price for it.

## Cancelling lazily

Cancels are the most common message on a real venue, so they should not scan the book. `Cancel`
looks the order up in the index, subtracts its quantity from the level and marks it with
quantity 0. The order stays in the queue until matching reaches it (it is skipped) or until
cancelled orders make up more than half of the queue, when the queue is compacted. A level whose
live quantity drops to zero is removed from the side right away, so the best price is always
live.

## Validation

Orders with a non-positive quantity, a non-positive limit price or an unknown side are rejected
with `ErrInvalidQty`, `ErrInvalidPrice` or `ErrInvalidSide` before an id is assigned. Before
v0.1.0 a sell with price <= 0 would rest and later trade at that price.

## Complexity

L = price levels on a side, N = resting orders on a side, k = orders filled.

| operation | cost |
|---|---|
| `Limit` that rests | O(log L) search, plus an O(L) copy when it creates a level |
| matching (`Limit` or `Market`) | O(k), plus skipping cancelled orders once |
| `Cancel` | O(1) amortized, O(log L + L) when it empties a level |
| `Depth` | O(L) |

## Out of scope

- Self-trade prevention and order types beyond limit and market (IOC, FOK, post-only)
- Concurrency. A `Book` belongs to one goroutine, so put a channel or a mutex in front of it to share it
- Persistence, sequencing, fees
