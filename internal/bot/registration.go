// internal/bot/registration.go
package bot

import (
	"fmt"
	"roulette/internal/logger"
	"roulette/internal/models"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
)

const (
	CallbackAgeVerify           = "age_verify"
	CallbackAgeVerifiedYes      = "age_verified_yes"
	CallbackAgeVerifiedNo       = "age_verified_no"
	CallbackChangeNameYes       = "name_changeyes"
	CallbackChangeNameNo        = "name_changeno"
	CallbackReserveSubscription = "reservsubs"
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

// 1. handleStartCommand обрабатывает команду /start нового пользователя
func (b *Bot) handleStartCommandNewUser(message *telego.Message) {
	user := message.From

	// Отправляем стикер регистрация нового пользователя
	b.MakeRequestDeferred(message.Chat.ID, 0, MessageOptions{
		Text:       StickerRegistration,
		MethodName: sendSticker,
	})

	// Получаем источник
	userSource := ""
	if strings.Contains(message.Text, " ") {
		commandArgs := strings.Split(message.Text, " ")
		if len(commandArgs) > 1 {
			userSource = strings.TrimSpace(commandArgs[1]) // Получаем источник
		}
	}

	// Обновляем информацию
	_, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, userSource, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error registering user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: b.service.GetText("error_while_registering", user.LanguageCode),
		})
		return
	}
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст приветствия
	regButton := telego.InlineKeyboardButton{
		Text:         b.service.GetText("btn_reg", language),
		CallbackData: CallbackAgeVerify,
	}

	// Создаем inline клавиатуру для первого сообщения
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{regButton},
		},
	}

	// Отправляем первое приветственное сообщение с inline клавиатурой
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:           b.service.GetText("startmessage1_new", language),
		InlineKeyboard: inlineKeyboard,
	})
}

// 2. handleAgeVerifyCallback обрабатывает запуск запроса возраста
func (b *Bot) handleAgeVerifyCallback(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	b.sendAgeVerificationRequest(query.Message.Chat.ID, language)
}

// 2.1 sendAgeVerificationRequest отправляет запрос на подтверждение возраста
func (b *Bot) sendAgeVerificationRequest(chatID int64, language string) {
	// Получаем локализированный текст запроса возраста
	ageVerificationText := b.service.GetText("agemes", language)

	// Получаем локализированные тексты для кнопок
	yesText := b.service.GetText("yes18", language)
	noText := b.service.GetText("no18", language)

	// Создаем inline клавиатуру с кнопками Да/Нет
	ageVerificationKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: yesText, CallbackData: CallbackAgeVerifiedYes},
				{Text: noText, CallbackData: CallbackAgeVerifiedNo},
			},
		},
	}

	// Отправляем сообщение с запросом на подтверждение возраста
	b.SendMessage(chatID, MessageOptions{
		Text:           ageVerificationText,
		InlineKeyboard: ageVerificationKeyboard,
	})
}

// 3. handleAgeVerificationCallback обрабатывает ответ пользователя на запрос возраста
func (b *Bot) handleAgeVerificationCallback(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	var isAdult bool
	if query.Data == CallbackAgeVerifiedYes {
		isAdult = true
	} else {
		isAdult = false
	}

	// Получаем пользователя из базы данных
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		return
	}

	// Обновляем статус подтверждения возраста
	dbUser.AgeVerified = &isAdult
	err = b.service.UpdateUser(dbUser)
	if err != nil {
		logger.Error.Printf("Error updating user age verification: %v", err)
		return
	}

	if !isAdult {
		// Если пользователь младше 18 лет, баним его
		stopAgeText := b.service.GetText("stopage", language)

		// Обновляем статус бана
		dbUser.Banned = true
		err = b.service.UpdateUser(dbUser)
		if err != nil {
			logger.Error.Printf("Error banning underage user: %v", err)
		}

		// Сообщаем пользователю, что сервис недоступен
		if query.Message != nil {
			b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
				Text: stopAgeText,
			})
		}
		return
	}

	// Если пользователь подтвердил, что старше 18 лет, отправляем запрос на выбор страны
	if query.Message != nil {
		countryText := b.service.GetText("countrymes", language)

		// Создаем клавиатуру со странами - начинаем с первой страницы (1)
		countriesKeyboard := b.createCountriesKeyboard(1)

		// Обновляем сообщение с запросом на выбор страны
		b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
			Text:           countryText,
			InlineKeyboard: countriesKeyboard,
		})
	}
}

// 4. обработка стран в handleCallbackQuery

// 5. handleNicknamePrompt отправляет запрос на подтверждение/изменение никнейма
func (b *Bot) handleNicknamePrompt(chatID int64, userID int64, language string) {
	// Получаем пользователя
	user, err := b.service.GetUser(userID)
	if err != nil {
		logger.Error.Printf("Error getting user for nickname prompt: %v", err)
		return
	}

	// Определяем предварительный никнейм
	var profileName string
	if user.Nickname != "" {
		// Если никнейм уже задан, используем его
		profileName = user.Nickname
	} else if user.Username != "" {
		profileName = user.Username
	} else if user.FirstName != "" && len(user.FirstName) > 1 {
		profileName = user.FirstName
	} else {
		profileName = fmt.Sprintf(b.service.GetText("player_nickname_template", language), user.TelegramID)
	}

	// Получаем локализованный текст сообщения
	namePromptText := b.service.GetText("name_mes", language)
	// Заменяем placeholder profile_name на настоящее имя
	namePromptText = strings.Replace(namePromptText, "{profile_name}", profileName, -1)

	// Создаем inline-клавиатуру для выбора
	yesText := b.service.GetText("name_changeyes", language)
	noText := b.service.GetText("name_changeno", language)

	nicknameKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: yesText, CallbackData: CallbackChangeNameYes},
				{Text: noText, CallbackData: CallbackChangeNameNo},
			},
		},
	}

	// Отправляем сообщение с вопросом
	err = b.SendMessage(chatID, MessageOptions{
		Text:           namePromptText,
		InlineKeyboard: nicknameKeyboard,
	})

	if err != nil {
		logger.Error.Printf("Error sending nickname prompt: %v", err)
	}
}

// 6.1 handleChangeNameYes обрабатывает согласие на изменение никнейма
func (b *Bot) handleChangeNameYes(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback
	b.answerCallbackQuery(query.ID, "", false)

	// Получаем локализованный текст для ввода нового никнейма
	nameChangeOkText := b.service.GetText("name_changeok", language)

	// Обновляем сообщение
	if query.Message != nil {
		// Устанавливаем состояние ожидания никнейма
		b.stateManager.SetState(user.ID, StateInputNickname, query.Message.MessageID)

		// Обновляем сообщение с инструкцией для ввода никнейма
		b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
			Text: nameChangeOkText,
		})
	}
}

// 6.2 handleChangeNameNo обрабатывает отказ от изменения никнейма
func (b *Bot) handleChangeNameNo(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback
	b.answerCallbackQuery(query.ID, "", false)

	// Получаем пользователя
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		return
	}

	// Сохраняем текущий никнейм (если он не был установлен ранее)
	if dbUser.Nickname == "" {
		var profileName string
		if dbUser.Username != "" {
			profileName = dbUser.Username
		} else if dbUser.FirstName != "" && len(dbUser.FirstName) > 1 {
			profileName = dbUser.FirstName
		} else {
			profileName = fmt.Sprintf("Player%d", dbUser.TelegramID)
		}

		// Сохраняем никнейм
		dbUser.Nickname = profileName
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user nickname: %v", err)
		}
	}

	// Отправляем запрос на подписку
	b.sendSubscriptionRequest(query.Message.Chat.ID, language, "name_changeno_msg_start")
}

// 7. sendSubscriptionRequest отправляет запрос на подписку на резервный канал
func (b *Bot) sendSubscriptionRequest(chatID int64, language, textKey string) {
	// Формируем ссылку на канал в правильном формате
	channelButton := telego.InlineKeyboardButton{
		Text: b.service.GetText("go_to_channel", language),
		URL:  "https://t.me/" + strings.TrimPrefix(ReserveChannelID, "@"),
	}

	// Создаем inline клавиатуру с кнопкой подтверждения
	subscribeButton := telego.InlineKeyboardButton{
		Text:         b.service.GetText("reservsubs", language),
		CallbackData: CallbackReserveSubscription,
	}
	if textKey == "" {
		textKey = "startmessage2"
	}
	b.SendMessage(chatID, MessageOptions{
		Text: b.service.GetText(textKey, language),
		InlineKeyboard: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{channelButton},
				{subscribeButton},
			},
		},
	})
}

// 8. handleReserveSubscriptionCheck обрабатывает нажатие кнопки проверки подписки
func (b *Bot) handleReserveSubscriptionCheck(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback
	b.answerCallbackQuery(query.ID, "", false)

	// Проверяем подписку
	isSubscribed, err := b.checkChannelSubscription(user.ID, ReserveChannelID)
	if err != nil {
		logger.Error.Printf("Error checking subscription: %v", err)
		// В случае ошибки считаем, что пользователь не подписан
		isSubscribed = false
	}

	if isSubscribed {
		// Подписка подтверждена
		successText := b.service.GetText("reservok", language)

		// Обновляем сообщение успешной подпиской
		if query.Message != nil {
			b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
				Text:          successText,
				ReplyKeyboard: b.createMainReplyKeyboard(language),
			})
		}
	} else {
		// Подписка не подтверждена

		// Отправляем сообщение с информацией о неудаче
		if query.Message != nil {
			b.sendSubscriptionRequest(query.Message.Chat.ID, language, "reservno")
		}
	}
}

// 8.1 checkChannelSubscription отправляет запрос на проверку подписки
func (b *Bot) checkChannelSubscription(userID int64, channelUsername string) (bool, error) {

	if !strings.HasPrefix(channelUsername, "@") {
		channelUsername = "@" + channelUsername
	}

	logger.Info.Printf("Checking subscription for user %d to channel %s", userID, channelUsername)

	// Получаем статус подписки пользователя
	chatMember, err := b.bot.GetChatMember(&telego.GetChatMemberParams{
		ChatID: telego.ChatID{
			Username: channelUsername, // с символом @
		},
		UserID: userID,
	})

	if err != nil {
		logger.Error.Printf("Error checking subscription for user %d: %v", userID, err)
		return false, err
	}

	// Проверяем тип участника
	switch member := chatMember.(type) {
	case *telego.ChatMemberOwner:
		// Владелец канала
		return true, nil
	case *telego.ChatMemberAdministrator:
		// Администратор канала
		return true, nil
	case *telego.ChatMemberMember:
		// Обычный участник
		return true, nil
	case *telego.ChatMemberRestricted:
		// Участник с ограничениями
		return true, nil
	case *telego.ChatMemberLeft:
		// Покинул канал
		return false, nil
	case *telego.ChatMemberBanned:
		// Забанен в канале
		return false, nil
	default:
		logger.Error.Printf("Unknown chat member type for user %d: %T", userID, member)
		return false, nil
	}
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
			b.sendSubscriptionRequest(chatID, language, "")
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
		b.sendSubscriptionRequest(chatID, language, "")
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

// isRegistrationComplete проверяет, завершена ли первичная регистрация пользователя
func (b *Bot) isRegistrationComplete(user *models.User) bool {
	// Проверяем обязательные поля для завершения регистрации:
	// 1. Подтверждение возраста
	if user.AgeVerified == nil || !*user.AgeVerified {
		return false
	}

	// 2. Выбор страны
	if user.Country == "" {
		return false
	}

	// 3. Язык должен быть установлен
	if user.LanguageCode == "" {
		return false
	}

	// 4. Пользователь не должен быть забанен
	if user.Banned {
		return false
	}

	return true
}
