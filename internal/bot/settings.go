package bot

import (
	"fmt"
	"roulette/internal/data"
	"roulette/internal/logger"
	"strings"

	"github.com/mymmrac/telego"
)

// Константы для колбеков настроек
const (
	// Основные категории настроек
	CallbackSettingsLanguage = "settings_language"
	CallbackSettingsCountry  = "settings_country"
	CallbackSettingsName     = "settings_name"
	CallbackSettingsNickName = "settings_nickname"
	CallbackSettingsWallet   = "settings_wallet"

	// Выбор языка
	CallbackLanguageEN = "language_en"
	CallbackLanguageRU = "language_ru"
	CallbackLanguageUK = "language_uk"

	// Действия с настройками
	CallbackSettingsBack     = "settings_back"
	CallbackSettingsMainMenu = "settings_main_menu"
)

// Обработчик команды настроек
func (b *Bot) handleSettingsCommand(message *telego.Message) {
	user := message.From

	language, err := b.getUserLang(user.ID, user.LanguageCode)
	if err != nil {
		logger.Error.Printf("Error getting user %d: %v", user.ID, err)
		return
	}

	// Получаем локализованный текст для настроек
	options := b.prepareMessage("settings_message", language)

	// Создаем inline клавиатуру для настроек с отображением текущих значений
	options.InlineKeyboard = b.createSettingsKeyboard(language, user.ID)

	b.SendMessage(message.Chat.ID, options)
}

// Создает клавиатуру настроек
func (b *Bot) createSettingsKeyboard(language string, userID int64) *telego.InlineKeyboardMarkup {
	// Получаем текущие данные пользователя
	user, err := b.getUser(userID)
	if err != nil {
		logger.Error.Printf("Error getting user for settings keyboard: %v", err)
		// Возвращаем клавиатуру без текущих значений в случае ошибки
		return b.createBasicSettingsKeyboard(language)
	}

	// Получаем локализованные тексты для кнопок
	btnLanguageText := b.getText("btn_settings_language", language)
	btnCountryText := b.getText("btn_settings_country", language)
	btnNameText := b.getText("btn_settings_name", language)
	btnNicknameText := b.getText("btn_settings_nickname", language)
	btnWalletText := b.getText("btn_settings_wallet", language) // Новая локализация
	btnBackText := b.getText("btn_back_to_main", language)

	// Добавляем текущие значения в кнопки

	// Для языка отображаем название на выбранном языке
	languageName := ""
	switch language {
	case "en":
		languageName = "English"
	case "ru":
		languageName = "Русский"
	case "uk":
		languageName = "Українська"
	default:
		languageName = "English" // По умолчанию
	}

	// Для страны показываем текущий выбор (если есть)
	countryDisplay := user.Country
	if countryDisplay == "" {
		countryDisplay = "-"
	} else {
		// Добавляем эмодзи флага, если есть страна
		for _, c := range data.Countries {
			if c.Code == countryDisplay {
				countryDisplay = c.Emoji + " " + countryDisplay
				break
			}
		}
	}

	// Для имени и никнейма отображаем текущие значения
	firstName := user.FirstName
	if firstName == "" {
		firstName = "-"
	}

	nickname := user.Nickname
	if nickname == "" {
		nickname = "-"
	}

	// Для адреса кошелька отображаем текущее значение или маскированное
	walletAddress := user.WalletAddress
	if walletAddress == "" {
		walletAddress = "-"
	} else if len(walletAddress) > 10 {
		// Маскируем адрес для безопасности: показываем только первые 6 и последние 4 символа
		walletAddress = walletAddress[:6] + "..." + walletAddress[len(walletAddress)-4:]
	}

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("%s: %s", btnLanguageText, languageName), CallbackData: CallbackSettingsLanguage},
			},
			{
				{Text: fmt.Sprintf("%s: %s", btnCountryText, countryDisplay), CallbackData: CallbackSettingsCountry},
			},
			{
				{Text: fmt.Sprintf("%s: %s", btnNameText, firstName), CallbackData: CallbackSettingsName},
			},
			{
				{Text: fmt.Sprintf("%s: %s", btnNicknameText, nickname), CallbackData: CallbackSettingsNickName},
			},
			{
				{Text: fmt.Sprintf("%s: %s", btnWalletText, walletAddress), CallbackData: CallbackSettingsWallet},
			},
			{
				{Text: btnBackText, CallbackData: CallbackSettingsMainMenu},
			},
		},
	}
}

// Функция для базовой клавиатуры без текущих значений (будет использоваться в случае ошибки)
func (b *Bot) createBasicSettingsKeyboard(language string) *telego.InlineKeyboardMarkup {
	// Получаем локализованные тексты для кнопок
	btnLanguageText := b.getText("btn_settings_language", language)
	btnCountryText := b.getText("btn_settings_country", language)
	btnNameText := b.getText("btn_settings_name", language)
	btnNicknameText := b.getText("btn_settings_nickname", language)
	btnWalletText := b.getText("btn_settings_wallet", language) // Новая локализация
	btnBackText := b.getText("btn_back_to_main", language)

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: btnLanguageText, CallbackData: CallbackSettingsLanguage},
			},
			{
				{Text: btnCountryText, CallbackData: CallbackSettingsCountry},
			},
			{
				{Text: btnNameText, CallbackData: CallbackSettingsName},
			},
			{
				{Text: btnNicknameText, CallbackData: CallbackSettingsNickName},
			},
			{
				{Text: btnWalletText, CallbackData: CallbackSettingsWallet},
			},
			{
				{Text: btnBackText, CallbackData: CallbackSettingsMainMenu},
			},
		},
	}
}

// Создает клавиатуру выбора языка
func (b *Bot) createLanguageKeyboard(language string) *telego.InlineKeyboardMarkup {
	btnBackText := b.getText("btn_back", language)

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "English 🇬🇧", CallbackData: CallbackLanguageEN},
			},
			{
				{Text: "Русский 🇷🇺", CallbackData: CallbackLanguageRU},
			},
			{
				{Text: "Українська 🇺🇦", CallbackData: CallbackLanguageUK},
			},
			{
				{Text: btnBackText, CallbackData: CallbackSettingsBack},
			},
		},
	}
}

// handleInputNameState обрабатывает ввод имени в настройках
func (b *Bot) handleInputNameState(message *telego.Message, messageID int) {
	user := message.From

	// Обработка ввода имени
	if len(message.Text) > 0 {
		// Обновляем имя пользователя
		dbUser, err := b.getUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}
		language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

		// Валидация имени
		name := strings.TrimSpace(message.Text)
		if len(name) == 0 || len(name) > 100 {
			// Неверная длина имени
			b.SendMessage(message.Chat.ID, b.prepareMessage("invalid_name", language))
			return
		}

		dbUser.FirstName = name
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user name: %v", err)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, b.prepareMessage("update_error", language))
			return
		}
		b.updateUserCache(user.ID)

		// Отправляем сообщение об успешном обновлении
		successText := b.getText("name_saved", language)

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

	// Обработка ввода никнейма при обновлении в настройках
	if len(message.Text) > 0 {
		// Обновляем никнейм пользователя
		dbUser, err := b.getUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}
		language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

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
		b.updateUserCache(user.ID)

		// Отправляем сообщение об успешном обновлении
		successText := b.getText("nickname_saved", language)

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

	// Обработка ввода адреса кошелька
	if len(message.Text) > 0 {
		dbUser, err := b.getUser(user.ID)
		if err != nil {
			logger.Error.Printf("Error getting user: %v", err)
			b.stateManager.ClearState(user.ID)
			return
		}

		language := getLanguage(dbUser.LanguageCode, user.LanguageCode)

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

		dbUser.WalletAddress = walletAddress
		if err := b.service.UpdateUser(dbUser); err != nil {
			logger.Error.Printf("Error updating user wallet address: %v", err)

			// Отправляем сообщение об ошибке
			b.SendMessage(message.Chat.ID, b.prepareMessage("update_error", language))
			return
		}
		b.updateUserCache(user.ID)

		// Отправляем сообщение об успешном обновлении
		successText := b.getText("withdrawusdtchangeok", language)

		backBtn := b.createBackBtnKeyboard(language)

		b.UpdateMessage(message.Chat.ID, messageID, MessageOptions{
			Text:           successText,
			InlineKeyboard: backBtn,
		})

		// Очищаем состояние
		b.stateManager.ClearState(user.ID)
	}
}
