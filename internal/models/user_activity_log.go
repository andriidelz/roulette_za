package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// UserActivityLog represents a log entry for user actions
type UserActivityLog struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	TelegramID int64      `gorm:"not null;index:idx_activity_logs_telegram_id;index:idx_activity_logs_telegram_created" json:"telegram_id"`
	ChatID     int64      `gorm:"not null" json:"chat_id"`
	ActionType string     `gorm:"type:varchar(100);not null;index:idx_activity_logs_action_type" json:"action_type"`
	ActionData ActionData `gorm:"type:jsonb" json:"action_data,omitempty"`
	MessageID  *int64     `gorm:"" json:"message_id,omitempty"`
	CreatedAt  time.Time  `gorm:"not null;default:now();index:idx_activity_logs_created_at,priority:1;index:idx_activity_logs_telegram_created,priority:2;index:idx_activity_logs_cleanup,priority:1" json:"created_at"`
}

// ActionData is a custom type for JSONB field
type ActionData map[string]interface{}

// Value implements the driver.Valuer interface for ActionData
func (a ActionData) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface for ActionData
func (a *ActionData) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(ActionData)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*a = result
	return nil
}

// TableName specifies the table name for UserActivityLog
func (UserActivityLog) TableName() string {
	return "user_activity_logs"
}

// Action type constants
const (
	// Commands
	ActionCommandStart    = "command_start"
	ActionCommandBalance  = "command_balance"
	ActionCommandRating   = "command_rating"
	ActionCommandHistory  = "command_history"
	ActionCommandHelp     = "command_help"
	ActionCommandSettings = "command_settings"
	ActionCommandLanguage = "command_language"
	ActionCommandWithdraw = "command_withdraw"

	// Callbacks (generic, details in action_data)
	ActionCallbackBet     = "callback_bet"     // bet_red_123 -> action_data: {action: "red", param: "123"}
	ActionCallbackGet     = "callback_get"     // get_result_9 -> action_data: {action: "result", param: "9"}
	ActionCallbackRating  = "callback_rating"  // rating_weekly -> action_data: {action: "weekly"}
	ActionCallbackCaptcha = "callback_captcha" // captcha_verify_abc -> action_data: {action: "verify", param: "abc"}
	ActionCallback        = "callback"         // generic callback

	// Messages
	ActionMessageText     = "message_text"
	ActionMessageSticker  = "message_sticker"
	ActionMessageVideo    = "message_video"
	ActionMessagePhoto    = "message_photo"
	ActionMessageVoice    = "message_voice"
	ActionMessageDocument = "message_document"
	ActionMessageLocation = "message_location"
	ActionMessageEdited   = "message_edited"

	// Other
	ActionInlineQuery = "inline_query"
	ActionUnknown     = "unknown"
)
