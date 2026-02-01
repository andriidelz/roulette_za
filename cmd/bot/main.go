package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	_ "net/http/pprof"

	"roulette/internal/bot"
	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/repository"
	"roulette/internal/service"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func startDebug() {
	runtime.SetBlockProfileRate(10000) // 1 подія на ~10µs блокування (підкручується)
	runtime.SetMutexProfileFraction(10)

	go func() {
		_ = http.ListenAndServe("127.0.0.1:6060", nil)
	}()
}

func main() {
	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		logger.Error.Printf("Error loading .env file: %v", err)
	}

	// Инициализируем конфигурацию
	cfg := config.NewConfig()

	logger.Info.Printf("Starting bot with Telegram token: %s...", cfg.TelegramToken[:5])

	// Подключаемся к базе данных
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error.Fatalf("Failed to connect to database: %v", err)
	}

	// Оптимизируем настройки БД
	optimizeDatabase(db)

	// Создаем репозиторий
	repo := repository.NewRepository(db)

	// Создаем Redis клиент
	redisDB := bot.NewRedisClient(cfg)

	// Создаем сервис
	svc := service.NewService(repo, cfg.TelegramToken, redisDB)

	// Создаем бота с URL для RabbitMQ (метрики инициализируются внутри бота)
	telegramBot, err := bot.NewBot(cfg.TelegramToken, svc, cfg)
	if err != nil {
		logger.Error.Fatalf("Failed to create bot: %v", err)
	}

	// Запускаем бота
	if err := telegramBot.Start(); err != nil {
		logger.Error.Fatalf("Failed to start bot: %v", err)
	}

	startDebug()

	logger.Info.Printf("Bot started successfully and listening for updates")

	// Ожидаем сигнал для завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Останавливаем бота (метрики остановятся автоматически)
	telegramBot.Stop()

	logger.Info.Println("Bot stopped gracefully")
}

func optimizeDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error.Printf("Failed to get database instance: %v", err)
		return
	}

	sqlDB.SetMaxOpenConns(180)
	sqlDB.SetMaxIdleConns(90)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	db = db.Session(&gorm.Session{
		PrepareStmt:          true,
		FullSaveAssociations: false,
	})
}
