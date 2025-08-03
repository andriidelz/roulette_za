package bot

import (
	"context"
	"fmt"
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

	CallbackAgeVerifiedYes      = "age_verified_yes"
	CallbackAgeVerifiedNo       = "age_verified_no"
	CallbackChangeNameYes       = "name_changeyes"
	CallbackChangeNameNo        = "name_changeno"
	CallbackReserveSubscription = "reservsubs"
	CallbackBetRed              = "bet_red"
	CallbackBetBlack            = "bet_black"
	CallbackBetZero             = "bet_zero"
	CallbackBack                = "back"
	CallbackCaptchaCorrect      = "captcha_correct"
	CallbackCaptchaIncorrect    = "captcha_incorrect"

	StickerNoBids    = "CAACAgUAAxkBAAEORLpn9lEBwqSME7WwehtZBLt5ybqSrAACKRUAAvWxqVeH8hhzfq9SEjYE" // nomorebids
	StickerWin       = "CAACAgUAAxkBAAEORLxn9lEJolSTKIZrUxOLZbkMChpdWwACuBcAArzBqVdjiSsft06GCjYE" // win
	StickerLose      = "CAACAgUAAxkBAAEORL5n9lEOq_kczbL1CGpgN5-UhhhgqQAC3BIAAtGwqVdlepoFId2tMzYE" // lose
	StickerBlackRes1 = "CAACAgUAAxkBAAEORMBn9lEUB8KMRJ8nduCQ-y32y5ns4AACNBUAArIsqVfUvoMXgG8VvzYE" // blackresult (вариант 1)
	StickerBlackRes2 = "CAACAgUAAxkBAAEORMJn9lEXC6ByJRCY4_8Mu5vQQP-1zgACOxYAAnWOqVesBnNzFycGfDYE" // blackresult (вариант 2)
	StickerRedRes1   = "CAACAgUAAxkBAAEORMhn9lEfopgbb8y7qi__V8deZr0MpAACYBcAAs4bqVeFX-l3HDBIFjYE" // redresult (вариант 1)
	StickerRedRes2   = "CAACAgUAAxkBAAEORMpn9lEiRobEQnz4qg6GFSmfZQmjbwACiRgAAhuTqVdgysjb-Y-sLTYE" // redresult (вариант 2)
	StickerZeroRes1  = "CAACAgUAAxkBAAEORMRn9lEar58eDwvent8Lp3TvMRvF5AACtxEAAlRRsFdySRXPzXyVqzYE" // zeroresult (вариант 1)
	StickerZeroRes2  = "CAACAgUAAxkBAAEORMZn9lEd12gNsWFFxGXLAZoeJbSEsgACCxYAAmDwqVdsE7WC-rayWDYE" // zeroresult (вариант 2)
	// Стикер ошибки отправки сообщения, если словили 429 ошибку
	StickerError = "CAACAgUAAxkBAAEO5upob-zRQ5ptM0PmCYlvTra-KSbbiQACEBYAAkV9qVf5P89H45HU5zYE" // error
)

var ReserveChannelID = "@socialroulette_dev" // https://t.me/socialroulette_dev

func init() {
	// Инициализируем конфигурацию
	cfg := config.NewConfig()
	ReserveChannelID = cfg.TelegramReserveChannelID
}

func (b *Bot) getMetrics() *metrics.Metrics {
	if b == nil || b.metrics == nil {
		return nil
	}
	return b.metrics
}

// NewBot создает новый экземпляр бота
func NewBot(token string, service service.Service, cfg *config.Config) (*Bot, error) {
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
	me, err := b.bot.GetMe()
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
	updates, err := b.bot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
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

	// Останавливаем длинный поллинг
	b.bot.StopLongPolling()

	// Отменяем контекст
	b.cancel()

	// Останавливаем игровой обработчик
	b.gameHandler.Stop()

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

// handleNicknamePrompt отправляет запрос на подтверждение/изменение никнейма
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
	privacyPolicyText := b.service.GetText("privacypolicym", language)

	// Отправляем текст privacy policy
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          privacyPolicyText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
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
	contactText := b.service.GetText("contactm", language)

	// Отправляем текст о контакте с админом
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          contactText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

// sendSubscriptionRequest отправляет запрос на подписку на резервный канал
func (b *Bot) sendSubscriptionRequest(chatID int64, language string) {
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

	b.SendMessage(chatID, MessageOptions{
		Text: b.service.GetText("startmessage2", language),
		InlineKeyboard: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{channelButton},
				{subscribeButton},
			},
		},
	})
}

// handleReserveSubscriptionCheck обрабатывает нажатие кнопки проверки подписки
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
				Text: successText,
			})
		}

		// Отправляем главное меню после небольшой задержки
		go func() {
			time.Sleep(2 * time.Second)
			b.sendMainMenu(query.Message.Chat.ID, language)
		}()
	} else {
		// Подписка не подтверждена
		failText := b.service.GetText("reservno", language)

		// Обновляем сообщение с информацией о неудаче
		if query.Message != nil {
			b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
				Text: failText,
			})

			// Повторно отправляем запрос на подписку через 3 секунды
			go func() {
				time.Sleep(3 * time.Second)
				b.sendSubscriptionRequest(query.Message.Chat.ID, language)
			}()
		}
	}
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

	// Получаем доступное количество ставок
	betsBalance, err := b.service.GetUserRemainingBets(userID)
	if err != nil {
		logger.Error.Printf("Error getting user remaining bets: %v", err)
		betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
	}

	// Проверяем активность пользователя
	// - беспрерывная игра
	// - ставка на одну и ту же опцию
	switch b.captchaBetActivity(userID) {
	case "needCaptcha":
		b.SendMessage(userID, b.captchaMessage(userID, language))
		return
	}
	switch b.captchaBetDuplicate(userID, string(option)) {
	case "needCaptcha":
		b.SendMessage(userID, b.captchaMessage(userID, language))
		return
	}

	// Вызываем MakeBet и обрабатываем возможные ошибки
	err = b.gameHandler.MakeBet(userID, option)
	if err != nil {
		// Определяем тип ошибки и отправляем соответствующее сообщение
		var errorText string
		if strings.Contains(err.Error(), "already made a bet") {
			// Пользователь уже сделал ставку в этом раунде
			errorText = b.service.GetText("bet_already_made", language)
		} else if strings.Contains(err.Error(), "cannot bet on zero") {
			// Пользователь не может ставить на Zero
			canBetZero, remaining, _ := b.service.CanBetZero(userID)
			if !canBetZero {
				zeroLimitText := b.service.GetText("zero_limit", language)
				errorText = fmt.Sprintf(zeroLimitText, remaining)
			} else {
				errorText = b.service.GetText("bet_error", language)
			}
		} else if strings.Contains(err.Error(), "no bets left") {
			// У пользователя закончились ставки на сегодня
			errorText = b.service.GetText("betsbalancelow", language)
		} else {
			// Общая ошибка ставки

			logger.Error.Println("bet_error", err)
			errorText = b.service.GetText("bet_error", language)
		}

		b.SendMessage(userID, MessageOptions{
			Text:          errorText,
			ReplyKeyboard: b.gameHandler.createDetailedBetKeyboard(language, userID, betsBalance),
		})
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

	// Получаем данные пользователя из базы
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		// Если пользователь новый выполняется его предварительная регистрация,
		// независимо от того какую команду он выполнил
		// handleStartCommand только для обновления незаполненных полей

		// Получаем источник
		userSource := ""
		if strings.Contains(message.Text, " ") {
			commandArgs := strings.Split(message.Text, " ")
			if len(commandArgs) > 1 {
				userSource = strings.TrimSpace(commandArgs[1]) // Получаем источник
			}
		}
		dbUser, err = b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, userSource, user.LanguageCode)
		if err != nil {
			logger.Error.Printf("Error registering user: %v", err)
		} else {
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
			b.handleStartCommand(message)
			return
		case CommandPrivacy:
			// Команды privacy и contact доступны всегда (не требуют завершения регистрации)
			b.handlePrivacyCommand(message)
			return
		case CommandContact:
			b.handleContactCommand(message)
			return
		case CommandPlay:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				return
			}
			b.gameHandler.HandlePlayCommand(message)
			return
		case CommandStats:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				return
			}
			b.handleStatsCommand(message)
			return
		case CommandRating:
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				return
			}
			b.handleRatingCommand(message)
			return
		case CommandFAQ:
			// FAQ доступен всегда
			b.handleFAQCommand(message)
			return
		case CommandSettings:
			if !b.RequireCompleteRegistration(message.Chat.ID, user.ID) {
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
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputUpNicknameState(message, messageID)
			return
		case StateInputWallet:
			// Для ввода кошелька требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputWalletState(message, messageID)
			return
		case StateInputWithdrawAmount:
			// Для вывода средств требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputWithdrawAmountCommand(message)
			return
		case StateInputWithdrawWallet:
			// Для вывода средств требуется завершенная регистрация
			if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
				b.stateManager.ClearState(user.ID)
				return
			}
			b.handleInputWithdrawWalletCommand(message)
			return
		}
	}

	// Для всех остальных текстовых команд требуется завершенная регистрация
	if !b.RequireCompleteRegistration(message.Chat.ID, message.From.ID) {
		return
	}

	// Получаем локализированные тексты для кнопок
	btnPlayText := b.service.GetText("btn_play", language)
	btnStatisticsText := b.service.GetText("btn_statistics", language)
	btnRatingText := b.service.GetText("btn_rating", language)
	btnAccountText := b.service.GetText("btn_account", language)
	btnFAQText := b.service.GetText("btn_faq", language)

	btnRedText := b.service.GetText("btn_bet_red", language)
	btnBlackText := b.service.GetText("btn_bet_black", language)
	btnZeroText := b.service.GetText("btn_bet_zero", language)
	btnZeroLockedText := b.service.GetText("btn_bet_zero_locked", language)
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
	case btnRedText:
		b.handleMakeBet(user.ID, models.Red)
	case btnBlackText:
		b.handleMakeBet(user.ID, models.Black)
	case btnZeroText:
		b.handleMakeBet(user.ID, models.Zero)
	case btnZeroLockedText:
		// Обработка нажатия на заблокированную кнопку Zero
		canBetZero, remaining, _ := b.service.CanBetZero(user.ID)
		if !canBetZero {
			zeroLimitText := b.service.GetText("zero_limit", language)
			zeroLimitText = fmt.Sprintf(zeroLimitText, remaining)

			// Получаем доступное количество ставок
			betsBalance, err := b.service.GetUserRemainingBets(user.ID)
			if err != nil {
				logger.Error.Printf("Error getting user remaining bets: %v", err)
				betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
			}

			b.SendMessage(message.Chat.ID, MessageOptions{
				Text:          zeroLimitText,
				ReplyKeyboard: b.gameHandler.createDetailedBetKeyboard(language, user.ID, betsBalance),
			})
		}
	case b.service.GetText("availablebets", language):
		// Получаем доступное количество ставок
		betsBalance, err := b.service.GetUserRemainingBets(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user remaining bets: %v", err)
			betsBalance = -1 // Если ошибка, ставим отрицательное значение (безлимитное)
		}

		var messageText string
		if betsBalance <= 0 {
			messageText = b.service.GetText("betsbalancelow", language)
		} else {
			messageTemplate := b.service.GetText("betsbalanceok", language)
			messageText = fmt.Sprintf(messageTemplate, betsBalance)
		}

		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          messageText,
			ReplyKeyboard: b.gameHandler.createDetailedBetKeyboard(language, user.ID, betsBalance),
		})
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

	callbackData := query.Data

	// Обработка прохождения капчи
	switch callbackData {
	case CallbackCaptchaIncorrect:
		// ничего не отправляем пока капча не будет пройдена.
		// Слишком большое кол-во сообщений к пользователю повышает риск бана бота
		return
	case CallbackCaptchaCorrect:
		b.captchaCorrect(query)
		return
	}

	// Проверка на повышенную активность пользователя
	switch b.captchaUserActivity(user.ID) {
	case "wait":
		return
	case "needCaptcha":
		b.SendMessage(query.Message.Chat.ID, b.captchaMessage(user.ID, language))
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
			b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
			if banText == "stopcountry" { // если локализация не найдена
				banText = "Сервис не доступен для жителей россии или белларуси" // Запасной вариант
			}

			// Отправляем сообщение о недоступности сервиса
			if query.Message != nil {
				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.SendMessage(query.Message.Chat.ID, MessageOptions{
					Text: "Произошла ошибка при сохранении страны. Пожалуйста, попробуйте еще раз.",
				})
			}
			return
		}

		// Продолжаем обычную обработку для незабаненных пользователей
		if query.Message != nil {
			if !hadCountryBefore {
				// Если страны не было установлено ранее:
				// 1. Удаляем сообщение с выбором страны
				err = b.bot.DeleteMessage(&telego.DeleteMessageParams{
					ChatID:    telego.ChatID{ID: query.Message.Chat.ID},
					MessageID: query.Message.MessageID,
				})

				if err != nil {
					logger.Error.Printf("Error deleting country selection message: %v", err)
				}

				// Переходим к следующему шагу регистрации (выбор никнейма или проверка подписки)
				if b.handleNicknamePrompt != nil {
					// Если у вас есть функция для выбора никнейма
					b.handleNicknamePrompt(query.Message.Chat.ID, user.ID, language)
				} else {
					// Иначе сразу переходим к проверке подписки
					b.sendSubscriptionRequest(query.Message.Chat.ID, language)
				}
				return
			} else {
				// Если страна была установлена ранее:
				// Показываем подтверждение сохранения и кнопку назад
				successText := b.service.GetText("country_saved", language)

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.stateManager.SetState(user.ID, StateInputName, query.Message.MessageID)

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.stateManager.SetState(user.ID, StateInputUpNickname, query.Message.MessageID)

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.stateManager.SetState(user.ID, StateInputWallet, query.Message.MessageID)

				backBtn := b.createBackBtnKeyboard(language)

				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
				b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
					Text:           settingsText,
					InlineKeyboard: b.createSettingsKeyboard(language, user.ID), // передаем userID
				})
			}
			return

		case CallbackSettingsMainMenu:
			// Очищаем состояние пользователя
			b.stateManager.ClearState(user.ID)

			// Возвращаемся в главное меню
			b.sendMainMenu(query.Message.Chat.ID, language)
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

			b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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

	// Обработка других callback data
	switch callbackData {
	case CallbackAgeVerifiedYes, CallbackAgeVerifiedNo:
		b.handleAgeVerificationCallback(query)
		return
	case CallbackChangeNameYes:
		// Обработка согласия на изменение никнейма
		b.handleChangeNameYes(query)
		return
	case CallbackChangeNameNo:
		// Обработка отказа от изменения никнейма
		b.handleChangeNameNo(query)
		return
	case CallbackReserveSubscription:
		b.handleReserveSubscriptionCheck(query)
		return
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
	case "back_to_start":
		b.handleBackToStartMenu(query)
	case CallbackBetRed:
		b.handleMakeBet(user.ID, models.Red)
		b.answerCallbackQuery(query.ID, "", false)
	case CallbackBetBlack:
		b.handleMakeBet(user.ID, models.Black)
		b.answerCallbackQuery(query.ID, "", false)
	case CallbackBetZero:
		b.handleMakeBet(user.ID, models.Zero)
		b.answerCallbackQuery(query.ID, "", false)
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
		walletText := b.service.GetText("withdrawusdtchange", language)

		// Обновляем сообщение и запрашиваем адрес кошелька
		if query.Message != nil {
			// Устанавливаем состояние ожидания адреса кошелька
			b.stateManager.SetState(user.ID, StateInputWithdrawWallet, query.Message.MessageID)

			b.SendMessage(query.Message.Chat.ID, MessageOptions{
				Text:          walletText,
				ReplyKeyboard: b.createAccountKeyboard(language),
			})
		}
		return
	case CallbackCancelInput:
		b.stateManager.ClearState(user.ID)

		// Получаем локализованный текст раздела аккаунта
		accountStartText := b.service.GetText("accstart", language)
		b.SendMessage(query.Message.Chat.ID, MessageOptions{
			Text:          accountStartText,
			ReplyKeyboard: b.createAccountKeyboard(language),
		})
		return
	case CallbackCheckAmount:
		b.handleCheckAmountCallback(query)
	case CallbackSetAmount:
		withdrawText := b.service.GetText("withdrawusdtsumam", language)

		// Посылаем сообщение и запрашиваем сумму для вывода
		if query.Message != nil {
			// Устанавливаем состояние ожидания суммы для вывода
			b.stateManager.SetState(user.ID, StateInputWithdrawAmount, query.Message.MessageID)

			b.SendMessage(query.Message.Chat.ID, MessageOptions{
				Text:          withdrawText,
				ReplyKeyboard: b.createAccountKeyboard(language),
			})
		}
		return

	case CallbackProcessWithdraw:
		b.handleProcessWithdrawCallback(query)
	case "stop_game":
		b.gameHandler.HandleStopGameButton(user.ID)
		b.answerCallbackQuery(query.ID, "", false)
		b.handleHelpCommand(query.Message)
	default:
		// Неизвестный callback
		b.answerCallbackQuery(query.ID, "Unknown action", true)
	}
}

// Обработчики команд

// handleStartCommand обрабатывает команду /start
// сама регистрация пользователя выполняется в начале функции handleMessage
// в этой функции только дозаполнение полей
func (b *Bot) handleStartCommand(message *telego.Message) {
	user := message.From

	// Проверяем, существует ли пользователь
	_, err := b.service.GetUser(user.ID)
	isNewUser := err != nil // Флаг нового пользователя

	// Получаем источник
	userSource := ""
	if isNewUser {
		if strings.Contains(message.Text, " ") {
			commandArgs := strings.Split(message.Text, " ")
			if len(commandArgs) > 1 {
				userSource = strings.TrimSpace(commandArgs[1]) // Получаем источник
			}
		}
	}

	// Регистрируем пользователя или обновляем информацию
	dbUser, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, userSource, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error registering user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: b.service.GetText("error_while_registering", user.LanguageCode),
		})
		return
	}

	// Определяем язык пользователя из базы данных
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст приветствия
	welcomeText := b.service.GetText("startmessage1", language)

	// Создаем inline клавиатуру для первого сообщения
	inlineKeyboard := b.createStartInlineKeyboard(language)

	// Отправляем первое приветственное сообщение с inline клавиатурой
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:           welcomeText,
		InlineKeyboard: inlineKeyboard,
	})

	// Для нового пользователя или с неподтвержденным возрастом проверяем возраст
	if isNewUser || dbUser.AgeVerified == nil {
		// Отправляем запрос на подтверждение возраста
		b.sendAgeVerificationRequest(message.Chat.ID, language)
		return
	}

	// Если возраст подтвержден, но не установлена страна
	if *dbUser.AgeVerified && (isNewUser || dbUser.Country == "") {
		// Отправляем запрос на выбор страны
		countryText := b.service.GetText("countrymes", language)

		// Создаем клавиатуру со странами - начинаем с первой страницы (1)
		countriesKeyboard := b.createCountriesKeyboard(1)

		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:           countryText,
			InlineKeyboard: countriesKeyboard,
		})
	} else {
		// Если у пользователя уже выбрана страна и возраст подтвержден, отправляем главное меню
		b.sendMainMenu(message.Chat.ID, language)
	}
}

// Вспомогательный метод для отправки главного меню
func (b *Bot) sendMainMenu(chatID int64, language string) {
	b.SendMessage(chatID, MessageOptions{
		Text:          b.service.GetText("main_menu", language),
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

// sendAgeVerificationRequest отправляет запрос на подтверждение возраста
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

// handleAgeVerificationCallback обрабатывает ответ пользователя на запрос возраста
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

// handleChangeNameYes обрабатывает согласие на изменение никнейма
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

// handleChangeNameNo обрабатывает отказ от изменения никнейма
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

	// Получаем локализованный текст для подтверждения сохранения текущего никнейма
	nameChangeNoText := b.service.GetText("name_changeno_msg", language)

	// Обновляем сообщение
	if query.Message != nil {
		b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
			Text: nameChangeNoText,
		})

		// Продолжаем процесс регистрации
		go func() {
			// Небольшая задержка для чтения сообщения
			time.Sleep(2 * time.Second)
			// Отправляем запрос на подписку
			b.sendSubscriptionRequest(query.Message.Chat.ID, language)
		}()
	}
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
		b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
		b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
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
	statisticsStartText := b.service.GetText("statisticsstart", language)

	// Создаем клавиатуру выбора периода статистики
	statsKeyboard := b.createStatsKeyboard(language)

	// Отправляем сообщение с клавиатурой для выбора периода
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          statisticsStartText,
		ReplyKeyboard: statsKeyboard,
	})
}

// handleUnknownCommand обрабатывает неизвестные команды
func (b *Bot) handleUnknownCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отправляем сообщение о неизвестной команде
	unknownCommandText := b.service.GetText("unknown_command", language)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text: unknownCommandText,
	})

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
		b.sendMainMenu(query.Message.Chat.ID, language)
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
	statsTemplate := b.service.GetText(statsMsgKey, language)

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
	statsText := fmt.Sprintf(
		statsTemplate,
		totalBets, blackBets, redBets, zeroBets,
		wonBets, wonBlackBets, wonRedBets, wonZeroBets,
		lostBets, lostBlackBets, lostRedBets, lostZeroBets,
		totalPoints,
	)

	// Отправляем статистику пользователю
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          statsText,
		ReplyKeyboard: b.createStatsKeyboard(language),
	})

	// Отправляем сообщение с предложением выбрать другой период
	statisticsNextText := b.service.GetText("statistics_next", language)
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          statisticsNextText,
		ReplyKeyboard: b.createStatsKeyboard(language),
	})
}

// Допоміжні методи

// Відповідь на callback-запит
func (b *Bot) answerCallbackQuery(queryID string, text string, showAlert bool) {
	err := b.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
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

	// Если указан путь к фото или FileID
	if options.PhotoPath != "" || options.PhotoFileID != "" {
		options.MethodName = sendPhoto
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
		err := b.bot.DeleteMessage(&telego.DeleteMessageParams{
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
		err := b.bot.DeleteMessage(&telego.DeleteMessageParams{
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
		ChatID:                telego.ChatID{ID: chatID},
		Text:                  options.Text,
		ParseMode:             options.ParseMode,
		DisableWebPagePreview: options.DisableWebPagePreview,
		DisableNotification:   options.DisableNotification,
	}

	// Добавляем сущности, если они есть
	if len(options.Entities) > 0 {
		params.Entities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	msg, err := b.bot.SendMessage(params)
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
		ChatID:                telego.ChatID{ID: chatID},
		MessageID:             messageID,
		Text:                  options.Text,
		ParseMode:             options.ParseMode,
		DisableWebPagePreview: options.DisableWebPagePreview,
	}

	// Добавляем сущности, если они есть
	if len(options.Entities) > 0 {
		params.Entities = options.Entities
	}

	// Для обновления можно использовать только инлайн клавиатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageText(params)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return msg, nil
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
		return b.bot.SendPhoto(params)
	} else if options.PhotoPath != "" {
		// Для файла используем метод Upload
		return b.sendPhotoFile(chatID, options.PhotoPath, options.DelPhoto, params)
	}

	return nil, fmt.Errorf("no photo source specified")
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
	msg, err := b.bot.SendPhoto(params)
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

	msg, err := b.bot.EditMessageMedia(params)
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
	_, err := b.bot.SendSticker(&telego.SendStickerParams{
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
			invalidNicknameText := b.service.GetText("invalid_nickname", user.LanguageCode)
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: invalidNicknameText,
			})
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

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("name_changesave", user.LanguageCode)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: successText,
		})

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)

		// Продолжаем процесс регистрации
		go func() {
			// Небольшая задержка для чтения сообщения
			time.Sleep(2 * time.Second)
			// Отправляем запрос на подписку
			b.sendSubscriptionRequest(message.Chat.ID, user.LanguageCode)
		}()
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
			invalidNameText := b.service.GetText("invalid_name", language)
			if invalidNameText == "invalid_name" {
				invalidNameText = "Имя должно содержать от 1 до 100 символов"
			}
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: invalidNameText,
			})
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
			errorText := b.service.GetText("update_error", language)
			if errorText == "update_error" {
				errorText = "Ошибка при обновлении данных. Попробуйте еще раз."
			}
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: errorText,
			})
			return
		}

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("name_saved", language)
		if successText == "name_saved" {
			successText = "Имя успешно сохранено!"
		}

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
			invalidNicknameText := b.service.GetText("invalid_nickname", language)
			if invalidNicknameText == "invalid_nickname" {
				invalidNicknameText = "Никнейм должен содержать от 3 до 20 символов и состоять только из латинских букв, цифр и знака подчеркивания"
			}
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: invalidNicknameText,
			})
			return
		}

		// Обновляем никнейм пользователя
		dbUser.Nickname = nickname
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user nickname: %v", err)

			// Отправляем сообщение об ошибке
			errorText := b.service.GetText("update_error", language)
			if errorText == "update_error" {
				errorText = "Ошибка при обновлении никнейма. Попробуйте еще раз."
			}
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: errorText,
			})
			return
		}

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("nickname_saved", language)
		if successText == "nickname_saved" {
			successText = "Никнейм успешно сохранен!"
		}

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
			invalidWalletText := b.service.GetText("withdrawusdtchangeerror", language)
			if invalidWalletText == "withdrawusdtchangeerror" {
				invalidWalletText = "Неверный формат адреса кошелька. Адрес TRC20 должен начинаться с 'T' и содержать не менее 30 символов."
			}

			// Создаем клавиатуру с кнопкой назад
			backBtn := b.createBackBtnKeyboard(language)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text:           invalidWalletText,
				InlineKeyboard: backBtn,
			})
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
			errorText := b.service.GetText("update_error", language)
			if errorText == "update_error" {
				errorText = "Ошибка при обновлении адреса кошелька. Попробуйте еще раз."
			}
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: errorText,
			})
			return
		}

		// Отправляем сообщение об успешном обновлении
		successText := b.service.GetText("withdrawusdtchangeok", language)
		if successText == "withdrawusdtchangeok" {
			successText = "Адрес кошелька успешно обновлен!"
		}

		backBtn := b.createBackBtnKeyboard(language)

		b.UpdateMessage(message.Chat.ID, messageID, MessageOptions{
			Text:           successText,
			InlineKeyboard: backBtn,
		})

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)
	}
}
