package repository

import (
	"roulette/internal/models"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Repository інтерфейс для роботи з базою даних
type Repository interface {
	// Пользователи
	CreateUser(user *models.User) error
	GetUserByTelegramID(telegramID int64) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
	UpdateUser(user *models.User) error
	GetUserCount() (int64, error)
	GetUsers(page, perPage int) ([]models.User, int64, error)
	GetUserWithdrawals(userID uint, limit int) ([]models.Withdrawal, error)
	SearchUsers(query string, page, perPage int) ([]models.User, int64, error)
	UpdateUserActivity(userID uint) error

	// Статистика
	GetUserTotalBets(userID uint) (int, error)
	GetUserWonBets(userID uint) (int, error)
	GetUserTotalPoints(userID uint) (int, error)
	GetUserDailyBets(userID uint) (int, error)
	GetUserDailyStats(userID uint) (int, int, error)
	GetUserWeeklyStats(userID uint) (int, int, error)
	GetUserMonthlyStats(userID uint) (int, int, error)
	GetDetailedStatsByDate(userID uint, startDate string, endDate string) (map[string]int, error)
	GetTotalStats() (map[string]int64, error)
	GetSuccessRateStats() (map[string]float64, error)
	GetTopPlayersBySuccessRate(limit int) ([]map[string]interface{}, error)
	GetTopPlayersByAttempts(limit int) ([]map[string]interface{}, error)

	// Источники
	GetSource() ([]map[string]interface{}, error)
	GetSourceByDate(dateFrom, dateTo string) ([]map[string]interface{}, error)
	SetSourceKey(key string, name string) error
	CheckSourceKeyExists(key string) (bool, error)
	GetAllSourceKeys() ([]models.SourceKey, error)

	// Игры и стаки
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
	// UpdateSuperRating(rating *models.SuperRating) error
	FixPartiallyDistributedPrizes(year, week int, action string) error
	DeleteRating(userID uint) error
	UpdateWeeklyRatingForUser(userID uint) error
	UpdateWeeklyRatingForUsers(userIDs []uint, year, week int) error
	GetUserRankAndNeighbors(userID uint, year, week int, neighborsCount int) ([]models.WeeklyRating, int, error)
	GetPointsToReachPrizeZone(year, week, topCount int) (int, error)
	RefreshWeeklyRatingsPosition(year, week int) error
	CheckIfPrizesAlreadyDistributed(year, week int) (bool, error)
	GetPrizeFundWithoutCreation(year, week int) (*models.PrizeFund, error)
	GetRecentPrizeFunds(limit int) ([]models.PrizeFund, error)
	// CreatePrizeFund(fund *models.PrizeFund) error
	CancelPrizeDistribution(year, week int) error

	// Методы для работы с настройками
	GetSetting(key string) (*models.Setting, error)
	// UpdateSetting(key string, value string) error
	GetAllSettings() (map[string]*models.Setting, error)
	CreateOrUpdateSetting(key, value, defaultValue, description string) error
	// GetSettingWithDefault(key string) (*models.Setting, error)

	// Локализации
	GetLocalization(key string, language string) (models.Localization, error)
	SetLocalization(value models.Localization) error
	GetAllLocalizationsForLanguage(language string) ([]models.Localization, error)
	GetAllLocalizationsByKey(key string) ([]models.Localization, error) // ДОБАВЛЕН НЕДОСТАЮЩИЙ МЕТОД
	DeleteLocalization(key string) error
	GetLocalizationCount(language string) (int64, error)
	CheckLocalizationExists(key string) (bool, error)
	ImportLocalizations(localizations []models.Localization) error

	// Призовые фонды
	GetPrizeFund(year, week int) (*models.PrizeFund, error)
	UpdatePrizeFund(fund *models.PrizeFund) error

	// Уведомления
	CreateNotification(notification *models.Notification) error
	GetUserNotifications(userID uint, limit int) ([]models.Notification, error)
	MarkNotificationAsRead(id uint) error
	GetPendingNotificationTasks() ([]models.NotificationTask, error)
	MarkNotificationAsSent(id uint) error

	// Вывод средств
	CreateWithdrawal(withdrawal *models.Withdrawal) error
	GetPendingWithdrawals() ([]models.Withdrawal, error)
	GetProcessingWithdrawals() ([]models.Withdrawal, error)
	GetWithdrawalsHistory(limit int) ([]models.Withdrawal, error)
	UpdateWithdrawalStatus(id uint, status string) error
	GetWithdrawalByProviderID(providerName, providerID string) (*models.Withdrawal, error)
	UpdateWithdrawal(withdrawal *models.Withdrawal) error
	// Статистика по виплатам
	GetWithdrawalsStat(dateFrom, dateTo string) ([]models.WithdrawalStat, error)
	FindWithdrawalsStat(day string) (models.WithdrawalStat, error)
	RecalculateWithdrawalsStat(day string) (models.WithdrawalStat, error)
	UpdateWithdrawalsStat(data *models.WithdrawalStat) error

	// Админ-функции
	GetWithdrawalByID(id uint) (*models.Withdrawal, error)

	// Хеши и раунды
	SaveHashEntry(entry *models.HashEntry) error
	GetHashEntries(offset, limit int) ([]models.HashEntry, error)
	CountHashEntries() (int64, error)
	GetCurrentHashEntry() (*models.HashEntry, error)
	CreateHashEntry(entry *models.HashEntry) error
	CompleteHashEntry(id uint, revealedAt time.Time) error
	GetHashEntryByID(id uint) (*models.HashEntry, error)
	// GetBetsByHashEntryID(hashEntryID uint) ([]models.Bet, error)
	GetActiveHashEntry() (*models.HashEntry, error)
	GetUserBetsForHashEntry(userID, hashEntryID uint) ([]models.Bet, error)

	// Работа со страной пользователя
	// SetUserCountry(userID uint, country string) error
	GetUserCountry(userID uint) (string, error)

	GetBetsByHashEntryIDWithUsers(hashEntryID uint) ([]models.Bet, error)

	GetCurrentYearWeek() (int, int) // Возвращает текущий год и неделю

	// Уведомления
	CreateNotificationRecipientsBatch(recipients []models.NotificationRecipient) error
	CreateNotificationTask(task *models.NotificationTask) error
	CreateNotificationTemplate(template *models.NotificationTemplate) error
	DeleteNotificationTask(id uint) error
	CreateNotificationRecipient(recipient *models.NotificationRecipient) error
	DeleteNotificationTemplate(id uint) error
	// GetActivityFiltersWithUserCounts() ([]models.ActivityFilterOption, error)
	GetCountriesWithUserCounts() ([]models.CountryOption, error)
	GetNotificationRecipients(taskID uint, status string, page, pageSize int) ([]models.NotificationRecipient, int64, error)
	GetNotificationTaskByID(id uint) (*models.NotificationTask, error)
	GetNotificationTasks(status string, page, pageSize int) ([]models.NotificationTask, int64, error)
	GetNotificationTasksStats(period string) (*models.NotificationStatistics, error)
	GetNotificationTemplateByID(id uint) (*models.NotificationTemplate, error)
	GetNotificationTemplates(templateType string, page, pageSize int) ([]models.NotificationTemplate, int64, error)
	GetTemplateWithLocalizations(templateID uint) (*models.NotificationTemplateWithLocalizations, error)
	GetUsersForNotificationTask(task *models.NotificationTask) ([]models.User, error)
	UpdateNotificationRecipient(recipient *models.NotificationRecipient) error
	UpdateNotificationTask(task *models.NotificationTask) error
	UpdateNotificationTemplate(template *models.NotificationTemplate) error
	UpdateTaskProgress(taskID uint, sentCount, deliveredCount, readCount int) error
	CheckNotificationSent(userID uint, notificationType string, date string) (bool, error)
	SaveNotificationSent(userID uint, notificationType string, date string) error

	// Activity Analyzer methods
	GetActivityDashboardStats() (*models.ActivityDashboardStats, error)
	GetTopActivityUsers(limit int, timeFrom, timeTo *time.Time, minActions int) ([]models.UserActivityStats, error)
	GetUserActivityStats(telegramID int64, timeFrom, timeTo *time.Time) (*models.UserActivityStats, error)
	GetUserActivityLogsForAnalyzer(telegramID int64, limit int, actionType string) ([]models.UserActivityLog, error)
	GetUserActivityTimeline(telegramID int64, interval string, timeFrom, timeTo *time.Time, actionType string) ([]models.ActionTimeSeries, error)
	GetUserActionDistribution(telegramID int64, timeFrom, timeTo *time.Time) ([]models.ActionTypeDistribution, error)
	GetUserActivityIntervals(telegramID int64, limit int) ([]models.IntervalStats, error)
	GetAllActionTypes() ([]string, error)
	GetOverallActivityTimeline(interval string, timeFrom, timeTo *time.Time, limit int) ([]models.ActionTimeSeries, error)
	GetTopActionTypes(limit int) ([]models.ActionTypeDistribution, error)
	DeleteOldActivityLogs(olderThan time.Time) (int64, error)

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

// Реалізація методів для рейтингів

func (r *PostgresRepository) GetWeeklyRating(year, week int, limit int) ([]models.WeeklyRating, error) {
	var ratings []models.WeeklyRating
	query := r.db.Where("year = ? AND week = ?", year, week).
		Order("position ASC").
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
			UserID:   userID,
			Year:     year,
			Week:     week,
			Position: r.getLastWeeklyRatingPosition() + 1, // Устанавливаем последнюю позицию,
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

// unused
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
		FROM bets b INNER JOIN users u 
		ON b.user_id = u.id
		WHERE u.banned IS NOT TRUE AND DATE_PART('week', b.created_at) = ? AND DATE_PART('year', b.created_at) = ?
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
	return r.RefreshWeeklyRatingsPosition(year, week)
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

// unused
func (r *PostgresRepository) UpdateSuperRating(rating *models.SuperRating) error {
	return r.db.Save(rating).Error
}

// Реалізація методів для призових фондів

func (r *PostgresRepository) GetPrizeFund(year, week int) (*models.PrizeFund, error) {
	var fund models.PrizeFund
	err := r.db.Where("year = ? AND week = ?", year, week).First(&fund).Error

	// Якщо фонду немає, створюємо новий на основе настроек
	if err == gorm.ErrRecordNotFound {
		// Получаем настройки для призового фонда
		prizeAmountSetting, err := r.GetSetting("weekly_prize_amount")
		if err != nil {
			// Если настройка не найдена, используем значение по умолчанию
			prizeAmountSetting = &models.Setting{Value: "1000"}
		}

		topCountSetting, err := r.GetSetting("weekly_prize_top")
		if err != nil {
			// Если настройка не найдена, используем значение по умолчанию
			topCountSetting = &models.Setting{Value: "100"}
		}

		// Преобразуем значения в нужные типы
		prizeAmount, err := strconv.ParseFloat(prizeAmountSetting.Value, 64)
		if err != nil {
			prizeAmount = 1000 // Значение по умолчанию
		}

		topCount, err := strconv.Atoi(topCountSetting.Value)
		if err != nil {
			topCount = 100 // Значение по умолчанию
		}

		// Создаем новый фонд с полученными настройками
		fund = models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   prizeAmount,
			TopCount: topCount,
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

func (r *PostgresRepository) GetCurrentYearWeek() (int, int) {
	return time.Now().ISOWeek()
}

// Close закриває з'єднання з базою даних
func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
