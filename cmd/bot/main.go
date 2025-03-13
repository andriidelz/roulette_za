package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"roulette/internal/bot"
	"roulette/internal/config"
	"roulette/internal/repository"
	"roulette/internal/rotator"
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

	// Получаем обработчик игры из бота
	gameHandler := telegramBot.GetGameHandler()

	// Создаем ротатор с интервалом 30 секунд
	rotationInterval := 30 * time.Second
	hashRotator := rotator.NewRotator(svc, rotationInterval)

	// Регистрируем игровой обработчик для получения уведомлений о новых хешах
	hashRotator.RegisterNotifier(gameHandler)

	// Запускаем ротатор в фоновом режиме
	go hashRotator.Start()

	// Запускаем бота
	if err := telegramBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// Ожидаем сигнал для завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Останавливаем бота и ротатор
	telegramBot.Stop()
	hashRotator.Stop()

	log.Println("Bot and rotator stopped gracefully")
}
