package main

import (
	"os"
	"os/signal"
	"syscall"

	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/repository"
	"roulette/internal/rotator"
	"roulette/internal/service"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Завантажуємо змінні оточення
	if err := godotenv.Load(); err != nil {
		logger.Error.Printf("Error loading .env file: %v", err)
	}

	// Ініціалізуємо конфігурацію
	cfg := config.NewConfig()

	// Підключаємося до бази даних
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error.Fatalf("Failed to connect to database: %v", err)
	}

	// Створюємо репозиторій
	repo := repository.NewRepository(db)

	// Створюємо сервіс
	svc := service.NewService(repo, cfg.TelegramToken)

	// Створюємо ротатор з налаштуваннями інтервалу та RabbitMQ
	hashRotator, err := rotator.NewRotator(svc, cfg.RotationInterval, cfg.RabbitMQURL)
	if err != nil {
		logger.Error.Fatalf("Failed to create rotator: %v", err)
	}

	// Запускаємо ротатор у фоновому режимі
	go hashRotator.Start()

	// Виводимо повідомлення про запуск
	logger.Info.Printf("Rotator started with interval: %s, connected to RabbitMQ: %s",
		cfg.RotationInterval, cfg.RabbitMQURL)

	// Очікуємо сигнал для завершення
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info.Println("Shutting down rotator...")

	// Зупиняємо ротатор
	hashRotator.Stop()

	// Закриваємо з'єднання з базою даних
	if err := repo.Close(); err != nil {
		logger.Error.Printf("Error closing database connection: %v", err)
	} else {
		logger.Info.Println("Database connection closed")
	}

	logger.Info.Println("Rotator shutdown complete")
}
