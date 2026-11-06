# crypto-parser

Go backtester and parameter optimizer for crypto strategies on historical candles.

This is research tooling: **no API keys, no live orders, no wallet**. Binance public klines only.

## Run

```bash
make test
make run-ui          # http://localhost:8080
make fetch-btc       # BTCUSDT 1h via public REST
make backtest        # EMA crossover on fetched candles
make optimize        # grid / random sweep, top-N in SQLite
```

Look here first: `internal/domain` (entities), `internal/strategy` (interface + EMA crossover), `internal/backtest` (engine), `internal/exchange/binance` (adapter). Domain does not import HTTP.

## Layout

```
cmd/            CLI + dashboard
internal/domain
internal/strategy
internal/backtest
internal/optimize
internal/exchange   adapters
internal/storage    Parquet / SQLite
configs/            YAML runs
```

Optional Postgres: `docker compose up -d`. Default storage is SQLite under `results/` (gitignored).

Pine input parser: `go run ./cmd/parse-strategy --in strategies/example.pine`.

## Docs

- [Architecture](docs/ARCHITECTURE.md)
- [YAML schema](docs/YAML_SCHEMA.md)
- [Stack](docs/STACK.md)
