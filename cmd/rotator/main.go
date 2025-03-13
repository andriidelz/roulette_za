package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"roulette/internal/config"
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

	// Створюємо ротатор з налаштуваннями інтервалу
	hashRotator := rotator.NewRotator(svc, cfg.RotationInterval)

	// Запускаємо ротатор у фоновому режимі
	go hashRotator.Start()

	// Виводимо повідомлення про запуск
	log.Printf("Rotator started with interval: %s", cfg.RotationInterval)

	// Очікуємо сигнал для завершення
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down rotator...")

	// Зупиняємо ротатор
	hashRotator.Stop()

	// Закриваємо з'єднання з базою даних
	if err := repo.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	} else {
		log.Println("Database connection closed")
	}

	log.Println("Rotator shutdown complete")
}
