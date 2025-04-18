package config

import (
	"fmt"
	"roulette/internal/payment"
	"roulette/internal/payment/providers/oxapay"

	oxapayclient "roulette/pkg/oxapay"

	"github.com/gin-gonic/gin"
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
	APIKey      string
	WebhookKey  string
	CallbackURL string
}

// InitPaymentProviders инициализирует провайдеры платежей
func InitPaymentProviders(db *gorm.DB, router *gin.Engine) (*payment.Factory, error) {
	paymentConfig := getPaymentConfig()

	// Создаем фабрику провайдеров
	factory := payment.NewFactory()

	// Инициализируем OxaPay
	if paymentConfig.OxaPay.APIKey != "" {
		oxaClient := oxapayclient.NewClient(oxapayclient.Config{
			APIKey:      paymentConfig.OxaPay.APIKey,
			WebhookKey:  paymentConfig.OxaPay.WebhookKey,
			CallbackURL: paymentConfig.OxaPay.CallbackURL,
			DB:          db,
		})

		// Инициализируем таблицы для OxaPay
		if err := oxaClient.InitializeTables(); err != nil {
			return nil, fmt.Errorf("failed to initialize OxaPay tables: %w", err)
		}

		// Настраиваем вебхуки для OxaPay
		oxaClient.SetupWebhookHandler(router, "/webhooks/payment/oxapay")

		// Создаем и регистрируем провайдер OxaPay
		oxaProvider := oxapay.NewProvider(
			paymentConfig.OxaPay.APIKey,
			paymentConfig.OxaPay.WebhookKey,
			paymentConfig.OxaPay.CallbackURL,
			db,
		)

		factory.RegisterProvider("oxapay", oxaProvider)
	}

	// Здесь можно инициализировать другие провайдеры

	return factory, nil
}

// getPaymentConfig получает конфигурацию платежных систем из переменных окружения
func getPaymentConfig() PaymentConfig {
	return PaymentConfig{
		DefaultProvider: getEnv("DEFAULT_PAYMENT_PROVIDER", "oxapay"),
		OxaPay: OxaPayConfig{
			APIKey:      getEnv("OXAPAY_API_KEY", ""),
			WebhookKey:  getEnv("OXAPAY_WEBHOOK_KEY", ""),
			CallbackURL: getEnv("OXAPAY_CALLBACK_URL", ""),
		},
	}
}
