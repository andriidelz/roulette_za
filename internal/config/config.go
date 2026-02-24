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
	TelegramAPIURL           string
	TelegramDebugMode        bool
	TelegramTestMode         bool
	TelegramToken            string
	TelegramName             string // название бота
	TelegramReserveChannelID string

	// Webhook configuration
	WebhookEnabled     bool   // Enable webhook mode (false = long polling)
	WebhookURL         string // Public webhook URL (e.g., https://yourdomain.com)
	WebhookListenAddr  string // Local address to listen on (e.g., :8443)
	WebhookPath        string // Path for webhook (e.g., /webhook)
	WebhookSecretToken string // Secret token for X-Telegram-Bot-Api-Secret-Token header validation

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

	// Profiling/Debug
	PprofEnabled              bool
	PprofAddr                 string
	PprofBlockProfileRate     int
	PprofMutexProfileFraction int

	DebugMode bool
}

const (
	UserStatusActive   string = "ACTIVE"
	UserStatusDisabled string = "DISABLED" //присвоюється користувачу який забанив бота
	UserStatusBanned   string = "BANNED"
	UserStatusLockout  string = "LOCKOUT"
	UserStatusCaptcha  string = "CAPTCHA"
)

// NewConfig створює новий екземпляр конфігурації
func NewConfig() *Config {
	// Інтервал ротації за замовчуванням - 30 секунд
	rotationInterval := 30 * time.Second

	return &Config{
		// База даних
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/roulette?sslmode=disable"),

		// Телеграм

		TelegramAPIURL:           getEnv("TELEGRAM_API_URL", "https://api.telegram.org"),
		TelegramDebugMode:        getEnvBool("TELEGRAM_DEBUG_MODE", false),
		TelegramTestMode:         getEnvBool("TELEGRAM_TEST_MODE", false),
		TelegramToken:            getEnv("TELEGRAM_TOKEN", ""),
		TelegramName:             getEnv("TELEGRAM_NAME", ""),
		TelegramReserveChannelID: getEnv("TELEGRAM_RESERVE_CHANNEL_ID", ""),

		// Webhook settings
		WebhookEnabled:     getEnv("WEBHOOK_ENABLED", "false") == "true",
		WebhookURL:         getEnv("WEBHOOK_URL", ""),
		WebhookListenAddr:  getEnv("WEBHOOK_LISTEN_ADDR", ":8443"),
		WebhookPath:        getEnv("WEBHOOK_PATH", "/webhook"),
		WebhookSecretToken: getEnv("WEBHOOK_SECRET_TOKEN", ""),

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

		// Profiling/Debug
		PprofEnabled:              getEnvBool("PPROF_ENABLED", false),
		PprofAddr:                 getEnv("PPROF_ADDR", "0.0.0.0:6060"),
		PprofBlockProfileRate:     getEnvInt("PPROF_BLOCK_PROFILE_RATE", 10000),
		PprofMutexProfileFraction: getEnvInt("PPROF_MUTEX_PROFILE_FRACTION", 10),

		DebugMode: getEnvBool("DEBUG_MODE", false),
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
