# Схемы YAML-конфигов

Все конфиги лежат в `configs/`. Валидируются при загрузке в Go.

---

## 1. Coin Pool — `configs/pools/*.yaml`

Список монет для массовых прогонов.

```yaml
pool:
  name: major           # уникальное имя пула
  description: "Top crypto by market cap"
  exchange: binance     # binance | bybit

symbols:
  - BTCUSDT
  - ETHUSDT
  - BNBUSDT
  - SOLUSDT
  - XRPUSDT
  - ADAUSDT
  - DOGEUSDT
  - AVAXUSDT
  - DOTUSDT
  - LINKUSDT

timeframes:             # какие TF качать и тестировать
  - 15m
  - 1h
  - 4h
  - 1d
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| `pool.name` | string | да | Идентификатор пула |
| `pool.exchange` | string | да | Биржа-источник |
| `symbols` | []string | да | Торговые пары |
| `timeframes` | []string | да | `1m`,`5m`,`15m`,`1h`,`4h`,`1d` |

---

## 2. Strategy — `configs/strategies/*.yaml`

Описание стратегии и пространства параметров для оптимизации.

```yaml
strategy:
  name: ema_crossover
  version: "1.0"
  description: "EMA fast/slow crossover with volume filter"

parameters:
  ema_fast:
    type: int
    default: 12
    range: [5, 50]      # для optimize: min, max
    step: 1
  ema_slow:
    type: int
    default: 26
    range: [20, 200]
    step: 2
  volume_multiplier:
    type: float
    default: 1.5
    range: [1.0, 3.0]
    step: 0.1

filters:
  min_volume_24h: 1000000
  exclude_weekends: false

risk:
  stop_loss_pct: 2.0
  take_profit_pct: 4.0
  max_position_pct: 10.0
  commission_pct: 0.1
  slippage_pct: 0.05

backtest:
  initial_balance: 10000
```

| Поле | Тип | Описание |
|------|-----|----------|
| `parameters.*.type` | `int` \| `float` \| `bool` | Тип параметра |
| `parameters.*.range` | [min, max] | Диапазон для grid search |
| `parameters.*.step` | number | Шаг перебора |
| `risk.commission_pct` | float | Комиссия биржи, % |
| `risk.slippage_pct` | float | Проскальзывание, % |

---

## 3. Run — `configs/runs/*.yaml`

Конкретный прогон: монета + стратегия + период.

```yaml
run:
  name: btc_1h_ema_v1
  strategy: ema_crossover    # имя файла из configs/strategies/
  symbol: BTCUSDT
  exchange: binance
  timeframe: 1h
  date_from: "2023-01-01"
  date_to: "2025-05-01"

# Переопределение параметров (опционально)
params:
  ema_fast: 12
  ema_slow: 26

optimize:
  enabled: true
  method: grid              # grid | random
  metric: sharpe_ratio      # sharpe_ratio | profit_factor | total_return | max_drawdown
  top_n: 10
  max_iterations: 1000      # для random
```

| Поле | Тип | Описание |
|------|-----|----------|
| `run.strategy` | string | Ссылка на `configs/strategies/{name}.yaml` |
| `optimize.method` | string | Алгоритм подбора параметров |
| `optimize.metric` | string | Метрика для ранжирования |
| `optimize.top_n` | int | Сколько лучших сохранить |

---

## 4. App — `configs/app.yaml`

Глобальные настройки приложения.

```yaml
app:
  name: crypto-parser
  log_level: info           # debug | info | warn | error

paths:
  data_dir: ./data
  results_dir: ./results

database:
  driver: sqlite            # sqlite | postgres
  dsn: ./results/crypto-parser.db
  # postgres: postgres://parser:parser@localhost:5432/crypto_parser?sslmode=disable

ui:
  port: 8080
  enabled: true

fetch:
  rate_limit_ms: 200        # пауза между запросами к API
  retry_count: 3
```

---

## Маппинг таймфреймов → Binance interval

| YAML | Binance API |
|------|-------------|
| `1m` | `1m` |
| `5m` | `5m` |
| `15m` | `15m` |
| `1h` | `1h` |
| `4h` | `4h` |
| `1d` | `1d` |

---

## Примеры команд

```bash
# Загрузить данные по run-конфигу
go run ./cmd/fetch --run configs/runs/btc_1h_ema.yaml

# Бэктест
go run ./cmd/backtest --run configs/runs/btc_1h_ema.yaml

# Оптимизация
go run ./cmd/optimize --run configs/runs/btc_1h_ema.yaml
```
