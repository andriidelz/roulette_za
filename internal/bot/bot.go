package bot

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/metrics"
	"roulette/internal/models"
	"roulette/internal/service"
	"roulette/internal/utils"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/redis/go-redis/v9"
)

// Структура бота
type Bot struct {
	bot               *telego.Bot
	updates           <-chan telego.Update
	service           service.Service
	initialized       bool
	ctx               context.Context
	cancel            context.CancelFunc
	gameHandler       *GameHandler       // Обработчик игры
	stateManager      *StateManager      // Менеджер состояний
	subscriptionCache *SubscriptionCache // Кеш подписок на каналы
	redisDB           *redis.Client      // Клиент Redis
	metrics           *metrics.Metrics
}

// Константы для команд и callback-запитов
const (
	CommandStart    = "start"
	CommandPrivacy  = "privacy"
	CommandContact  = "contact"
	CommandPlay     = "play"
	CommandStats    = "statistics"
	CommandRating   = "rating"
	CommandAccount  = "account"
	CommandFAQ      = "faq"
	CommandSettings = "settings"

	CallbackBack    = "back"
	CallbackCaptcha = "captcha_"

	// Стикер ошибки отправки сообщения, если словили 429 ошибку
	StickerError        = "CAACAgUAAxkBAAEO5upob-zRQ5ptM0PmCYlvTra-KSbbiQACEBYAAkV9qVf5P89H45HU5zYE" // error
	StickerRegistration = "CAACAgUAAxkBAAEBgIpokq-UIqudmGRVogN-Mu28MQQ6UwACwRIAAs9XsFejY2Cqcm0SBDYE" // registration (регистрация нового пользователя)
)

var ReserveChannelID = "@socialroulette_dev" // https://t.me/socialroulette_dev

func (b *Bot) getMetrics() *metrics.Metrics {
	if b == nil || b.metrics == nil {
		return nil
	}
	return b.metrics
}

// NewBot создает новый экземпляр бота
func NewBot(token string, service service.Service, cfg *config.Config) (*Bot, error) {
	ReserveChannelID = cfg.TelegramReserveChannelID

	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	b := &Bot{
		bot:          bot,
		service:      service,
		ctx:          ctx,
		cancel:       cancel,
		stateManager: NewStateManager(),
		redisDB:      NewRedisClient(cfg),
		metrics:      nil,
	}

	// Инициализируем обработчик игры после создания бота с поддержкой RabbitMQ
	gameHandler, err := NewGameHandler(b, service, cfg.RabbitMQURL)
	if err != nil {
		cancel() // Освобождаем ресурсы в случае ошибки
		return nil, fmt.Errorf("failed to create game handler: %w", err)
	}

	b.gameHandler = gameHandler

	return b, nil
}

func NewRedisClient(cfg *config.Config) *redis.Client {
	// Create Redis client with options
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPass,
		DB:           cfg.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     50,
		PoolTimeout:  30 * time.Second,
		MinIdleConns: 10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Error.Printf("failed to connect to Redis: %v", err)
	}

	logger.Info.Printf("Successfully connected to Redis at %s:%s", cfg.RedisHost, cfg.RedisPort)

	return rdb
}

// Start запускает бота
func (b *Bot) Start() error {
	if b.initialized {
		return fmt.Errorf("bot already started")
	}

	// Инициализируем метрики
	b.metrics = metrics.NewMetrics("roulette-bot", 9101, metrics.AppTypeBot)
	go func() {
		if err := b.metrics.Start(); err != nil {
			logger.Error.Printf("Failed to start metrics server: %v", err)
		}
	}()

	// Получаем информацию о боте
	me, err := b.bot.GetMe(b.ctx)
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}
	logger.Info.Printf("Bot started: https://t.me/%s", me.Username)

	// Настраиваем обработчик уведомлений
	if err := b.setupNotificationHandler(); err != nil {
		logger.Warning.Printf("Warning: Failed to setup notification handler: %v", err)
		// Продолжаем работу, так как это не критическая ошибка
	}

	// Начало получения обновлений
	updates, err := b.bot.UpdatesViaLongPolling(b.ctx, &telego.GetUpdatesParams{
		Timeout: 60,
		Offset:  0,
	})
	if err != nil {
		return fmt.Errorf("failed to get updates: %w", err)
	}
	b.updates = updates

	// Запускаем обработку обновлений в фоновом режиме
	go b.processUpdates()

	// Запускаем отправку сообщений
	go b.sendBotQueue()

	// Запускаем планировщик для обновления рейтингов
	b.StartRatingScheduler()
	// Запускаем планировщик для обновления капч
	b.StartUpdateCaptcha()

	// Запускам емуляцию ставок по заданиям для пользователей
	b.gameHandler.initEmulate()

	b.initialized = true
	return nil
}

// GetGameHandler возвращает обработчик игры для регистрации в ротаторе
func (b *Bot) GetGameHandler() *GameHandler {
	return b.gameHandler
}

// Stop останавливает бота
func (b *Bot) Stop() {
	if !b.initialized {
		return
	}

	// Отменяем контекст - это остановит длинный поллинг
	if b.cancel != nil {
		b.cancel()
	}

	// Останавливаем игровой обработчик
	if b.gameHandler != nil {
		b.gameHandler.Stop()
	}

	// Останавливаем метрики
	if b.metrics != nil {
		if err := b.metrics.Stop(); err != nil {
			logger.Error.Printf("Error stopping metrics server: %v", err)
		}
	}

	b.initialized = false
	logger.Info.Println("Bot stopped")
}

// processUpdates обрабатывает обновления от телеграма
func (b *Bot) processUpdates() {
	for update := range b.updates {
		b.handleUpdate(update)
	}
}

// handleUpdate обрабатывает одно обновление
func (b *Bot) handleUpdate(update telego.Update) {
	// Обработка сообщений
	if update.Message != nil && update.Message.Text != "" {
		b.handleMessage(update.Message)
		return
	}

	// Обработка callback-запитов
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}
}

// handlePrivacyCommand обрабатывает команду /privacy
func (b *Bot) handlePrivacyCommand(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for privacy policy: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст privacy policy
	options := b.prepareMessage("privacypolicym", language)
	options.ReplyKeyboard = b.createMainReplyKeyboard(language)

	// Отправляем текст privacy policy
	b.SendMessage(message.Chat.ID, options)
}

// handleContactCommand обрабатывает команду /contact
func (b *Bot) handleContactCommand(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for contact: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Контакт с админом"
	options := b.prepareMessage("contactm", language)
	options.ReplyKeyboard = b.createMainReplyKeyboard(language)
	// Отправляем текст о контакте с админом
	b.SendMessage(message.Chat.ID, options)
}

// MakeBet делает ставку в текущем раунде
func (b *Bot) handleMakeBet(userID int64, option models.BetOption) {
	// Получаем пользователя для определения языка
	user, userErr := b.service.GetUser(userID)
	if userErr != nil {
		logger.Error.Printf("Error getting user %d: %v", userID, userErr)
		return
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Проверяем активность пользователя
	// - беспрерывная игра
	// - ставка на одну и ту же опцию
	switch b.captchaBetActivity(userID) {
	case "needCaptcha":

		// виводимо капчу у рендомний час з першої секунди 4 хвилини по 59 секунду 5 хвилини
		go func() {
			time.Sleep(time.Duration(rand.Intn(120)) * time.Second)
			b.SendMessage(userID, b.captchaMessage(userID, language))
		}()
	}
	switch b.captchaBetDuplicate(userID, string(option)) {
	case "needCaptcha":
		b.SendMessage(userID, b.captchaMessage(userID, language))
		return
	}

	// Вызываем MakeBet и обрабатываем возможные ошибки
	err := b.gameHandler.MakeBet(userID, option)
	if err != nil {
		// Определяем тип ошибки и отправляем соответствующее сообщение
		var errorKey string
		var remain int

		if strings.Contains(err.Error(), "already made a bet") {
			// Пользователь уже сделал ставку в этом раунде
			errorKey = "bet_already_made"
		} else if strings.Contains(err.Error(), "cannot bet on zero") {
			// Пользователь не может ставить на Zero
			canBetZero, remaining, _ := b.service.CanBetZero(userID)
			if !canBetZero {
				errorKey = "zero_limit"
				remain = remaining
			} else {
				errorKey = "bet_error"
			}
		} else if strings.Contains(err.Error(), "no bets left") {
			// У пользователя закончились ставки на сегодня
			errorKey = "betsbalancelow"
		} else {
			// Общая ошибка ставки

			logger.Error.Println("bet_error", err)
			errorKey = "bet_error"
		}
		options := b.prepareMessage(errorKey, language)
		if errorKey == "zero_limit" {
			options.Text = fmt.Sprintf(options.Text, remain)
		}

		options.InlineKeyboard = b.gameHandler.createBetKeyboard(language, userID)
		b.SendMessage(userID, options)
		return
	}
}

// handleMessage обрабатывает сообщения
func (b *Bot) handleMessage(message *telego.Message) {
	startTime := time.Now()
	defer func() {
		// Записываем латентность команды
		if metrics := b.getMetrics(); metrics != nil && metrics.Bot != nil {
			duration := time.Since(startTime).Seconds()
			command := "message" // дефолтный тип

			text := message.Text
			if len(text) > 0 && text[0] == '/' {
				command = strings.Split(text[1:], " ")[0]
				command = strings.ToLower(command)
			}

			metrics.Bot.RecordCommandLatency(command, duration)
		}
	}()

	user := message.From
	go b.service.UpdateUserActivity(user.ID)

	// Режим эмуляции
	originalUserID := user.ID
	if emulatedID, ok := emulatedUsers[originalUserID]; ok {
		if !strings.HasPrefix(message.Text, "/"+CommandStopEmulateID) {
			user.ID = emulatedID
			defer func() {
				user.ID = originalUserID
			}()
		}
	}

	// Если не найден в базе считается как новый пользователь
	// на будущее переписать на статусы пользователя
	isNewUser := false

	// Получаем данные пользователя из базы
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		// Если пользователь новый выполняется его предварительная регистрация,
		// независимо от того какую команду он выполнил
		// handleStartCommand только для обновления незаполненных полей

		dbUser, err = b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, "", user.LanguageCode)
		if err != nil {
			logger.Error.Printf("Error registering user: %v", err)
		} else {

			isNewUser = true

			// Записываем метрику регистрации нового пользователя
			if metrics := b.getMetrics(); metrics != nil && metrics.Business != nil {
				metrics.Business.RecordUserRegistration()
			}
		}
	}
	if err == nil && dbUser.Banned {
		// Если пользователь забанен, молча игнорируем сообщение
		return
	}

	// Всегда используем язык из базы данных
	language := dbUser.LanguageCode
	if language == "" {
		// Если в базе не указан язык пробуем получить его у юзера
		if user.LanguageCode != "" {
			language = user.LanguageCode
		} else {
			language = "en"
		}
	}

	if b.captchaBan(user.ID) {
		return
	}

	// Проверка на повышенную активность пользователя
	switch b.captchaUserActivity(user.ID) {
	case "wait":
		return
	case "needCaptcha":
		b.SendMessage(message.Chat.ID, b.captchaMessage(user.ID, language))
		return
	}

	// Обновляем язык пользователя из API, если он отличается
	if user.LanguageCode != language {
		user.LanguageCode = language
	}

	text := message.Text

	// Обработка команд, начинающихся с /
	if len(text) > 0 && text[0] == '/' {
		command := strings.Split(text[1:], " ")[0] // Получаем команду без аргументов
		command = strings.ToLower(command)

		switch command {
		case CommandStart:
			if isNewUser {
				b.handleStartCommandNewUser(message) // регистрация нового пользователя
			} else {
				b.handleStartCommand(message) // start от уже зарегистрированного пользователя
			}
			return
		case CommandPrivacy:
			// Команды privacy и contact доступны всегда (не требуют завершения регистрации)
			b.handlePrivacyCommand(message)
			return
		case CommandContact:
			b.handleContactCommand(message)
			return
		case CommandPlay:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				return
			}
			b.gameHandler.HandlePlayCommand(message)
			return
		case CommandStats:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				return
			}
			b.handleStatsCommand(message)
			return
		case CommandRating:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				return
			}
			b.handleRatingCommand(message)
			return
		case CommandAccount:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				return
			}
			b.handleAccountCommand(message)
			return
		case CommandFAQ:
			// FAQ доступен всегда
			b.handleFAQCommand(message)
			return
		case CommandSettings:
			if !b.RequireCompleteRegistration(message.Chat.ID, user.ID, "") {
				return
			}
			b.handleSettingsCommand(message)
			return
		case CommandMyID:
			b.handleMyIDCommand(message)
			return
		case CommandEmulateID: //  /emulateid 123456789
			b.handleEmulateIDCommand(message)
			return
		case CommandStopEmulateID:
			b.handleStopEmulateIDCommand(message)
			return

		default:
			// Неизвестная команда
			b.handleUnknownCommand(message)
		}
		return
	}

	// Проверяем состояние пользователя для обработки ввода
	state, messageID, exists := b.stateManager.GetState(user.ID)
	if exists && state != StateNone {
		// Обработка состояний ввода данных при регистрации (НЕ требует завершенной регистрации)
		switch state {
		case StateInputNickname:
			b.handleInputNicknameState(message)
			return
		case StateInputName:
			b.handleInputNameState(message, messageID)
			return
		case StateInputUpNickname:
			// Для обновления никнейма требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputUpNicknameState(message, messageID)
			return
		case StateInputWallet:
			// Для ввода кошелька требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputWalletState(message, messageID)
			return
		case StateInputWithdrawAmount:
			// Для вывода средств требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputWithdrawAmountCommand(message)
			return
		case StateInputWithdrawWallet:
			// Для вывода средств требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputWithdrawWalletCommand(message)
			return
		}
	}

	// Для всех остальных текстовых команд требуется завершенная регистрация
	if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID, "") {
		return
	}

	// Получаем локализированные тексты для кнопок
	btnPlayText := b.service.GetText("btn_play", language)
	btnStatisticsText := b.service.GetText("btn_statistics", language)
	btnRatingText := b.service.GetText("btn_rating", language)
	btnAccountText := b.service.GetText("btn_account", language)
	btnFAQText := b.service.GetText("btn_faq", language)

	btnBackText := b.service.GetText("btn_back", language)
	btnStopText := b.service.GetText("stop", language)

	weekRatingText := b.service.GetText("weekrat", language)
	personalRatingText := b.service.GetText("personalrat", language)
	exitRatingText := b.service.GetText("exitrat", language)

	// Обработка клавиатуры главного меню
	switch text {
	case btnPlayText:
		b.gameHandler.HandlePlayCommand(message)
	case btnStatisticsText:
		b.handleStatsCommand(message)
	case btnRatingText:
		b.handleRatingCommand(message)
	case btnAccountText:
		b.handleAccountCommand(message)
		return
	case btnFAQText:
		b.handleFAQCommand(message)
	// Обработка ставок по тексту кнопки
	case btnBackText:
		// Возврат в главное меню и удаление из активных игроков
		b.gameHandler.HandleBackButton(user.ID)
		b.handleHelpCommand(message)
	case btnStopText:
		// Остановка игры и возврат в главное меню
		b.gameHandler.HandleStopGameButton(user.ID)
		b.handleHelpCommand(message)

		// Обработка кнопок статистики
	case b.service.GetText("daystat", language):
		b.handleDayStatistics(message)
	case b.service.GetText("weekstat", language):
		b.handleWeekStatistics(message)
	case b.service.GetText("monthstat", language):
		b.handleMonthStatistics(message)
	case b.service.GetText("allstat", language):
		b.handleAllStatistics(message)
	case b.service.GetText("exitstat", language):
		b.handleHelpCommand(message) // Возврат в главное меню

		// Обработка кнопок рейтинга
	case weekRatingText:
		b.handleWeeklyRating(message)
	case personalRatingText:
		b.handlePersonalRating(message)
	case exitRatingText:
		b.handleHelpCommand(message) // Возврат в главное меню

	// Обработка кнопок аккаунта
	case b.service.GetText("balance", language):
		b.handleBalanceCommand(message)
		return
	case b.service.GetText("withdraw", language):
		b.handleWithdrawCommand(message)
	case b.service.GetText("bonus", language):
		// Временно просто возвращаем в меню аккаунта
		b.handleAccountCommand(message)
	case b.service.GetText("buybets", language):
		// Временно просто возвращаем в меню аккаунта
		b.handleAccountCommand(message)
	case b.service.GetText("exitacc", language):
		b.handleHelpCommand(message) // Возврат в главное меню

		// Обработка кнопок меню FAQ
	case b.service.GetText("faqrules", language):
		b.handleFAQRules(message)
	case b.service.GetText("faqawards", language):
		b.handleFAQAwards(message)
	case b.service.GetText("faqpayments", language):
		b.handleFAQPayments(message)
	case b.service.GetText("faqfairplay", language):
		b.handleFAQFairPlay(message)
	case b.service.GetText("privacypolicy", language):
		b.handleFAQPrivacyPolicy(message)
	case b.service.GetText("contact", language):
		b.handleFAQContact(message)
	case b.service.GetText("faqexit", language):
		b.handleHelpCommand(message) // Возврат в главное меню

	default:
		// Обработка других текстовых сообщений
		b.handleGenericMessage(message)
	}
}

// handleCallbackQuery обрабатывает callback-запиты
func (b *Bot) handleCallbackQuery(query *telego.CallbackQuery) {
	// Валидация пользователя
	user := query.From
	go b.service.UpdateUserActivity(user.ID)

	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		// Регистрация пользователя, если он не найден
		dbUser, err = b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, "", user.LanguageCode)
		if err != nil {
			logger.Error.Printf("Error registering user: %v", err)
			return
		}
	}

	// Режим эмуляции
	if emulatedID, ok := emulatedUsers[user.ID]; ok {
		originalID := user.ID
		user.ID = emulatedID
		defer func() {
			user.ID = originalID
		}()
	}

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обновим язык пользователя из API, если он отличается от базы данных
	// Это обеспечит синхронизацию между API Telegram и нашей БД
	if user.LanguageCode != "" && user.LanguageCode != dbUser.LanguageCode {
		user.LanguageCode = dbUser.LanguageCode
	}

	if b.captchaBan(user.ID) {
		return
	}

	callbackData := query.Data

	// Обработка прохождения капчи
	if strings.HasPrefix(callbackData, CallbackCaptcha) {
		b.captchaCheck(query)
		return
	}

	// Проверка на повышенную активность пользователя
	switch b.captchaUserActivity(user.ID) {
	case "wait":
		return
	case "needCaptcha":
		b.SendMessage(query.Message.GetChat().ID, b.captchaMessage(user.ID, language))
		return
	}

	// Обработка нажатия на кнопку пагинации стран
	if strings.HasPrefix(callbackData, "country_page:") {
		// Извлекаем номер страницы
		pageStr := strings.TrimPrefix(callbackData, "country_page:")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			logger.Error.Printf("Error parsing page number: %v", err)
			b.answerCallbackQuery(query.ID, "Error", true)
			return
		}

		// Отвечаем на callback
		b.answerCallbackQuery(query.ID, "", false)

		// Обновляем сообщение с новой страницей стран
		if query.Message != nil {
			countryText := b.service.GetText("countrymes", language)
			b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
				Text:           countryText,
				InlineKeyboard: b.createCountriesKeyboard(page),
			})
		}
		return
	}

	// Проверяем, это выбор страны или другой callback
	if strings.HasPrefix(callbackData, "country:") {
		// Извлекаем код страны
		countryCode := strings.TrimPrefix(callbackData, "country:")

		// Получаем пользователя
		dbUser, err := b.service.GetUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.answerCallbackQuery(query.ID, "Error getting user info", true)
			return
		}

		// Проверяем, была ли у пользователя установлена страна ранее
		hadCountryBefore := dbUser.Country != ""

		// Отвечаем на callback
		b.answerCallbackQuery(query.ID, "", false)

		// Проверяем, является ли выбранная страна заблокированной (RU или BY)
		if countryCode == "RU" || countryCode == "BY" {
			// Получаем локализованный текст сообщения о блокировке
			banText := b.service.GetText("stopcountry", language)

			// Отправляем сообщение о недоступности сервиса
			if query.Message != nil {
				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text: banText,
				})
			}

			// Сохраняем выбранную страну и устанавливаем флаг бана
			dbUser.Country = countryCode
			dbUser.Banned = true
			if err := b.service.UpdateUser(dbUser); err != nil {
				logger.Error.Printf("Error updating user: %v", err)
			}

			return
		}

		// Для других стран - просто сохраняем выбор
		dbUser.Country = countryCode
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user country: %v", err)
			if query.Message != nil {
				b.SendMessage(query.Message.GetChat().ID, MessageOptions{
					Text: b.service.GetText("error", language),
				})
			}
			return
		}

		// Продолжаем обычную обработку для незабаненных пользователей
		if query.Message != nil {
			if !hadCountryBefore {
				// Если страны не было установлено ранее:
				// 1. Удаляем сообщение с выбором страны
				err = b.bot.DeleteMessage(b.ctx, &telego.DeleteMessageParams{
					ChatID:    telego.ChatID{ID: query.Message.GetChat().ID},
					MessageID: query.Message.GetMessageID(),
				})
				if err != nil {
					logger.Error.Printf("Error deleting country selection message: %v", err)
				}

				// Переходим к следующему шагу регистрации (выбор никнейма или проверка подписки)
				b.handleNicknamePrompt(query.Message.GetChat().ID, user.ID, language)
				return
			} else {
				// Если страна была установлена ранее:
				// Показываем подтверждение сохранения и кнопку назад
				successText := b.service.GetText("country_saved", language)

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           successText,
					InlineKeyboard: backBtn,
				})
			}
		}
	}

	// Обработка настроек
	if strings.HasPrefix(callbackData, "settings_") {
		// Отвечаем на callback
		b.answerCallbackQuery(query.ID, "", false)

		switch callbackData {
		case CallbackSettingsLanguage:
			languageText := b.service.GetText("settings_language", language)

			// Обновляем сообщение с выбором языка
			if query.Message != nil {
				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           languageText,
					InlineKeyboard: b.createLanguageKeyboard(language),
				})
			}
			return

		case CallbackSettingsCountry:
			countryText := b.service.GetText("countrymes", language)

			// Создаем клавиатуру со странами - начинаем с первой страницы
			countriesKeyboard := b.createCountriesKeyboard(1)

			// Обновляем сообщение с выбором страны
			if query.Message != nil {
				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           countryText,
					InlineKeyboard: countriesKeyboard,
				})
			}
			return

		case CallbackSettingsName:
			nameText := b.service.GetText("settings_name", language)

			// Обновляем сообщение и запрашиваем имя
			if query.Message != nil {
				// Устанавливаем состояние ожидания имени
				b.stateManager.SetState(user.ID, StateInputName, query.Message.GetMessageID())

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           nameText,
					InlineKeyboard: backBtn,
				})
			}
			return

		case CallbackSettingsNickName:
			nameText := b.service.GetText("settings_nickname", language)

			// Обновляем сообщение и запрашиваем публичное имя
			if query.Message != nil {
				// Устанавливаем состояние ожидания публичного имени
				b.stateManager.SetState(user.ID, StateInputUpNickname, query.Message.GetMessageID())

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           nameText,
					InlineKeyboard: backBtn,
				})
			}
			return

		case CallbackSettingsWallet:
			walletText := b.service.GetText("withdrawusdtchange", language)

			// Обновляем сообщение и запрашиваем адрес кошелька
			if query.Message != nil {
				// Устанавливаем состояние ожидания адреса кошелька
				b.stateManager.SetState(user.ID, StateInputWallet, query.Message.GetMessageID())

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           walletText,
					InlineKeyboard: backBtn,
				})
			}
			return

		case CallbackSettingsBack:
			// Возвращаемся к меню настроек
			settingsText := b.service.GetText("settings_message", language)

			// Очищаем состояние пользователя
			b.stateManager.ClearState(user.ID)

			if query.Message != nil {
				b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
					Text:           settingsText,
					InlineKeyboard: b.createSettingsKeyboard(language, user.ID), // передаем userID
				})
			}
			return

		case CallbackSettingsMainMenu:
			// Очищаем состояние пользователя
			b.stateManager.ClearState(user.ID)

			// Возвращаемся в главное меню
			b.sendMainMenu(query.Message.GetChat().ID, language)
			return
		}
	}

	// Обработка выбора языка
	if strings.HasPrefix(callbackData, "language_") {
		// Извлекаем код языка
		langCode := strings.TrimPrefix(callbackData, "language_")

		// Обновляем язык пользователя
		dbUser, err := b.service.GetUser(user.ID)
		if err != nil {
			b.answerCallbackQuery(query.ID, "Error updating language", true)
			return
		}

		// Обновляем язык пользователя в базе данных
		user.LanguageCode = langCode
		dbUser.LanguageCode = langCode
		if err := b.service.UpdateUserLanguage(dbUser.TelegramID, langCode); err != nil {
			b.answerCallbackQuery(query.ID, "Error saving language", true)
			return
		}

		// Обновляем язык пользователя для текущей сессии
		language = langCode // Обновляем локальную переменную language

		// Получаем локализованный текст для успешного сохранения языка
		// используя новый язык
		successText := b.service.GetText("language_saved", langCode)

		// Отвечаем на callback
		b.answerCallbackQuery(query.ID, "", false)

		// Обновляем сообщение подтверждением изменения языка
		if query.Message != nil {
			// Создаем кнопку назад с новым языком
			backBtn := b.createBackBtnKeyboard(langCode)

			b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
				Text:           successText,
				InlineKeyboard: backBtn,
			})
		}
		return
	}

	// Игнорируем пустой callback (обычно это кнопка индикатора текущей страницы)
	if callbackData == "noop" {
		b.answerCallbackQuery(query.ID, "", false)
		return
	}

	switch callbackData {
	// Обработка регистрации и заполнения данных пользователя
	case CallbackAgeVerify:
		b.handleAgeVerifyCallback(query)
		return
	case CallbackAgeVerifiedYes, CallbackAgeVerifiedNo:
		b.handleAgeVerificationCallback(query)
		return
	case CallbackCountry:
		b.handleCountryCallback(query)
		return
	case CallbackChangeName:
		b.handleNicknameCallback(query)
		return
	case CallbackChangeNameYes:
		// Обработка согласия на изменение никнейма
		b.handleChangeNameYes(query)
		return
	case CallbackChangeNameNo:
		// Обработка отказа от изменения никнейма
		b.handleChangeNameNo(query)
		return
	case CallbackReserveSubs:
		b.handleReserveSubscription(query)
		return
	case CallbackReserveSubsCheck:
		b.handleReserveSubscriptionCheck(query)
		return
	case "back_to_start":
		b.handleBackToStartMenu(query)
		return

	// Обработка других callback data
	case "rules":
		text := b.service.GetText("rules", language)
		b.updateOrSendMessage(query, text)
	case "awards":
		text := b.service.GetText("awards", language)
		b.updateOrSendMessage(query, text)
	case "payments":
		text := b.service.GetText("payments", language)
		b.updateOrSendMessage(query, text)
	case "fairplay":
		text := b.service.GetText("fairplay", language)
		b.updateOrSendMessage(query, text)

		// Гра
	case CallbackStartRound:
		b.gameHandler.handleStartRound(query)
	case CallbackBetRed:
		b.handleMakeBet(user.ID, models.Red)
		b.answerCallbackQuery(query.ID, "", false)
	case CallbackBetBlack:
		b.handleMakeBet(user.ID, models.Black)
		b.answerCallbackQuery(query.ID, "", false)
	case CallbackBetZero:
		b.handleMakeBet(user.ID, models.Zero)
		b.answerCallbackQuery(query.ID, "", false)
	case CallbackBetZeroLocked:
		// Обработка нажатия на заблокированную кнопку Zero
		_, remaining, _ := b.service.CanBetZero(user.ID)
		zeroText := b.service.GetText("zero_limit", language)
		zeroText = fmt.Sprintf(zeroText, remaining)
		// відправка повідомлення як toast pop-up
		b.answerCallbackQuery(query.ID, zeroText, false)
	case CallbackBetAvailable:
		b.gameHandler.handleAvailableBets(query)

	case CallbackBack:
		b.handleBackToMainMenu(query)
	case "view_rating":
		b.handleRatingCallbackQuery(query)
		b.answerCallbackQuery(query.ID, "", false)
	case CallbackRequestWithdraw:
		b.handleRequestWithdrawCallback(query)
	case CallbackCheckWallet:
		b.handleCheckWalletCallback(query)
	case CallbackChangeWallet:
		options := b.prepareMessage("withdrawusdtchange", language)

		// Обновляем сообщение и запрашиваем адрес кошелька
		if query.Message != nil {
			// Устанавливаем состояние ожидания адреса кошелька
			b.stateManager.SetState(user.ID, StateInputWithdrawWallet, query.Message.GetMessageID())
			options.ReplyKeyboard = b.createAccountKeyboard(language)
			b.SendMessage(query.Message.GetChat().ID, options)
		}
		return
	case CallbackCancelInput:
		b.stateManager.ClearState(user.ID)

		// Получаем локализованный текст раздела аккаунта
		options := b.prepareMessage("accstart", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		b.SendMessage(query.Message.GetChat().ID, options)
		return
	case CallbackCheckAmount:
		b.handleCheckAmountCallback(query)
	case CallbackSetAmount:
		options := b.prepareMessage("withdrawusdtsumam", language)

		// Посылаем сообщение и запрашиваем сумму для вывода
		if query.Message != nil {
			// Устанавливаем состояние ожидания суммы для вывода
			b.stateManager.SetState(user.ID, StateInputWithdrawAmount, query.Message.GetMessageID())
			options.ReplyKeyboard = b.createAccountKeyboard(language)
			b.SendMessage(query.Message.GetChat().ID, options)
		}
		return

	case CallbackProcessWithdraw:
		b.handleProcessWithdrawCallback(query)
	case "stop_game":
		b.gameHandler.HandleStopGameButton(user.ID)
		b.answerCallbackQuery(query.ID, "", false)

		if message, ok := query.Message.(*telego.Message); ok {
			b.handleHelpCommand(message)
		}
	default:
		// Неизвестный callback
		b.answerCallbackQuery(query.ID, "Unknown action", true)
	}
}

// Обработчики команд

// Вспомогательный метод для отправки главного меню
func (b *Bot) sendMainMenu(chatID int64, language string) {
	options := b.prepareMessage("main_menu", language)
	options.ReplyKeyboard = b.createMainReplyKeyboard(language)
	b.SendMessage(chatID, options)
}

// handleBackToStartMenu обработка callback для возврата к стартовому меню
func (b *Bot) handleBackToStartMenu(query *telego.CallbackQuery) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback
	b.answerCallbackQuery(query.ID, "", false)

	// Получаем локализированный текст приветствия
	welcomeText := b.service.GetText("startmessage1", language)

	// Создаем inline клавиатуру для стартового сообщения
	inlineKeyboard := b.createStartInlineKeyboard(language)

	// Обновляем сообщение
	if query.Message != nil {
		b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
			Text:           welcomeText,
			InlineKeyboard: inlineKeyboard,
		})
	} else {
		// Если сообщение недоступно, отправляем новое
		b.SendMessage(query.From.ID, MessageOptions{
			Text:           welcomeText,
			InlineKeyboard: inlineKeyboard,
		})
	}
}

// updateOrSendMessage обновляет существующее сообщение или отправляет новое
func (b *Bot) updateOrSendMessage(query *telego.CallbackQuery, text string) {
	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Создаем кнопку "Назад"
	backButton := telego.InlineKeyboardButton{
		Text:         b.service.GetText("btn_back", language),
		CallbackData: "back_to_start",
	}
	keyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{backButton},
		},
	}

	if query.Message != nil {
		// Обновляем существующее сообщение
		b.UpdateMessage(query.Message.GetChat().ID, query.Message.GetMessageID(), MessageOptions{
			Text:           text,
			InlineKeyboard: keyboard,
		})
	} else {
		// Отправляем новое сообщение
		b.SendMessage(query.From.ID, MessageOptions{
			Text:           text,
			InlineKeyboard: keyboard,
		})
	}

	// Отвечаем на callback
	b.answerCallbackQuery(query.ID, "", false)
}

func (b *Bot) handleHelpCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отправляем главное меню
	b.sendMainMenu(message.Chat.ID, language)
}

func (b *Bot) handleStatsCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для стартового сообщения статистики
	options := b.prepareMessage("statisticsstart", language)

	// Создаем клавиатуру выбора периода статистики
	options.ReplyKeyboard = b.createStatsKeyboard(language)

	// Отправляем сообщение с клавиатурой для выбора периода
	b.SendMessage(message.Chat.ID, options)
}

// handleUnknownCommand обрабатывает неизвестные команды
func (b *Bot) handleUnknownCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отправляем сообщение о неизвестной команде
	b.SendMessage(message.Chat.ID, b.prepareMessage("unknown_command", language))

	// Отправляем главное меню
	b.sendMainMenu(message.Chat.ID, language)
}

// handleGenericMessage обрабатывает обычные текстовые сообщения
func (b *Bot) handleGenericMessage(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отправляем главное меню
	b.sendMainMenu(message.Chat.ID, language)
}

// handleBackToMainMenu обрабатывает возврат к главному меню
func (b *Bot) handleBackToMainMenu(query *telego.CallbackQuery) {
	b.answerCallbackQuery(query.ID, "", false)

	user := query.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Если у сообщения есть чат, отправляем в него главное меню
	if query.Message != nil {
		b.sendMainMenu(query.Message.GetChat().ID, language)
	} else {
		// Иначе отправляем в личку пользователю
		b.sendMainMenu(user.ID, language)
	}
}

// handleDayStatistics обрабатывает показ статистики за день
func (b *Bot) handleDayStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "day")
}

// handleWeekStatistics обрабатывает показ статистики за неделю
func (b *Bot) handleWeekStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "week")
}

// handleMonthStatistics обрабатывает показ статистики за месяц
func (b *Bot) handleMonthStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "month")
}

// handleAllStatistics обрабатывает показ статистики за все время
func (b *Bot) handleAllStatistics(message *telego.Message) {
	b.showStatisticsForPeriod(message, "all")
}

// showStatisticsForPeriod показывает статистику для выбранного периода
func (b *Bot) showStatisticsForPeriod(message *telego.Message, period string) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем подробную статистику пользователя
	detailedStats, err := b.service.GetDetailedUserStats(user.ID, period)
	if err != nil {
		logger.Error.Printf("Error getting detailed stats: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error retrieving statistics. Please try again.",
		})
		return
	}

	// Формируем ключ для текста статистики в зависимости от периода
	statsMsgKey := period + "statm"

	// Получаем шаблон для соответствующего периода
	options := b.prepareMessage(statsMsgKey, language)

	// Сформируем текст для вставки в шаблон
	totalBets := detailedStats["totalBets"]
	blackBets := detailedStats["blackBets"]
	redBets := detailedStats["redBets"]
	zeroBets := detailedStats["zeroBets"]

	wonBets := detailedStats["wonBets"]
	wonBlackBets := detailedStats["wonBlackBets"]
	wonRedBets := detailedStats["wonRedBets"]
	wonZeroBets := detailedStats["wonZeroBets"]

	lostBets := detailedStats["lostBets"]
	lostBlackBets := detailedStats["lostBlackBets"]
	lostRedBets := detailedStats["lostRedBets"]
	lostZeroBets := detailedStats["lostZeroBets"]

	totalPoints := detailedStats["totalPoints"]

	// Заполняем шаблон данными
	options.Text = fmt.Sprintf(
		options.Text,
		totalBets, blackBets, redBets, zeroBets,
		wonBets, wonBlackBets, wonRedBets, wonZeroBets,
		lostBets, lostBlackBets, lostRedBets, lostZeroBets,
		totalPoints,
	)
	options.ReplyKeyboard = b.createStatsKeyboard(language)

	// Отправляем статистику пользователю
	b.SendMessage(message.Chat.ID, options)

	// Отправляем сообщение с предложением выбрать другой период
	options = b.prepareMessage("statistics_next", language)
	options.ReplyKeyboard = b.createStatsKeyboard(language)
	b.SendMessage(message.Chat.ID, options)
}

// Допоміжні методи

// Відповідь на callback-запит
func (b *Bot) answerCallbackQuery(queryID string, text string, showAlert bool) {
	err := b.bot.AnswerCallbackQuery(b.ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: queryID,
		Text:            text,
		ShowAlert:       showAlert,
	})
	if err != nil {
		logger.Error.Printf("Error answering callback query: %v", err)
	}
}

const (
	sendMessage      = "sendMessage"
	editMessageText  = "editMessageText"
	sendPhoto        = "sendPhoto"
	sendVideo        = "sendVideo"
	editMessageMedia = "editMessageMedia"
	sendSticker      = "sendSticker"
)

// MessageOptions содержит опции для отправки или обновления сообщения
type MessageOptions struct {
	// Время создания сообщения Unix (Нужно чтобы sorted set не перезаписывал сообщение)
	CreatedAt int64
	// Время жизни сообщения, если не указано будет взято из const userQueueExpiration
	// Используется для определения очередности доставки сообщения!
	// Если указать время то внутри очереди пользователя на отправку оно переместися
	// ORDER отправки == TTL
	TTL time.Duration

	// MethodName - Метод телеграма
	MethodName string

	// MessageID - ID сообщения для изменения или удаления
	MessageID int

	// Text - текст сообщения
	Text string

	// PhotoPath - путь к фото (если установлен, будет отправлено фото с подписью Text)
	PhotoPath string
	DelPhoto  bool // true для временных фото, после отправки запускается удаление через какое то время после отправки
	// PhotoFileID - FileID фото (если установлен, будет отправлено фото с подписью Text)
	PhotoFileID string

	// VideoPath - путь к видео (если установлен, будет отправлено видео с подписью Text)
	VideoPath string
	// VideoFileID - FileID видео (если установлен, будет отправлено видео с подписью Text)
	VideoFileID string

	// InlineKeyboard - инлайн клавиатура (если установлена, будет добавлена к сообщению)
	InlineKeyboard *telego.InlineKeyboardMarkup

	// ReplyKeyboard - клавиатура ответа (если установлена, будет добавлена к сообщению)
	ReplyKeyboard *telego.ReplyKeyboardMarkup

	// RemoveKeyboard - если true, клавиатура ответа будет удалена
	RemoveKeyboard bool

	// OneTimeKeyboard - если true и установлено ReplyKeyboard, клавиатура будет одноразовой
	OneTimeKeyboard bool

	// Selective - если true и установлено ReplyKeyboard, клавиатура будет показана только определенным пользователям
	Selective bool

	// ParseMode - режим форматирования текста (HTML, Markdown, MarkdownV2)
	ParseMode string

	// Entities - специальные сущности (эмодзи, форматирование и т.д.)
	Entities []telego.MessageEntity

	// DisableWebPagePreview - если true, превью ссылок будет отключено
	DisableWebPagePreview bool

	// DisableNotification - если true, сообщение будет отправлено беззвучно
	DisableNotification bool
}

// prepareMessage - подготовка сообщения - установка текста и фото/видео если указаны
func (b *Bot) prepareMessage(key, languageCode string) (options MessageOptions) {
	res, _ := b.service.GetRepo().GetLocalization(key, languageCode)

	return MessageOptions{
		Text:        res.Value,
		PhotoFileID: res.Image,
		VideoFileID: res.Video,
	}
}

// SendMessage отправляет новое сообщение с указанными опциями
func (b *Bot) SendMessage(chatID int64, options MessageOptions) error {
	// Обрабатываем текст, заменяя литеральные \r\n на реальные переносы строк
	// Используем двойной проход для избежания проблем с экранированием
	processedText := strings.ReplaceAll(options.Text, "\\r\\n", "\n")
	processedText = strings.ReplaceAll(processedText, "\r\n", "\n")
	options.Text = processedText

	// Собираем параметры для замены макросов
	params := make(map[string]interface{})

	// Добавляем глобальные макросы
	for key, value := range b.service.GetGlobalMacros() {
		params[key] = value
	}
	// Заменяем макросы в текстах с помощью общей функции
	_, options.Text, _ = utils.ReplaceMacrosInTexts("", options.Text, "", params)

	// Проверка на наличие шаблонов эмодзи в тексте
	if strings.Contains(options.Text, "{{emoji:") {
		// Обрабатываем кастомные эмодзи в формате {{emoji:id}}
		emojiText, emojiEntities := utils.BuildMessageWithCustomEmojis(options.Text)
		options.Text = emojiText

		// Объединяем существующие сущности с новыми эмодзи-сущностями
		if len(emojiEntities) > 0 {
			if len(options.Entities) > 0 {
				options.Entities = append(options.Entities, emojiEntities...)
			} else {
				options.Entities = emojiEntities
			}
		}
	}

	// Определяем тип сообщения
	if options.PhotoPath != "" || options.PhotoFileID != "" {
		options.MethodName = sendPhoto
	} else if options.VideoPath != "" || options.VideoFileID != "" {
		options.MethodName = sendVideo
	} else {
		options.MethodName = sendMessage
	}

	// Устанавливаем в очередь на отправку
	return b.MakeRequestDeferred(chatID, 0, options)
}

// UpdateMessage обновляет существующее сообщение с указанными опциями
func (b *Bot) UpdateMessage(chatID int64, messageID int, options MessageOptions) error {
	// Если указан путь к фото
	if options.PhotoPath != "" {
		// Для фото с локального источника необходимо удалить старое сообщение и отправить новое
		err := b.bot.DeleteMessage(b.ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else if options.PhotoFileID != "" {
		// Обновление фото по FileID
		options.MethodName = editMessageMedia
		options.MessageID = messageID
		// Устанавливаем в очередь на отправку
		return b.MakeRequestDeferred(chatID, 0, options)

	} else if options.ReplyKeyboard != nil || options.RemoveKeyboard {
		// Для ReplyKeyboard необходимо удалить старое сообщение и отправить новое
		err := b.bot.DeleteMessage(b.ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else {
		options.MethodName = editMessageText
		options.MessageID = messageID

		// Устанавливаем в очередь на отправку
		return b.MakeRequestDeferred(chatID, 0, options)
	}
}

// sendText отправляет текстовое сообщение
func (b *Bot) sendText(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      options.Text,
		ParseMode: options.ParseMode,
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: options.DisableWebPagePreview,
		},
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности, если они есть
	if len(options.Entities) > 0 {
		params.Entities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	msg, err := b.bot.SendMessage(b.ctx, params)
	if err != nil {
		logger.Error.Printf("Error sending message: %v", err)
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return msg, nil
}

// updateText обновляет текстовое сообщение
func (b *Bot) updateText(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	params := &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      options.Text,
		ParseMode: options.ParseMode,
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: options.DisableWebPagePreview,
		},
	}

	// Добавляем сущности, если они есть
	if len(options.Entities) > 0 {
		params.Entities = options.Entities
	}

	// Для обновления можно использовать только инлайн клавиатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageText(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return msg, nil
}

// sendVideo отправляет видео с подписью
func (b *Bot) sendVideo(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Параметры для отправки
	params := &telego.SendVideoParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		params.CaptionEntities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	// Выбираем источник видео
	if options.VideoFileID != "" {
		// Для FileID используем его непосредственно
		// params.Video = telego.InputFile{FileID: options.VideoFileID}
		params.Video = tu.FileFromID(options.VideoFileID)
		return b.bot.SendVideo(b.ctx, params)
	} else if options.VideoPath != "" {
		// Для файла используем метод Upload
		return b.sendVideoFile(chatID, options.VideoPath, params)
	}

	return nil, fmt.Errorf("no video source specified")
}

// sendPhoto отправляет фото с подписью
func (b *Bot) sendPhoto(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Параметры для отправки
	params := &telego.SendPhotoParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		params.CaptionEntities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	// Выбираем источник фото
	if options.PhotoFileID != "" {
		// Для FileID используем его непосредственно
		// params.Photo = telego.InputFile{FileID: options.PhotoFileID}
		params.Photo = tu.FileFromID(options.PhotoFileID)
		return b.bot.SendPhoto(b.ctx, params)
	} else if options.PhotoPath != "" {
		// Для файла используем метод Upload
		return b.sendPhotoFile(chatID, options.PhotoPath, options.DelPhoto, params)
	}

	return nil, fmt.Errorf("no photo source specified")
}

// sendVideoFile отправляет видео с локального файла
func (b *Bot) sendVideoFile(chatID int64, videoPath string, params *telego.SendVideoParams) (*telego.Message, error) {
	// Открываем файл
	file, err := os.Open(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()
	params.Video = tu.File(file)

	// Отправляем фото
	msg, err := b.bot.SendVideo(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send video: %w", err)
	}

	return msg, nil
}

// sendPhotoFile отправляет фото с локального файла
func (b *Bot) sendPhotoFile(chatID int64, photoPath string, delPhoto bool, params *telego.SendPhotoParams) (*telego.Message, error) {
	// Открываем файл
	file, err := os.Open(photoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open photo file: %w", err)
	}
	defer file.Close()

	if delPhoto {
		// Если был передан параметр на удаление
		// то через какое то время запускаем функцию удаления фото
		go func() {
			time.Sleep(20 * time.Second)
			e := os.Remove(photoPath)
			logger.Error.Println("Remove photo file")
			if e != nil {
				logger.Error.Println("failed to remove photo file: %w", err)
			}
		}()
	}

	// Устанавливаем загруженный файл
	params.Photo = tu.File(file)

	// Отправляем фото
	msg, err := b.bot.SendPhoto(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send photo: %w", err)
	}

	return msg, nil
}

// updatePhotoByFileID обновляет фото по FileID
func (b *Bot) updatePhotoByFileID(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Создаем объект InputMediaPhoto с FileID
	mediaPhoto := &telego.InputMediaPhoto{
		Type:      "photo",
		Media:     tu.FileFromID(options.PhotoFileID),
		Caption:   options.Text,
		ParseMode: options.ParseMode,
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		mediaPhoto.CaptionEntities = options.Entities
	}

	params := &telego.EditMessageMediaParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Media:     mediaPhoto,
	}

	// Для обновления можно использовать только инлайн клавиатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageMedia(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}

	return msg, nil
}

// getReplyMarkup возвращает соответствующую клавиатуру на основе опций
func (b *Bot) getReplyMarkup(options MessageOptions) telego.ReplyMarkup {
	if options.InlineKeyboard != nil {
		return options.InlineKeyboard
	}

	if options.RemoveKeyboard {
		return &telego.ReplyKeyboardRemove{
			RemoveKeyboard: true,
			Selective:      options.Selective,
		}
	}

	if options.ReplyKeyboard != nil {
		// Клонируем клавиатуру, чтобы не изменять оригинал
		keyboard := *options.ReplyKeyboard
		keyboard.OneTimeKeyboard = options.OneTimeKeyboard
		keyboard.Selective = options.Selective
		return &keyboard
	}

	return nil
}

// SendSticker отправляет стикер
func (b *Bot) SendSticker(chatID int64, stickerFileID string) error {
	_, err := b.bot.SendSticker(b.ctx, &telego.SendStickerParams{
		ChatID:  telego.ChatID{ID: chatID},
		Sticker: tu.FileFromID(stickerFileID),
	})
	return err
}

// Случайный выбор стикера из двух вариантов
func getRandomSticker(sticker1, sticker2 string) string {
	if time.Now().UnixNano()%2 == 0 {
		return sticker1
	}
	return sticker2
}

// handleInputNicknameState обрабатывает ввод никнейма при регистрации
func (b *Bot) handleInputNicknameState(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обработка ввода никнейма при регистрации
	if len(message.Text) > 0 {
		// Проверяем валидность никнейма (только латинские буквы и цифры)
		nickname := strings.TrimSpace(message.Text)
		isValid := true

		// Проверяем, что никнейм состоит только из разрешенных символов
		for _, r := range nickname {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				isValid = false
				break
			}
		}

		if !isValid || len(nickname) < 3 || len(nickname) > 20 {
			// Никнейм невалиден, отправляем сообщение об ошибке

			b.SendMessage(message.Chat.ID, b.prepareMessage("invalid_nickname", user.LanguageCode))
			return
		}

		// Обновляем никнейм пользователя
		dbUser, err := b.service.GetUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}

		// Сохраняем никнейм в отдельное поле Nickname
		dbUser.Nickname = nickname
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user nickname: %v", err)
		}

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)

		// Отправляем запрос на подписку
		b.sendSubscriptionRequest(message.Chat.ID, user.LanguageCode, "name_changesave_msg_start")
	}
}

// handleInputNameState обрабатывает ввод имени в настройках
func (b *Bot) handleInputNameState(message *telego.Message, messageID int) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обработка ввода имени
	if len(message.Text) > 0 {
		// Валидация имени
		name := strings.TrimSpace(message.Text)
		if len(name) == 0 || len(name) > 100 {
			// Неверная длина имени
			b.SendMessage(message.Chat.ID, b.prepareMessage("invalid_name", language))
			return
		}

		// Обновляем имя пользователя
		dbUser, err := b.service.GetUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}

		dbUser.FirstName = name
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user name: %v", err)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, b.prepareMessage("update_error", language))
			return
		}

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("name_saved", language)

		backBtn := b.createBackBtnKeyboard(language)

		b.UpdateMessage(message.Chat.ID, messageID, MessageOptions{
			Text:           successText,
			InlineKeyboard: backBtn,
		})

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)
	}
}

// handleInputUpNicknameState обрабатывает ввод никнейма в настройках
func (b *Bot) handleInputUpNicknameState(message *telego.Message, messageID int) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обработка ввода никнейма при обновлении в настройках
	if len(message.Text) > 0 {
		// Обновляем никнейм пользователя
		dbUser, err := b.service.GetUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}

		// Проверяем валидность никнейма (только латинские буквы, цифры и подчеркивание)
		nickname := strings.TrimSpace(message.Text)
		isValid := true

		// Проверяем, что никнейм состоит только из разрешенных символов
		for _, r := range nickname {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				isValid = false
				break
			}
		}

		if !isValid || len(nickname) < 3 || len(nickname) > 20 {
			// Никнейм невалиден, отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, b.prepareMessage("invalid_nickname", language))
			return
		}

		// Обновляем никнейм пользователя
		dbUser.Nickname = nickname
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user nickname: %v", err)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, b.prepareMessage("update_error", language))
			return
		}

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("nickname_saved", language)

		backBtn := b.createBackBtnKeyboard(language)

		b.UpdateMessage(message.Chat.ID, messageID, MessageOptions{
			Text:           successText,
			InlineKeyboard: backBtn,
		})

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)
	}
}

// handleInputWalletState обрабатывает ввод адреса кошелька в настройках
func (b *Bot) handleInputWalletState(message *telego.Message, messageID int) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обработка ввода адреса кошелька
	if len(message.Text) > 0 {
		// Проверка валидности адреса кошелька (базовая проверка)
		walletAddress := strings.TrimSpace(message.Text)

		// Базовая валидация адреса TRC20
		if !strings.HasPrefix(walletAddress, "T") || len(walletAddress) < 30 {
			// Неверный формат кошелька
			options := b.prepareMessage("withdrawusdtchangeerror", language)

			// Создаем клавиатуру с кнопкой назад
			options.InlineKeyboard = b.createBackBtnKeyboard(language)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, options)
			return
		}

		// Обновляем адрес кошелька пользователя
		dbUser, err := b.service.GetUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}

		dbUser.WalletAddress = walletAddress
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user wallet address: %v", err)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, b.prepareMessage("update_error", language))
			return
		}

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("withdrawusdtchangeok", language)

		backBtn := b.createBackBtnKeyboard(language)

		b.UpdateMessage(message.Chat.ID, messageID, MessageOptions{
			Text:           successText,
			InlineKeyboard: backBtn,
		})

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)
	}
}
