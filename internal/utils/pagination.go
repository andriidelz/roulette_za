package utils

import (
	"fmt"
	"math"

	"github.com/mymmrac/telego"
)

// PaginatedKeyboardButton представляет кнопку с данными для постраничного отображения
type PaginatedKeyboardButton struct {
	Text         string // Текст кнопки
	CallbackData string // Callback-данные
}

// CreatePaginatedKeyboard создает клавиатуру с разбивкой по страницам
func CreatePaginatedKeyboard(
	buttons []PaginatedKeyboardButton, // Все кнопки
	page int, // Текущая страница (начиная с 1)
	rowSize int, // Количество кнопок в строке
	pageSize int, // Количество кнопок на странице
	prefix string, // Префикс для callback data кнопок навигации
) *telego.InlineKeyboardMarkup {
	totalButtons := len(buttons)

	if totalButtons == 0 {
		return &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{},
		}
	}

	// Вычисляем общее количество страниц
	totalPages := int(math.Ceil(float64(totalButtons) / float64(pageSize)))

	// Проверяем, что текущая страница в допустимом диапазоне
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Вычисляем начальный и конечный индексы для текущей страницы
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > totalButtons {
		endIdx = totalButtons
	}

	// Получаем кнопки для текущей страницы
	currentPageButtons := buttons[startIdx:endIdx]

	// Создаем клавиатуру
	var keyboard [][]telego.InlineKeyboardButton

	// Добавляем кнопки текущей страницы
	currentRow := []telego.InlineKeyboardButton{}
	for i, button := range currentPageButtons {
		currentRow = append(currentRow, telego.InlineKeyboardButton{
			Text:         button.Text,
			CallbackData: button.CallbackData,
		})

		// Если достигли конца строки или это последняя кнопка, добавляем строку в клавиатуру
		if (i+1)%rowSize == 0 || i == len(currentPageButtons)-1 {
			keyboard = append(keyboard, currentRow)
			currentRow = []telego.InlineKeyboardButton{}
		}
	}

	// Добавляем навигационные кнопки
	navRow := []telego.InlineKeyboardButton{}

	// Кнопка "Назад" (если не на первой странице)
	if page > 1 {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "◀️",
			CallbackData: fmt.Sprintf("%s_page:%d", prefix, page-1),
		})
	}

	// Индикатор текущей страницы
	navRow = append(navRow, telego.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", page, totalPages),
		CallbackData: "noop", // Кнопка без действия
	})

	// Кнопка "Вперед" (если не на последней странице)
	if page < totalPages {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "▶️",
			CallbackData: fmt.Sprintf("%s_page:%d", prefix, page+1),
		})
	}

	// Добавляем навигационные кнопки в клавиатуру
	if totalPages > 1 {
		keyboard = append(keyboard, navRow)
	}

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}
