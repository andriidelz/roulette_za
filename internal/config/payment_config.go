package config

import (
	"fmt"
	"roulette/internal/logger"
	"roulette/internal/payment"
	"roulette/internal/payment/providers/oxapay"

	oxapayclient "roulette/pkg/oxapay"

	"gorm.io/gorm"
)

// PaymentConfig содержит конфигурацию для платежных систем
type PaymentConfig struct {
	DefaultProvider string
	OxaPay          OxaPayConfig
	// Можно добавить другие провайдеры здесь
}

// OxaPayConfig содержит конфигурацию для OxaPay
type OxaPayConfig struct {
	APIKey string
}

// InitPaymentProviders инициализирует провайдеры платежей
func InitPaymentProviders(db *gorm.DB) (*payment.Factory, string, error) {
	paymentConfig := getPaymentConfig()
	logger.Info.Printf("Initializing payment providers with default: %s", paymentConfig.DefaultProvider)

	// Создаем фабрику провайдеров
	factory := payment.NewFactory()

	// Инициализируем мок-провайдер по умолчанию
	mockProvider := payment.NewMockProvider()
	factory.RegisterProvider("mock", mockProvider)

	defaultProvider := paymentConfig.DefaultProvider

	// Если defaultProvider не указан, используем "mock"
	if defaultProvider == "" {
		defaultProvider = "mock"
	}

	// Инициализируем OxaPay только если есть API ключ
	if paymentConfig.OxaPay.APIKey != "" {
		logger.Info.Printf("Found OxaPay API key, initializing OxaPay provider")

		oxaClient := oxapayclient.NewClient(oxapayclient.Config{
			APIKey: paymentConfig.OxaPay.APIKey,
			DB:     db,
		})

		// Инициализируем таблицы для OxaPay
		if err := oxaClient.InitializeTables(); err != nil {
			logger.Error.Printf("Failed to initialize OxaPay tables: %v", err)
			return nil, defaultProvider, fmt.Errorf("failed to initialize OxaPay tables: %w", err)
		}

		// Создаем и регистрируем провайдер OxaPay
		oxaProvider := oxapay.NewProvider(
			paymentConfig.OxaPay.APIKey,
			db,
		)

		factory.RegisterProvider("oxapay", oxaProvider)
		logger.Info.Printf("Registered OxaPay payment provider")

		// Если defaultProvider не установлен или установлен в "oxapay",
		// и OxaPay успешно инициализирован - используем его по умолчанию
		if defaultProvider == "mock" || defaultProvider == "oxapay" {
			defaultProvider = "oxapay"
		}
	}

	logger.Info.Printf("Payment providers initialized, using default provider: %s", defaultProvider)
	return factory, defaultProvider, nil
}

// getPaymentConfig получает конфигурацию платежных систем из переменных окружения
func getPaymentConfig() PaymentConfig {
	return PaymentConfig{
		DefaultProvider: getEnv("DEFAULT_PAYMENT_PROVIDER", ""),
		OxaPay: OxaPayConfig{
			APIKey: getEnv("OXAPAY_API_KEY", ""),
		},
	}
}
