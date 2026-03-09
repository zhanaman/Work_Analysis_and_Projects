# AI Context: PBM Partner Analytics Bot

## 1. Project Overview
This is a Telegram Bot built in Go for HPE Partner Business Managers (PBM). 
It parses giant HPE Excel reports (96MB+) locally or via Docker and stores the data in PostgreSQL.
The bot provides instant fuzzy search and formatted analytical cards with actionable Insights (Targets, Gaps, Readiness).

## 2. Core Architecture
- **Language**: Go 1.22+
- **Bot Framework**: `go-telegram/bot` (v1)
- **Database**: PostgreSQL 16
- **DB Driver**: `jackc/pgx/v5`
- **Excel Parser**: `qax-os/excelize` (StreamRowReader is MANDATORY for performance)
- **Deployment**: Docker Compose on `31.44.6.132` (Debian VPS, 2GB RAM + 2GB Swap)

## 3. Data Domain (Centers & Tiers)
The data is segregated into 3 Centers of Expertise:
- **Compute**
- **Hybrid Cloud**
- **Networking**
  
Within each center, partners can pursue specific Tiers:
- Business Partner (BP)
- Silver
- Gold
- Platinum

## 4. Key UX Principles
- **Progressive Disclosure**: Show high-level summary first, hide dense data unless asked.
- **Inline Cards**: The `/search` command returns a single message with an inline text card. No drill-down inline buttons are used for the main partner details to reduce chat clutter.
- **Single-Message Mutation**: Dashboard interactions (e.g. `/stats`) use `EditMessageText` to rewrite the exact same message instead of sending new ones.
- **Actionable Data**: Always show *why* a partner is failing ("X gaps", "❌ Volume $X / $Y", "❌ Certs X/Y"). Do not show ✅ for passing criteria if it clutters the view.

## 5. Deployment / Makefile Commands
- `make upload FILE=...`: Uploads an `.xlsx` file from the host machine to the VPS via SCP, executes `/app/importer` inside the running Docker container, and immediately deletes the `.xlsx` file from the server. The data date is extracted from the Excel's `Refresh_date` column (not from the filename).
- `make deploy`: Triggers a `git pull` and `docker compose up --build -d` on the remote VPS.
- `make import-local FILE=...`: Runs the parser locally against local postgres (port 5433).

## 6. Known Trade-Offs & Quirks
- The `excelize` library parses files fully into memory unless `StreamRowReader` is used. Stream parsing is mandatory.
- The VPS has 2GB RAM + 2GB persistent SWAP. This is enough to parse 80MB+ Excel files without OOM.
- `data_date` is extracted from the Excel's `Refresh_date` / `Refreshed date` column (found in Compute, HC, Networking sheets). Multiple date formats are handled: `MM-DD-YY`, `M/D/YY HH:MM`, `YYYY-MM-DD`.
