package bot

import (
	"fmt"
	"roulette/internal/logger"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
)

const (
	CommandMyID          = "myid"      // Команда для просмотра своих ID
	CommandEmulateID     = "emulateid" // Команда для эмуляции другого пользователя
	CommandStopEmulateID = "stopemulateid"
)

var emulatedUsers = make(map[int64]int64)

// handleMyIDCommand обрабатывает команду /myid
func (b *Bot) handleMyIDCommand(message *telego.Message) {
	user := message.From
	originalUserID := user.ID

	// Получаем информацию о пользователе из базы данных
	dbUser, err := b.getUser(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user: %v", err)
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "Error getting user information.",
		})
		return
	}

	// Формируем сообщение с информацией о пользователе
	idInfoText := fmt.Sprintf("📋 Your identifiers:\n\nUser ID (DB): %d\nTelegram ID: %d",
		dbUser.ID, dbUser.TelegramID)

	// Проверяем, эмулирует ли пользователь кого-то
	if emulatedID, ok := emulatedUsers[originalUserID]; ok {
		// Получаем информацию об эмулируемом пользователе
		emulatedUser, emulErr := b.getUser(emulatedID)
		if emulErr == nil {
			idInfoText += fmt.Sprintf("\n\n⚠️ WARNING: Emulation is active!\n"+
				"You are emulating user:\n"+
				"User ID (DB): %d\n"+
				"Telegram ID: %d\n"+
				"👤 Name: %s\n\n"+
				"To stop emulation, use /stopemulateid",
				emulatedUser.ID, emulatedUser.TelegramID, emulatedUser.FirstName)
		} else {
			idInfoText += fmt.Sprintf("\n\n⚠️ WARNING: Emulation is active!\n"+
				"You are emulating user with Telegram ID: %d\n"+
				"(Error getting user data)\n\n"+
				"To stop emulation, use /stopemulateid",
				emulatedID)
		}
	} else {
		idInfoText += "\n\n✅ No emulation active"
	}

	// Отправляем сообщение
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text: idInfoText,
	})
}

// handleEmulateIDCommand обрабатывает команду /emulateid
func (b *Bot) handleEmulateIDCommand(message *telego.Message) {
	user := message.From

	// Парсим команду и аргументы
	// В Telegram команды с параметрами приходят в формате "/command param1 param2"
	commandText := message.Text
	parts := strings.SplitN(commandText, " ", 2)

	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		// Отправляем инструкцию по использованию
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "⚠️ You must specify a Telegram ID.\n\nUsage: /emulateid <telegram_id>",
		})
		return
	}

	// Получаем ID из аргумента
	targetIDStr := strings.TrimSpace(parts[1])
	targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil {
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "⚠️ Invalid ID format. Please use a numeric identifier.",
		})
		return
	}

	// Пытаемся найти пользователя по Telegram ID
	targetUser, err := b.getUser(targetID)
	if err != nil {
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: fmt.Sprintf("⚠️ User with Telegram ID %d not found.", targetID),
		})
		return
	}

	// Записываем эмуляцию в глобальную карту
	emulatedUsers[user.ID] = targetID

	// Отправляем сообщение об успешной эмуляции
	successText := fmt.Sprintf("✅ You are now emulating user:\n\n"+
		"User ID: %d\n"+
		"Telegram ID: %d\n"+
		"👤 Name: %s\n\n"+
		"To stop emulation, use /stopemulateid",
		targetUser.ID, targetUser.TelegramID, targetUser.FirstName)

	b.SendMessage(message.Chat.ID, MessageOptions{
		Text: successText,
	})
}

// handleStopEmulateIDCommand обрабатывает команду /stopemulateid
func (b *Bot) handleStopEmulateIDCommand(message *telego.Message) {
	user := message.From

	// Проверяем, активна ли эмуляция для этого пользователя
	// Используем реальный ID пользователя, а не эмулируемый
	originalUserID := user.ID

	if _, exists := emulatedUsers[originalUserID]; !exists {
		b.SendMessage(message.Chat.ID, MessageOptions{
			Text: "⚠️ You don't have an active emulation.",
		})
		return
	}

	// Удаляем данные эмуляции
	delete(emulatedUsers, originalUserID)

	// Отправляем сообщение о прекращении эмуляции
	b.SendMessage(message.Chat.ID, MessageOptions{
		Text: "✅ Emulation stopped. You are now using your own account again.",
	})
}
