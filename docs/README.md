# Библиотека документации crypto-parser

Центральное хранилище знаний проекта. Читай документы в порядке ниже.

| Документ | Описание |
|----------|----------|
| [STACK.md](./STACK.md) | Стек технологий, зависимости, окружение |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Архитектура, слои, паттерны |
| [YAML_SCHEMA.md](./YAML_SCHEMA.md) | Схемы всех YAML-конфигов |
| [SPRINTS.md](./SPRINTS.md) | Подробный план спринтов для разработчика |
| `strategies/example.pine` | Короткий Pine-пример для парсера `input.*` |

## Быстрый старт

```bash
# 1. Установить Go 1.22+
brew install go

# 2. Склонировать и перейти в проект
cd crypto-parser

# 3. Запустить UI (без Docker)
make run-ui

# 4. (Опционально) PostgreSQL в Docker
docker compose up -d
```

## Структура данных на диске

```
crypto-parser/
├── configs/          # YAML — настройки (в git)
├── data/             # Parquet свечи (gitignore)
├── results/          # SQLite / отчёты (gitignore)
└── docs/             # Эта библиотека (в git)
```
