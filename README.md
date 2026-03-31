# Roulette Bot

Telegram bot for social casino on roulette game principle.

## Project's description

Roulette Bot - is a Telegram-bot, which allows users to play a simplified version of roulette with the option to bet on red, black, or zero. The system includes weekly rankings, prize pool distribution, and a multilingual interface.

## Main possibilities

- **Simplified Roulette**: bet on red, black, or zero
- **Weekly player rankings** with prize pool distribution
- **Super ranking**: formed based on weekly rankings
- **Multilingual support**: Ukrainian, English, and Russian
- **Fairness verification system**: transparent mechanism for verifying game results
- **Withdrawal of winnings**: automated and controlled withdrawals
- **Admin panel**: user-friendly project management interface

## Project's architecture

It consists of three main components:

1. **Telegram Bot**: The main user interface
2. **Hash Rotator**: Generates and manages roulette rounds
3. **Admin Panel**: A web interface for managing the project

Components interact via PostgreSQL and RabbitMQ.

## Project's structure

```
roulette-bot/
├── cmd/                    # Executable points
│   ├── bot/                # Bot launching
│   ├── admin/              # Admin-panel launching
│   └── rotator/            # Hashes' rotator launching
├── internal/               # Inner code
│   ├── admin/              # Admin-panel
│   ├── bot/                # Bot's logic
│   ├── config/             # Configuration
│   ├── conv/               # Conversion utilities
│   ├── data/               # Datas (countries' and etc.)
│   ├── i18n/               # Internationalization
│   ├── logger/             # Logging
│   ├── messaging/          # Work with RabbitMQ
│   ├── models/             # Data's models
│   ├── repository/         # Work with DB
│   ├── rotator/            # hashes' rotator
│   ├── service/            # Business logic
│   └── utils/              # Utilities
├── migrations/             # PostgreSQL migrations
├── shared-data/            # Docker volumes data + files
├── web/                    # Web-resourses
│   ├── templates/          # Templates
│   └── static/             # Static files
├── .env.example            # Sample of alternate environment
├── docker-compose.yml      # Docker Compose configuration
├── Dockerfile              # Dockerfile
├── go.mod                  # Go module
├── go.sum                  # Go module checksum
├── Makefile                # Makefile for convenient command
└── README.md               # Documentation
```

## Tech requirement

- Go 1.18+
- PostgreSQL 12+
- Telegram Bot API Token
- Docker (for containerization)

## Setting and launching

### Local launch

1. Clone repo:

   ```bash
   git clone https://github.com/your-username/roulette-bot.git
   cd roulette-bot
   ```

2. Init project:

   ```bash
   make init
   ```

3. Edit file `.env` with your devices:

   ```
   TELEGRAM_TOKEN=your_token_here
   TELEGRAM_NAME=your_bot_name
   DATABASE_URL=postgres://postgres:postgres@localhost:5432/roulette?sslmode=disable
   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=secure_password
   SESSION_SECRET=your_session_secret
   ```

   
The folder needs to be filled in:
shared-data/files - contains files necessary for the bot to function.
Currently, it contains videos, the source files of which are sent from the bot in messages and stored in Redis, with the key game:animation

4. Create database and accomplish migrations:

   ```bash
   make migrate
   ```

5. Launch comments:

   ```bash
   # Launch all components
   make run

   # Launch segregate components
   make run-bot
   make run-admin
   make run-rotator
   ```

### Launch via Docker

1. Clone repo:

   ```bash
   git clone https://github.com/your-username/roulette-bot.git
   cd roulette-bot
   ```

2. Create and set up file `.env`:

   ```
   TELEGRAM_TOKEN=your_token_here
   TELEGRAM_NAME=your_bot_name
   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=secure_password
   SESSION_SECRET=your_session_secret
   ```

   The folder needs to be filled in:
shared-data/files - contains files necessary for the bot to function.
Currently, it contains videos, the source files of which are sent from the bot in messages and stored in Redis, with the key game:animation

   When copying the entire shared-data folder, keep in mind that redis contains local data that is specific to the telegram bot (each telegram bot has its own unique file_id) - when transferring from another bot, you need to delete the `DEL "game:animation"` keys

4. Launch via Docker Compose:

   ```bash
   docker-compose up -d
   ```

5. Admin-panel is available on address: <http://localhost:8080>

## Usage

### Bot's commands

- `/start` - Beginning of work with a bot
- `/play` - Launch game
- `/statistics` - Watch statistics
- `/rating` - Watch weekly rate
- `/account` - Account management
- `/faq` - Frequent questions
- `/settings` - Profile settings

### Admin-panel

The admin panel provides the following features:

- **User Management**: View, block/unblock, edit
- **Statistics**: Overview of overall stats, betting success, top players
- **Ratings**: Manage weekly and super-ratings
- **Prize Pools**: Configure and distribute prizes
- **Localizations**: Manage language strings
- **Withdrawal Processing**: Review and approve withdrawal requests
- **Hash Verification**: Verify roulette results

## Fair play system

To ensure the fairness of the game, a pre-hashing system of results is used:

1. Before the start of the round, a random number between 0 and 36 is generated.
2. The number is hashed along with a cryptographic salt.
3. The hash is published at the start of the round.
4. After the round ends, the number and salt are revealed.
5. Users can verify that the hash matches the revealed number and salt.

With this system, the outcome cannot be changed once the round has begun and remains unpredictable for players until its completion.

### Checking the result

Users and administrators can independently verify the fairness of the results:

1. Obtain the number and salt from the bot after the round is completed.
2. Create a string in the format: `[number]:[salt]` (e.g., `5:a1b2c3d4e5f6`).
3. Calculate the SHA-256 hash of this string.
4. Compare the resulting hash with the hash provided at the beginning of the round.

If the hashes match, the result is fair.

#### Test example (Go)

```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func main() {
    number := 5                  // The number that came up
    salt := "a1b2c3d4e5f6"       // Salt from the bot message
    originalHash := "e37f...0840" // Hash from the beginning of the round
    
    // We form a string and calculate the hash
    data := fmt.Sprintf("%d:%s", number, salt)
    hasher := sha256.New()
    hasher.Write([]byte(data))
    computedHash := hex.EncodeToString(hasher.Sum(nil))
    
    // Checking the match
    if computedHash == originalHash {
        fmt.Println("Result is right!")
    } else {
        fmt.Println("The result does not match the hash!")
    }
}
```

## Setup and administration

### Configuration parameters

Basic settings are available through environment variables:

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `TELEGRAM_TOKEN` | Telegram bot token | - |
| `TELEGRAM_NAME` | Telegram bot name | - |
| `DATABASE_URL` | PostgreSQL connection URL | `postgres://postgres:postgres@localhost:5432/roulette?sslmode=disable` |
| `ADMIN_PORT` | Admin panel port | `8080` |
| `ADMIN_USERNAME` | Admin panel username | `admin` |
| `ADMIN_PASSWORD` | Admin panel password | `admin` |
| `SESSION_SECRET` | Session Secret | `super-secret-key` |
| `RABBITMQ_URL` | URL for connecting to RabbitMQ | `amqp://guest:guest@rabbitmq:5672/` |
| `ROTATION_INTERVAL` | Round rotation interval | `30s` |

### Game Settings

The following parameters can be configured through the admin panel:

- **Bet Limit** per user per day
- **Zero Bet Limit**: the minimum number of regular bets to bet on Zero
- **Prize Pool Amount** for the Weekly Leaderboard
- **Number of Prize Places** in the Leaderboard
- **Minimum Withdrawal Amount**
- **Prize Distribution Day and Time**

## Useful commands Makefile

- `make build` - Build the project
- `make run-bot` - Run the bot
- `make run-admin` - Run the admin panel
- `make run-rotator` - Run the hash rotator
- `make run` - Run all services
- `make migrate` - Apply initial migrations
- `make migrate-update` - Apply the schema update migration
- `make docker` - Build Docker images
- `make docker-up` - Run via Docker Compose
- `make docker-down` - Stop Docker Compose
- `make clean` - Clean builds
- `make test` - Run tests
- `make init` - Initialize the project

## Technologies

- [Go](https://golang.org/) - Primary programming language
- [Telego](https://github.com/mymmrac/telego) - Library for the Telegram Bot API
- [Gin](https://github.com/gin-gonic/gin) - Web framework for the admin panel
- [GORM](https://gorm.io/) - ORM for working with the database
- [PostgreSQL](https://www.postgresql.org/) - Relational database
- [RabbitMQ](https://www.rabbitmq.com/) - Messaging system
- [Docker](https://www.docker.com/) - Containerization
- [Bootstrap](https://getbootstrap.com/) - Framework for the admin panel interface

## Scalability and performance

The project's architecture allows for easy system scaling as the load increases:

1. **Horizontal scaling**: multiple instances of the bot and admin panel can be launched
2. **Database partitioning**: records can be separated by time or by user
3. **RabbitMQ optimization**: customization for specific usage patterns

## Monitoring and maintenance

For effective monitoring, we recommend:

1. Configure logging to files or a log collection service.
2. Create regular database backups.
3. Monitor resource usage using Prometheus/Grafana.
4. Set up alerts for critical situations (service failures, database problems).

## Additional information

For further information, please contact the author.
