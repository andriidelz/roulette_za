package bot

import (
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

	// Создаем inline клавиатуру для настроек
	inlineKeyboard := b.createSettingsKeyboard(language)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text:           settingsText,
		InlineKeyboard: inlineKeyboard,
	})
}

// Создает клавиатуру настроек
func (b *Bot) createSettingsKeyboard(language string) *telego.InlineKeyboardMarkup {
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
