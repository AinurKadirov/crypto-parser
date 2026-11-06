# План спринтов

> Для разработчика без опыта. Каждый спринт = 1–2 недели.  
> Не переходи к следующему, пока не выполнены **критерии приёмки**.

---

## Sprint 0 — Фундамент и окружение

**Цель:** проект собирается, UI показывает конфиги, документация на месте.

### Задачи

1. Установить Go 1.22+: `brew install go`, проверить `go version`
2. Клонировать репо, выполнить `make build`
3. Прочитать `docs/README.md`, `docs/STACK.md`, `docs/ARCHITECTURE.md`
4. Запустить UI: `make run-ui`, открыть http://localhost:8080
5. Убедиться, что на dashboard видны pools, strategies, runs

### Файлы (уже созданы в репо)

```
cmd/server/main.go
internal/domain/bar.go
internal/config/loader.go
internal/server/handlers.go
web/index.html
configs/app.yaml
configs/pools/major.yaml
configs/strategies/ema_crossover.yaml
configs/runs/btc_1h_ema.yaml
```

### Критерии приёмки

- [x] `go build ./...` без ошибок
- [x] UI открывается на :8080
- [x] Dashboard показывает список YAML-конфигов
- [x] API `GET /api/health` возвращает `{"status":"ok"}`

### Частые ошибки

- Go не в PATH → добавить в `~/.zshrc`: `export PATH=$PATH:/usr/local/go/bin`
- Порт занят → `CRYPTO_PARSER_UI_PORT=8081 make run-ui`

---

## Sprint 1 — Загрузка данных с Binance

**Цель:** скачать исторические свечи и сохранить в Parquet.

### Предварительно

- Sprint 0 завершён
- Прочитать [Binance Klines API](https://binance-docs.github.io/apidocs/spot/en/#kline-candlestick-data)

### Задачи (пошагово)

1. **Изучить** `internal/exchange/provider.go` — интерфейс `MarketDataProvider`
2. **Реализовать** `internal/exchange/binance/client.go`:
   - `FetchKlines(symbol, interval, start, end)` → `[]domain.Bar`
   - Пагинация: max 1000 свечей за запрос, цикл while
   - Rate limit: sleep 200ms между запросами
3. **Реализовать** `internal/storage/parquet/candle_repo.go`:
   - `Save(symbol, timeframe, bars)` → файл `data/binance/BTCUSDT/1h.parquet`
   - `Load(symbol, timeframe)` → `[]domain.Bar`
4. **CLI** `cmd/fetch/main.go`:
   - Флаги: `--symbol`, `--timeframe`, `--from`, `--to`
   - Или: `--run configs/runs/btc_1h_ema.yaml`
5. **Тест:** `make fetch-btc` → файл появился в `data/`

### Критерии приёмки

- [x] `go run ./cmd/fetch --symbol BTCUSDT --timeframe 1h --from 2024-01-01 --to 2024-06-01`
- [x] Parquet-файл создан, размер > 0
- [x] Повторный fetch не дублирует данные (merge по timestamp)
- [x] UI показывает статус данных (symbol, bars count, date range)

### Файлы создать/изменить

```
internal/exchange/binance/client.go
internal/exchange/binance/client_test.go
internal/storage/parquet/candle_repo.go
cmd/fetch/main.go
```

---

## Sprint 2 — Индикаторы

**Цель:** расчёт MA, EMA, RSI, MACD, Volume SMA.

### Задачи

1. Создать `internal/indicator/indicator.go` — интерфейс `Calculator`
2. Реализовать:
   - `sma.go` — Simple Moving Average
   - `ema.go` — Exponential MA
   - `rsi.go` — RSI(14)
   - `macd.go` — MACD(12,26,9)
   - `volume_sma.go` — средний объём
3. Каждый индикатор принимает `[]float64` или `[]domain.Bar`, возвращает `[]float64`
4. Unit-тесты с эталонными значениями (можно сверить с TradingView)
5. Добавить в UI вкладку «Indicators» — расчёт на загруженных данных

### Критерии приёмки

- [x] `go test ./internal/indicator/...` — все тесты зелёные
- [x] EMA(12) на BTC 1h совпадает с TV с точностью 0.01%
- [x] UI вкладка Indicators показывает расчёт на загруженных данных

### Подсказка

```go
// EMA: первое значение = SMA(period), далее
// ema[i] = price[i] * k + ema[i-1] * (1-k), k = 2/(period+1)
```

---

## Sprint 3 — Backtester (движок симуляции)

**Цель:** прогнать сделки по истории с учётом комиссий и SL/TP.

### Задачи

1. `internal/domain/trade.go` — Trade, Position, Order
2. `internal/domain/metrics.go` — BacktestResult (Sharpe, drawdown, win rate...)
3. `internal/backtest/engine.go`:
   - Вход: `[]domain.Bar`, конфиг risk
   - На каждом баре вызывает `strategy.OnBar()`
   - Открывает/закрывает позиции
   - Учитывает commission + slippage
4. `internal/backtest/metrics.go` — расчёт метрик после прогона
5. CLI-заглушка: `cmd/backtest` пока без стратегии, только buy-and-hold для проверки

### Критерии приёмки

- [x] Buy-and-hold BTC → результат ≈ изменению цены минус комиссии
- [x] SL 2% срабатывает на баре, где low <= entry * 0.98
- [x] Метрики: total_return, sharpe_ratio, max_drawdown, win_rate, profit_factor

### Метрики (формулы)

| Метрика | Формула |
|---------|---------|
| Total Return | `(final - initial) / initial * 100` |
| Max Drawdown | max падение от peak equity |
| Sharpe | `mean(returns) / std(returns) * sqrt(252)` |
| Profit Factor | `gross_profit / gross_loss` |

---

## Sprint 4 — Первая стратегия: EMA Crossover

**Цель:** end-to-end: конфиг → fetch → backtest → результат в UI.

### Задачи

1. `internal/strategy/strategy.go` — интерфейс
2. `internal/strategy/ema_crossover.go` — реализация
3. `internal/strategy/registry.go` — factory по имени
4. Подключить backtester к стратегии
5. `cmd/backtest/main.go` — полный CLI
6. Сохранение результата в SQLite (`internal/storage/sqlite/result_repo.go`)
7. UI: страница результатов с таблицей метрик

### Логика EMA Crossover

```
IF ema_fast crosses above ema_slow AND volume > volume_sma * multiplier
  → BUY
IF ema_fast crosses below ema_slow
  → SELL / CLOSE
```

### Критерии приёмки

- [x] `go run ./cmd/backtest --run configs/runs/btc_1h_ema.yaml`
- [x] Результат в SQLite, виден в UI
- [x] Лог содержит: trades count, final balance, sharpe

---

## Sprint 5 — Оптимизатор параметров

**Цель:** grid search — найти лучшие параметры для монеты/TF.

### Задачи

1. `internal/optimize/grid.go` — перебор по `range`/`step` из YAML
2. `internal/optimize/runner.go` — параллельный запуск (worker pool, `runtime.NumCPU()`)
3. Ранжирование по метрике из конфига
4. Сохранение top-N в `param_sweeps` таблицу
5. `cmd/optimize/main.go`
6. UI: таблица top-N параметров

### Критерии приёмки

- [x] Grid 10×10 параметров на BTC 1h завершается < 2 мин на M3
- [x] Top-10 сохранены с метриками (`param_sweeps`)
- [x] UI показывает лучшую комбинацию (таблица Optimization Leaderboard)

### Осторожно

- **Overfitting:** хороший результат на истории ≠ прибыль в будущем
- В Sprint 8 добавим walk-forward validation

---

## Sprint 5A — Pine Parser + MAT AI Sniper

**Цель:** парсить Pine `input.*` в YAML и подключать стратегию к бэктесту. В публичном снимке пример — `strategies/example.pine`.

### Задачи

1. `internal/strategyparser/pine.go` — парсер `input.int/float/bool/string` из Pine
2. `cmd/parse-strategy/main.go` — CLI генерации YAML из `.txt` стратегии
3. `configs/strategies/mat_ai_sniper.yaml` — версия стратегии для бэктеста/оптимизации
4. `configs/runs/mat_ai_sniper_btc_15m.yaml` — стартовый run-конфиг
5. Интеграция с Sprint 5/6:
   - Sprint 5: оптимизация параметров `fastLength/slowLength/atrMultiplier/...`
   - Sprint 6: запуск по pool монет и набору TF

### Критерии приёмки

- [x] `go run ./cmd/parse-strategy --in strategies/mat-ai-sniper.txt ...` генерирует YAML
- [x] `configs/strategies/mat_ai_sniper.yaml` добавлен в проект
- [x] Sprint 5 optimizer использует параметры из strategy YAML для подбора
- [ ] Sprint 6 batch-run формирует top-N параметры per coin/per timeframe

---

## Sprint 6 — Multi-coin и batch runs

**Цель:** прогнать стратегию по всему пулу монет.

### Задачи

1. `cmd/optimize/main.go` — флаг `--pool configs/pools/major.yaml`
2. Batch runner: для каждого symbol × timeframe
3. Summary report: CSV/JSON с лучшими параметрами per coin
4. UI: heatmap или таблица «монета × метрика»

### Критерии приёмки

- [ ] `go run ./cmd/optimize --pool configs/pools/major.yaml --strategy ema_crossover --timeframe 1h`
- [ ] Отчёт для 10 монет
- [ ] Прогресс в логах (1/10, 2/10...)

---

## Sprint 7 — WebSocket live data

**Цель:** стрим свечей Binance + дописывание Parquet.

### Задачи

1. `internal/exchange/binance/ws.go` — подписка на kline stream
2. Reconnect с exponential backoff
3. `cmd/stream/main.go` — слушает N символов
4. UI: индикатор «live» / «offline», последняя свеча

### Критерии приёмки

- [ ] Stream BTCUSDT 1m — в логе новая свеча каждую минуту
- [ ] Parquet обновляется
- [ ] Reconnect после обрыва

---

## Sprint 8 — Walk-forward validation

**Цель:** защита от overfitting.

### Задачи

1. Разбить период на train/test окна (например 70/30)
2. Оптимизация на train → проверка на test
3. Конфиг: `optimize.walk_forward: { train_pct: 70, windows: 3 }`
4. UI: сравнение in-sample vs out-of-sample метрик

### Критерии приёмки

- [ ] Out-of-sample Sharpe < in-sample (ожидаемо)
- [ ] Отчёт показывает обе метрики

---

## Sprint 9 — Доп. стратегии и TradingView sync

**Цель:** RSI mean reversion, volume breakout; сверка с TV.

### Задачи

1. `internal/strategy/rsi_reversion.go`
2. `internal/strategy/volume_breakout.go`
3. YAML-конфиги для каждой
4. (Опционально) Python-скрипт tvkit для сверки EMA

---

## Sprint 10 — PostgreSQL + AI prep

**Цель:** production DB, экспорт для ML.

### Задачи

1. `docker compose up -d` → Postgres
2. `internal/storage/postgres/` — adapter
3. Переключение через `configs/app.yaml` → `driver: postgres`
4. Экспорт результатов в CSV для Python/LLM анализа

---

## Чеклист качества (каждый спринт)

- [ ] Код проходит `go vet ./...`
- [ ] Новые пакеты имеют `_test.go` где логика нетривиальна
- [ ] Нет hardcode путей — только через config/env
- [ ] Коммит-сообщение описывает «зачем», не «что»
- [ ] README обновлён если добавлена команда

## Карта зависимостей спринтов

```
S0 → S1 → S2 → S3 → S4 → S5 → S6
                  ↘ S7 (параллельно после S4)
S5 → S5A → S6
S5 → S8 → S9 → S10
```
