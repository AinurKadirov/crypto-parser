# Стек технологий

## Ядро

| Компонент | Технология | Зачем |
|-----------|------------|-------|
| Язык | **Go 1.22+** | Производительность, один бинарник, vibe-coding |
| Конфиги | **YAML** | Читаемые настройки стратегий и прогонов |
| Свечи (история) | **Parquet** | Компактное columnar-хранилище OHLCV |
| Результаты (dev) | **SQLite** | Файл `results/crypto-parser.db`, без Docker |
| Результаты (prod) | **PostgreSQL 16** | Docker Compose, масштабирование |
| UI | **Go embed + HTML/CSS/JS** | Минимальный dashboard без фреймворков |
| HTTP | **net/http** | Встроенный сервер, без Gin/Echo на старте |

## Внешние API

| Источник | Тип | Статус |
|----------|-----|--------|
| Binance Spot | REST + WebSocket | Primary, бесплатно |
| Bybit | REST + WebSocket | Backup (Sprint 7+) |
| TradingView | Неофициальный WS | Только сверка (Sprint 9+) |

## Go-зависимости

```
gopkg.in/yaml.v3              — парсинг YAML
github.com/parquet-go/parquet-go — Parquet I/O
modernc.org/sqlite            — SQLite без CGO
github.com/jackc/pgx/v5       — PostgreSQL (опционально)
```

## Docker

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: crypto_parser
      POSTGRES_USER: parser
      POSTGRES_PASSWORD: parser
```

PostgreSQL **не обязателен** на старте. SQLite работает из коробки.

## Железо (целевое)

- MacBook M3 Air, 16 GB RAM
- ~8 goroutines для параллельной оптимизации
- ~500 MB диска на 20 монет × 5 таймфреймов × 2 года

## Что НЕ используем на старте

- C++ — overkill для бэктеста
- Node.js — только если позже понадобится отдельный UI
- Kubernetes — локальная разработка
- Redis — не нужен до live-trading

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `CRYPTO_PARSER_CONFIG_DIR` | `./configs` | Папка YAML |
| `CRYPTO_PARSER_DATA_DIR` | `./data` | Parquet |
| `CRYPTO_PARSER_RESULTS_DIR` | `./results` | SQLite / отчёты |
| `CRYPTO_PARSER_DB_DSN` | `sqlite:results/crypto-parser.db` | SQLite или `postgres://...` |
| `CRYPTO_PARSER_UI_PORT` | `8080` | Порт dashboard |
| `CRYPTO_PARSER_BINANCE_BASE_URL` | `https://api.binance.com` | Binance API |
