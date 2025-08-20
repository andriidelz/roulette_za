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
	CallbackAgeVerify        = "age_verify"
	CallbackAgeVerifiedYes   = "age_verified_yes"
	CallbackAgeVerifiedNo    = "age_verified_no"
	CallbackCountry          = "country_change"
	CallbackChangeName       = "name_change"
	CallbackChangeNameYes    = "name_changeyes"
	CallbackChangeNameNo     = "name_changeno"
	CallbackReserveSubs      = "reserve_subs"
	CallbackReserveSubsCheck = "reserve_subs_check"
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
		b.SendMessage(message.Chat.ID, b.prepareMessage("error_while_registering", user.LanguageCode))
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
	options := b.prepareMessage("startmessage1_new", language)
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{regButton},
		},
	}
	// Отправляем первое приветственное сообщение с inline клавиатурой
	b.SendMessage(message.Chat.ID, options)
}

// handleStartCommand обрабатывает команду /start
// сама регистрация пользователя выполняется в начале функции handleMessage
// в этой функции только дозаполнение полей
func (b *Bot) handleStartCommand(message *telego.Message) {
	user := message.From

	// Регистрируем пользователя или обновляем информацию
	dbUser, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, "", user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error registering user: %v", err)
		b.SendMessage(message.Chat.ID, b.prepareMessage("error_while_registering", user.LanguageCode))
		return
	}

	// Проверка полностью ли завершена регистрация. Если нет то будет отправлено предложение завершить ее
	if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "start") {
		return
	}

	// Определяем язык пользователя из базы данных
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Если полностью завершена регистрация
	// Отправляем текст приветствия и главное меню
	options := b.prepareMessage("startmessage1", language)
	options.ReplyKeyboard = b.createMainReplyKeyboard(language)
	b.SendMessage(message.Chat.ID, options)
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
	options := b.prepareMessage("agemes", language)

	// Получаем локализированные тексты для кнопок
	yesText := b.service.GetText("yes18", language)
	noText := b.service.GetText("no18", language)

	// Создаем inline клавиатуру с кнопками Да/Нет
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: yesText, CallbackData: CallbackAgeVerifiedYes},
				{Text: noText, CallbackData: CallbackAgeVerifiedNo},
			},
		},
	}

	// Отправляем сообщение с запросом на подтверждение возраста
	b.SendMessage(chatID, options)
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

// 4. handleCountryCallback обрабатывает запуск запроса страны
func (b *Bot) handleCountryCallback(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	options := b.prepareMessage("countrymes", language)
	options.InlineKeyboard = b.createCountriesKeyboard(1)
	b.SendMessage(query.Message.Chat.ID, options)
}

// 4.1 обработка стран в handleCallbackQuery

// 5. handleNicknamePrompt обрабатывает запуск запроса подтверждение/изменение никнейма
func (b *Bot) handleNicknameCallback(query *telego.CallbackQuery) {

	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	b.handleNicknamePrompt(query.Message.Chat.ID, user.ID, language)
}

// 5.1 handleNicknamePrompt отправляет запрос на подтверждение/изменение никнейма
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
	options := b.prepareMessage("name_mes", language)
	// Заменяем placeholder profile_name на настоящее имя
	options.Text = strings.Replace(options.Text, "{profile_name}", profileName, -1)

	// Создаем inline-клавиатуру для выбора
	yesText := b.service.GetText("name_changeyes", language)
	noText := b.service.GetText("name_changeno", language)

	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: yesText, CallbackData: CallbackChangeNameYes},
				{Text: noText, CallbackData: CallbackChangeNameNo},
			},
		},
	}

	// Отправляем сообщение с вопросом
	err = b.SendMessage(chatID, options)

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

// 7. handleReserveSubscription отправляет запрос на подписку на резервный канал
func (b *Bot) handleReserveSubscription(query *telego.CallbackQuery) {

	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)
	b.sendSubscriptionRequest(query.Message.Chat.ID, language, "startmessage3_exist")
}

// 7.1 sendSubscriptionRequest отправляет запрос на подписку на резервный канал
func (b *Bot) sendSubscriptionRequest(chatID int64, language, textKey string) {
	// Формируем ссылку на канал в правильном формате
	channelButton := telego.InlineKeyboardButton{
		Text: b.service.GetText("go_to_channel", language),
		URL:  "https://t.me/" + strings.TrimPrefix(ReserveChannelID, "@"),
	}

	// Создаем inline клавиатуру с кнопкой подтверждения
	subscribeButton := telego.InlineKeyboardButton{
		Text:         b.service.GetText("reservsubs", language),
		CallbackData: CallbackReserveSubsCheck,
	}
	if textKey == "" {
		textKey = "startmessage2"
	}

	options := b.prepareMessage(textKey, language)
	options.InlineKeyboard = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{channelButton},
			{subscribeButton},
		},
	}

	b.SendMessage(chatID, options)
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
func (b *Bot) RequireCompleteRegistration(chatID, userID int64, command string) bool {
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

		if dbUser.AgeVerified != nil && !*dbUser.AgeVerified {
			// Пользователь не подтвердил совершеннолетие - показываем сообщение о блокировке

			b.SendMessage(chatID, b.prepareMessage("stopage", language))
			return false
		}

		if dbUser.Banned {
			// Пользователь забанен (RU/BY или несовершеннолетний)
			if dbUser.Country == "RU" || dbUser.Country == "BY" {
				b.SendMessage(chatID, b.prepareMessage("stopcountry", language))
			} else {

				b.SendMessage(chatID, b.prepareMessage("stopage", language))
			}
			return false
		}

		// Пользователь не заполнил все данные
		call := ""

		switch {
		case dbUser.AgeVerified == nil:
			// Нужно подтвердить возраст
			call = CallbackAgeVerify
		case dbUser.Country == "":
			// Нужно выбрать страну
			call = CallbackCountry
		case dbUser.Nickname == "":
			// Нужно выбрать никнейм
			call = CallbackChangeName
		case b.checkSubscriptionWithCache(userID, ReserveChannelID):
			// Не подписан на резервный канал
			call = CallbackReserveSubs
		default:
			call = CallbackBack
		}

		regButton := telego.InlineKeyboardButton{
			Text:         b.service.GetText("btn_reg_continue", language),
			CallbackData: call,
		}

		// Создаем inline клавиатуру для первого сообщения
		options := b.prepareMessage("startmessage1_continue", language)
		options.InlineKeyboard = &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{regButton},
			},
		}

		// Отправляем сообщение с предложением завершить регистрацию
		b.SendMessage(chatID, options)
		return false
	}

	// Дополнительная проверка на подписку, даже если регистрация завершена
	subscribed := b.checkSubscriptionWithCache(userID, ReserveChannelID)
	if !subscribed {

		if command == "start" {

			// Получаем локализированный текст приветствия
			regButton := telego.InlineKeyboardButton{
				Text:         b.service.GetText("play_exist", language),
				CallbackData: CallbackReserveSubs,
			}

			// Создаем inline клавиатуру для первого сообщения
			options := b.prepareMessage("startmessage1_exist", language)
			options.InlineKeyboard = &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{regButton},
				},
			}
			// Отправляем первое приветственное сообщение с inline клавиатурой
			b.SendMessage(chatID, options)

		} else {
			b.sendSubscriptionRequest(chatID, language, "startmessage3_exist")
		}
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
