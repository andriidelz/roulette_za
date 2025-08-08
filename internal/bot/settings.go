package bot

import (
	"fmt"
	"roulette/internal/data"
	"roulette/internal/logger"

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
	language := user.LanguageCode
	if language == "" {
		language = "en"
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
	user, err := b.service.GetUser(userID)
	if err != nil {
		logger.Error.Printf("Error getting user for settings keyboard: %v", err)
		// Возвращаем клавиатуру без текущих значений в случае ошибки
		return b.createBasicSettingsKeyboard(language)
	}

	// Получаем локализованные тексты для кнопок
	btnLanguageText := b.service.GetText("btn_settings_language", language)
	btnCountryText := b.service.GetText("btn_settings_country", language)
	btnNameText := b.service.GetText("btn_settings_name", language)
	btnNicknameText := b.service.GetText("btn_settings_nickname", language)
	btnWalletText := b.service.GetText("btn_settings_wallet", language) // Новая локализация
	btnBackText := b.service.GetText("btn_back_to_main", language)

	// Добавляем текущие значения в кнопки

	// Для языка отображаем название на выбранном языке
	languageName := ""
	switch user.LanguageCode {
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
	btnLanguageText := b.service.GetText("btn_settings_language", language)
	btnCountryText := b.service.GetText("btn_settings_country", language)
	btnNameText := b.service.GetText("btn_settings_name", language)
	btnNicknameText := b.service.GetText("btn_settings_nickname", language)
	btnWalletText := b.service.GetText("btn_settings_wallet", language) // Новая локализация
	btnBackText := b.service.GetText("btn_back_to_main", language)

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
	btnBackText := b.service.GetText("btn_back", language)

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
