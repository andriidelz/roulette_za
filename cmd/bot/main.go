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
	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Инициализируем конфигурацию
	cfg := config.NewConfig()

	// Подключаемся к базе данных
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Создаем репозиторий
	repo := repository.NewRepository(db)

	// Создаем сервис
	svc := service.NewService(repo)

	// Создаем бота
	telegramBot, err := bot.NewBot(cfg.TelegramToken, svc)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Удалены строки создания и запуска ротатора

	// Запускаем бота
	if err := telegramBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// Ожидаем сигнал для завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Останавливаем бота
	telegramBot.Stop()

	log.Println("Bot stopped gracefully")
}
