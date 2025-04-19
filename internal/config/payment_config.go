package config

import (
	"fmt"
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
		oxaClient := oxapayclient.NewClient(oxapayclient.Config{
			APIKey: paymentConfig.OxaPay.APIKey,
			DB:     db,
		})

		// Инициализируем таблицы для OxaPay
		if err := oxaClient.InitializeTables(); err != nil {
			return nil, defaultProvider, fmt.Errorf("failed to initialize OxaPay tables: %w", err)
		}

		// Создаем и регистрируем провайдер OxaPay
		oxaProvider := oxapay.NewProvider(
			paymentConfig.OxaPay.APIKey,
			db,
		)

		factory.RegisterProvider("oxapay", oxaProvider)

		// Если defaultProvider не установлен или установлен в "oxapay",
		// и OxaPay успешно инициализирован - используем его по умолчанию
		if defaultProvider == "mock" || defaultProvider == "oxapay" {
			defaultProvider = "oxapay"
		}
	}

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
