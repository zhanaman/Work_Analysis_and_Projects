# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binaries
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bot ./cmd/bot/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /importer ./cmd/import/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /partner-bot ./cmd/partner-bot/

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

# Security: run as non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bot /app/bot
COPY --from=builder /importer /app/importer
COPY --from=builder /partner-bot /app/partner-bot
COPY migrations/ /app/migrations/

# Data directory for Excel uploads
RUN mkdir -p /app/data && chown appuser:appuser /app/data

USER appuser

ENTRYPOINT ["/app/bot"]
