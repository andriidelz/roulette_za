package repository

import (
	"roulette/internal/models"
	"time"

	"gorm.io/gorm"
)

// Repository інтерфейс для роботи з базою даних
type Repository interface {
	// Користувачі
	CreateUser(user *models.User) error
	GetUserByTelegramID(telegramID int64) (*models.User, error)
	UpdateUser(user *models.User) error
	GetUserCount() (int64, error)
	GetUserStats(userID uint) (*models.UserStats, error)
	UpdateUserStats(stats *models.UserStats) error
	ResetDailyBets() error

	// Ігри і ставки
	CreateBet(bet *models.Bet) error
	GetUserBets(userID uint, limit int) ([]models.Bet, error)
	GetUserBetsCount(userID uint) (int, error)
	UpdateBet(bet *models.Bet) error

	// Рейтинги
	GetWeeklyRating(year, week int, limit int) ([]models.WeeklyRating, error)
	GetUserWeeklyRating(userID uint, year, week int) (*models.WeeklyRating, error)
	UpdateWeeklyRating(rating *models.WeeklyRating) error
	CalculateWeeklyRatings(year, week int) error
	GetSuperRating(period string, limit int) ([]models.SuperRating, error)
	UpdateSuperRating(rating *models.SuperRating) error

	// Налаштування
	GetSetting(key string) (*models.Setting, error)
	UpdateSetting(key string, value string) error

	// Локалізації
	GetLocalization(key string, language string) (string, error)
	SetLocalization(key string, language string, value string) error
	GetAllLocalizationsForLanguage(language string) ([]models.Localization, error)
	DeleteLocalization(key string) error
	GetLocalizationCount(language string) (int64, error)
	CheckLocalizationExists(key string) (bool, error)
	ImportLocalizations(localizations []models.Localization) error

	// Призові фонди
	GetPrizeFund(year, week int) (*models.PrizeFund, error)
	UpdatePrizeFund(fund *models.PrizeFund) error

	// Сповіщення
	CreateNotification(notification *models.Notification) error
	GetUserNotifications(userID uint, limit int) ([]models.Notification, error)
	MarkNotificationAsRead(id uint) error

	// Виведення коштів
	CreateWithdrawal(withdrawal *models.Withdrawal) error
	GetPendingWithdrawals() ([]models.Withdrawal, error)
	UpdateWithdrawalStatus(id uint, status string) error

	GetUsers(page, perPage int) ([]models.User, int64, error)
	GetUserByID(id uint) (*models.User, error)
	GetWithdrawalByID(id uint) (*models.Withdrawal, error)

	SaveHashEntry(entry *models.HashEntry) error
	GetHashEntries(offset, limit int) ([]models.HashEntry, error)
	CountHashEntries() (int64, error)
	GetCurrentHashEntry() (*models.HashEntry, error)
	CreateHashEntry(entry *models.HashEntry) error
	CompleteHashEntry(id uint, revealedAt time.Time) error
	GetHashEntryByID(id uint) (*models.HashEntry, error)
	GetBetsByHashEntryID(hashEntryID uint) ([]models.Bet, error)
	GetActiveHashEntry() (*models.HashEntry, error)
	GetUserBetsForHashEntry(userID, hashEntryID uint) ([]models.Bet, error)

	// Работа со страной пользователя
	SetUserCountry(userID uint, country string) error
	GetUserCountry(userID uint) (string, error)

	Close() error
}

type PostgresRepository struct {
	db *gorm.DB
}

// NewRepository створює новий екземпляр репозиторію
func NewRepository(db *gorm.DB) Repository {
	return &PostgresRepository{db: db}
}

// Реалізація методів для ігор і ставок

func (r *PostgresRepository) CreateBet(bet *models.Bet) error {
	tx := r.db.Begin()

	// Створюємо ставку
	if err := tx.Create(bet).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Оновлюємо статистику користувача
	var stats models.UserStats
	if err := tx.Where("user_id = ?", bet.UserID).First(&stats).Error; err != nil {
		tx.Rollback()
		return err
	}

	stats.TotalBets++
	stats.DailyBets++
	stats.WeeklyBets++
	stats.MonthlyBets++

	if bet.Won {
		stats.WonBets++
		stats.TotalPoints += bet.Points
		stats.DailyPoints += bet.Points
		stats.WeeklyPoints += bet.Points
		stats.MonthlyPoints += bet.Points
	}

	if err := tx.Save(&stats).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Збільшуємо лічильник ставок за день у користувача
	if err := tx.Model(&models.User{}).Where("id = ?", bet.UserID).
		Update("today_bets", gorm.Expr("today_bets + ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *PostgresRepository) GetUserBets(userID uint, limit int) ([]models.Bet, error) {
	var bets []models.Bet
	query := r.db.Where("user_id = ?", userID).Order("created_at desc")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&bets).Error; err != nil {
		return nil, err
	}

	return bets, nil
}

func (r *PostgresRepository) GetUserBetsCount(userID uint) (int, error) {
	var count int64
	if err := r.db.Model(&models.Bet{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// Реалізація методів для рейтингів

func (r *PostgresRepository) GetWeeklyRating(year, week int, limit int) ([]models.WeeklyRating, error) {
	var ratings []models.WeeklyRating
	query := r.db.Where("year = ? AND week = ?", year, week).
		Order("points desc, efficiency desc").
		Preload("User")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&ratings).Error; err != nil {
		return nil, err
	}

	return ratings, nil
}

func (r *PostgresRepository) GetUserWeeklyRating(userID uint, year, week int) (*models.WeeklyRating, error) {
	var rating models.WeeklyRating
	err := r.db.Where("user_id = ? AND year = ? AND week = ?", userID, year, week).First(&rating).Error

	// Якщо рейтингу немає, створюємо новий
	if err == gorm.ErrRecordNotFound {
		rating = models.WeeklyRating{
			UserID: userID,
			Year:   year,
			Week:   week,
		}
		if err := r.db.Create(&rating).Error; err != nil {
			return nil, err
		}
		return &rating, nil
	} else if err != nil {
		return nil, err
	}

	return &rating, nil
}

func (r *PostgresRepository) UpdateWeeklyRating(rating *models.WeeklyRating) error {
	return r.db.Save(rating).Error
}

func (r *PostgresRepository) CalculateWeeklyRatings(year, week int) error {
	// Виконуємо складний запит для розрахунку рейтингу за вказаний тиждень
	query := `
		INSERT INTO weekly_ratings (user_id, week, year, points, bets, efficiency, position, created_at, updated_at)
		SELECT 
			us.user_id,
			?,                         -- week
			?,                         -- year
			us.weekly_points,          -- points
			us.weekly_bets,            -- bets
			CASE WHEN us.weekly_bets > 0 THEN us.weekly_points::float / us.weekly_bets ELSE 0 END, -- efficiency
			0,                         -- position (буде оновлено пізніше)
			NOW(),                     -- created_at
			NOW()                      -- updated_at
		FROM user_stats us
		WHERE us.weekly_points > 0
		ON CONFLICT (user_id, week, year) 
		DO UPDATE SET
			points = EXCLUDED.points,
			bets = EXCLUDED.bets,
			efficiency = EXCLUDED.efficiency,
			updated_at = NOW()
	`

	if err := r.db.Exec(query, week, year).Error; err != nil {
		return err
	}

	// Оновлюємо позиції в рейтингу
	positionQuery := `
		WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY points DESC, efficiency DESC) AS new_position
			FROM weekly_ratings
			WHERE week = ? AND year = ?
		)
		UPDATE weekly_ratings wr
		SET position = r.new_position
		FROM ranked r
		WHERE wr.id = r.id AND wr.week = ? AND wr.year = ?
	`

	return r.db.Exec(positionQuery, week, year, week, year).Error
}

func (r *PostgresRepository) GetSuperRating(period string, limit int) ([]models.SuperRating, error) {
	var ratings []models.SuperRating
	query := r.db.Where("period = ?", period).
		Order("points desc, positions desc").
		Preload("User")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&ratings).Error; err != nil {
		return nil, err
	}

	return ratings, nil
}

func (r *PostgresRepository) UpdateSuperRating(rating *models.SuperRating) error {
	return r.db.Save(rating).Error
}

// Реалізація методів для призових фондів

func (r *PostgresRepository) GetPrizeFund(year, week int) (*models.PrizeFund, error) {
	var fund models.PrizeFund
	err := r.db.Where("year = ? AND week = ?", year, week).First(&fund).Error

	// Якщо фонду немає, створюємо новий
	if err == gorm.ErrRecordNotFound {
		fund = models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   1000,
			TopCount: 100,
		}
		if err := r.db.Create(&fund).Error; err != nil {
			return nil, err
		}
		return &fund, nil
	} else if err != nil {
		return nil, err
	}

	return &fund, nil
}

func (r *PostgresRepository) UpdatePrizeFund(fund *models.PrizeFund) error {
	return r.db.Save(fund).Error
}

// Реалізація методів для сповіщень

func (r *PostgresRepository) CreateNotification(notification *models.Notification) error {
	return r.db.Create(notification).Error
}

func (r *PostgresRepository) GetUserNotifications(userID uint, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.Where("user_id = ?", userID).Order("created_at desc")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&notifications).Error; err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *PostgresRepository) MarkNotificationAsRead(id uint) error {
	return r.db.Model(&models.Notification{}).Where("id = ?", id).
		Update("read", true).Error
}

// Реализация методов для работы со страной пользователя
func (r *PostgresRepository) SetUserCountry(userID uint, country string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("country", country).Error
}

func (r *PostgresRepository) GetUserCountry(userID uint) (string, error) {
	var user models.User
	err := r.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return "", err
	}
	return user.Country, nil
}

// Close закриває з'єднання з базою даних
func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
