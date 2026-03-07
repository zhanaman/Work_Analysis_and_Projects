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
