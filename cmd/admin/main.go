package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"roulette/internal/admin"
	"roulette/internal/config"
	"roulette/internal/payment"
	"roulette/internal/payment/providers/oxapay"
	"roulette/internal/repository"
	"roulette/internal/service"
	oxapayclient "roulette/pkg/oxapay"

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
	svc := service.NewService(repo, cfg.TelegramToken)

	// Инициализируем фабрику платежных провайдеров
	paymentFactory := payment.NewFactory()

	// Регистрируем провайдеры (здесь должна быть аналогичная логика из InitPaymentProviders)
	// Для простоты можем оставить пустую фабрику или зарегистрировать моки

	// Регистрируем мок-провайдер для тестирования
	mockProvider := payment.NewMockProvider()
	paymentFactory.RegisterProvider("mock", mockProvider)

	defaultProvider := "mock" // Используем мок по умолчанию

	oxapayAPIKey := os.Getenv("OXAPAY_API_KEY")
	oxapayWebhookKey := os.Getenv("OXAPAY_WEBHOOK_KEY")
	oxapayCallbackURL := os.Getenv("OXAPAY_CALLBACK_URL")

	if oxapayAPIKey != "" && oxapayWebhookKey != "" {
		// Инициализируем клиент OxaPay
		oxaConfig := oxapayclient.Config{
			APIKey:      oxapayAPIKey,
			WebhookKey:  oxapayWebhookKey,
			CallbackURL: oxapayCallbackURL, // Передаем полный URL-колбэка
			DB:          db,
		}

		oxaClient := oxapayclient.NewClient(oxaConfig)

		// Инициализируем таблицы для OxaPay, если необходимо
		if err := oxaClient.InitializeTables(); err != nil {
			log.Printf("Warning: Failed to initialize OxaPay tables: %v", err)
		}

		// Создаем провайдер OxaPay с полным URL-колбэка
		oxaProvider := oxapay.NewProvider(
			oxapayAPIKey,
			oxapayWebhookKey,
			oxapayCallbackURL,
			db,
		)

		// Регистрируем провайдер в фабрике
		paymentFactory.RegisterProvider("oxapay", oxaProvider)

		// Используем OxaPay как провайдер по умолчанию, если он настроен
		defaultProvider = "oxapay"

		log.Printf("OxaPay payment provider initialized successfully with callback URL: %s", oxapayCallbackURL)
	} else {
		log.Printf("OxaPay API credentials not found or incomplete, using mock provider")
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
	}

	// Створюємо адмін-панель
	adminPanel := admin.NewAdminPanel(svc, repo, adminSettings, paymentSvc)

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

// Функция для регистрации мок-провайдера в фабрике
func registerMockProvider(factory *payment.Factory) {
	mockProvider := payment.NewMockProvider()
	factory.RegisterProvider("mock", mockProvider)
}
