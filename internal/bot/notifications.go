package bot

import (
	"encoding/json"
	"fmt"
	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/messaging"
	"roulette/internal/service"
	"strings"

	"github.com/mymmrac/telego"
)

// setupNotificationHandler настраивает обработчик уведомлений
func (b *Bot) setupNotificationHandler() error {
	// Создаем подключение к RabbitMQ
	rmq, err := messaging.NewRabbitMQ(b.getRabbitMQURL(), "roulette_events", "bot")
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ for notifications: %w", err)
	}

	// Подписываемся на очередь уведомлений
	err = rmq.SubscribeToQueue("user_notifications", []string{service.RoutingUserNotification}, b.handleNotificationMessage)
	if err != nil {
		return fmt.Errorf("failed to subscribe to notifications queue: %w", err)
	}

	logger.Info.Println("Notification handler initialized")
	return nil
}

// handleNotificationMessage обрабатывает сообщения с уведомлениями
func (b *Bot) handleNotificationMessage(message messaging.RouletteMessage) error {
	// Логируем полное сообщение для отладки
	logger.Info.Printf("Notification message data type: %T", message.Data)

	// Если данные в виде строки, выводим первые 100 символов для отладки
	if str, ok := message.Data.(string); ok {
		if len(str) > 100 {
			logger.Info.Printf("Message data preview: %s...", str[:100])
		} else {
			logger.Info.Printf("Message data full: %s", str)
		}
	}

	// Приводим данные к типу NotificationData
	var notificationData service.NotificationData
	notifyType := "notify"

	// Проверяем тип данных и обрабатываем соответствующим образом
	switch data := message.Data.(type) {
	case map[string]interface{}:
		// Данные уже в формате map, преобразуем их в структуру
		if telegramID, ok := data["telegram_id"].(float64); ok {
			notificationData.TelegramID = int64(telegramID)
		} else {
			return fmt.Errorf("invalid telegram_id format in map data")
		}

		if userID, ok := data["user_id"].(float64); ok {
			notificationData.UserID = uint(userID)
		}

		if title, ok := data["type"].(string); ok {
			notifyType = title
		}

		if title, ok := data["title"].(string); ok {
			notificationData.Title = title
		}

		if msg, ok := data["message"].(string); ok {
			notificationData.Message = msg
		}

		if imageURL, ok := data["image_url"].(string); ok {
			notificationData.ImageURL = imageURL
		}

		if buttonText, ok := data["button_text"].(string); ok {
			notificationData.ButtonText = buttonText
		}

		if buttonURL, ok := data["button_url"].(string); ok {
			notificationData.ButtonURL = buttonURL
		}

		if buttonCallback, ok := data["button_callback"].(string); ok {
			notificationData.ButtonCallback = buttonCallback
		}

		if notificationID, ok := data["notification_id"].(float64); ok {
			notificationData.NotificationID = uint(notificationID)
		}

	case []byte:
		// Логируем данные для отладки
		logger.Info.Printf("Message data as bytes: %s", string(data))

		// Если данные в байтовом формате, десериализуем их
		if err := json.Unmarshal(data, &notificationData); err != nil {
			return fmt.Errorf("error unmarshaling byte notification data: %w", err)
		}

	case string:
		// Проверим, не содержит ли строка префикс или не является ли она необработанным JSON-представлением
		cleanData := strings.TrimSpace(data)

		// Если строка начинается с '{', считаем ее JSON
		if strings.HasPrefix(cleanData, "{") {
			if err := json.Unmarshal([]byte(cleanData), &notificationData); err != nil {
				return fmt.Errorf("error unmarshaling JSON string notification data: %w (%s)", err, cleanData)
			}
		} else {
			// Возможно, данные закодированы или содержат служебные символы
			// Пробуем отправить сообщение напрямую без десериализации
			logger.Info.Printf("Sending notification directly with raw data: %s", cleanData)

			// Отправляем сообщение с информацией об ошибке
			b.SendMessage(77039720, MessageOptions{ // Замените на реальный ID администратора
				Text: fmt.Sprintf("⚠️ Получено неформатированное уведомление:\n\n%s", cleanData),
			})

			return fmt.Errorf("received unformatted notification data: %s", cleanData)
		}

	default:
		return fmt.Errorf("unsupported data type: %T", message.Data)
	}

	// Проверяем наличие TelegramID (необходимо для отправки)
	if notificationData.TelegramID == 0 {
		return fmt.Errorf("missing telegram_id in notification data")
	}

	// оновлюєм кеш
	b.updateUserCache(notificationData.TelegramID)

	// Логируем получение уведомления
	logger.Info.Printf("Processing notification for user %d: %s", notificationData.TelegramID, notificationData.Title)

	// Создаем inline клавиатуру, если есть кнопка
	var inlineKeyboard *telego.InlineKeyboardMarkup
	if notificationData.ButtonText != "" {
		button := telego.InlineKeyboardButton{
			Text: notificationData.ButtonText,
		}

		// Устанавливаем URL или callback данные
		if notificationData.ButtonURL != "" {
			button.URL = notificationData.ButtonURL
		} else if notificationData.ButtonCallback != "" {
			button.CallbackData = notificationData.ButtonCallback
		}

		inlineKeyboard = &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{button},
			},
		}
	}

	// Отправляем уведомление пользователю
	if notificationData.ImageURL != "" {
		// Отправляем сообщение с изображением
		err := b.SendMessage(notificationData.TelegramID, MessageOptions{
			Text:           fmt.Sprintf("<b>%s</b>\n\n%s", notificationData.Title, notificationData.Message),
			PhotoFileID:    notificationData.ImageURL, // или PhotoPath, если это локальный путь
			InlineKeyboard: inlineKeyboard,
			ParseMode:      "HTML",
			Type:           notifyType,
		})
		if err != nil {
			return fmt.Errorf("error sending notification with image: %w", err)
		}
	} else {
		// Отправляем текстовое сообщение
		err := b.SendMessage(notificationData.TelegramID, MessageOptions{
			Text:           fmt.Sprintf("<b>%s</b>\n\n%s", notificationData.Title, notificationData.Message),
			InlineKeyboard: inlineKeyboard,
			ParseMode:      "HTML",
			Type:           notifyType,
		})
		if err != nil {
			return fmt.Errorf("error sending notification: %w", err)
		}
	}

	logger.Info.Printf("Successfully sent notification to user %d", notificationData.TelegramID)
	return nil
}

// getRabbitMQURL возвращает URL для подключения к RabbitMQ
func (b *Bot) getRabbitMQURL() string {
	// Получаем URL из конфигурации
	cfg := config.NewConfig()
	return cfg.RabbitMQURL
}
