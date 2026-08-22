# ---------- build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Отдельный слой с зависимостями: инвалидируется только при изменении
# go.mod/go.sum, а не при каждой правке .go-файла (Docker layer cache).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 — статическая линковка, обязательна для запуска на musl (alpine)
# и попутно форсирует чисто Go DNS-резолвер вместо cgo-резолвера.
# -ldflags="-s -w" — убирает symbol table и DWARF debug info (меньше бинарник).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server \
    ./cmd/api

# ---------- final stage ----------
FROM alpine:3.24

# CA-сертификаты — не входят в базовый alpine по умолчанию, нужны для
# валидации TLS (например, sslmode=verify-full к Postgres в будущем,
# или любые внешние HTTPS-вызовы).
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/server .

USER appuser

# Метадата: сам по себе порт не публикует — публикация через
# `docker run -p` или `ports:` в docker-compose.
EXPOSE 8080

# Exec-форма (не "CMD /app/server") — критично для graceful shutdown
# (решение #35): в shell-форме PID 1 внутри контейнера — это /bin/sh,
# который не форвардит SIGTERM дочернему процессу.
ENTRYPOINT ["/app/server"]