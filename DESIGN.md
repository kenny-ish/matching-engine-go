# Design notes

The aim was an order book short enough to read in one go that still follows the rules of a real
continuous-trading venue. These notes explain the choices.

## Data structures

- Each side is a slice of price levels sorted best first: bids descending, asks ascending.
- A level is a FIFO queue (a slice) of the resting orders at one price.
- `index` maps a resting order id to its side, so `Cancel` knows which side to search.

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

## Validation

Orders with a non-positive quantity, a non-positive limit price or an unknown side are rejected
with `ErrInvalidQty`, `ErrInvalidPrice` or `ErrInvalidSide` before an id is assigned. Before
v0.1.0 a sell with price <= 0 would rest and later trade at that price.

## Complexity

L = price levels on a side, N = resting orders on a side, k = orders filled.

| operation | cost |
|---|---|
| `Limit` that rests | O(log L) search, plus an O(L) copy when it creates a level |
| matching (`Limit` or `Market`) | O(k) |
| `Cancel` | O(N) scan of the side |
| `Depth` | O(N) |

## Out of scope

- Self-trade prevention and order types beyond limit and market (IOC, FOK, post-only)
- Concurrency. A `Book` belongs to one goroutine, so put a channel or a mutex in front of it to share it
- Persistence, sequencing, fees
