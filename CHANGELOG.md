# Changelog

All notable changes to this project are documented in this file.

## 0.1.0 - 2026-09-27

- `orderbook` package: limit and market orders, price-time priority, trades at the maker's price, cancel by id, depth
- `cmd/matching-engine` interactive shell
- Orders with a non-positive quantity or price, or an unknown side, are rejected with typed errors (a sell at price <= 0 used to rest and trade at that price)
- DESIGN.md with the data structures, matching rules and complexity
