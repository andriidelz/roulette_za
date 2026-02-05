package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/models"

	"github.com/mymmrac/telego"
)

const (
	CallbackRequestWithdraw = "request_withdraw"
	CallbackCheckWallet     = "withdraw_check_wallet"
	CallbackChangeWallet    = "withdraw_change_wallet"
	CallbackCheckAmount     = "withdraw_check_amount"
	CallbackSetAmount       = "withdraw_set_amount"
	CallbackProcessWithdraw = "process_withdraw"
	CallbackCancelInput     = "withdraw_cancel_input"
)

// Константа с минимальной суммой для вывода, которую далее заменяем на получение из настроек
const MinWithdrawalAmount = 10.0

// Константа с комиссией
const FeeAmount = 1.0

// handleAccountCommand обрабатывает команду "Аккаунт" из главного меню
func (b *Bot) handleAccountCommand(message *telego.Message) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Получаем локализованный текст раздела аккаунта
	options := b.prepareMessage("accstart", language)

	// Создаем reply клавиатуру для аккаунта
	options.ReplyKeyboard = b.createAccountKeyboard(language)

	// Отправляем сообщение с клавиатурой
	b.SendMessage(message.Chat.ID, options)
}

// createAccountKeyboard создает клавиатуру для раздела аккаунта
func (b *Bot) createAccountKeyboard(language string) *telego.ReplyKeyboardMarkup {
	// Получаем локализованные тексты для кнопок
	balanceText := b.getText("balance", language)
	withdrawText := b.getText("withdraw", language)
	bonusText := b.getText("bonus", language)
	buyBetsText := b.getText("buybets", language)
	exitAccText := b.getText("exitacc", language)

	// Создаем клавиатуру
	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: balanceText},
				{Text: withdrawText},
			},
			{
				{Text: bonusText},
				{Text: buyBetsText},
			},
			{
				{Text: exitAccText},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
}

// handleBonusCommand - вивід бонусів
func (b *Bot) handleBonusCommand(message *telego.Message) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Получаем локализированный текст для раздела "Bonus"
	options := b.prepareMessage("bonusm", language)
	options.ReplyKeyboard = b.createAccountKeyboard(language)

	// Отправляем текст правил
	b.SendMessage(message.Chat.ID, options)
}

// handleBuyBetsCommand - отримання ставок
func (b *Bot) handleBuyBetsCommand(message *telego.Message) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Получаем локализированный текст для раздела "BuyBets"
	options := b.prepareMessage("buybetsm", language)
	options.ReplyKeyboard = b.createAccountKeyboard(language)

	// Отправляем текст правил
	b.SendMessage(message.Chat.ID, options)
}

// Модифицируем существующий метод handleBalanceCommand
func (b *Bot) handleBalanceCommand(message *telego.Message) {
	user := message.From

	// Получаем информацию о пользователе для проверки баланса
	dbUser, err := b.getUser(user.ID)
	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, b.prepareMessage("error_retrieving_balance", language))
		return
	}

	// Проверяем, достаточно ли денег для вывода
	minWithdrawal := MinWithdrawalAmount

	// Получаем настройку минимального вывода (если есть)
	settings, err := b.service.GetSettings()
	if err == nil {
		if minWithdrawalStr, exists := settings["minimum_withdrawal"]; exists && minWithdrawalStr != "" {
			// Пытаемся преобразовать значение
			if val, err := strconv.ParseFloat(minWithdrawalStr, 64); err == nil {
				minWithdrawal = val
			}
		}
	}

	// Отправляем соответствующее сообщение в зависимости от баланса
	if dbUser.Balance >= minWithdrawal {
		// Баланс достаточен для вывода
		options := b.prepareMessage("balanceaccok", language)
		options.Text = fmt.Sprintf(options.Text, dbUser.Balance)

		// Создаем кнопку для запроса вывода
		withdrawButtonText := b.getText("balaccokwith", language)
		withdrawButton := telego.InlineKeyboardButton{
			Text:         withdrawButtonText,
			CallbackData: CallbackRequestWithdraw,
		}

		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{withdrawButton},
			},
		}
		options.InlineKeyboard = inlineKeyboard
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		// Отправляем сообщение с балансом и кнопкой
		b.SendMessage(message.Chat.ID, options)
	} else {
		// Баланс недостаточен для вывода
		options := b.prepareMessage("balanceacclow", language)
		options.Text = fmt.Sprintf(options.Text, dbUser.Balance)
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		// Отправляем сообщение без кнопки вывода
		b.SendMessage(message.Chat.ID, options)
	}

	// Отправляем сообщение с предложением выбрать следующее действие через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		options := b.prepareMessage("balancenext", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		b.SendMessage(message.Chat.ID, options)
	}()
}

// Обновим существующий метод handleWithdrawCommand
func (b *Bot) handleWithdrawCommand(message *telego.Message) {
	user := message.From

	// Получаем информацию о пользователе для проверки баланса
	dbUser, err := b.getUser(user.ID)
	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, b.prepareMessage("error_retrieving_data", language))
		return
	}

	// Минимальная сумма для вывода
	minWithdrawal := MinWithdrawalAmount

	// Получаем настройку минимального вывода (если есть)
	settings, err := b.service.GetSettings()
	if err == nil {
		if minWithdrawalStr, exists := settings["minimum_withdrawal"]; exists && minWithdrawalStr != "" {
			// Пытаемся преобразовать значение
			if val, err := strconv.ParseFloat(minWithdrawalStr, 64); err == nil {
				minWithdrawal = val
			}
		}
	}

	// Проверяем достаточно ли денег для вывода
	if dbUser.Balance >= minWithdrawal {
		// Баланс достаточен для вывода
		options := b.prepareMessage("withdrawok", language)
		options.Text = fmt.Sprintf(options.Text, dbUser.Balance)

		// Создаем кнопку для проверки кошелька
		processButtonText := b.getText("withdrawproc", language)
		processButton := telego.InlineKeyboardButton{
			Text:         processButtonText,
			CallbackData: CallbackCheckWallet,
		}

		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{processButton},
			},
		}

		options.InlineKeyboard = inlineKeyboard
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		// Отправляем сообщение с балансом и кнопкой
		b.SendMessage(message.Chat.ID, options)
	} else {
		// Баланс недостаточен для вывода
		options := b.prepareMessage("withdrawlow", language)
		options.Text = fmt.Sprintf(options.Text, dbUser.Balance)
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		// Отправляем сообщение без кнопки
		b.SendMessage(message.Chat.ID, options)
	}

	// Отправляем сообщение с предложением выбрать следующее действие через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		options := b.prepareMessage("withdrawlownext", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		b.SendMessage(message.Chat.ID, options)
	}()
}

// handleInputWithdrawAmountCommand обрабатывает введение сумы на вывод
func (b *Bot) handleInputWithdrawAmountCommand(message *telego.Message) {
	user := message.From

	// Получаем информацию о пользователе для проверки баланса
	dbUser, err := b.getUser(user.ID)
	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, b.prepareMessage("error_retrieving_data", language))
		b.stateManager.ClearState(user.ID)
		return
	}

	// Проверка валидности суммы
	amountText := strings.TrimSpace(message.Text)

	withdrawAmount, err := strconv.ParseFloat(amountText, 64)
	if err != nil {
		// Неверный формат суммы
		options := b.prepareMessage("withdrawusdtsumerror", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		// Отправляем сообщение об ошибке
		b.SendMessage(message.Chat.ID, options)

		// Отправляем сообщение с предложением остановить вывод и вернуться в меню через 1 секунду
		go func() {
			time.Sleep(1 * time.Second)
			b.sendCancelMessage(message.Chat.ID, language)
		}()
		return
	}

	if dbUser.Balance < withdrawAmount {
		// Если сумма недостаточна, сообщаем об этом
		options := b.prepareMessage("withdrawusdtsumbig", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		b.SendMessage(message.Chat.ID, options)

		// Отправляем сообщение с предложением остановить вывод и вернуться в меню через 1 секунду
		go func() {
			time.Sleep(1 * time.Second)
			b.sendCancelMessage(message.Chat.ID, language)
		}()
		return
	}

	// Проверяем, что сумма больше минимальной
	minWithdrawal := MinWithdrawalAmount

	// Получаем настройку минимального вывода (если есть)
	settings, err := b.service.GetSettings()
	if err == nil {
		if minWithdrawalStr, exists := settings["minimum_withdrawal"]; exists && minWithdrawalStr != "" {
			// Пытаемся преобразовать значение
			if val, err := strconv.ParseFloat(minWithdrawalStr, 64); err == nil {
				minWithdrawal = val
			}
		}
	}

	if withdrawAmount < minWithdrawal {
		// Если сумма недостаточна, сообщаем об этом
		options := b.prepareMessage("insufficient_balance", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		b.SendMessage(message.Chat.ID, options)

		// Отправляем сообщение с предложением остановить вывод и вернуться в меню через 1 секунду
		go func() {
			time.Sleep(1 * time.Second)
			b.sendCancelMessage(message.Chat.ID, language)
		}()
		return
	}

	// Создаем запрос на вывод средств
	withdrawal := &models.Withdrawal{
		UserID:    dbUser.ID,
		Amount:    withdrawAmount,
		Status:    "pending",
		Wallet:    dbUser.WalletAddress,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.stateManager.ClearState(user.ID)

	if err := b.service.CreateWithdrawal(withdrawal); err != nil {
		logger.Error.Printf("Error creating withdrawal request: %v", err)
		options := b.prepareMessage("withdrawal_error", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		b.SendMessage(message.Chat.ID, options)
		return
	}

	// Уменьшаем баланс пользователя
	dbUser.Balance -= withdrawAmount // Уменьшаем баланс на введенную суму
	if err := b.service.UpdateUser(dbUser); err != nil {
		logger.Error.Printf("Error updating user balance: %v", err)
	}
	b.updateUserCache(user.ID)

	// Отправляем сообщение об успешном создании запроса
	options := b.prepareMessage("withdrawsumok", language)
	options.Text = fmt.Sprintf(options.Text, withdrawal.Amount, dbUser.WalletAddress)

	// Создаем кнопку для возврата в главное меню
	exitAccText := b.getText("exitacc", language)
	exitAccButton := telego.InlineKeyboardButton{
		Text:         exitAccText,
		CallbackData: CallbackSettingsMainMenu,
	}
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{exitAccButton},
		},
	}
	options.InlineKeyboard = inlineKeyboard
	options.ReplyKeyboard = b.createAccountKeyboard(language)

	b.SendMessage(message.Chat.ID, options)
}

// handleInputWithdrawWalletCommand обрабатывает введение кошелька
func (b *Bot) handleInputWithdrawWalletCommand(message *telego.Message) {
	user := message.From

	// Получаем информацию о пользователе
	dbUser, err := b.getUser(user.ID)
	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, b.prepareMessage("error_retrieving_data", language))
		b.stateManager.ClearState(user.ID)
		return
	}

	// Проверка валидности адреса кошелька (базовая проверка)
	walletAddress := strings.TrimSpace(message.Text)

	// Базовая валидация адреса TRC20
	if !strings.HasPrefix(walletAddress, "T") || len(walletAddress) < 30 {
		// Неверный формат кошелька
		options := b.prepareMessage("withdrawusdtchangeerror", language)
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		// Отправляем сообщение об ошибке
		b.SendMessage(message.Chat.ID, options)

		// Отправляем сообщение с предложением остановить вывод и вернуться в меню через 1 секунду
		go func() {
			time.Sleep(1 * time.Second)
			b.sendCancelMessage(message.Chat.ID, language)
		}()

		return
	}

	// Обновляем адрес кошелька пользователя
	dbUser.WalletAddress = walletAddress
	if err := b.service.UpdateUser(dbUser); err != nil {
		logger.Error.Printf("Error updating user wallet address: %v", err)
	}
	b.updateUserCache(user.ID)

	// Отправляем сообщение об успешном обновлении
	options := b.prepareMessage("withdrawusdtchangeok", language)

	// Создаем кнопку для подтверждения кошелька
	walletOKButtonText := b.getText("usdtok", language)
	walletOKButton := telego.InlineKeyboardButton{
		Text:         walletOKButtonText,
		CallbackData: CallbackCheckAmount,
	}

	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				walletOKButton,
			},
		},
	}
	options.InlineKeyboard = inlineKeyboard
	options.ReplyKeyboard = b.createAccountKeyboard(language)
	b.SendMessage(message.Chat.ID, options)

	// Очищаем состояние
	b.stateManager.ClearState(user.ID)
}

// Новые обработчики для колбэков

// handleRequestWithdrawCallback обрабатывает нажатие на кнопку "Заказать вывод" в разделе баланса
func (b *Bot) handleRequestWithdrawCallback(query *telego.CallbackQuery) {
	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	user := query.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Переводим пользователя в раздел вывода
	if query.Message != nil {
		b.handleWithdrawCommand(&telego.Message{
			From: &telego.User{
				ID:           query.From.ID,
				IsBot:        query.From.IsBot,
				FirstName:    query.From.FirstName,
				LastName:     query.From.LastName,
				Username:     query.From.Username,
				LanguageCode: language,
			},
			Chat: query.Message.GetChat(),
		})
	}
}

// handleCheckWalletCallback обрабатывает нажатие на кнопку "Запросить вывод" в разделе вывода
func (b *Bot) handleCheckWalletCallback(query *telego.CallbackQuery) {
	user := query.From

	// Получаем информацию о пользователе
	dbUser, err := b.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for withdrawal: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	// Проверяем наличие адреса кошелька
	if dbUser.WalletAddress == "" {
		// Если адрес кошелька не указан, отправляем сообщение об этом
		// и предлагаем заполнить его в настройках
		options := b.prepareMessage("no_wallet_address", language)

		// Создаем кнопку для указания кошелька
		walletChangeButtonText := b.getText("usdtchange", language)
		walletChangeButton := telego.InlineKeyboardButton{
			Text:         walletChangeButtonText,
			CallbackData: CallbackChangeWallet,
		}

		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{walletChangeButton},
			},
		}

		if query.Message != nil {
			options.InlineKeyboard = inlineKeyboard
			options.ReplyKeyboard = b.createAccountKeyboard(language)
			b.SendMessage(query.Message.GetChat().ID, options)
		}
		return
	}

	options := b.prepareMessage("withdrawusdtcheck", language)
	options.Text = fmt.Sprintf(options.Text, dbUser.WalletAddress)

	// Создаем кнопку для подтверждения кошелька
	walletOKButtonText := b.getText("usdtok", language)
	walletOKButton := telego.InlineKeyboardButton{
		Text:         walletOKButtonText,
		CallbackData: CallbackCheckAmount,
	}

	// Создаем кнопку для изменения кошелька
	walletChangeButtonText := b.getText("usdtchange", language)
	walletChangeButton := telego.InlineKeyboardButton{
		Text:         walletChangeButtonText,
		CallbackData: CallbackChangeWallet,
	}

	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				walletOKButton,
				walletChangeButton,
			},
		},
	}

	if query.Message != nil {
		options.InlineKeyboard = inlineKeyboard
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		b.SendMessage(query.Message.GetChat().ID, options)
	}
}

// handleCheckAmountCallback обрабатывает нажатие на кнопку "Подтвердить адрес" в разделе вывода
func (b *Bot) handleCheckAmountCallback(query *telego.CallbackQuery) {
	user := query.From

	// Получаем информацию о пользователе
	dbUser, err := b.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for withdrawal: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	// Выводим сообщение с доступной суммой, комиссией и предложением вывести все или часть сумы
	options := b.prepareMessage("withdrawusdtsumcheck", language)
	options.Text = fmt.Sprintf(options.Text, dbUser.Balance, FeeAmount)

	// Создаем кнопку для запроса вывода всей суммы
	amountAllButtonText := b.getText("withdrawusdtall", language)
	amountAllButton := telego.InlineKeyboardButton{
		Text:         amountAllButtonText,
		CallbackData: CallbackProcessWithdraw,
	}

	// Создаем кнопку для указания суммы для вывода
	amountAmountButtonText := b.getText("withdrawusdtamount", language)
	amountAmountButton := telego.InlineKeyboardButton{
		Text:         amountAmountButtonText,
		CallbackData: CallbackSetAmount,
	}

	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				amountAllButton,
				amountAmountButton,
			},
		},
	}

	if query.Message != nil {
		options.InlineKeyboard = inlineKeyboard
		options.ReplyKeyboard = b.createAccountKeyboard(language)

		b.SendMessage(query.Message.GetChat().ID, options)
	}
}

// handleProcessWithdrawCallback обрабатывает нажатие на кнопку "Вывести всю суму" в разделе вывода
func (b *Bot) handleProcessWithdrawCallback(query *telego.CallbackQuery) {
	user := query.From

	// Получаем информацию о пользователе
	dbUser, err := b.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user for withdrawal: %v", err)
		return
	}

	language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	// Проверяем, что сумма больше минимальной
	minWithdrawal := MinWithdrawalAmount

	// Получаем настройку минимального вывода (если есть)
	settings, err := b.service.GetSettings()
	if err == nil {
		if minWithdrawalStr, exists := settings["minimum_withdrawal"]; exists && minWithdrawalStr != "" {
			// Пытаемся преобразовать значение
			if val, err := strconv.ParseFloat(minWithdrawalStr, 64); err == nil {
				minWithdrawal = val
			}
		}
	}

	if dbUser.Balance < minWithdrawal {
		// Если сумма недостаточна, сообщаем об этом
		options := b.prepareMessage("insufficient_balance", language)

		if query.Message != nil {
			options.ReplyKeyboard = b.createAccountKeyboard(language)
			b.SendMessage(query.Message.GetChat().ID, options)
		}
		return
	}

	// Создаем запрос на вывод средств
	withdrawal := &models.Withdrawal{
		UserID:    dbUser.ID,
		Amount:    dbUser.Balance,
		Status:    "pending",
		Wallet:    dbUser.WalletAddress,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := b.service.CreateWithdrawal(withdrawal); err != nil {
		logger.Error.Printf("Error creating withdrawal request: %v", err)
		options := b.prepareMessage("withdrawal_error", language)

		if query.Message != nil {
			options.ReplyKeyboard = b.createAccountKeyboard(language)
			b.SendMessage(query.Message.GetChat().ID, options)
		}
		return
	}

	// Уменьшаем баланс пользователя
	dbUser.Balance = 0 // Обнуляем баланс, так как выводим всю сумму
	if err := b.service.UpdateUser(dbUser); err != nil {
		logger.Error.Printf("Error updating user balance: %v", err)
	}
	b.updateUserCache(user.ID)

	// Отправляем сообщение об успешном создании запроса
	options := b.prepareMessage("withdrawsumok", language)
	options.Text = fmt.Sprintf(options.Text, withdrawal.Amount, dbUser.WalletAddress)

	if query.Message != nil {
		options.ReplyKeyboard = b.createAccountKeyboard(language)
		b.SendMessage(query.Message.GetChat().ID, options)
	}
}

// sendChancelMessage отправляет сообщение с предложением остановить вывод и вернуться в меню
func (b *Bot) sendCancelMessage(chatID int64, language string) {
	options := b.prepareMessage("withdrawusdtsumstop", language)

	// Создаем кнопку для возврата в главное меню
	exitAccText := b.getText("exitacc", language)
	exitAccButton := telego.InlineKeyboardButton{
		Text:         exitAccText,
		CallbackData: CallbackSettingsMainMenu,
	}
	// Создаем кнопку отмены ввода
	cancelText := b.getText("withdrawcancel", language)
	cancelButton := telego.InlineKeyboardButton{
		Text:         cancelText,
		CallbackData: CallbackCancelInput,
	}
	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{exitAccButton, cancelButton},
		},
	}
	options.InlineKeyboard = inlineKeyboard
	options.ReplyKeyboard = b.createAccountKeyboard(language)
	b.SendMessage(chatID, options)
}

// UserStatusDisabled присвоюється користувачу який забанив бота
func (b *Bot) disableUser(telegramID int64, reason, meta string) {

	dbUser, err := b.getUser(telegramID)
	if err != nil {
		logger.Error.Printf("Failed to getUser: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Redis - видалити всі черги / ключі, пов’язані з user_id
	err = b.redisDB.ZRem(ctx, userSendTaskKey, telegramID).Err()
	if err != nil {
		logger.Error.Printf("Error ZRem %d: %v", telegramID, err)
	}

	_, err = b.redisDB.Del(ctx, fmt.Sprintf(userQueueKeyPrefix, telegramID)).Result()
	if err != nil {
		logger.Error.Printf("Error Del %d: %v", telegramID, err)
	}

	if reason != "" {
		// create new record
		banLog := &models.UserBanLog{
			UserID:     dbUser.ID,
			TypeStatus: config.UserStatusDisabled,
			Reason:     reason,
			ReasonMeta: meta,
			Active:     false,
			UntilTo:    time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Save to database
		if err := b.service.GetRepo().CreateBanLog(banLog); err != nil {
			logger.Error.Printf("Failed to create ban log: %v", err)
		}
	}

	if dbUser.Status != config.UserStatusBanned {
		// Оновлюєм статус на відключений
		dbUser.Status = config.UserStatusDisabled
		err = b.service.UpdateUser(dbUser)
		if err != nil {
			logger.Error.Printf("Error updating user: %v", err)
		}
		b.updateUserCache(telegramID)
	}
}
