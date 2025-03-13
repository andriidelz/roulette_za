# Телеграм-бот Рулетка (MVP)

Телеграм-бот для соціального казино на основі спрощеного принципу гри в рулетку.
Користувачі можуть робити ставки на червоне, чорне або зеро, змагатися в рейтингу та отримувати призи.

## Особливості

- Спрощена гра в рулетку (червоне/чорне/зеро)
- Тижневий рейтинг гравців
- Супер-рейтинг (на основі позицій у тижневих рейтингах)
- Розподіл призового фонду
- Система локалізації (українська та англійська)
- Адмін-панель для керування проектом

## Структура проекту

```
roulette-bot/
├── cmd/                    # Точки входу
│   ├── bot/                # Запуск бота
│   └── admin/              # Запуск адмін-панелі
├── internal/               # Внутрішній код
│   ├── config/             # Конфігурація
│   ├── models/             # Моделі даних
│   ├── repository/         # Робота з БД
│   ├── service/            # Бізнес-логіка
│   ├── bot/                # Логіка бота
│   └── admin/              # Адмін-панель
├── migrations/             # Міграції PostgreSQL
├── web/                    # Веб-ресурси
│   ├── templates/          # Шаблони
│   └── static/             # Статичні файли
├── .env.example            # Приклад змінних оточення
├── docker-compose.yml      # Docker Compose
├── Dockerfile              # Dockerfile
├── go.mod                  # Go модуль
├── go.sum                  # Go модуль чексуми
├── Makefile                # Makefile для зручних команд
└── README.md               # Документація
```

## Вимоги

- Go 1.18+
- PostgreSQL 12+
- Telegram Bot API Token

## Встановлення та запуск

### Локальний запуск

1. Клонуйте репозиторій:

   ```bash
   git clone https://roulette.git
   cd roulette-bot
   ```

2. Ініціалізуйте проект:

   ```bash
   make init
   ```

3. Відредагуйте файл `.env` і додайте свій Telegram Bot Token:

   ```
   TELEGRAM_TOKEN=your_token_here
   ```

4. Створіть базу даних і виконайте міграції:

   ```bash
   make migrate
   ```

5. Запустіть бота та адмін-панель:

   ```bash
   make run
   ```

### Запуск через Docker

1. Клонуйте репозиторій:

   ```bash
   git clone https://roulette.git
   cd roulette-bot
   ```

2. Створіть файл `.env` з вашим Telegram Bot Token:

   ```
   TELEGRAM_TOKEN=your_token_here
   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=secure_password
   SESSION_SECRET=your_secret_key
   ```

3. Запустіть через Docker Compose:

   ```bash
   docker-compose up -d
   ```

4. Адмін-панель буде доступна за адресою: <http://localhost:8080>

## Використання

### Команди бота

- `/start` - Початок роботи з ботом
- `/play` - Почати гру
- `/profile` - Переглянути профіль
- `/stats` - Переглянути статистику
- `/rating` - Переглянути тижневий рейтинг
- `/superrating` - Переглянути супер-рейтинг
- `/balance` - Переглянути баланс
- `/withdraw` - Вивести кошти
- `/faq` - Часті питання

### Адмін-панель

- Управління користувачами
- Перегляд статистики
- Управління рейтингами
- Налаштування призових фондів
- Управління локалізаціями
- Обробка запитів на виведення коштів

## Технології

- [Go](https://golang.org/) - Мова програмування
- [Telego](https://github.com/mymmrac/telego) - Бібліотека для Telegram Bot API
- [Gin](https://github.com/gin-gonic/gin) - Веб-фреймворк для адмін-панелі
- [GORM](https://gorm.io/) - ORM для роботи з базою даних
- [PostgreSQL](https://www.postgresql.org/) - База даних
- [Docker](https://www.docker.com/) - Контейнеризація

## Розробка

### Корисні команди

- `make build` - Зібрати проект
- `make run-bot` - Запустити бота
- `make run-admin` - Запустити адмін-панель
- `make run` - Запустити обидва сервіси
- `make migrate` - Застосувати міграції
- `make docker` - Зібрати Docker образи
- `make docker-up` - Запустити через Docker Compose
- `make docker-down` - Зупинити Docker Compose
- `make clean` - Очистити збірки
