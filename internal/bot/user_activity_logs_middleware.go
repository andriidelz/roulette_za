package bot

import (
	"fmt"
	"strings"

	"roulette/internal/logger"
	"roulette/internal/models"
	"roulette/internal/repository"

	"github.com/mymmrac/telego"
)

// ActivityLoggerMiddleware is a proper middleware that intercepts all updates
type ActivityLoggerMiddleware struct {
	repo *repository.PostgresRepository
	next UpdateHandler
}

// UpdateHandler is a function that processes telego updates
type UpdateHandler func(update telego.Update)

// NewActivityLoggerMiddleware creates a new activity logger middleware
func NewActivityLoggerMiddleware(repo *repository.PostgresRepository, next UpdateHandler) UpdateHandler {
	middleware := &ActivityLoggerMiddleware{
		repo: repo,
		next: next,
	}
	return middleware.Handle
}

// Handle intercepts the update, logs it, and passes to next handler
func (m *ActivityLoggerMiddleware) Handle(update telego.Update) {
	// Log activity asynchronously (non-blocking)
	go m.logActivity(update)

	// Pass to next handler (business logic)
	m.next(update)
}

// logActivity extracts info from update and saves to database
func (m *ActivityLoggerMiddleware) logActivity(update telego.Update) {
	// Extract telegram_id
	telegramID := m.extractTelegramID(update)
	if telegramID == 0 {
		// Cannot log without telegram_id
		return
	}

	// Build activity log entry
	log := m.buildActivityLog(update, telegramID)
	if log == nil {
		return
	}

	// Save to database
	if err := m.repo.CreateActivityLog(log); err != nil {
		logger.Error.Printf("Failed to create activity log: %v", err)
	}
}

// extractTelegramID gets telegram_id from update
func (m *ActivityLoggerMiddleware) extractTelegramID(update telego.Update) int64 {
	// Extract telegram_id from update
	if update.Message != nil {
		return update.Message.From.ID
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID
	} else if update.EditedMessage != nil {
		return update.EditedMessage.From.ID
	} else if update.InlineQuery != nil {
		return update.InlineQuery.From.ID
	}

	return 0
}

// buildActivityLog creates activity log entry from update
func (m *ActivityLoggerMiddleware) buildActivityLog(update telego.Update, telegramID int64) *models.UserActivityLog {
	// Callback query (inline buttons)
	if update.CallbackQuery != nil {
		return m.buildCallbackLog(update.CallbackQuery, telegramID)
	}

	// Edited message
	if update.EditedMessage != nil {
		return m.buildEditedMessageLog(update.EditedMessage, telegramID)
	}

	// Regular message
	if update.Message != nil {
		return m.buildMessageLog(update.Message, telegramID)
	}

	// Inline query
	if update.InlineQuery != nil {
		return m.buildInlineQueryLog(update.InlineQuery, telegramID)
	}

	return nil
}

// buildCallbackLog creates log entry for callback query
func (m *ActivityLoggerMiddleware) buildCallbackLog(callback *telego.CallbackQuery, telegramID int64) *models.UserActivityLog {
	data := models.ActionData{
		"callback_data": callback.Data,
		"callback_id":   callback.ID,
	}

	// Determine action type with strict rules for specific patterns
	actionType := m.parseCallbackActionType(callback.Data)

	messageID := int64(callback.Message.GetMessageID())
	chatID := callback.Message.GetChat().ID

	return &models.UserActivityLog{
		TelegramID: telegramID,
		ChatID:     chatID,
		ActionType: actionType,
		ActionData: data,
		MessageID:  &messageID,
	}
}

// parseCallbackActionType determines action_type from callback_data with strict rules
func (m *ActivityLoggerMiddleware) parseCallbackActionType(callbackData string) string {
	if callbackData == "" {
		return "callback"
	}

	parts := strings.Split(callbackData, "_")
	if len(parts) == 0 {
		return "callback"
	}

	// Strict rules for specific patterns
	switch {
	// bet_red, bet_black, bet_zero -> callback_bet_red, callback_bet_black, callback_bet_zero
	case parts[0] == "bet" && len(parts) >= 2:
		color := parts[1]
		return fmt.Sprintf("callback_bet_%s", color)

	// get_result_* -> callback_get_result
	case parts[0] == "get" && len(parts) >= 2 && parts[1] == "result":
		return "callback_get_result"

	// captcha_* -> callback_captcha
	case parts[0] == "captcha":
		return "callback_captcha"

	// Everything else -> callback_{full_callback_data}
	default:
		return fmt.Sprintf("callback_%s", callbackData)
	}
}

// buildMessageLog creates log entry for regular message
func (m *ActivityLoggerMiddleware) buildMessageLog(message *telego.Message, telegramID int64) *models.UserActivityLog {
	messageID := int64(message.MessageID)
	chatID := message.Chat.ID

	// Command
	if message.Text != "" && strings.HasPrefix(message.Text, "/") {
		return m.buildCommandLog(message, telegramID, chatID, messageID)
	}

	// Sticker
	if message.Sticker != nil {
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessageSticker,
			ActionData: models.ActionData{
				"file_id":  message.Sticker.FileID,
				"emoji":    message.Sticker.Emoji,
				"set_name": message.Sticker.SetName,
			},
			MessageID: &messageID,
		}
	}

	// Video
	if message.Video != nil {
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessageVideo,
			ActionData: models.ActionData{
				"file_id":   message.Video.FileID,
				"duration":  message.Video.Duration,
				"width":     message.Video.Width,
				"height":    message.Video.Height,
				"file_size": message.Video.FileSize,
			},
			MessageID: &messageID,
		}
	}

	// Photo
	if message.Photo != nil && len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessagePhoto,
			ActionData: models.ActionData{
				"file_id":   photo.FileID,
				"width":     photo.Width,
				"height":    photo.Height,
				"file_size": photo.FileSize,
			},
			MessageID: &messageID,
		}
	}

	// Voice
	if message.Voice != nil {
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessageVoice,
			ActionData: models.ActionData{
				"file_id":   message.Voice.FileID,
				"duration":  message.Voice.Duration,
				"file_size": message.Voice.FileSize,
			},
			MessageID: &messageID,
		}
	}

	// Document
	if message.Document != nil {
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessageDocument,
			ActionData: models.ActionData{
				"file_id":   message.Document.FileID,
				"file_name": message.Document.FileName,
				"mime_type": message.Document.MimeType,
				"file_size": message.Document.FileSize,
			},
			MessageID: &messageID,
		}
	}

	// Location
	if message.Location != nil {
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessageLocation,
			ActionData: models.ActionData{
				"latitude":  message.Location.Latitude,
				"longitude": message.Location.Longitude,
			},
			MessageID: &messageID,
		}
	}

	// Text message
	if message.Text != "" {
		return &models.UserActivityLog{
			TelegramID: telegramID,
			ChatID:     chatID,
			ActionType: models.ActionMessageText,
			ActionData: models.ActionData{
				"text": message.Text,
			},
			MessageID: &messageID,
		}
	}

	// Unknown message type
	return &models.UserActivityLog{
		TelegramID: telegramID,
		ChatID:     chatID,
		ActionType: models.ActionUnknown,
		ActionData: models.ActionData{
			"message_type": "unknown",
		},
		MessageID: &messageID,
	}
}

// buildCommandLog creates log entry for command
func (m *ActivityLoggerMiddleware) buildCommandLog(message *telego.Message, telegramID, chatID, messageID int64) *models.UserActivityLog {
	text := message.Text
	if !strings.HasPrefix(text, "/") {
		return nil
	}

	parts := strings.SplitN(text, " ", 2)
	command := strings.TrimPrefix(parts[0], "/")

	// Remove @botname if present
	if idx := strings.Index(command, "@"); idx != -1 {
		command = command[:idx]
	}

	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	actionType := fmt.Sprintf("command_%s", command)
	data := models.ActionData{
		"command": fmt.Sprintf("/%s", command),
	}

	if args != "" {
		data["args"] = args
		data["full_text"] = text

		// Special handling for /start command with ref key
		if command == "start" && args != "" {
			data["ref_key"] = args
		}
	}

	return &models.UserActivityLog{
		TelegramID: telegramID,
		ChatID:     chatID,
		ActionType: actionType,
		ActionData: data,
		MessageID:  &messageID,
	}
}

// buildEditedMessageLog creates log entry for edited message
func (m *ActivityLoggerMiddleware) buildEditedMessageLog(message *telego.Message, telegramID int64) *models.UserActivityLog {
	messageID := int64(message.MessageID)
	chatID := message.Chat.ID

	data := models.ActionData{
		"edit_date": message.EditDate,
	}

	if message.Text != "" {
		data["new_text"] = message.Text
	}

	return &models.UserActivityLog{
		TelegramID: telegramID,
		ChatID:     chatID,
		ActionType: models.ActionMessageEdited,
		ActionData: data,
		MessageID:  &messageID,
	}
}

// buildInlineQueryLog creates log entry for inline query
func (m *ActivityLoggerMiddleware) buildInlineQueryLog(query *telego.InlineQuery, telegramID int64) *models.UserActivityLog {
	return &models.UserActivityLog{
		TelegramID: telegramID,
		ChatID:     query.From.ID,
		ActionType: models.ActionInlineQuery,
		ActionData: models.ActionData{
			"query_id": query.ID,
			"query":    query.Query,
			"offset":   query.Offset,
		},
	}
}
