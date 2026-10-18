# matching-engine-go

[![CI](https://github.com/kenny-ish/matching-engine-go/actions/workflows/ci.yml/badge.svg)](https://github.com/kenny-ish/matching-engine-go/actions/workflows/ci.yml)

A limit order book with price-time priority in a few hundred lines of Go, using only the standard
library. It follows the matching rules of real exchanges.

- limit orders and market orders (immediate-or-cancel)
- price-time priority: best price first, then first come, first served within a price level
- trades execute at the maker's price
- partial fills, resting remainders, O(1) cancel by id
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

More in the runnable examples (`orderbook/example_test.go`), which `go test` checks against their
expected output.

## Interactive shell

```
$ go run ./cmd/matching-engine
commands: buy P Q | sell P Q | market buy|sell Q | cancel ID | book | quit
> buy 100 5
order id 1
> buy 99 4
order id 2
> sell 101 3
order id 3
> sell 102 6
order id 4
> book
  ask     102 x 6      cum 9
  ask     101 x 3      cum 3
  ---- spread 1 ----
  bid     100 x 5      cum 5
  bid      99 x 4      cum 9
> market buy 4
TRADE 3 @ 101 (maker 3, taker 5)
TRADE 1 @ 102 (maker 4, taker 5)
> book
  ask     102 x 5      cum 5
  ---- spread 2 ----
  bid     100 x 5      cum 5
  bid      99 x 4      cum 9
```

## Tests

```bash
go test ./...                                               # unit, edge-case, example and fuzz seed tests
go test -run '^$' -fuzz FuzzBook -fuzztime 30s ./orderbook  # random operation sequences
go test -run '^$' -bench . ./orderbook                      # benchmarks
```

The fuzz target drives the book with random limit, market and cancel operations and checks after
every step that the book is never crossed, levels are strictly ordered and never empty, every
trade prints at the maker's price within the taker's limit, and the resting quantity at each price
matches a simple model of the orders.

Release notes are in [CHANGELOG.md](CHANGELOG.md).
