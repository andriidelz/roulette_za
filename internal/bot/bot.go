package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"roulette/internal/models"
	"roulette/internal/service"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Структура бота
type Bot struct {
	bot         *telego.Bot
	updates     <-chan telego.Update
	service     service.Service
	initialized bool
	ctx         context.Context
	cancel      context.CancelFunc
	gameHandler *GameHandler // Обработчик игры
}

// Константы для команд и callback-запитов
const (
	CommandStart       = "start"
	CommandHelp        = "help"
	CommandPlay        = "play"
	CommandProfile     = "profile"
	CommandStats       = "stats"
	CommandRating      = "rating"
	CommandSuperRating = "superrating"
	CommandBalance     = "balance"
	CommandWithdraw    = "withdraw"
	CommandFAQ         = "faq"

	CallbackBetRed   = "bet_red"
	CallbackBetBlack = "bet_black"
	CallbackBetZero  = "bet_zero"
	CallbackBack     = "back"
)

// NewBot создает новый экземпляр бота
func NewBot(token string, service service.Service) (*Bot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	b := &Bot{
		bot:     bot,
		service: service,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Инициализируем обработчик игры после создания бота
	b.gameHandler = NewGameHandler(b, service)

	return b, nil
}

// Start запускает бота
func (b *Bot) Start() error {
	if b.initialized {
		return fmt.Errorf("bot already started")
	}

	// Получаем информацию о боте
	me, err := b.bot.GetMe()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}
	log.Printf("Bot started: @%s", me.Username)

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

	b.initialized = false
	log.Println("Bot stopped")
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

// MakeBet делает ставку в текущем раунде
func (b *Bot) handleMakeBet(userID int64, option models.BetOption) {
	// Получаем пользователя для определения языка
	user, userErr := b.service.GetUser(userID)
	if userErr != nil {
		log.Printf("Error getting user %d: %v", userID, userErr)
		return
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Вызываем MakeBet и обрабатываем возможные ошибки
	err := b.gameHandler.MakeBet(userID, option)
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
		} else {
			// Общая ошибка ставки
			errorText = b.service.GetText("bet_error", language)
		}

		b.SendMessage(userID, MessageOptions{
			Text:          errorText,
			ReplyKeyboard: b.gameHandler.createBetKeyboard(language, userID),
		})
		return
	}

	// Если ставка успешно сделана, отправляем сообщение о принятии ставки
	nomorebidsText := b.service.GetText("nomorebids", language)
	b.SendMessage(userID, MessageOptions{
		Text:          nomorebidsText,
		ReplyKeyboard: b.gameHandler.createBetKeyboard(language, userID),
	})
}

// handleMessage обрабатывает сообщения
func (b *Bot) handleMessage(message *telego.Message) {
	user := message.From

	// Регистрация пользователя, если он новый
	_, err := b.service.GetUser(user.ID)
	if err != nil {
		_, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, user.LanguageCode)
		if err != nil {
			log.Printf("Error registering user: %v", err)
		}
	}

	text := message.Text

	// Обработка команд, начинающихся с /
	if len(text) > 0 && text[0] == '/' {
		command := strings.Split(text[1:], " ")[0] // Получаем команду без аргументов
		command = strings.ToLower(command)

		switch command {
		case CommandStart:
			b.handleStartCommand(message)
		case CommandHelp:
			b.handleHelpCommand(message)
		case CommandPlay:
			b.gameHandler.HandlePlayCommand(message)
		case CommandProfile:
			b.handleProfileCommand(message)
		case CommandStats:
			b.handleStatsCommand(message)
		case CommandRating:
			b.handleRatingCommand(message)
		case CommandSuperRating:
			b.handleSuperRatingCommand(message)
		case CommandBalance:
			b.handleBalanceCommand(message)
		case CommandWithdraw:
			b.handleWithdrawCommand(message)
		case CommandFAQ:
			b.handleFAQCommand(message)
		default:
			// Неизвестная команда
			b.handleUnknownCommand(message)
		}
		return
	}

	// Обработка текстовых сообщений (не команд)
	language := user.LanguageCode
	if language == "" {
		language = "en"
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

	// Обработка клавиатуры главного меню
	switch text {
	case btnPlayText:
		b.gameHandler.HandlePlayCommand(message)
	case btnStatisticsText:
		b.handleStatsCommand(message)
	case btnRatingText:
		b.handleRatingCommand(message)
	case btnAccountText:
		b.handleBalanceCommand(message)
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

			b.SendMessage(message.Chat.ID, MessageOptions{
				Text:          zeroLimitText,
				ReplyKeyboard: b.gameHandler.createBetKeyboard(language, user.ID),
			})
		}
	case btnBackText:
		// Возврат в главное меню и удаление из активных игроков
		b.gameHandler.HandleBackButton(user.ID)
		b.handleHelpCommand(message)
	default:
		// Обработка других текстовых сообщений
		b.handleGenericMessage(message)
	}
}

// handleCallbackQuery обрабатывает callback-запиты
func (b *Bot) handleCallbackQuery(query *telego.CallbackQuery) {
	// Валидация пользователя
	user := query.From
	_, err := b.service.GetUser(user.ID)
	if err != nil {
		// Регистрация пользователя, если он не найден
		_, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, user.LanguageCode)
		if err != nil {
			log.Printf("Error registering user: %v", err)
			return
		}
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	callbackData := query.Data

	// Проверяем, это выбор страны или другой callback
	if strings.HasPrefix(callbackData, "country:") {
		// Извлекаем код страны
		countryCode := strings.TrimPrefix(callbackData, "country:")

		// Сохраняем выбранную страну
		if err := b.service.SetUserCountry(user.ID, countryCode); err != nil {
			log.Printf("Error saving user country: %v", err)
			b.answerCallbackQuery(query.ID, "Error saving country", true)
			return
		}

		// Отправляем успешный статус пользователю
		successText := b.service.GetText("country_saved", language)

		// Отвечаем на callback
		b.answerCallbackQuery(query.ID, "", false)

		// Обновляем сообщение, убирая кнопки выбора страны и добавляя подтверждение
		if query.Message != nil {
			b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
				Text: successText,
			})
		}

		// Отправляем главное меню
		if query.Message != nil {
			b.sendMainMenu(query.Message.Chat.ID, language)
		} else {
			b.sendMainMenu(user.ID, language)
		}
		return
	}

	// Обработка других callback data
	switch callbackData {
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
	default:
		// Неизвестный callback
		b.answerCallbackQuery(query.ID, "Unknown action", true)
	}
}

// Обработчики команд

// handleStartCommand обрабатывает команду /start
func (b *Bot) handleStartCommand(message *telego.Message) {
	user := message.From

	// Регистрируем пользователя или обновляем информацию
	_, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, user.LanguageCode)
	if err != nil {
		log.Printf("Error registering user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error while registering. Please try again.",
		})
		return
	}

	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст приветствия
	welcomeText := b.service.GetText("startmessage1", language)

	// Создаем inline клавиатуру для первого сообщения
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: b.service.GetText("btn_rules", language), CallbackData: "rules"},
				{Text: b.service.GetText("btn_awards", language), CallbackData: "awards"},
			},
			{
				{Text: b.service.GetText("btn_payments", language), CallbackData: "payments"},
				{Text: b.service.GetText("btn_fairplay", language), CallbackData: "fairplay"},
			},
		},
	}

	// Отправляем первое приветственное сообщение с inline клавиатурой
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:           welcomeText,
		InlineKeyboard: inlineKeyboard,
	})

	// Проверяем, известна ли нам страна пользователя
	country, err := b.service.GetUserCountry(user.ID)
	if err != nil || country == "" {
		// Если страна неизвестна, отправляем запрос на выбор страны
		countryText := b.service.GetText("countrymes", language)

		// Создаем клавиатуру со странами
		countriesKeyboard := b.createCountriesKeyboard()

		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:           countryText,
			InlineKeyboard: countriesKeyboard,
		})
	} else {
		// Если страна уже известна, отправляем главное меню
		b.sendMainMenu(message.Chat.ID, language)
	}
}

// Вспомогательный метод для отправки главного меню
func (b *Bot) sendMainMenu(chatID int64, language string) {
	// Создаем reply клавиатуру для главного меню
	replyKeyboard := &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: b.service.GetText("btn_play", language)},
				{Text: b.service.GetText("btn_statistics", language)},
			},
			{
				{Text: b.service.GetText("btn_rating", language)},
				{Text: b.service.GetText("btn_account", language)},
			},
			{
				{Text: b.service.GetText("btn_faq", language)},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}

	// Отправляем сообщение с главным меню и минимальным текстом
	menuText := b.service.GetText("main_menu", language)
	b.SendMessage(chatID, MessageOptions{
		Text:          menuText,
		ReplyKeyboard: replyKeyboard,
	})
}

// createCountriesKeyboard создает клавиатуру с флагами стран
func (b *Bot) createCountriesKeyboard() *telego.InlineKeyboardMarkup {
	// Массив стран с кодами ISO 3166-1 alpha-2 и эмодзи флагов
	countries := []struct {
		Code  string
		Emoji string
		Name  string
	}{
		{"RU", "🇷🇺", "Russia"},
		{"UA", "🇺🇦", "Ukraine"},
		{"BY", "🇧🇾", "Belarus"},
		{"KZ", "🇰🇿", "Kazakhstan"},
		{"US", "🇺🇸", "USA"},
		{"GB", "🇬🇧", "United Kingdom"},
		{"DE", "🇩🇪", "Germany"},
		{"FR", "🇫🇷", "France"},
		{"IT", "🇮🇹", "Italy"},
		{"ES", "🇪🇸", "Spain"},
		{"CN", "🇨🇳", "China"},
		{"JP", "🇯🇵", "Japan"},
		{"KR", "🇰🇷", "South Korea"},
		{"IN", "🇮🇳", "India"},
		{"TR", "🇹🇷", "Turkey"},
		{"AE", "🇦🇪", "UAE"},
		{"IL", "🇮🇱", "Israel"},
		{"BR", "🇧🇷", "Brazil"},
	}

	// Формируем 6 строк по 3 страны в каждой
	var keyboard [][]telego.InlineKeyboardButton
	row := make([]telego.InlineKeyboardButton, 0, 3)

	for i, country := range countries {
		buttonText := country.Emoji + " " + country.Code
		buttonData := "country:" + country.Code

		row = append(row, telego.InlineKeyboardButton{
			Text:         buttonText,
			CallbackData: buttonData,
		})

		// Добавляем строку после каждых 3 стран
		if (i+1)%3 == 0 || i == len(countries)-1 {
			keyboard = append(keyboard, row)
			row = make([]telego.InlineKeyboardButton, 0, 3)
		}
	}

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
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
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: b.service.GetText("btn_rules", language), CallbackData: "rules"},
				{Text: b.service.GetText("btn_awards", language), CallbackData: "awards"},
			},
			{
				{Text: b.service.GetText("btn_payments", language), CallbackData: "payments"},
				{Text: b.service.GetText("btn_fairplay", language), CallbackData: "fairplay"},
			},
		},
	}

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
	// Создаем кнопку "Назад"
	backButton := telego.InlineKeyboardButton{
		Text:         "◀️ Назад",
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

func (b *Bot) handleProfileCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем информацию о пользователе и его статистику
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error retrieving profile. Please try again.",
		})
		return
	}

	stats, err := b.service.GetUserStats(user.ID)
	if err != nil {
		log.Printf("Error getting user stats: %v", err)
		stats = &models.UserStats{} // Пустая статистика, если не удалось получить
	}

	// Расчет эффективности
	efficiency := 0.0
	if stats.TotalBets > 0 {
		efficiency = float64(stats.WonBets) / float64(stats.TotalBets) * 100
	}

	// Получаем шаблон профиля и форматируем его
	profileTemplate := b.service.GetText("profile_template", language)
	profileText := fmt.Sprintf(
		profileTemplate,
		dbUser.Username,
		dbUser.Balance,
		stats.TotalBets,
		stats.WonBets,
		efficiency,
		stats.TotalPoints,
	)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          profileText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

func (b *Bot) handleStatsCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем статистику пользователя
	stats, err := b.service.GetUserStats(user.ID)
	if err != nil {
		log.Printf("Error getting user stats: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error retrieving statistics. Please try again.",
		})
		return
	}

	// Получаем шаблон статистики и форматируем его
	statsTemplate := b.service.GetText("stats_template", language)
	statsText := fmt.Sprintf(
		statsTemplate,
		stats.DailyBets,
		stats.DailyPoints,
		stats.WeeklyBets,
		stats.WeeklyPoints,
		stats.MonthlyBets,
		stats.MonthlyPoints,
		stats.TotalBets,
		stats.TotalPoints,
	)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          statsText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

func (b *Bot) handleRatingCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обработка команды рейтинга
	// Добавить позже логику получения и отображения рейтинга
	ratingText := "The rating system is under development."

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          ratingText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

func (b *Bot) handleSuperRatingCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Обработка команды супер-рейтинга
	// Добавить позже логику получения и отображения супер-рейтинга
	superRatingText := "The super-rating system is under development."

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          superRatingText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

func (b *Bot) handleBalanceCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем информацию о пользователе для проверки баланса
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error retrieving balance. Please try again.",
		})
		return
	}

	// Получаем шаблон баланса
	var balanceText string
	minWithdrawal := 10.0 // Значение по умолчанию

	// Проверяем достаточно ли денег для вывода
	if dbUser.Balance >= minWithdrawal {
		balanceTemplate := b.service.GetText("balanceaccok", language)
		balanceText = fmt.Sprintf(balanceTemplate, dbUser.Balance)
	} else {
		balanceTemplate := b.service.GetText("balanceacclow", language)
		balanceText = fmt.Sprintf(balanceTemplate, dbUser.Balance)
	}

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          balanceText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

func (b *Bot) handleWithdrawCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем информацию о пользователе для проверки баланса
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error retrieving data. Please try again.",
		})
		return
	}

	minWithdrawal := 10.0 // Значение по умолчанию

	// Проверяем достаточно ли денег для вывода
	var withdrawText string
	if dbUser.Balance >= minWithdrawal {
		withdrawTemplate := b.service.GetText("withdrawok", language)
		withdrawText = fmt.Sprintf(withdrawTemplate, dbUser.Balance)
	} else {
		withdrawTemplate := b.service.GetText("withdrawlow", language)
		withdrawText = fmt.Sprintf(withdrawTemplate, dbUser.Balance)
	}

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          withdrawText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
	})
}

func (b *Bot) handleFAQCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст FAQ
	faqText := b.service.GetText("faq", language)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          faqText,
		ReplyKeyboard: b.createMainReplyKeyboard(language),
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

// Допоміжні методи

// createMainKeyboard создает основную inline клавиатуру
func (b *Bot) createMainKeyboard(language string) *telego.InlineKeyboardMarkup {
	// Получаем локализированные тексты для кнопок
	btnPlayText := b.service.GetText("btn_play", language)
	btnProfileText := b.service.GetText("btn_profile", language)
	btnStatsText := b.service.GetText("btn_stats", language)
	btnRatingText := b.service.GetText("btn_rating", language)
	btnBalanceText := b.service.GetText("btn_balance", language)
	btnFAQText := b.service.GetText("btn_faq", language)

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: btnPlayText, CallbackData: CommandPlay},
				{Text: btnProfileText, CallbackData: CommandProfile},
			},
			{
				{Text: btnStatsText, CallbackData: CommandStats},
				{Text: btnRatingText, CallbackData: CommandRating},
			},
			{
				{Text: btnBalanceText, CallbackData: CommandBalance},
				{Text: btnFAQText, CallbackData: CommandFAQ},
			},
		},
	}
}

// createMainReplyKeyboard создает основную reply клавиатуру
func (b *Bot) createMainReplyKeyboard(language string) *telego.ReplyKeyboardMarkup {
	// Получаем локализированные тексты для кнопок
	btnPlayText := b.service.GetText("btn_play", language)
	btnStatisticsText := b.service.GetText("btn_statistics", language)
	btnRatingText := b.service.GetText("btn_rating", language)
	btnAccountText := b.service.GetText("btn_account", language)
	btnFAQText := b.service.GetText("btn_faq", language)

	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: btnPlayText},
				{Text: btnStatisticsText},
			},
			{
				{Text: btnRatingText},
				{Text: btnAccountText},
			},
			{
				{Text: btnFAQText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
	}
}

// Відповідь на callback-запит
func (b *Bot) answerCallbackQuery(queryID string, text string, showAlert bool) {
	err := b.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: queryID,
		Text:            text,
		ShowAlert:       showAlert,
	})
	if err != nil {
		log.Printf("Error answering callback query: %v", err)
	}
}

// MessageOptions содержит опции для отправки или обновления сообщения
type MessageOptions struct {
	// Text - текст сообщения
	Text string

	// PhotoPath - путь к фото (если установлен, будет отправлено фото с подписью Text)
	PhotoPath string

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

	// DisableWebPagePreview - если true, превью ссылок будет отключено
	DisableWebPagePreview bool

	// DisableNotification - если true, сообщение будет отправлено беззвучно
	DisableNotification bool
}

// SendMessage отправляет новое сообщение с указанными опциями
func (b *Bot) SendMessage(chatID int64, options MessageOptions) (*telego.Message, error) {
	// Обрабатываем текст, заменяя литеральные \r\n на реальные переносы строк
	// Используем двойной проход для избежания проблем с экранированием
	processedText := strings.ReplaceAll(options.Text, "\\r\\n", "\n")
	processedText = strings.ReplaceAll(processedText, "\r\n", "\n")
	options.Text = processedText

	// Если указан путь к фото или FileID
	if options.PhotoPath != "" || options.PhotoFileID != "" {
		return b.sendPhoto(chatID, options)
	}

	// Иначе отправляем текстовое сообщение
	return b.sendText(chatID, options)
}

// UpdateMessage обновляет существующее сообщение с указанными опциями
func (b *Bot) UpdateMessage(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	// Если указан путь к фото
	if options.PhotoPath != "" {
		// Для фото с локального источника необходимо удалить старое сообщение и отправить новое
		err := b.bot.DeleteMessage(&telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else if options.PhotoFileID != "" {
		// Обновление фото по FileID
		return b.updatePhotoByFileID(chatID, messageID, options)
	} else if options.ReplyKeyboard != nil || options.RemoveKeyboard {
		// Для ReplyKeyboard необходимо удалить старое сообщение и отправить новое
		err := b.bot.DeleteMessage(&telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else {
		// Обновляем текстовое сообщение
		return b.updateText(chatID, messageID, options)
	}
}

// sendText отправляет текстовое сообщение
func (b *Bot) sendText(chatID int64, options MessageOptions) (*telego.Message, error) {
	params := &telego.SendMessageParams{
		ChatID:                telego.ChatID{ID: chatID},
		Text:                  options.Text,
		ParseMode:             options.ParseMode,
		DisableWebPagePreview: options.DisableWebPagePreview,
		DisableNotification:   options.DisableNotification,
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	msg, err := b.bot.SendMessage(params)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return msg, nil
}

// updateText обновляет текстовое сообщение
func (b *Bot) updateText(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	params := &telego.EditMessageTextParams{
		ChatID:                telego.ChatID{ID: chatID},
		MessageID:             messageID,
		Text:                  options.Text,
		ParseMode:             options.ParseMode,
		DisableWebPagePreview: options.DisableWebPagePreview,
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
	// Параметры для отправки
	params := &telego.SendPhotoParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
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
		return b.sendPhotoFile(chatID, options.PhotoPath, params)
	}

	return nil, fmt.Errorf("no photo source specified")
}

// sendPhotoFile отправляет фото с локального файла
func (b *Bot) sendPhotoFile(chatID int64, photoPath string, params *telego.SendPhotoParams) (*telego.Message, error) {
	// Открываем файл
	file, err := os.Open(photoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open photo file: %w", err)
	}
	defer file.Close()

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
	// Создаем объект InputMediaPhoto с FileID
	mediaPhoto := &telego.InputMediaPhoto{
		Type:      "photo",
		Media:     tu.FileFromID(options.PhotoFileID),
		Caption:   options.Text,
		ParseMode: options.ParseMode,
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
