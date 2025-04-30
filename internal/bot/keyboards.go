package bot

import (
	"fmt"
	"roulette/internal/data"
	"roulette/internal/utils"
	"sort"

	"github.com/mymmrac/telego"
)

// createStatsKeyboard создает клавиатуру для выбора периода статистики
func (b *Bot) createStatsKeyboard(language string) *telego.ReplyKeyboardMarkup {
	return &telego.ReplyKeyboardMarkup{
		Keyboard: [][]telego.KeyboardButton{
			{
				{Text: b.service.GetText("daystat", language)},
				{Text: b.service.GetText("weekstat", language)},
			},
			{
				{Text: b.service.GetText("monthstat", language)},
				{Text: b.service.GetText("allstat", language)},
			},
			{
				{Text: b.service.GetText("exitstat", language)},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
}

// createMainReplyKeyboard создает основную reply клавиатуру
func (b *Bot) createMainReplyKeyboard(language string) *telego.ReplyKeyboardMarkup {
	return &telego.ReplyKeyboardMarkup{
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
		Selective:       false,
	}
}

// createCountriesKeyboard создает клавиатуру с флагами стран и постраничной навигацией
// page - номер страницы (начиная с 1)
func (b *Bot) createCountriesKeyboard(page int) *telego.InlineKeyboardMarkup {
	// Сортировка:
	// 1. Сначала избранные страны
	// 2. Затем остальные страны по коду (в алфавитном порядке)
	// 3. RU и BY всегда в конце списка
	sort.Slice(data.Countries, func(i, j int) bool {

		// Специальная обработка для Украины - она всегда в начале списка
		if data.Countries[i].Code == "UA" {
			return true // i идет после j
		}
		if data.Countries[j].Code == "UA" {
			return false // i идет перед j
		}

		// Специальная обработка для России и Беларуси - они всегда в конце списка
		if data.Countries[i].Code == "RU" || data.Countries[i].Code == "BY" {
			return false // i идет после j
		}
		if data.Countries[j].Code == "RU" || data.Countries[j].Code == "BY" {
			return true // i идет перед j
		}

		// Избранные страны идут первыми
		if data.Countries[i].Favorite && !data.Countries[j].Favorite {
			return true
		}
		if !data.Countries[i].Favorite && data.Countries[j].Favorite {
			return false
		}

		// Обычная сортировка по коду (алфавитная)
		return data.Countries[i].Code < data.Countries[j].Code
	})

	// Создаем массив кнопок для постраничной навигации
	var buttons []utils.PaginatedKeyboardButton
	for _, country := range data.Countries {
		buttonText := fmt.Sprintf("%s %s", country.Emoji, country.Code)
		buttonData := fmt.Sprintf("country:%s", country.Code)

		buttons = append(buttons, utils.PaginatedKeyboardButton{
			Text:         buttonText,
			CallbackData: buttonData,
		})
	}

	// Параметры пагинации
	const rowSize = 5   // Кнопок в строке
	const pageSize = 50 // Кнопок на странице

	// Создаем пагинированную клавиатуру с префиксом "country" для навигации
	return utils.CreatePaginatedKeyboard(buttons, page, rowSize, pageSize, "country")
}

func (b *Bot) createBackBtnKeyboard(language string) *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: b.service.GetText("btn_back", language), CallbackData: CallbackSettingsBack},
			},
		},
	}
}

func (b *Bot) createStartInlineKeyboard(language string) *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{
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
}
