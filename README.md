# PBM Partner Analytics Bot 🤖

Telegram бот для HPE PBM — мгновенный поиск и аналитика по партнёрам.

## Features

- 🔍 **Fuzzy Search** — поиск партнёра по имени с нечётким совпадением
- 📊 **Partner Card** — детальная карточка: tier, сертификации, revenue, gap analysis
- 👥 **Multi-User** — система авторизации с ролями (admin / user / pending)
- 📥 **Excel Import** — парсинг огромных Excel файлов (96MB+) через streaming
- 🐳 **Docker Ready** — деплой через Docker Compose

## Quick Start

```bash
# 1. Copy env and configure
cp .env.example .env
# Edit .env with your Telegram token and admin ID

# 2. Start with Docker
make docker-up

# 3. Talk to your bot in Telegram!
```

## Development

```bash
# Install Go dependencies
go mod tidy

# Run locally (requires PostgreSQL)
make run

# Run tests
make test
```

## Tech Stack

| Component  | Technology           |
|-----------|----------------------|
| Language  | Go 1.22+             |
| Telegram  | go-telegram/bot      |
| Excel     | excelize (streaming)  |
| Database  | PostgreSQL 16 + pgx  |
| Deploy    | Docker Compose       |

## Commands

| Command          | Description                    |
|-----------------|--------------------------------|
| `/start`        | Приветствие + регистрация      |
| `/search <имя>` | Поиск партнёра                 |
| `/stats`        | Статистика базы                |
| `/help`         | Справка                        |
| `/users`        | Список ожидающих (admin only)  |
