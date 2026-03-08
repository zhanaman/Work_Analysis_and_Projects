.PHONY: build run dev test clean docker-up docker-down migrate-up

# Default Go binary name
BINARY=pbm-partner-bot

## build: Build the Go binary
build:
	go build -o $(BINARY) ./cmd/bot/

## run: Build and run the bot
run: build
	./$(BINARY)

## dev: Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

## test: Run all tests
test:
	go test -v ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	go clean

## docker-up: Start all services with Docker Compose
docker-up:
	docker compose up --build -d

## docker-down: Stop all Docker services
docker-down:
	docker compose down

## docker-logs: Show Docker logs
docker-logs:
	docker compose logs -f bot

## migrate-up: Apply database migrations (requires running PostgreSQL)
migrate-up:
	@echo "Migrations are auto-applied via docker-entrypoint-initdb.d"
	@echo "For manual apply, use: psql \$$POSTGRES_DSN -f migrations/001_init.up.sql"

## migrate-down: Rollback database migrations
migrate-down:
	@echo "Run: psql \$$POSTGRES_DSN -f migrations/001_init.down.sql"

## tidy: Go mod tidy
tidy:
	go mod tidy

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'

# === Remote deployment ===
VPS_HOST=root@31.44.6.132
VPS_KEY=~/.ssh/id_vps_ynab
VPS_DIR=/root/pbm_bot

## upload: Upload Excel to production and import (usage: make upload FILE=path/to/file.xlsx)
upload:
ifndef FILE
	$(error Usage: make upload FILE=path/to/file.xlsx)
endif
	@echo "📤 Uploading $(FILE) to server..."
	scp -i $(VPS_KEY) "$(FILE)" $(VPS_HOST):$(VPS_DIR)/data/
	@echo "📊 Running import..."
	ssh -i $(VPS_KEY) $(VPS_HOST) "docker exec pbm-partner-bot /app/importer -file /app/data/$$(basename '$(FILE)') -dsn \$$(cat $(VPS_DIR)/.env | grep POSTGRES_DSN | cut -d= -f2-)"
	@echo "🗑️  Cleaning up file on server..."
	ssh -i $(VPS_KEY) $(VPS_HOST) "rm -f $(VPS_DIR)/data/$$(basename '$(FILE)')"
	@echo "✅ Import complete!"

## deploy: Deploy latest code to production
deploy:
	git push
	ssh -i $(VPS_KEY) $(VPS_HOST) "cd $(VPS_DIR) && git pull && docker compose up --build -d"
	@echo "✅ Deployed!"

## import-local: Import Excel locally (usage: make import-local FILE=path/to/file.xlsx)
import-local:
ifndef FILE
	$(error Usage: make import-local FILE=path/to/file.xlsx)
endif
	go run ./cmd/import/ -file "$(FILE)"

