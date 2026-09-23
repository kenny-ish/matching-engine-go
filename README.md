# matching-engine-go

A compact exchange matching engine to understand how order books really work.

- limit and market orders
- **price-time priority**: best price first, then first-come-first-served within a level
- partial fills, resting remainders, cancels by id
- integer prices (ticks) and quantities, so no floating point surprises

```bash
go run .
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
```

Trades execute at the **maker's** price, as on real exchanges. Unfilled market order
quantity is discarded rather than resting.

```bash
go test ./...
```
