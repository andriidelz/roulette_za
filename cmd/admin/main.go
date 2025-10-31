package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"roulette/internal/admin"
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
	// Завантажуємо змінні оточення
	if err := godotenv.Load(); err != nil {
		logger.Error.Printf("Error loading .env file: %v", err)
	}

	// Ініціалізуємо конфігурацію
	cfg := config.NewConfig()

	// Инициализируем сервер метрик
	appMetrics := metrics.NewMetrics("roulette-admin", 9103, metrics.AppTypeAdmin)
	go func() {
		if err := appMetrics.Start(); err != nil {
			logger.Error.Printf("Failed to start metrics server: %v", err)
			// НЕ используем Fatalf - просто логируем ошибку
		}
	}()

	// Підключаємося до бази даних
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error.Fatalf("Failed to connect to database: %v", err)
	}

	// Створюємо репозиторій
	repo := repository.NewRepository(db)

	// Створюємо сервіс
	svc := service.NewService(repo, cfg.TelegramToken, nil)

	// IMPORTANT: Ensure RabbitMQ connection is closed on shutdown
	defer func() {
		logger.Info.Println("Closing service connections...")
		if err := svc.Close(); err != nil {
			logger.Error.Printf("Error closing service: %v", err)
		}
	}()

	// Инициализируем фабрику платежных провайдеров
	paymentFactory, defaultProvider, err := config.InitPaymentProviders(db)
	if err != nil {
		logger.Error.Printf("Warning: Failed to initialize payment providers: %v", err)
	}

	// Создаем PaymentService
	paymentSvc := service.NewPaymentService(repo, paymentFactory, defaultProvider)

	// Створюємо налаштування для адмін-панелі
	adminSettings := &admin.Settings{
		Port:             cfg.AdminPort,
		SessionSecret:    cfg.SessionSecret,
		AdminUsername:    cfg.AdminUsername,
		AdminPassword:    cfg.AdminPassword,
		AllowedIPs:       cfg.AllowedIPs,
		DisableIPFilters: cfg.DisableIPFilters,
		BotName:          cfg.TelegramName,
	}

	// Створюємо адмін-панель
	adminPanel := admin.NewAdminPanel(svc, repo, adminSettings, paymentSvc)

	// Запускаємо адмін-панель у фоновому режимі
	go func() {
		if err := adminPanel.Start(); err != nil {
			logger.Error.Fatalf("Failed to start admin panel: %v", err)
		}
	}()

	// Запускаем фоновую проверку статусов выплат (каждые 3 минуты)
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			logger.Info.Println("Starting scheduled check of pending withdrawals...")
			if err := paymentSvc.CheckPendingWithdrawals(); err != nil {
				logger.Error.Printf("Error checking pending withdrawals: %v", err)
			}
		}
	}()

	// Запускаем планировщик проверки пользователей в топе рейтинга (каждый час)
	topRatingTicker := time.NewTicker(2 * time.Minute)
	go func() {
		for range topRatingTicker.C {
			now := time.Now()
			hour := now.Hour()
			minute := now.Minute()

			if hour == 20 && minute >= 0 && minute <= 30 { // Окно отправки 20:00-20:30
				logger.Info.Println("Starting scheduled check of top rating entries...")
				if err := svc.CheckTopRatingEntries(); err != nil {
					logger.Error.Printf("Error checking top rating entries: %v", err)
				}

				logger.Info.Println("Starting scheduled unregistered entries...")
				if err := svc.CheckPendingRegistration(); err != nil {
					logger.Error.Printf("Error checking top rating entries: %v", err)
				}
			}
		}
	}()

	// Запускаем планировщик проверки запланированных уведомлений (каждую минуту)
	notificationTaskTicker := time.NewTicker(1 * time.Minute)
	go func() {
		for range notificationTaskTicker.C {
			logger.Info.Println("Starting scheduled check of pending notification tasks...")
			// Получаем задачи, запланированные на текущее время
			pendingTasks, err := svc.GetPendingNotificationTasks()
			if err != nil {
				logger.Error.Printf("Error getting pending notification tasks: %v", err)
				continue
			}

			// Запускаем отправку для каждой задачи
			for _, task := range pendingTasks {
				if err := svc.SendNotifications(task.ID); err != nil {
					logger.Error.Printf("Error sending notifications for task %d: %v", task.ID, err)
				} else {
					logger.Info.Printf("Successfully started notification task %d", task.ID)
				}
			}
		}
	}()

	// Виводимо повідомлення про запуск
	logger.Info.Printf("Admin panel started on http://localhost:%s", cfg.AdminPort)

	// Очікуємо сигнал для завершення
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Останавливаем ticker перед выходом
	ticker.Stop()
	topRatingTicker.Stop()
	notificationTaskTicker.Stop()

	// Останавливаем сервер метрик
	if appMetrics != nil {
		if err := appMetrics.Stop(); err != nil {
			logger.Error.Printf("Error stopping metrics server: %v", err)
		}
	}

	logger.Info.Println("Admin panel stopped gracefully")
}
