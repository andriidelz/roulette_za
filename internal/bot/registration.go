// internal/bot/registration.go
package bot

import (
	"roulette/internal/logger"
	"sync"
	"time"
)

// SubscriptionCache кеширует результаты проверки подписки
type SubscriptionCache struct {
	cache     map[int64]bool
	timestamp map[int64]time.Time
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewSubscriptionCache создает новый кеш подписок с заданным TTL
func NewSubscriptionCache(ttl time.Duration) *SubscriptionCache {
	return &SubscriptionCache{
		cache:     make(map[int64]bool),
		timestamp: make(map[int64]time.Time),
		ttl:       ttl,
	}
}

// Get получает результат проверки подписки из кеша
func (c *SubscriptionCache) Get(userID int64) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	subscribed, exists := c.cache[userID]
	if !exists {
		return false, false
	}

	// Проверяем, не истек ли TTL
	if time.Since(c.timestamp[userID]) > c.ttl {
		return false, false
	}

	return subscribed, true
}

// Set сохраняет результат проверки подписки в кеш
func (c *SubscriptionCache) Set(userID int64, subscribed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[userID] = subscribed
	c.timestamp[userID] = time.Now()
}

// RequireCompleteRegistration проверяет завершенность регистрации пользователя
// и перенаправляет на соответствующий шаг при необходимости
func (b *Bot) RequireCompleteRegistration(chatID int64, userID int64) bool {
	// Получаем пользователя
	dbUser, err := b.service.GetUser(userID)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", userID, err)
		return false
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Проверяем полноту регистрации
	if !b.isRegistrationComplete(dbUser) {
		// Определяем, на каком этапе регистрации находится пользователь
		if dbUser.AgeVerified == nil {
			// Нужно подтвердить возраст
			b.sendAgeVerificationRequest(chatID, language)
			return false
		}

		if dbUser.AgeVerified != nil && !*dbUser.AgeVerified {
			// Пользователь не подтвердил совершеннолетие - показываем сообщение о блокировке
			stopAgeText := b.service.GetText("stopage", language)
			b.SendMessage(chatID, MessageOptions{
				Text: stopAgeText,
			})
			return false
		}

		if dbUser.Country == "" {
			// Нужно выбрать страну
			countryText := b.service.GetText("countrymes", language)
			countriesKeyboard := b.createCountriesKeyboard(1)
			b.SendMessage(chatID, MessageOptions{
				Text:           countryText,
				InlineKeyboard: countriesKeyboard,
			})
			return false
		}

		if dbUser.Nickname == "" {
			// Нужно выбрать никнейм
			b.handleNicknamePrompt(chatID, userID, language)
			return false
		}

		// Проверяем подписку на резервный канал
		subscribed := b.checkSubscriptionWithCache(userID, ReserveChannelID)
		if !subscribed {
			b.sendSubscriptionRequest(chatID, language)
			return false
		}

		if dbUser.Banned {
			// Пользователь забанен (RU/BY или несовершеннолетний)
			if dbUser.Country == "RU" || dbUser.Country == "BY" {
				banText := b.service.GetText("stopcountry", language)
				b.SendMessage(chatID, MessageOptions{
					Text: banText,
				})
			} else {
				stopAgeText := b.service.GetText("stopage", language)
				b.SendMessage(chatID, MessageOptions{
					Text: stopAgeText,
				})
			}
			return false
		}
	}

	// Дополнительная проверка на подписку, даже если регистрация завершена
	subscribed := b.checkSubscriptionWithCache(userID, ReserveChannelID)
	if !subscribed {
		b.sendSubscriptionRequest(chatID, language)
		return false
	}

	return true
}

// checkSubscriptionWithCache проверяет подписку с использованием кеша
func (b *Bot) checkSubscriptionWithCache(userID int64, channelUsername string) bool {
	// Проверяем кеш
	if b.subscriptionCache == nil {
		b.subscriptionCache = NewSubscriptionCache(1 * time.Minute) // TTL 1 минута
	}

	if subscribed, exists := b.subscriptionCache.Get(userID); exists {
		return subscribed
	}

	// Если в кеше нет, проверяем через API
	subscribed, err := b.checkChannelSubscription(userID, channelUsername)
	if err != nil {
		logger.Error.Printf("Error checking subscription for user %d: %v", userID, err)
		return false
	}

	// Сохраняем результат в кеш
	b.subscriptionCache.Set(userID, subscribed)

	return subscribed
}
