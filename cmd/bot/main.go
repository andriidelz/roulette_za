package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"roulette/internal/bot"
	"roulette/internal/config"
	"roulette/internal/repository"
	"roulette/internal/service"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Завантажуємо змінні оточення
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Ініціалізуємо конфігурацію
	cfg := config.NewConfig()

	// Підключаємося до бази даних
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Створюємо репозиторій
	repo := repository.NewRepository(db)

	// Створюємо сервіс
	svc := service.NewService(repo)

	// Створюємо бота
	telegramBot, err := bot.NewBot(cfg.TelegramToken, svc)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Запускаємо бота
	if err := telegramBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// Очікуємо сигнал для завершення
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Зупиняємо бота
	telegramBot.Stop()
}
