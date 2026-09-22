# syntax=docker/dockerfile:1.7

## Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /src

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static-ish binary; CGO not required (pgx is pure Go).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/bot \
    ./cmd/bot

## Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget tzdata \
    && addgroup -S app \
    && adduser -S app -G app -h /app

WORKDIR /app

COPY --from=builder /out/bot /app/bot

USER app

ENV PORT=8080 \
    ENVIRONMENT=production \
    LOG_FORMAT=json \
    LOG_LEVEL=info \
    TELEGRAM_UPDATE_MODE=webhook

EXPOSE 8080

# Liveness: the HTTP server answers /health on the configured PORT.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/health" || exit 1

ENTRYPOINT ["/app/bot"]
