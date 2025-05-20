package bot

import (
	"fmt"
	"log"
	"roulette/internal/models"
	"strconv"
	"time"

	"github.com/mymmrac/telego"
)

const (
	CallbackRequestWithdraw = "request_withdraw"
	CallbackCheckWallet     = "check_wallet"
	CallbackCheckAmount     = "withdraw_check_amount"
	CallbackSetAmount       = "withdraw_set_amount"
	CallbackProcessWithdraw = "process_withdraw"
)

// Константа с минимальной суммой для вывода, которую далее заменяем на получение из настроек
const MinWithdrawalAmount = 10.0

// Константа с комиссией
const FeeAmount = 1.0

// handleAccountCommand обрабатывает команду "Аккаунт" из главного меню
func (b *Bot) handleAccountCommand(message *telego.Message) {
	user := message.From
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализованный текст раздела аккаунта
	accountStartText := b.service.GetText("accstart", language)

	// Создаем reply клавиатуру для аккаунта
	accountKeyboard := b.createAccountKeyboard(language)

	// Отправляем сообщение с клавиатурой
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:          accountStartText,
		ReplyKeyboard: accountKeyboard,
	})
}

// createAccountKeyboard создает клавиатуру для раздела аккаунта
func (b *Bot) createAccountKeyboard(language string) *telego.ReplyKeyboardMarkup {
	// Получаем локализованные тексты для кнопок
	balanceText := b.service.GetText("balance", language)
	withdrawText := b.service.GetText("withdraw", language)
	bonusText := b.service.GetText("bonus", language)
	buyBetsText := b.service.GetText("buybets", language)
	exitAccText := b.service.GetText("exitacc", language)

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

// Модифицируем существующий метод handleBalanceCommand
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
			Text: b.service.GetText("error_retrieving_balance", language),
		})
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
		balanceTemplate := b.service.GetText("balanceaccok", language)
		balanceText := fmt.Sprintf(balanceTemplate, dbUser.Balance)

		// Создаем кнопку для запроса вывода
		withdrawButtonText := b.service.GetText("balaccokwith", language)
		withdrawButton := telego.InlineKeyboardButton{
			Text:         withdrawButtonText,
			CallbackData: CallbackRequestWithdraw,
		}

		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{withdrawButton},
			},
		}

		// Отправляем сообщение с балансом и кнопкой
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:           balanceText,
			InlineKeyboard: inlineKeyboard,
			ReplyKeyboard:  b.createAccountKeyboard(language),
		})
	} else {
		// Баланс недостаточен для вывода
		balanceTemplate := b.service.GetText("balanceacclow", language)
		balanceText := fmt.Sprintf(balanceTemplate, dbUser.Balance)

		// Отправляем сообщение без кнопки вывода
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          balanceText,
			ReplyKeyboard: b.createAccountKeyboard(language),
		})
	}

	// Отправляем сообщение с предложением выбрать следующее действие через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		balanceNextText := b.service.GetText("balancenext", language)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          balanceNextText,
			ReplyKeyboard: b.createAccountKeyboard(language),
		})
	}()
}

// Обновим существующий метод handleWithdrawCommand
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
			Text: b.service.GetText("error_retrieving_data", language),
		})
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
		withdrawTemplate := b.service.GetText("withdrawok", language)
		withdrawText := fmt.Sprintf(withdrawTemplate, dbUser.Balance)

		// Создаем кнопку для проверки кошелька
		processButtonText := b.service.GetText("withdrawproc", language)
		processButton := telego.InlineKeyboardButton{
			Text:         processButtonText,
			CallbackData: CallbackCheckWallet,
		}

		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{processButton},
			},
		}

		// Отправляем сообщение с балансом и кнопкой
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:           withdrawText,
			InlineKeyboard: inlineKeyboard,
			ReplyKeyboard:  b.createAccountKeyboard(language),
		})
	} else {
		// Баланс недостаточен для вывода
		withdrawTemplate := b.service.GetText("withdrawlow", language)
		withdrawText := fmt.Sprintf(withdrawTemplate, dbUser.Balance)

		// Отправляем сообщение без кнопки
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          withdrawText,
			ReplyKeyboard: b.createAccountKeyboard(language),
		})
	}

	// Отправляем сообщение с предложением выбрать следующее действие через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		withdrawNextText := b.service.GetText("withdrawlownext", language)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text:          withdrawNextText,
			ReplyKeyboard: b.createAccountKeyboard(language),
		})
	}()
}

// Новые обработчики для колбэков

// handleRequestWithdrawCallback обрабатывает нажатие на кнопку "Заказать вывод" в разделе баланса
func (b *Bot) handleRequestWithdrawCallback(query *telego.CallbackQuery) {
	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	user := query.From
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		// Регистрация пользователя, если он не найден
		dbUser, err = b.service.RegisterUser(user.ID, user.Username, user.FirstName, user.LastName, user.LanguageCode)
		if err != nil {
			log.Printf("Error registering user: %v", err)
			return
		}
	}

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
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
			Chat: query.Message.Chat,
		})
	}
}

// handleCheckWalletCallback обрабатывает нажатие на кнопку "Запросить вывод" в разделе вывода
func (b *Bot) handleCheckWalletCallback(query *telego.CallbackQuery) {
	user := query.From

	// Получаем информацию о пользователе
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user for withdrawal: %v", err)
		return
	}

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	// Проверяем наличие адреса кошелька
	if dbUser.WalletAddress == "" {
		// Если адрес кошелька не указан, отправляем сообщение об этом
		// и предлагаем заполнить его в настройках
		noWalletText := b.service.GetText("no_wallet_address", language)

		// Создаем кнопку для перехода в настройки
		settingsButtonText := b.service.GetText("go_to_settings", language)
		settingsButton := telego.InlineKeyboardButton{
			Text:         settingsButtonText,
			CallbackData: CallbackSettingsWallet, // Это колбэк для перехода в настройки кошелька
		}

		inlineKeyboard := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{settingsButton},
			},
		}

		if query.Message != nil {
			b.SendMessage(query.Message.Chat.ID, MessageOptions{
				Text:           noWalletText,
				InlineKeyboard: inlineKeyboard,
				ReplyKeyboard:  b.createAccountKeyboard(language),
			})
		}
		return
	}

	walletTemplate := b.service.GetText("withdrawusdtcheck", language)
	checkWalletText := fmt.Sprintf(walletTemplate, dbUser.WalletAddress)

	// Создаем кнопку для подтверждения суммы
	walletOKButtonText := b.service.GetText("usdtok", language)
	walletOKButton := telego.InlineKeyboardButton{
		Text:         walletOKButtonText,
		CallbackData: CallbackCheckAmount,
	}

	// Создаем кнопку для перехода в настройки
	walletNoButtonText := b.service.GetText("go_to_settings", language)
	walletNoButton := telego.InlineKeyboardButton{
		Text:         walletNoButtonText,
		CallbackData: CallbackSettingsWallet, // Это колбэк для перехода в настройки кошелька
	}

	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				walletOKButton,
				walletNoButton,
			},
		},
	}

	if query.Message != nil {
		b.SendMessage(query.Message.Chat.ID, MessageOptions{
			Text:           checkWalletText,
			InlineKeyboard: inlineKeyboard,
			ReplyKeyboard:  b.createAccountKeyboard(language),
		})
	}
}

// handleCheckAmountCallback обрабатывает нажатие на кнопку "Подтвердить адрес" в разделе вывода
func (b *Bot) handleCheckAmountCallback(query *telego.CallbackQuery) {
	user := query.From

	// Получаем информацию о пользователе
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user for withdrawal: %v", err)
		return
	}

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

	// Отвечаем на callback, чтобы убрать индикатор загрузки
	b.answerCallbackQuery(query.ID, "", false)

	// Выводим сообщение с доступной суммой, комиссией и предложением вывести все или часть сумы
	checkAmountTemplate := b.service.GetText("withdrawusdtsumcheck", language)
	checkAmountText := fmt.Sprintf(checkAmountTemplate, dbUser.Balance, FeeAmount)

	// Создаем кнопку для запроса вывода всей суммы
	amountAllButtonText := b.service.GetText("withdrawusdtall", language)
	amountAllButton := telego.InlineKeyboardButton{
		Text:         amountAllButtonText,
		CallbackData: CallbackProcessWithdraw,
	}

	// Создаем кнопку для указания суммы для вывода
	amountAmountButtonText := b.service.GetText("withdrawusdtamount", language)
	amountAmountButton := telego.InlineKeyboardButton{
		Text:         amountAmountButtonText,
		CallbackData: CallbackSetAmount,
	}

	inlineKeyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				amountAllButton,
				amountAmountButton, // не работает
			},
		},
	}

	if query.Message != nil {
		b.SendMessage(query.Message.Chat.ID, MessageOptions{
			Text:           checkAmountText,
			InlineKeyboard: inlineKeyboard,
			ReplyKeyboard:  b.createAccountKeyboard(language),
		})
	}
}

// handleProcessWithdrawCallback обрабатывает нажатие на кнопку "Вывести всю суму" в разделе вывода
func (b *Bot) handleProcessWithdrawCallback(query *telego.CallbackQuery) {
	user := query.From

	// Получаем информацию о пользователе
	dbUser, err := b.service.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user for withdrawal: %v", err)
		return
	}

	// Всегда используем язык из базы данных, т.к. он может быть обновлен
	language := dbUser.LanguageCode
	if language == "" {
		language = "en"
	}

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
		insufficientText := b.service.GetText("insufficient_balance", language)

		if query.Message != nil {
			b.SendMessage(query.Message.Chat.ID, MessageOptions{
				Text:          insufficientText,
				ReplyKeyboard: b.createAccountKeyboard(language),
			})
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
		log.Printf("Error creating withdrawal request: %v", err)
		errorText := b.service.GetText("withdrawal_error", language)

		if query.Message != nil {
			b.SendMessage(query.Message.Chat.ID, MessageOptions{
				Text:          errorText,
				ReplyKeyboard: b.createAccountKeyboard(language),
			})
		}
		return
	}

	// Уменьшаем баланс пользователя
	dbUser.Balance = 0 // Обнуляем баланс, так как выводим всю сумму
	if err := b.service.UpdateUser(dbUser); err != nil {
		log.Printf("Error updating user balance: %v", err)
	}

	// Отправляем сообщение об успешном создании запроса
	successTemplate := b.service.GetText("withdrawal_success", language)
	successText := fmt.Sprintf(successTemplate, withdrawal.Amount, dbUser.WalletAddress)

	if query.Message != nil {
		b.SendMessage(query.Message.Chat.ID, MessageOptions{
			Text:          successText,
			ReplyKeyboard: b.createAccountKeyboard(language),
		})
	}
}
