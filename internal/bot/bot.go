// internal/bot/bot.go
package bot

import (
	"context"
	"fmt"
	"log"
	"os"

	"roulette/internal/service"
	"roulette/internal/utils"

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
}

// Константи для команд і callback-запитів
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

// NewBot створює новий екземпляр бота
func NewBot(token string, service service.Service) (*Bot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		bot:     bot,
		service: service,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start запускає бота
func (b *Bot) Start() error {
	if b.initialized {
		return fmt.Errorf("bot already started")
	}

	// Отримуємо інформацію про бота
	me, err := b.bot.GetMe()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}
	log.Printf("Bot started: @%s", me.Username)

	// Початок отримання оновлень
	updates, err := b.bot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
		Timeout: 60,
		Offset:  0,
	})
	if err != nil {
		return fmt.Errorf("failed to get updates: %w", err)
	}
	b.updates = updates

	// Запускаємо обробку оновлень у фоновому режимі
	go b.processUpdates()

	b.initialized = true
	return nil
}

// Stop зупиняє бота
func (b *Bot) Stop() {
	if !b.initialized {
		return
	}

	// Зупиняємо довгий поллінг
	b.bot.StopLongPolling()

	// Відміняємо контекст
	b.cancel()

	b.initialized = false
	log.Println("Bot stopped")
}

// processUpdates обробляє оновлення від телеграма
func (b *Bot) processUpdates() {
	for update := range b.updates {
		b.handleUpdate(update)
	}
}

// handleUpdate обробляє одне оновлення
func (b *Bot) handleUpdate(update telego.Update) {
	// Обробка повідомлень
	if update.Message != nil && update.Message.Text != "" {
		b.handleMessage(update.Message)
		return
	}

	// Обробка callback-запитів
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}
}

// handleMessage обробляє повідомлення
func (b *Bot) handleMessage(message *telego.Message) {
	// Обробка команд
	if message.Text[0] == '/' {
		command := message.Text[1:] // Видаляємо символ '/'

		// Визначаємо команду
		switch command {
		case CommandStart:
			b.handleStartCommand(message)
		default:
			// Невідома команда
			b.SendMessage(message.Chat.ID, MessageOptions{
				Text: "Невідома команда.",
			})
		}
	}
}

// handleCallbackQuery обробляє callback-запити
func (b *Bot) handleCallbackQuery(query *telego.CallbackQuery) {
	switch query.Data {
	case CallbackBack:
		b.handleBackToMainMenu(query)
	default:
		// Невідомий callback
		b.answerCallbackQuery(query.ID, "Невідома дія", true)
	}
}

// Обробники команд

func (b *Bot) handleStartCommand(message *telego.Message) {
	user := message.From

	u, _ := b.service.GetUser(user.ID)
	utils.DumpInterface(u)

	// Реєструємо користувача
	_, err := b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, user.LanguageCode)
	if err != nil {
		log.Printf("Error registering user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Помилка при реєстрації. Спробуйте ще раз.",
		})
		return
	}

	// Відправляємо привітальне повідомлення
	welcomeText := "Вітаємо в грі Рулетка! Тут ви можете робити ставки на червоне, чорне або зеро."
	helpText := "Доступні команди:\n/play - Почати гру\n/profile - Ваш профіль\n/stats - Ваша статистика\n/rating - Тижневий рейтинг\n/balance - Ваш баланс"

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text: fmt.Sprintf("%s\n\n%s", welcomeText, helpText),
		// InlineKeyboard: b.createMainKeyboard(),
		ReplyKeyboard: b.createMainReplyKeyboard(),
	})

	u, _ = b.service.GetUser(user.ID)
	utils.DumpInterface(u)
}

// Обробники callback-запитів

func (b *Bot) handleBackToMainMenu(query *telego.CallbackQuery) {
	b.answerCallbackQuery(query.ID, "", false)

	helpText := "Доступні команди:\n/play - Почати гру\n/profile - Ваш профіль\n/stats - Ваша статистика\n/rating - Тижневий рейтинг\n/balance - Ваш баланс"

	b.UpdateMessage(query.Message.Chat.ID, query.Message.MessageID, MessageOptions{
		Text:           helpText,
		InlineKeyboard: b.createMainKeyboard(),
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
		log.Printf("Error answering callback query: %v", err)
	}
}

// Створення основної клавіатури
func (b *Bot) createMainKeyboard() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🎮 Грати", CallbackData: CommandPlay},
				{Text: "👤 Профіль", CallbackData: CommandProfile},
			},
			{
				{Text: "📊 Статистика", CallbackData: CommandStats},
				{Text: "🏆 Рейтинг", CallbackData: CommandRating},
			},
			{
				{Text: "💰 Баланс", CallbackData: CommandBalance},
				{Text: "❓ FAQ", CallbackData: CommandFAQ},
			},
		},
	}
}

// // Створення клавіатури відповіді (приклад)
func (b *Bot) createMainReplyKeyboard() *telego.ReplyKeyboardMarkup {
	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: "🎮 Грати"},
				{Text: "👤 Профіль"},
			},
			{
				{Text: "📊 Статистика"},
				{Text: "🏆 Рейтинг"},
			},
			{
				{Text: "💰 Баланс"},
				{Text: "❓ FAQ"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
	}
}

// MessageOptions містить опції для відправки або оновлення повідомлення
type MessageOptions struct {
	// Text - текст повідомлення
	Text string

	// PhotoPath - шлях до фото (якщо встановлено, буде відправлено фото з підписом Text)
	PhotoPath string

	// PhotoFileID - FileID фото (якщо встановлено, буде відправлено фото з підписом Text)
	PhotoFileID string

	// InlineKeyboard - інлайн клавіатура (якщо встановлено, буде додана до повідомлення)
	InlineKeyboard *telego.InlineKeyboardMarkup

	// ReplyKeyboard - клавіатура відповіді (якщо встановлено, буде додана до повідомлення)
	ReplyKeyboard *telego.ReplyKeyboardMarkup

	// RemoveKeyboard - якщо true, клавіатура відповіді буде видалена
	RemoveKeyboard bool

	// OneTimeKeyboard - якщо true і встановлено ReplyKeyboard, клавіатура буде одноразовою
	OneTimeKeyboard bool

	// Selective - якщо true і встановлено ReplyKeyboard, клавіатура буде показана тільки певним користувачам
	Selective bool

	// ParseMode - режим форматування тексту (HTML, Markdown, MarkdownV2)
	ParseMode string

	// DisableWebPagePreview - якщо true, превью посилань буде вимкнено
	DisableWebPagePreview bool

	// DisableNotification - якщо true, повідомлення буде відправлено беззвучно
	DisableNotification bool
}

// SendMessage відправляє нове повідомлення з вказаними опціями
func (b *Bot) SendMessage(chatID int64, options MessageOptions) (*telego.Message, error) {
	// Якщо вказано шлях до фото або FileID
	if options.PhotoPath != "" || options.PhotoFileID != "" {
		return b.sendPhoto(chatID, options)
	}

	// Інакше відправляємо текстове повідомлення
	return b.sendText(chatID, options)
}

// UpdateMessage оновлює існуюче повідомлення з вказаними опціями
func (b *Bot) UpdateMessage(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	// Якщо вказано шлях до фото
	if options.PhotoPath != "" {
		// Для фото з локального джерела необхідно видалити старе повідомлення і відправити нове
		err := b.bot.DeleteMessage(&telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else if options.PhotoFileID != "" {
		// Оновлення фото по FileID
		return b.updatePhotoByFileID(chatID, messageID, options)
	} else if options.ReplyKeyboard != nil || options.RemoveKeyboard {
		// Для ReplyKeyboard необхідно видалити старе повідомлення і відправити нове
		err := b.bot.DeleteMessage(&telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else {
		// Оновлюємо текстове повідомлення
		return b.updateText(chatID, messageID, options)
	}
}

// sendText відправляє текстове повідомлення
func (b *Bot) sendText(chatID int64, options MessageOptions) (*telego.Message, error) {
	params := &telego.SendMessageParams{
		ChatID:                telego.ChatID{ID: chatID},
		Text:                  options.Text,
		ParseMode:             options.ParseMode,
		DisableWebPagePreview: options.DisableWebPagePreview,
		DisableNotification:   options.DisableNotification,
	}

	// Встановлюємо відповідну клавіатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	msg, err := b.bot.SendMessage(params)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return msg, nil
}

// updateText оновлює текстове повідомлення
func (b *Bot) updateText(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	params := &telego.EditMessageTextParams{
		ChatID:                telego.ChatID{ID: chatID},
		MessageID:             messageID,
		Text:                  options.Text,
		ParseMode:             options.ParseMode,
		DisableWebPagePreview: options.DisableWebPagePreview,
	}

	// Для оновлення можна використовувати тільки інлайн клавіатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageText(params)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return msg, nil
}

// sendPhoto відправляє фото з підписом
func (b *Bot) sendPhoto(chatID int64, options MessageOptions) (*telego.Message, error) {
	// Параметри для відправки
	params := &telego.SendPhotoParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
	}

	// Встановлюємо відповідну клавіатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	// Вибираємо джерело фото
	if options.PhotoFileID != "" {
		// Для FileID використовуємо його безпосередньо
		// params.Photo = telego.InputFile{FileID: options.PhotoFileID}
		params.Photo = tu.FileFromID(options.PhotoFileID)
		return b.bot.SendPhoto(params)
	} else if options.PhotoPath != "" {
		// Для файлу використовуємо метод Upload
		return b.sendPhotoFile(chatID, options.PhotoPath, params)
	}

	return nil, fmt.Errorf("no photo source specified")
}

// sendPhotoFile відправляє фото з локального файлу
func (b *Bot) sendPhotoFile(chatID int64, photoPath string, params *telego.SendPhotoParams) (*telego.Message, error) {
	// Відкриваємо файл
	file, err := os.Open(photoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open photo file: %w", err)
	}
	defer file.Close()

	// Встановлюємо завантажений файл
	params.Photo = tu.File(file)

	// Відправляємо фото
	msg, err := b.bot.SendPhoto(params)
	if err != nil {
		return nil, fmt.Errorf("failed to send photo: %w", err)
	}

	return msg, nil
}

// updatePhotoByFileID оновлює фото за FileID
func (b *Bot) updatePhotoByFileID(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	// Створюємо об'єкт InputMediaPhoto з FileID
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

	// Для оновлення можна використовувати тільки інлайн клавіатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageMedia(params)
	if err != nil {
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}

	return msg, nil
}

// getReplyMarkup повертає відповідну клавіатуру на основі опцій
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
		// Клонуємо клавіатуру, щоб не змінювати оригінал
		keyboard := *options.ReplyKeyboard
		keyboard.OneTimeKeyboard = options.OneTimeKeyboard
		keyboard.Selective = options.Selective
		return &keyboard
	}

	return nil
}
