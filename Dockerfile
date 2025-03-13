FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копіюємо go.mod та go.sum
COPY go.mod go.sum ./

# Завантажуємо залежності
RUN go mod download

# Копіюємо код
COPY . .

# Створюємо необхідні директорії, якщо вони відсутні
RUN mkdir -p /app/web/templates /app/web/static

# Збираємо бот
RUN CGO_ENABLED=0 GOOS=linux go build -o roulette-bot ./cmd/bot/main.go

# Збираємо адмін-панель
RUN CGO_ENABLED=0 GOOS=linux go build -o roulette-admin ./cmd/admin/main.go

# Збираємо ротатор
RUN CGO_ENABLED=0 GOOS=linux go build -o roulette-rotator ./cmd/rotator/main.go

# Кінцевий образ
FROM alpine:latest

# Встановлюємо необхідні пакети
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Створюємо необхідні директорії
RUN mkdir -p /app/web/templates /app/web/static /app/migrations

# Копіюємо бінарні файли
COPY --from=builder /app/roulette-bot /app/roulette-bot
COPY --from=builder /app/roulette-admin /app/roulette-admin
COPY --from=builder /app/roulette-rotator /app/roulette-rotator

# Копіюємо міграції, якщо вони існують
COPY --from=builder /app/migrations/* /app/migrations/

# Копіюємо web директорії, якщо вони існують
COPY --from=builder /app/web/templates /app/web/templates
COPY --from=builder /app/web/static /app/web/static

# Копіюємо .env.example як .env, якщо існує
COPY --from=builder /app/.env.example /app/.env

# Змінюємо права на виконання
RUN chmod +x /app/roulette-bot /app/roulette-admin /app/roulette-rotator

# Порт для адмін-панелі
EXPOSE 8080

# Запускаємо за замовчуванням бота
CMD ["./roulette-bot"]