package models

import (
	"time"

	"gorm.io/gorm"
)

// Опція вибору у рулетці
type BetOption string

const (
	Red   BetOption = "red"
	Black BetOption = "black"
	Zero  BetOption = "zero"
)

// Пользователь
type User struct {
	ID             uint    `gorm:"primaryKey"`
	TelegramID     int64   `gorm:"uniqueIndex"`
	Username       string  `gorm:"size:255"`
	Nickname       string  `gorm:"size:50"`
	FirstName      string  `gorm:"size:255"`
	LastName       string  `gorm:"size:255"`
	Source         string  `gorm:"size:20"` // источник по реферальной ссылке
	RefKey         string  `gorm:"size:20"` // источник по реферальной ссылке
	LanguageCode   string  `gorm:"size:10"`
	Country        string  `gorm:"size:2"` // ISO 3166-1 alpha-2 код страны
	WalletAddress  string  `gorm:"size:255"`
	AvatarURL      string  `gorm:"size:512"`
	Balance        float64 `gorm:"default:0"`
	Banned         bool    `gorm:"default:false"`
	Registered     bool    `gorm:"default:false"` // true if user finish registration
	AgeVerified    *bool   `gorm:"default:null"`  // Указатель на bool для возможности хранения NULL значения
	LastActivityAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HashEntry представляє запис хешу (раунд) в базі даних
type HashEntry struct {
	gorm.Model
	Number      int64      // Випадкове число (0-36)
	SaltHEX     string     // Сіль у шістнадцятковому форматі
	Hash        string     // Хеш, обчислений на основі числа та солі
	IsCompleted bool       `gorm:"default:false"` // Завершен ли раунд
	RevealedAt  *time.Time // Время раскрытия результата
	Bets        []Bet      `gorm:"foreignKey:HashEntryID"` // Ставки в этом раунде
}

// Ставка користувача
type Bet struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"index"`
	User        User      `gorm:"foreignKey:UserID"`
	HashEntryID uint      `gorm:"index"` // ID раунда (hash_entry)
	HashEntry   HashEntry `gorm:"foreignKey:HashEntryID"`
	Option      BetOption `gorm:"size:10"`
	Won         bool      `gorm:"default:false"`
	GetResult   bool      `gorm:"default:false"` // Історія того чи отримував користувач результат своєї ставки
	Points      int       `gorm:"default:0"`     // Отримані бали
	CreatedAt   time.Time
}

// Статистика користувача
type UserStats struct {
	ID            uint `gorm:"primaryKey"`
	UserID        uint `gorm:"uniqueIndex"`
	User          User `gorm:"foreignKey:UserID"`
	TotalBets     int  `gorm:"default:0"`
	WonBets       int  `gorm:"default:0"`
	TotalPoints   int  `gorm:"default:0"`
	DailyBets     int  `gorm:"default:0"`
	WeeklyBets    int  `gorm:"default:0"`
	MonthlyBets   int  `gorm:"default:0"`
	DailyPoints   int  `gorm:"default:0"`
	WeeklyPoints  int  `gorm:"default:0"`
	MonthlyPoints int  `gorm:"default:0"`
	LastReset     time.Time
	UpdatedAt     time.Time
}

// Тижневий рейтинг
type WeeklyRating struct {
	ID         uint    `gorm:"primaryKey"`
	UserID     uint    `gorm:"index"`
	User       User    `gorm:"foreignKey:UserID"`
	Week       int     `gorm:"index"` // Номер тижня року
	Year       int     `gorm:"index"`
	Points     int     `gorm:"default:0"`
	Bets       int     `gorm:"default:0"`
	Efficiency float64 `gorm:"default:0"` // Points / Bets
	Position   int     `gorm:"default:0"` // Позиція в рейтингу
	Prize      float64 `gorm:"default:0"` // Виграш
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Супер рейтинг (на основі позицій в тижневих рейтингах)
type SuperRating struct {
	ID        uint    `gorm:"primaryKey"`
	UserID    uint    `gorm:"index"`
	User      User    `gorm:"foreignKey:UserID"`
	Period    string  `gorm:"size:20;index"` // Квартал або півріччя (напр. "2025-Q1", "2025-H1")
	Points    int     `gorm:"default:0"`     // Сума балів за входження в топ-100
	Positions int     `gorm:"default:0"`     // Кількість входжень у топ-100
	Position  int     `gorm:"default:0"`     // Позиція в супер-рейтингу
	Prize     float64 `gorm:"default:0"`     // Виграш
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Налаштування бота
type Setting struct {
	ID           uint   `gorm:"primaryKey"`
	Key          string `gorm:"size:255;uniqueIndex"`
	Value        string `gorm:"type:text"`
	DefaultValue string `gorm:"type:text"`
	Description  string `gorm:"type:text"`
}

// SettingInfo представляет информацию о настройке с типизированным значением
type SettingInfo struct {
	Key          string
	Value        string
	DefaultValue string
	Description  string
	Type         string // "int", "float", "string", "bool", "time", "day"
}

// Мовні локалізації
type Localization struct {
	ID       uint   `gorm:"primaryKey"`
	Key      string `gorm:"size:255;index"`
	Language string `gorm:"size:10;index"`
	Value    string `gorm:"type:text"`
	Image    string `gorm:"type:text"`
	Video    string `gorm:"type:text"`
}

// Источники
type SourceKey struct {
	ID        uint      `gorm:"primaryKey"`
	Key       string    `gorm:"size:255;index"`
	Name      string    `gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Призовий фонд за тиждень
type PrizeFund struct {
	ID        uint    `gorm:"primaryKey"`
	Week      int     `gorm:"index"` // Номер тижня року
	Year      int     `gorm:"index"`
	Amount    float64 `gorm:"default:0"`
	TopCount  int     `gorm:"default:100"` // Кількість призових місць
	Processed bool    `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Сповіщення користувача
type Notification struct {
	ID             uint       `gorm:"primaryKey"`
	UserID         uint       `gorm:"index"`
	User           User       `gorm:"foreignKey:UserID"`
	Type           string     `gorm:"size:50;index"`
	Message        string     `gorm:"type:text"`
	Title          string     `gorm:"type:text"`
	ImageURL       string     `gorm:"type:text;column:image_url"`
	ButtonText     string     `gorm:"type:text;column:button_text"`
	ButtonURL      string     `gorm:"type:text;column:button_url"`
	ButtonCallback string     `gorm:"type:text;column:button_callback"`
	Read           bool       `gorm:"default:false"`
	Delivered      bool       `gorm:"default:false"`
	ReadAt         *time.Time `gorm:"column:read_at"`
	CreatedAt      time.Time
}

// Запит на виведення коштів
type Withdrawal struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index"`
	User      User `gorm:"foreignKey:UserID"`
	Amount    float64
	Status    string `gorm:"size:20;index"` // pending, approved, rejected
	Wallet    string `gorm:"size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time

	ProviderName    string `gorm:"size:50;index"`
	ProviderID      string `gorm:"size:255;index"`
	TransactionHash string `gorm:"size:255"`
}
