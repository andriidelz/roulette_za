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

	// Статистика (нові методи замість UserStats)
	GetUserTotalBets(userID uint) (int, error)
	GetUserWonBets(userID uint) (int, error)
	GetUserTotalPoints(userID uint) (int, error)
	GetUserDailyBets(userID uint) (int, error)
	GetUserDailyStats(userID uint) (int, int, error)
	GetUserWeeklyStats(userID uint) (int, int, error)
	GetUserMonthlyStats(userID uint) (int, int, error)
	GetDetailedStatsByDate(userID uint, startDate string, endDate string) (map[string]int, error)

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

	// Адмін функції
	GetUsers(page, perPage int) ([]models.User, int64, error)
	GetUserByID(id uint) (*models.User, error)
	GetWithdrawalByID(id uint) (*models.Withdrawal, error)

	// Хеші та раунди
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

	// Закрытие соединения
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
	return r.db.Create(bet).Error
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
	// Выполняем прямой SQL запрос для расчета рейтингов на основе ставок
	query := `
		INSERT INTO weekly_ratings (user_id, week, year, points, bets, efficiency, position, created_at, updated_at)
		SELECT 
			b.user_id,
			?,                         -- week
			?,                         -- year
			COALESCE(SUM(CASE WHEN b.won THEN b.points ELSE 0 END), 0) as points,
			COUNT(*) as bets,
			CASE WHEN COUNT(*) > 0 THEN COALESCE(SUM(CASE WHEN b.won THEN b.points ELSE 0 END), 0)::float / COUNT(*) ELSE 0 END, -- efficiency
			0,                         -- position (будет обновлено позже)
			NOW(),                     -- created_at
			NOW()                      -- updated_at
		FROM bets b
		WHERE DATE_PART('week', b.created_at) = ? AND DATE_PART('year', b.created_at) = ?
		GROUP BY b.user_id
		ON CONFLICT (user_id, week, year) 
		DO UPDATE SET
			points = EXCLUDED.points,
			bets = EXCLUDED.bets,
			efficiency = EXCLUDED.efficiency,
			updated_at = NOW()
	`

	if err := r.db.Exec(query, week, year, week, year).Error; err != nil {
		return err
	}

	// Обновляем позиции в рейтинге
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

// Методы для настроек (перенесенные из settings.go)
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

// Методы статистики (новые)
func (r *PostgresRepository) GetUserTotalBets(userID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.Bet{}).Where("user_id = ?", userID).Count(&count).Error
	return int(count), err
}

func (r *PostgresRepository) GetUserWonBets(userID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.Bet{}).Where("user_id = ? AND won = ?", userID, true).Count(&count).Error
	return int(count), err
}

func (r *PostgresRepository) GetUserTotalPoints(userID uint) (int, error) {
	var total int
	err := r.db.Model(&models.Bet{}).Where("user_id = ? AND won = ?", userID, true).Select("COALESCE(SUM(points), 0)").Scan(&total).Error
	return total, err
}

func (r *PostgresRepository) GetUserDailyBets(userID uint) (int, error) {
	var count int64
	today := time.Now().Format("2006-01-02")
	err := r.db.Model(&models.Bet{}).Where("user_id = ? AND DATE(created_at) = ?", userID, today).Count(&count).Error
	return int(count), err
}

func (r *PostgresRepository) GetUserDailyStats(userID uint) (int, int, error) {
	var count int64
	var points int

	today := time.Now().Format("2006-01-02")

	// Количество ставок за день
	err := r.db.Model(&models.Bet{}).Where("user_id = ? AND DATE(created_at) = ?", userID, today).Count(&count).Error
	if err != nil {
		return 0, 0, err
	}

	// Количество баллов за день
	err = r.db.Model(&models.Bet{}).Where("user_id = ? AND won = ? AND DATE(created_at) = ?", userID, true, today).Select("COALESCE(SUM(points), 0)").Scan(&points).Error

	return int(count), points, err
}

func (r *PostgresRepository) GetUserWeeklyStats(userID uint) (int, int, error) {
	var count int64
	var points int

	// Начало недели (понедельник)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // Воскресенье в Go имеет номер 0
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1).Format("2006-01-02")

	// Количество ставок за неделю
	err := r.db.Model(&models.Bet{}).Where("user_id = ? AND DATE(created_at) >= ?", userID, startOfWeek).Count(&count).Error
	if err != nil {
		return 0, 0, err
	}

	// Количество баллов за неделю
	err = r.db.Model(&models.Bet{}).Where("user_id = ? AND won = ? AND DATE(created_at) >= ?", userID, true, startOfWeek).Select("COALESCE(SUM(points), 0)").Scan(&points).Error

	return int(count), points, err
}

func (r *PostgresRepository) GetUserMonthlyStats(userID uint) (int, int, error) {
	var count int64
	var points int

	// Начало месяца
	startOfMonth := time.Now().Format("2006-01") + "-01"

	// Количество ставок за месяц
	err := r.db.Model(&models.Bet{}).Where("user_id = ? AND DATE(created_at) >= ?", userID, startOfMonth).Count(&count).Error
	if err != nil {
		return 0, 0, err
	}

	// Количество баллов за месяц
	err = r.db.Model(&models.Bet{}).Where("user_id = ? AND won = ? AND DATE(created_at) >= ?", userID, true, startOfMonth).Select("COALESCE(SUM(points), 0)").Scan(&points).Error

	return int(count), points, err
}

// Close закриває з'єднання з базою даних
func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
