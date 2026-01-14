# Roulette Bot - Architecture Overview

Telegram-бот для игры в рулетку с административной панелью и системой ротации хэшей.

## Архитектура проекта

Проект состоит из трёх основных сервисов:

### Основные сервисы

1. **Bot Service** (`cmd/bot`) - Telegram бот
   - Обрабатывает команды пользователей
   - Управляет игровой логикой
   - Интеграция с платежными системами
   - Metrics port: 9101

2. **Admin Service** (`cmd/admin`) - Административная панель
   - Web-интерфейс для управления ботом
   - Управление пользователями и играми
   - Отправка уведомлений
   - Управление платежами
   - Port: 8080, Metrics port: 9103

3. **Rotator Service** (`cmd/rotator`) - Ротатор хэшей
   - Периодическая смена хэшей для честной игры
   - Интеграция с RabbitMQ для уведомлений
   - Metrics port: 9102

### Структура директорий

```
roulette/
├── cmd/                    # Entry points для сервисов
│   ├── bot/               # Telegram bot
│   ├── admin/             # Admin panel
│   └── rotator/           # Hash rotator
├── internal/              # Внутренняя логика
│   ├── admin/             # Admin panel logic
│   ├── bot/               # Bot logic
│   ├── config/            # Configuration
│   ├── logger/            # Logging
│   ├── metrics/           # Prometheus metrics
│   ├── repository/        # Database layer (GORM)
│   ├── rotator/           # Rotation logic
│   ├── service/           # Business logic
│   └── captcha-go/        # Captcha generation
├── web/                   # Static web assets
├── migrations/            # Database migrations
├── shared-data/           # Docker volumes data + files
├── compose.yml           # Production compose
├── compose.dev.yml       # Development compose
└── Dockerfile            # Multi-stage build
```

## Контейнеры

### Общее количество контейнеров: 16

**Основные сервисы (3):**
- postgres
- redis  
- rabbitmq

**Приложения (4):**
- migrations (запускается один раз)
- bot
- admin
- rotator

**Инструменты разработки - profile: tools (2):**
- pgadmin
- redis-commander

**Мониторинг - profile: monitoring (7):**
- prometheus
- grafana
- postgres-exporter
- rabbitmq-exporter
- redis-exporter
- node-exporter

### Последовательность запуска

1. **Инфраструктурные сервисы запускаются параллельно:**
   - postgres (с healthcheck)
   - redis (с healthcheck)
   - rabbitmq (с healthcheck)

2. **Миграции базы данных:**
   - migrations контейнер запускается после postgres healthcheck
   - Выполняется один раз (restart: "no")
   - Использует migrate/migrate:v4.16.2
   - Применяет миграции из ./migrations директории

3. **Приложения запускаются после успешных миграций:**
   - bot (depends_on: postgres healthy, migrations completed, rabbitmq healthy, redis healthy)
   - admin (depends_on: postgres healthy, migrations completed, redis healthy)
   - rotator (depends_on: postgres healthy, migrations completed, rabbitmq healthy, redis healthy)

4. **Инструменты запускаются после основных сервисов:**
   - pgadmin (depends_on: postgres healthy)
   - redis-commander (depends_on: redis healthy)

5. **Мониторинг запускается после экспортеров:**
   - Экспортеры запускаются после соответствующих сервисов
   - grafana запускается после prometheus и всех экспортеров

### Процесс миграции БД

Миграции выполняются автоматически при запуске через отдельный контейнер `migrate/migrate:v4.16.2`. Файлы миграций находятся в директории `./migrations` и применяются в алфавитном порядке.

## Доступные интерфейсы

### Основные сервисы
- **Admin Panel**: http://localhost:8080
  - Username/Password: настраиваются в `.env` (по умолчанию admin/admin)
  - Управление ботом, пользователями, играми, платежами
  - Отправка уведомлений пользователям

### Инструменты разработки (profile: tools)
- **PgAdmin**: http://localhost:8181/pgadmin
  - Email/Password: postgres@postgres.com/postgres
  - Управление PostgreSQL базой данных

- **Redis Commander**: http://localhost:8282/redis  
  - Username/Password: admin/admin
  - Управление Redis кешем

- **RabbitMQ Management**: http://localhost:15672
  - Username/Password: guest/guest
  - Управление очередями сообщений

### Мониторинг (profile: monitoring)
- **Grafana**: http://localhost:3000/grafana
  - Username/Password: admin/admin
  - Дашборды для мониторинга всех компонентов
  - Готовые дашборды для PostgreSQL, Redis, RabbitMQ, приложений

- **Prometheus**: http://localhost:9090/prometheus
  - Сборка метрик со всех сервисов
  - Targets: applications, databases, system metrics

## Система метрик

### Система метрик

Каждый сервис экспортирует метрики:

**Bot Service (port 9101)**
- Количество обработанных команд
- Время ответа бота
- Количество активных пользователей
- Ошибки обработки сообщений

**Admin Service (port 9103)**  
- HTTP request metrics
- Session metrics
- Admin actions counter
- Payment processing metrics

**Rotator Service (port 9102)**
- Hash rotation frequency
- RabbitMQ connection status
- Rotation success/error rate

### Infrastructure Metrics
- **PostgreSQL Exporter** (port 9187): база данных, подключения, запросы
- **Redis Exporter** (port 9121): кеш, memory usage, операции
- **RabbitMQ Exporter** (port 9419): очереди, сообщения, подключения  
- **Node Exporter** (port 9100): системные метрики сервера

## База данных

### PostgreSQL
- Основная база данных: пользователи, игры, транзакции
- Миграции: автоматические при старте через migrate/migrate
- Connection Pool: GORM

### Redis  
- Кеширование: сессии, временные данные
- Persistence: AOF включен

## Переменные окружения

```bash
# Delegate builds to bake for better performance
COMPOSE_BAKE=true

# Docker образ
# for IMAGE_TAG=stage/release => IMAGE_NAME=git.traff.tools/sprut/roulette/apps
IMAGE_NAME=local/roulette-apps
IMAGE_TAG=dev

# Профили запуска
COMPOSE_PROFILES=tools,monitoring

# Database
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=roulette

# RabbitMQ
RABBITMQ_USER=guest
RABBITMQ_PASS=guest

# PgAdmin (optional)
# Also check file pgadmin4.config.json
PGADMIN_EMAIL=admin@admin.com
PGADMIN_PASSWORD=admin

# Redis Configuration
REDIS_PASSWORD=redis123
REDIS_DB=0

# Redis Commander
REDIS_COMMANDER_USER=admin
REDIS_COMMANDER_PASSWORD=admin

# Telegram
TELEGRAM_TOKEN=your_telegram_bot_token_here
TELEGRAM_NAME=your_telegram_bot_name_here
TELEGRAM_RESERVE_CHANNEL_ID=@your_telegram_channel

# Admin panel
ADMIN_USERNAME=admin
ADMIN_PASSWORD=secure_password_here
SESSION_SECRET=super_secret_key_here
ADMIN_PORT=8080
ALLOWED_IPS=127.0.0.1,::1,192.168.1.100
DISABLE_IP_FILTERS=true

# OxaPay
DEFAULT_PAYMENT_PROVIDER=mock
OXAPAY_API_KEY=your_api_key

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin

# Postgres Exporter
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=roulette
```

