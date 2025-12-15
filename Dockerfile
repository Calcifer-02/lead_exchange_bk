# --- Stage 1: Build ---
FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git bash

# Включаем автоматическое обновление toolchain для совместимости
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Устанавливаем goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Собираем бинарь приложения
RUN go build -o app ./cmd/main.go

# --- Stage 2: Runtime ---
FROM alpine:3.18

WORKDIR /app
RUN apk add --no-cache bash ca-certificates

# Копируем приложение
COPY --from=builder /app/app .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/pkg ./pkg

# Копируем goose
COPY --from=builder /go/bin/goose /usr/local/bin/goose

ENV BACKEND_PORT=8081
ENV POSTGRES_SETUP="user=postgres password=password dbname=lead_exchange host=postgres_db port=5432 sslmode=disable"

EXPOSE 8081

# CMD с запуском миграций и приложения
CMD ["sh", "-c", "goose -dir ./migrations postgres \"$POSTGRES_SETUP\" up && ./app"]
