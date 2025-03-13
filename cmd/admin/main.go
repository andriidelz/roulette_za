package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"roulette/internal/admin"
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

	// Створюємо налаштування для адмін-панелі
	adminSettings := &admin.Settings{
		Port:             cfg.AdminPort,
		SessionSecret:    cfg.SessionSecret,
		AdminUsername:    cfg.AdminUsername,
		AdminPassword:    cfg.AdminPassword,
		AllowedIPs:       cfg.AllowedIPs,
		DisableIPFilters: cfg.DisableIPFilters,
	}

	// Створюємо адмін-панель
	adminPanel := admin.NewAdminPanel(svc, repo, adminSettings)

	// Запускаємо адмін-панель у фоновому режимі
	go func() {
		if err := adminPanel.Start(); err != nil {
			log.Fatalf("Failed to start admin panel: %v", err)
		}
	}()

	// Виводимо повідомлення про запуск
	log.Printf("Admin panel started on http://localhost:%s", cfg.AdminPort)

	// Очікуємо сигнал для завершення
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down admin panel...")
}
