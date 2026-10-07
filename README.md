# matching-engine-go

[![CI](https://github.com/kenny-ish/matching-engine-go/actions/workflows/ci.yml/badge.svg)](https://github.com/kenny-ish/matching-engine-go/actions/workflows/ci.yml)

A limit order book with price-time priority in a few hundred lines of Go, using only the standard
library. It follows the matching rules of real exchanges.

- limit orders and market orders (immediate-or-cancel)
- price-time priority: best price first, then first come, first served within a price level
- trades execute at the maker's price
- partial fills, resting remainders, cancel by id
- integer prices and quantities, and invalid orders are rejected with typed errors

[DESIGN.md](DESIGN.md) explains the data structures, the matching rules and what each operation
costs.

## Library

```bash
go get github.com/kenny-ish/matching-engine-go/orderbook
```

```go
b := orderbook.NewBook()
b.Limit(orderbook.Sell, 101, 5)
b.Limit(orderbook.Sell, 100, 5)
id, trades, err := b.Limit(orderbook.Buy, 101, 7) // fills 5 @ 100, then 2 @ 101
```

## Interactive shell

```
$ go run ./cmd/matching-engine
commands: buy P Q | sell P Q | market buy|sell Q | cancel ID | book | quit
> buy 100 5
order id 1
> sell 101 3
order id 2
> sell 99 7
TRADE 5 @ 100 (maker 1, taker 3)
order id 3
> book
  ask     101 x 3
  ask      99 x 2
> market buy 4
TRADE 2 @ 99 (maker 3, taker 4)
TRADE 2 @ 101 (maker 2, taker 4)
> cancel 2
cancelled: true
> buy 100 0
rejected: orderbook: quantity must be positive
```

## Tests

```bash
go test ./...                                               # unit, edge-case and fuzz seed tests
go test -run '^$' -fuzz FuzzBook -fuzztime 30s ./orderbook  # random operation sequences
go test -run '^$' -bench . ./orderbook                      # benchmarks
```

The fuzz target drives the book with random limit, market and cancel operations and checks after
every step that the book is never crossed, levels are strictly ordered and never empty, every
trade prints at the maker's price within the taker's limit, and the resting quantity at each price
matches a simple model of the orders.

Release notes are in [CHANGELOG.md](CHANGELOG.md).
