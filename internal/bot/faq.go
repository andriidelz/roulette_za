package bot

import (
	"roulette/internal/logger"
	"time"

	"github.com/mymmrac/telego"
)

// handleFAQCommand обрабатывает команду /faq и показывает стартовое меню FAQ
func (b *Bot) handleFAQCommand(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ menu: %v", err)
		return
	}

	// Используем язык из базы данных
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для стартового сообщения FAQ
	options := b.prepareMessage("faqstart", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем сообщение с клавиатурой для выбора раздела FAQ
	b.SendMessage(message.Chat.ID, options)
}

// createFAQKeyboard создает клавиатуру для меню FAQ
func (b *Bot) createFAQKeyboard(language string) *telego.ReplyKeyboardMarkup {
	// Получаем локализированные тексты для кнопок FAQ
	btnRulesText := b.service.GetText("faqrules", language)
	btnAwardsText := b.service.GetText("faqawards", language)
	btnPaymentsText := b.service.GetText("faqpayments", language)
	btnFairPlayText := b.service.GetText("faqfairplay", language)
	btnPrivacyPolicyText := b.service.GetText("privacypolicy", language)
	btnContactText := b.service.GetText("contact", language)
	btnExitText := b.service.GetText("faqexit", language)

	// Создаем клавиатуру
	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: btnRulesText},
				{Text: btnAwardsText},
			},
			{
				{Text: btnPaymentsText},
				{Text: btnFairPlayText},
			},
			{
				{Text: btnPrivacyPolicyText},
				{Text: btnContactText},
			},
			{
				{Text: btnExitText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
}

// handleFAQRules обрабатывает нажатие на кнопку "Правила" в меню FAQ
func (b *Bot) handleFAQRules(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ rules: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Правила"
	options := b.prepareMessage("faqrulesm", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем текст правил
	b.SendMessage(message.Chat.ID, options)

	// Отправляем последующее сообщение для продолжения навигации после небольшой задержки
	go b.sendFAQNextPrompt(message.Chat.ID, language)
}

// handleFAQAwards обрабатывает нажатие на кнопку "Распределение наград" в меню FAQ
func (b *Bot) handleFAQAwards(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ awards: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Распределение наград"
	options := b.prepareMessage("faqawardsm", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем текст о распределении наград
	b.SendMessage(message.Chat.ID, options)

	// Отправляем последующее сообщение для продолжения навигации после небольшой задержки
	go b.sendFAQNextPrompt(message.Chat.ID, language)
}

// handleFAQPayments обрабатывает нажатие на кнопку "Выплаты наград" в меню FAQ
func (b *Bot) handleFAQPayments(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ payments: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Выплаты наград"
	options := b.prepareMessage("faqpaymentsm", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем текст о выплатах наград
	b.SendMessage(message.Chat.ID, options)

	// Отправляем последующее сообщение для продолжения навигации после небольшой задержки
	go b.sendFAQNextPrompt(message.Chat.ID, language)
}

// handleFAQFairPlay обрабатывает нажатие на кнопку "Принципы честной игры" в меню FAQ
func (b *Bot) handleFAQFairPlay(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ fair play: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Принципы честной игры"
	options := b.prepareMessage("faqfairplaym", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем текст о принципах честной игры
	b.SendMessage(message.Chat.ID, options)

	// Отправляем последующее сообщение для продолжения навигации после небольшой задержки
	go b.sendFAQNextPrompt(message.Chat.ID, language)
}

// handleFAQPrivacyPolicy обрабатывает нажатие на кнопку "Privacy policy" в меню FAQ
func (b *Bot) handleFAQPrivacyPolicy(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ privacy policy: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Privacy policy"
	options := b.prepareMessage("privacypolicym", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем текст privacy policy
	b.SendMessage(message.Chat.ID, options)

	// Отправляем последующее сообщение для продолжения навигации после небольшой задержки
	go b.sendFAQNextPrompt(message.Chat.ID, language)
}

// handleFAQContact обрабатывает нажатие на кнопку "Контакт с админом" в меню FAQ
func (b *Bot) handleFAQContact(message *telego.Message) {
	dbUser, err := b.service.GetUser(message.From.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for FAQ contact: %v", err)
		return
	}

	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализированный текст для раздела "Контакт с админом"
	options := b.prepareMessage("contactm", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем текст о контакте с админом
	b.SendMessage(message.Chat.ID, options)

	// Отправляем последующее сообщение для продолжения навигации после небольшой задержки
	go b.sendFAQNextPrompt(message.Chat.ID, language)
}

// sendFAQNextPrompt отправляет сообщение с приглашением выбрать следующий раздел FAQ
func (b *Bot) sendFAQNextPrompt(chatID int64, language string) {
	// Задержка перед отправкой сообщения
	time.Sleep(5 * time.Second)

	// Получаем локализированный текст для сообщения
	options := b.prepareMessage("faqnext", language)
	options.ReplyKeyboard = b.createFAQKeyboard(language)

	// Отправляем сообщение
	err := b.SendMessage(chatID, options)

	if err != nil {
		logger.Error.Printf("Error sending FAQ next prompt: %v", err)
	}
}
