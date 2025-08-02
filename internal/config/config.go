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
	TelegramToken            string
	TelegramName             string // название бота
	TelegramReserveChannelID string

	// Адмін-панель
	AdminPort        string
	AdminUsername    string
	AdminPassword    string
	SessionSecret    string
	AllowedIPs       []string
	DisableIPFilters bool

	// Rotator
	RotationInterval time.Duration

	// RabbitMQ
	RabbitMQURL string

	// Redis
	RedisHost string
	RedisPort string
	RedisPass string
	RedisDB   int
}

// NewConfig створює новий екземпляр конфігурації
func NewConfig() *Config {
	// Інтервал ротації за замовчуванням - 30 секунд
	rotationInterval := 30 * time.Second

	return &Config{
		// База даних
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/roulette?sslmode=disable"),

		// Телеграм
		TelegramToken:            getEnv("TELEGRAM_TOKEN", ""),
		TelegramName:             getEnv("TELEGRAM_NAME", ""),
		TelegramReserveChannelID: getEnv("TELEGRAM_RESERVE_CHANNEL_ID", ""),

		// Адмін-панель
		AdminPort:        getEnv("ADMIN_PORT", "8080"),
		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin"),
		SessionSecret:    getEnv("SESSION_SECRET", "super-secret-key"),
		AllowedIPs:       getEnvStringSlice("ALLOWED_IPS", []string{"127.0.0.1"}),
		DisableIPFilters: getEnvBool("DISABLE_IP_FILTERS", false),

		// Rotator
		RotationInterval: rotationInterval,

		// RabbitMQ
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),
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

// getEnvInt отримує int значення змінної оточення або повертає значення за замовчуванням
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.Atoi(value)
		if err == nil {
			return intValue
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
