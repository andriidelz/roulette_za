package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config структура конфігурації проекту
type Config struct {
	// База даних
	DatabaseURL string

	// Телеграм
	TelegramToken string

	// Адмін-панель
	AdminPort        string
	AdminUsername    string
	AdminPassword    string
	SessionSecret    string
	AllowedIPs       []string
	DisableIPFilters bool

	RotationInterval time.Duration
}

// NewConfig створює новий екземпляр конфігурації
func NewConfig() *Config {

	// Інтервал ротації за замовчуванням - 10 секунд
	rotationInterval := 30 * time.Second

	return &Config{
		// База даних
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/roulette?sslmode=disable"),

		// Телеграм
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),

		// Адмін-панель
		AdminPort:        getEnv("ADMIN_PORT", "8080"),
		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin"),
		SessionSecret:    getEnv("SESSION_SECRET", "super-secret-key"),
		AllowedIPs:       getEnvStringSlice("ALLOWED_IPS", []string{"127.0.0.1"}),
		DisableIPFilters: getEnvBool("DISABLE_IP_FILTERS", false),

		RotationInterval: rotationInterval,
	}
}

// getEnv отримує значення змінної оточення або повертає значення за замовчуванням
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvBool отримує булеве значення змінної оточення або повертає значення за замовчуванням
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvStringSlice отримує масив рядків з змінної оточення або повертає значення за замовчуванням
func getEnvStringSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.Split(value, ",")
	}
	return defaultValue
}
