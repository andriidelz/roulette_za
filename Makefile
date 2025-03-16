.PHONY: build run run-bot run-admin run-rotator migrate migrate-update clean docker docker-up docker-down test init

# Збірка
build:
	go build -o bin/roulette-bot ./cmd/bot/main.go
	go build -o bin/roulette-admin ./cmd/admin/main.go
	go build -o bin/roulette-rotator ./cmd/rotator/main.go

# Запуск
run: run-bot run-admin run-rotator

run-bot:
	go run ./cmd/bot/main.go

run-admin:
	go run ./cmd/admin/main.go

run-rotator:
	go run ./cmd/rotator/main.go

# Міграції
migrate:
	PGPASSWORD=postgres psql -h localhost -U postgres -d roulette -f migrations/001_initial_schema.sql

# Застосування міграції оновлення схеми
migrate-update:
	PGPASSWORD=postgres psql -h localhost -U postgres -d roulette -f migrations/002_update_schema.sql

# Очистка
clean:
	rm -rf bin/

# Docker
docker:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Тести
test:
	go test -v ./...

# Ініціалізація проекту
init:
	cp .env.example .env
	go mod tidy
	mkdir -p bin
	mkdir -p web/static
	@echo "Ініціалізація завершена. Відредагуйте файл .env і запустіть 'make migrate'"
