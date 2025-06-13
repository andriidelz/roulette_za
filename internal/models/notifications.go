package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NotificationTemplate представляет шаблон для уведомлений
type NotificationTemplate struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	Type           string    `gorm:"size:50;not null" json:"type"`    // 'manual', 'automatic'
	TriggerEvent   string    `gorm:"size:50" json:"trigger_event"`    // Событие для автоматических уведомлений
	TitleKey       string    `gorm:"size:255" json:"title_key"`       // Ключ локализации для заголовка
	MessageKey     string    `gorm:"size:255" json:"message_key"`     // Ключ локализации для сообщения
	ImageURLs      string    `gorm:"type:jsonb" json:"-"`             // JSON строка с URL изображений для разных языков
	ButtonTextKey  string    `gorm:"size:255" json:"button_text_key"` // Ключ локализации для текста кнопки
	ButtonURL      string    `json:"button_url"`                      // URL для кнопки
	ButtonCallback string    `json:"button_callback"`                 // Callback для кнопки
	Active         bool      `gorm:"default:true" json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NotificationTargetParams представляет параметры таргетинга для уведомлений
type NotificationTargetParams struct {
	Countries       []string               `json:"countries,omitempty"`        // Список стран для таргетинга
	ActivityFilters []string               `json:"activity_filters,omitempty"` // Фильтры по активности (последняя игра)
	UserIDs         []uint                 `json:"user_ids,omitempty"`         // Конкретные ID пользователей
	TimeZone        string                 `json:"time_zone,omitempty"`        // Часовой пояс для отправки
	SendTimeStart   string                 `json:"send_time_start,omitempty"`  // Начало периода отправки (по местному времени)
	SendTimeEnd     string                 `json:"send_time_end,omitempty"`    // Конец периода отправки (по местному времени)
	Macros          map[string]interface{} `json:"macros,omitempty"`           // Макросы для замены в тексте
}

// Value реализует интерфейс driver.Valuer для сохранения в БД
func (p NotificationTargetParams) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan реализует интерфейс sql.Scanner для чтения из БД
func (p *NotificationTargetParams) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &p)
}

// NotificationTask представляет задачу на отправку уведомлений
type NotificationTask struct {
	ID             uint                     `gorm:"primaryKey" json:"id"`
	TemplateID     uint                     `json:"template_id"`
	Template       NotificationTemplate     `gorm:"foreignKey:TemplateID" json:"template"`
	Status         string                   `gorm:"size:50;not null;default:'pending'" json:"status"` // 'pending', 'processing', 'completed', 'failed'
	TargetType     string                   `gorm:"size:50;not null" json:"target_type"`              // 'all', 'country', 'activity', 'custom'
	TargetParams   NotificationTargetParams `gorm:"type:jsonb" json:"target_params"`                  // Параметры таргетинга в JSON формате
	ScheduledAt    *time.Time               `json:"scheduled_at"`                                     // Время запланированной отправки
	StartedAt      *time.Time               `json:"started_at"`                                       // Время начала отправки
	CompletedAt    *time.Time               `json:"completed_at"`                                     // Время завершения отправки
	TotalUsers     int                      `gorm:"default:0" json:"total_users"`                     // Общее количество пользователей для отправки
	SentCount      int                      `gorm:"default:0" json:"sent_count"`                      // Количество отправленных уведомлений
	DeliveredCount int                      `gorm:"default:0" json:"delivered_count"`                 // Количество доставленных уведомлений
	ReadCount      int                      `gorm:"default:0" json:"read_count"`                      // Количество прочитанных уведомлений
	Recipients     []NotificationRecipient  `gorm:"foreignKey:TaskID" json:"recipients,omitempty"`    // Получатели уведомления
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

// NotificationRecipient представляет получателя уведомления
type NotificationRecipient struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	TaskID       uint       `gorm:"not null" json:"task_id"`
	UserID       uint       `gorm:"not null" json:"user_id"`
	User         User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Status       string     `gorm:"size:50;not null;default:'pending'" json:"status"` // 'pending', 'sent', 'delivered', 'read', 'failed'
	ScheduledAt  *time.Time `json:"scheduled_at"`                                     // Время запланированной отправки для конкретного пользователя
	SentAt       *time.Time `json:"sent_at"`                                          // Время отправки
	DeliveredAt  *time.Time `json:"delivered_at"`                                     // Время доставки
	ReadAt       *time.Time `json:"read_at"`                                          // Время прочтения
	ErrorMessage string     `json:"error_message"`                                    // Сообщение об ошибке
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NotificationStatistics представляет статистику уведомлений
type NotificationStatistics struct {
	Period         string         `json:"period"`          // 'day', 'week', 'month'
	TotalSent      int            `json:"total_sent"`      // Всего отправлено
	TotalDelivered int            `json:"total_delivered"` // Всего доставлено
	TotalRead      int            `json:"total_read"`      // Всего прочитано
	CountryStats   []CountryStats `json:"country_stats"`   // Статистика по странам
}

// CountryStats представляет статистику по странам
type CountryStats struct {
	Country     string `json:"country"`      // Код страны
	CountryName string `json:"country_name"` // Название страны
	Sent        int    `json:"sent"`         // Отправлено
	Delivered   int    `json:"delivered"`    // Доставлено
	Read        int    `json:"read"`         // Прочитано
}

// NotificationTemplateWithLocalizations расширяет шаблон локализованными данными
type NotificationTemplateWithLocalizations struct {
	NotificationTemplate
	TitleLocalizations      map[string]string `json:"title_localizations"`       // Локализации заголовка
	MessageLocalizations    map[string]string `json:"message_localizations"`     // Локализации сообщения
	ButtonTextLocalizations map[string]string `json:"button_text_localizations"` // Локализации текста кнопки
}

// EnhancedNotificationTask расширяет задачу дополнительной информацией
type EnhancedNotificationTask struct {
	NotificationTask
	TemplateWithLocalizations NotificationTemplateWithLocalizations `json:"template_with_localizations"`
	TaskProgress              float64                               `json:"task_progress"`            // Прогресс выполнения задачи (0-100%)
	EstimatedTimeRemaining    string                                `json:"estimated_time_remaining"` // Оставшееся время в формате "2h 15m"
}

// ActivityFilterOption представляет опцию фильтра по активности
type ActivityFilterOption struct {
	ID    string `json:"id"`    // Идентификатор фильтра
	Label string `json:"label"` // Отображаемый текст
	Count int    `json:"count"` // Количество пользователей, соответствующих фильтру
}

// CountryOption представляет опцию выбора страны
type CountryOption struct {
	Code  string `json:"code"`  // Код страны
	Emoji string `json:"emoji"` // Эмодзи флага
	Name  string `json:"name"`  // Название страны
	Count int    `json:"count"` // Количество пользователей из этой страны
}

// GetImage возвращает map с URL изображений для разных языков
func (t *NotificationTemplate) GetImage() map[string]string {
	if t.ImageURLs == "" {
		return map[string]string{}
	}

	var imageURLs map[string]string
	if err := json.Unmarshal([]byte(t.ImageURLs), &imageURLs); err != nil {
		// В случае ошибки возвращаем пустую map
		return map[string]string{}
	}

	return imageURLs
}

// SetImage устанавливает map с URL изображений для разных языков
func (t *NotificationTemplate) SetImage(imageURLs map[string]string) error {
	data, err := json.Marshal(imageURLs)
	if err != nil {
		return err
	}

	t.ImageURLs = string(data)
	return nil
}

// MarshalJSON переопределяет метод для правильной сериализации
func (t *NotificationTemplateWithLocalizations) MarshalJSON() ([]byte, error) {
	type Alias NotificationTemplateWithLocalizations

	return json.Marshal(&struct {
		*Alias
		Image map[string]string `json:"image"`
	}{
		Alias: (*Alias)(t),
		Image: t.GetImage(),
	})
}

func (t *NotificationTemplateWithLocalizations) GetImage() map[string]string {
	return t.NotificationTemplate.GetImage()
}
