package repository

import (
	"time"

	"roulette/internal/models"

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
	CreateGame(game *models.Game) error
	GetLastGame() (*models.Game, error)
	CreateBet(bet *models.Bet) error
	GetUserBets(userID uint, limit int) ([]models.Bet, error)
	GetUserBetsCount(userID uint) (int, error)

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

	Close() error
}

type PostgresRepository struct {
	db *gorm.DB
}

// NewRepository створює новий екземпляр репозиторію
func NewRepository(db *gorm.DB) Repository {
	return &PostgresRepository{db: db}
}

// Реалізація методів для користувачів

func (r *PostgresRepository) CreateUser(user *models.User) error {
	tx := r.db.Begin()

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Створюємо статистику для користувача
	stats := &models.UserStats{
		UserID:    user.ID,
		LastReset: time.Now(),
	}

	if err := tx.Create(stats).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *PostgresRepository) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User
	err := r.db.Where("telegram_id = ?", telegramID).First(&user).Error

	// Перевіряємо, чи помилка є "record not found"
	if err == gorm.ErrRecordNotFound {
		// Повертаємо помилку, але не викликаємо panic
		return nil, err
	} else if err != nil {
		// Інша помилка
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *PostgresRepository) GetUserCount() (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) GetUserStats(userID uint) (*models.UserStats, error) {
	var stats models.UserStats
	err := r.db.Where("user_id = ?", userID).First(&stats).Error

	// Якщо статистики немає, створюємо нову
	if err == gorm.ErrRecordNotFound {
		stats = models.UserStats{
			UserID:    userID,
			LastReset: time.Now(),
		}
		if err := r.db.Create(&stats).Error; err != nil {
			return nil, err
		}
		return &stats, nil
	} else if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *PostgresRepository) UpdateUserStats(stats *models.UserStats) error {
	return r.db.Save(stats).Error
}

func (r *PostgresRepository) ResetDailyBets() error {
	return r.db.Model(&models.User{}).Update("today_bets", 0).Error
}

// Реалізація методів для ігор і ставок

func (r *PostgresRepository) CreateGame(game *models.Game) error {
	return r.db.Create(game).Error
}

func (r *PostgresRepository) GetLastGame() (*models.Game, error) {
	var game models.Game
	err := r.db.Order("created_at desc").First(&game).Error

	// Якщо ігор ще немає, повертаємо nil без помилки
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &game, nil
}

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

// Реалізація методів для налаштувань

func (r *PostgresRepository) GetSetting(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *PostgresRepository) UpdateSetting(key string, value string) error {
	return r.db.Model(&models.Setting{}).Where("key = ?", key).
		Update("value", value).Error
}

// Реалізація методів для локалізацій

func (r *PostgresRepository) GetLocalization(key string, language string) (string, error) {
	var loc models.Localization
	err := r.db.Where("key = ? AND language = ?", key, language).First(&loc).Error
	if err != nil {
		// Якщо локалізація не знайдена для вказаної мови, спробуємо знайти англійську
		if language != "en" {
			return r.GetLocalization(key, "en")
		}
		return "", err
	}
	return loc.Value, nil
}

func (r *PostgresRepository) SetLocalization(key string, language string, value string) error {
	var loc models.Localization
	err := r.db.Where("key = ? AND language = ?", key, language).First(&loc).Error

	if err == gorm.ErrRecordNotFound {
		// Створюємо нову локалізацію
		loc = models.Localization{
			Key:      key,
			Language: language,
			Value:    value,
		}
		return r.db.Create(&loc).Error
	} else if err != nil {
		return err
	}

	// Оновлюємо існуючу локалізацію
	loc.Value = value
	return r.db.Save(&loc).Error
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

// Реалізація методів для виведення коштів

func (r *PostgresRepository) CreateWithdrawal(withdrawal *models.Withdrawal) error {
	return r.db.Create(withdrawal).Error
}

func (r *PostgresRepository) GetPendingWithdrawals() ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	if err := r.db.Where("status = ?", "pending").
		Preload("User").
		Find(&withdrawals).Error; err != nil {
		return nil, err
	}
	return withdrawals, nil
}

func (r *PostgresRepository) UpdateWithdrawalStatus(id uint, status string) error {
	return r.db.Model(&models.Withdrawal{}).Where("id = ?", id).
		Update("status", status).Error
}

// GetUsers отримує список користувачів з пагінацією
func (r *PostgresRepository) GetUsers(page, perPage int) ([]models.User, int64, error) {
	var users []models.User
	var totalCount int64

	// Отримуємо загальну кількість користувачів
	if err := r.db.Model(&models.User{}).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Отримуємо користувачів з пагінацією
	offset := (page - 1) * perPage
	if err := r.db.Offset(offset).Limit(perPage).Order("created_at desc").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}

// GetUserByID отримує користувача за його ID
func (r *PostgresRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetWithdrawalByID отримує запит на виведення коштів за його ID
func (r *PostgresRepository) GetWithdrawalByID(id uint) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	if err := r.db.First(&withdrawal, id).Error; err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

// SaveHashEntry зберігає запис хешу в базу даних
func (r *PostgresRepository) SaveHashEntry(entry *models.HashEntry) error {
	return r.db.Create(entry).Error
}

// GetHashEntries отримує записи хешів з бази даних з пагінацією
func (r *PostgresRepository) GetHashEntries(offset, limit int) ([]models.HashEntry, error) {
	var entries []models.HashEntry
	err := r.db.Order("id desc").Offset(offset).Limit(limit).Find(&entries).Error
	return entries, err
}

// CountHashEntries підраховує загальну кількість записів хешів
func (r *PostgresRepository) CountHashEntries() (int64, error) {
	var count int64
	err := r.db.Model(&models.HashEntry{}).Count(&count).Error
	return count, err
}

// Close закриває з'єднання з базою даних
func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
