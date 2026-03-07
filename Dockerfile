# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bot ./cmd/bot/

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

# Security: run as non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bot /app/bot
COPY migrations/ /app/migrations/

USER appuser

ENTRYPOINT ["/app/bot"]
