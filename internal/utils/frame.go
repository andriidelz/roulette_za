package utils

import (
	"fmt"
	"strings"
)

// FrameStyle визначає стиль рамки
type FrameStyle struct {
	TopLeft      string
	TopRight     string
	BottomLeft   string
	BottomRight  string
	Horizontal   string
	Vertical     string
	LeftPadding  int
	RightPadding int
}

// KeyValue представляє пару ключ-значення для впорядкованого виведення
type KeyValue struct {
	Key   string
	Value string
}

// DefaultFrameStyle повертає стандартний стиль рамки з одинарними лініями
func DefaultFrameStyle() FrameStyle {
	return FrameStyle{
		TopLeft:      "┌",
		TopRight:     "┐",
		BottomLeft:   "└",
		BottomRight:  "┘",
		Horizontal:   "─",
		Vertical:     "│",
		LeftPadding:  1,
		RightPadding: 1,
	}
}

// DoubleBorderFrameStyle повертає стиль рамки з подвійними лініями
func DoubleBorderFrameStyle() FrameStyle {
	return FrameStyle{
		TopLeft:      "╔",
		TopRight:     "╗",
		BottomLeft:   "╚",
		BottomRight:  "╝",
		Horizontal:   "═",
		Vertical:     "║",
		LeftPadding:  1,
		RightPadding: 1,
	}
}

// RoundedFrameStyle повертає стиль рамки з заокругленими кутами
func RoundedFrameStyle() FrameStyle {
	return FrameStyle{
		TopLeft:      "╭",
		TopRight:     "╮",
		BottomLeft:   "╰",
		BottomRight:  "╯",
		Horizontal:   "─",
		Vertical:     "│",
		LeftPadding:  1,
		RightPadding: 1,
	}
}

// CreateTextFrame створює рамку навколо тексту
func CreateTextFrame(lines []string, style FrameStyle) string {
	if len(lines) == 0 {
		return ""
	}

	// Знаходимо найдовший рядок
	maxLength := 0
	for _, line := range lines {
		if len(line) > maxLength {
			maxLength = len(line)
		}
	}

	// Визначаємо загальну ширину рамки
	totalWidth := maxLength + style.LeftPadding + style.RightPadding

	// Створюємо горизонтальну лінію
	horizontalLine := strings.Repeat(style.Horizontal, totalWidth)

	// Формуємо результуючий рядок
	var result strings.Builder

	// Верхня межа
	result.WriteString(style.TopLeft)
	result.WriteString(horizontalLine)
	result.WriteString(style.TopRight)
	result.WriteString("\n")

	// Вміст рамки
	for _, line := range lines {
		result.WriteString(style.Vertical)
		result.WriteString(strings.Repeat(" ", style.LeftPadding))
		result.WriteString(line)

		// Додаємо пробіли справа для вирівнювання
		padding := totalWidth - len(line) - style.LeftPadding
		result.WriteString(strings.Repeat(" ", padding))

		result.WriteString(style.Vertical)
		result.WriteString("\n")
	}

	// Нижня межа
	result.WriteString(style.BottomLeft)
	result.WriteString(horizontalLine)
	result.WriteString(style.BottomRight)

	return result.String()
}

// PrintTextFrame створює рамку навколо тексту та виводить її в stdout
func PrintTextFrame(lines []string, style FrameStyle) {
	fmt.Println(CreateTextFrame(lines, style))
}

// OrderedTextInFrame генерує рамку навколо впорядкованого набору ключів та їх значень
func OrderedTextInFrame(pairs []KeyValue, style FrameStyle) string {
	var lines []string
	for _, pair := range pairs {
		lines = append(lines, fmt.Sprintf("%s: %s", pair.Key, pair.Value))
	}
	return CreateTextFrame(lines, style)
}

// PrintOrderedTextInFrame виводить рамку з впорядкованим набором ключів та їх значень
func PrintOrderedTextInFrame(pairs []KeyValue, style FrameStyle) {
	fmt.Println(OrderedTextInFrame(pairs, style))
}

// Для зворотної сумісності
// TextInFrame генерує рамку навколо набору ключів та їх значень (не гарантує порядок)
func TextInFrame(data map[string]string, style FrameStyle) string {
	var lines []string
	for key, value := range data {
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}
	return CreateTextFrame(lines, style)
}

// PrintTextInFrame виводить рамку з набором ключів та їх значень (не гарантує порядок)
func PrintTextInFrame(data map[string]string, style FrameStyle) {
	fmt.Println(TextInFrame(data, style))
}
