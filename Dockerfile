FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build apps
RUN CGO_ENABLED=0 GOOS=linux go build -o roulette-bot ./cmd/bot/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o roulette-admin ./cmd/admin/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o roulette-rotator ./cmd/rotator/main.go

# Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy builds
COPY --from=builder /app/roulette-bot /app/roulette-bot
COPY --from=builder /app/roulette-admin /app/roulette-admin
COPY --from=builder /app/roulette-rotator /app/roulette-rotator

# Copy web directories
COPY --from=builder /app/web /app/web

# Change access rights
RUN chmod +x /app/roulette-bot /app/roulette-admin /app/roulette-rotator

# Port for web interface
EXPOSE 8080

# Run bot
CMD ["./roulette-bot"]
