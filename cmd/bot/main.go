package main

import (
	"os"
	"os/signal"
	"syscall"

	"roulette/internal/bot"
	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/metrics"
	"roulette/internal/repository"
	"roulette/internal/service"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		logger.Error.Printf("Error loading .env file: %v", err)
	}

	// Инициализируем конфигурацию
	cfg := config.NewConfig()

	logger.Info.Printf("Starting bot with Telegram token: %s...", cfg.TelegramToken[:5])

	// Инициализируем сервер метрик
	appMetrics := metrics.NewMetrics("roulette-bot", 9101)
	if err := appMetrics.Start(); err != nil {
		logger.Error.Fatalf("Failed to start metrics server: %v", err)
	}

	// Подключаемся к базе данных
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error.Fatalf("Failed to connect to database: %v", err)
	}

	// Создаем репозиторий
	repo := repository.NewRepository(db)

	// Создаем сервис
	svc := service.NewService(repo, cfg.TelegramToken)

	// Создаем бота с URL для RabbitMQ
	telegramBot, err := bot.NewBot(cfg.TelegramToken, svc, cfg)
	if err != nil {
		logger.Error.Fatalf("Failed to create bot: %v", err)
	}

	// Запускаем бота
	if err := telegramBot.Start(); err != nil {
		logger.Error.Fatalf("Failed to start bot: %v", err)
	}

	logger.Info.Printf("Bot started successfully and listening for updates")

	// Ожидаем сигнал для завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Останавливаем бота
	telegramBot.Stop()

	// Останавливаем сервер метрик
	if err := appMetrics.Stop(); err != nil {
		logger.Error.Printf("Error stopping metrics server: %v", err)
	}

	logger.Info.Println("Bot stopped gracefully")
}
