# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git build-base

# Кеш модулей
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Исходники
COPY . .

# Общие флаги
ENV CGO_ENABLED=0 GOOS=linux
ARG VERSION
ARG COMMIT
ARG BUILT_AT
ENV LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}"

# Параллельная сборка (BuildKit кеш для компиляции)
RUN --mount=type=cache,target=/root/.cache/go-build \
    ( go build -trimpath -ldflags "$LDFLAGS" -o roulette-bot ./cmd/bot & \
      go build -trimpath -ldflags "$LDFLAGS" -o roulette-admin ./cmd/admin & \
      go build -trimpath -ldflags "$LDFLAGS" -o roulette-rotator ./cmd/rotator & \
      wait ) && \
    ls -la /app/roulette-* && \
    test -f /app/roulette-bot && \
    test -f /app/roulette-admin && \
    test -f /app/roulette-rotator

# Runtime: alpine (root по умолчанию)
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app

COPY --from=builder /app/roulette-bot /app/roulette-bot
COPY --from=builder /app/roulette-admin /app/roulette-admin
COPY --from=builder /app/roulette-rotator /app/roulette-rotator
COPY --from=builder /app/locales /app/locales
COPY --from=builder /app/web /app/web
COPY --from=builder /app/internal/captcha-go /app/internal/captcha-go

RUN chmod +x /app/roulette-bot /app/roulette-admin /app/roulette-rotator
EXPOSE 8080
CMD ["./roulette-bot"]
