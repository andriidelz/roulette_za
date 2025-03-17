package bot

import (
	"fmt"
	"log"

	"github.com/mymmrac/telego"
)

// Константы для колбеков настроек
const (
	// Основные категории настроек
	CallbackSettingsLanguage = "settings_language"
	CallbackSettingsCountry  = "settings_country"
	CallbackSettingsName     = "settings_name"
	CallbackSettingsLastName = "settings_lastname"

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
	settingsText := b.service.GetText("settings_message", language)

	// Создаем inline клавиатуру для настроек с отображением текущих значений
	inlineKeyboard := b.createSettingsKeyboard(language, user.ID)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:           settingsText,
		InlineKeyboard: inlineKeyboard,
	})
}

// Создает клавиатуру настроек
func (b *Bot) createSettingsKeyboard(language string, userID int64) *telego.InlineKeyboardMarkup {
	// Получаем текущие данные пользователя
	user, err := b.service.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user for settings keyboard: %v", err)
		// Возвращаем клавиатуру без текущих значений в случае ошибки
		return b.createBasicSettingsKeyboard(language)
	}

	// Получаем локализованные тексты для кнопок
	btnLanguageText := b.service.GetText("btn_settings_language", language)
	btnCountryText := b.service.GetText("btn_settings_country", language)
	btnNameText := b.service.GetText("btn_settings_name", language)
	btnLastNameText := b.service.GetText("btn_settings_lastname", language)
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
		for _, c := range countries {
			if c.Code == countryDisplay {
				countryDisplay = c.Emoji + " " + countryDisplay
				break
			}
		}
	}

	// Для имени и фамилии отображаем текущие значения
	firstName := user.FirstName
	if firstName == "" {
		firstName = "-"
	}

	lastName := user.LastName
	if lastName == "" {
		lastName = "-"
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
				{Text: fmt.Sprintf("%s: %s", btnLastNameText, lastName), CallbackData: CallbackSettingsLastName},
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
	btnLastNameText := b.service.GetText("btn_settings_lastname", language)
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
				{Text: btnLastNameText, CallbackData: CallbackSettingsLastName},
			},
			{
				{Text: btnBackText, CallbackData: CallbackSettingsMainMenu},
			},
		},
	}
}

// Создает клавиатуру выбора языка
func (b *Bot) createLanguageKeyboard() *telego.InlineKeyboardMarkup {
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
				{Text: "◀️ Back", CallbackData: CallbackSettingsBack},
			},
		},
	}
}
