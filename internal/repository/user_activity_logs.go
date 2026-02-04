package repository

import (
	"time"

	"roulette/internal/config"
	"roulette/internal/logger"
	"roulette/internal/models"
)

// CreateActivityLog creates a new activity log entry
func (r *PostgresRepository) CreateActivityLog(log *models.UserActivityLog) error {
	return r.db.Create(log).Error
}

// GetUserActivityLogs retrieves activity logs for a specific telegram user with optional filters
func (r *PostgresRepository) GetUserActivityLogs(telegramID int64, limit, offset int, actionType string, from, to time.Time) ([]models.UserActivityLog, error) {
	var logs []models.UserActivityLog

	query := r.db.Where("telegram_id = ?", telegramID)

	// Filter by action type if provided
	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	// Filter by time range
	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at <= ?", to)
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, err
}

// GetActivityLogsByTimeRange retrieves all activity logs within a time range
func (r *PostgresRepository) GetActivityLogsByTimeRange(from, to time.Time, limit, offset int) ([]models.UserActivityLog, error) {
	var logs []models.UserActivityLog

	query := r.db.Where("created_at >= ? AND created_at <= ?", from, to)

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, err
}

// CountUserActivityLogs counts activity logs for a specific telegram user with optional filters
func (r *PostgresRepository) CountUserActivityLogs(telegramID int64, actionType string, from, to time.Time) (int64, error) {
	var count int64

	query := r.db.Model(&models.UserActivityLog{}).Where("telegram_id = ?", telegramID)

	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at <= ?", to)
	}

	err := query.Count(&count).Error
	return count, err
}

// GetUserLastActivity retrieves the last activity log entry for a telegram user
func (r *PostgresRepository) GetUserLastActivity(telegramID int64) (*models.UserActivityLog, error) {
	var log models.UserActivityLog
	err := r.db.Where("telegram_id = ?", telegramID).
		Order("created_at DESC").
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// DeleteOldActivityLogs deletes activity logs older than the specified date
func (r *PostgresRepository) DeleteOldActivityLogs(olderThan time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", olderThan).Delete(&models.UserActivityLog{})
	return result.RowsAffected, result.Error
}

// GetActivityLogStats retrieves statistics about user activity by telegram_id
func (r *PostgresRepository) GetActivityLogStats(telegramID int64, from, to time.Time) (map[string]int64, error) {
	type ActionCount struct {
		ActionType string `json:"action_type"`
		Count      int64  `json:"count"`
	}

	var results []ActionCount
	stats := make(map[string]int64)

	query := r.db.Model(&models.UserActivityLog{}).
		Select("action_type, COUNT(*) as count").
		Where("telegram_id = ?", telegramID).
		Group("action_type")

	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at <= ?", to)
	}

	err := query.Find(&results).Error
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		stats[result.ActionType] = result.Count
	}

	return stats, nil
}

// GetUsersActivityCount retrieves activity counts for multiple telegram users
func (r *PostgresRepository) GetUsersActivityCount(telegramIDs []int64, from, to time.Time) (map[int64]int64, error) {
	type UserCount struct {
		TelegramID int64 `json:"telegram_id"`
		Count      int64 `json:"count"`
	}

	var results []UserCount
	counts := make(map[int64]int64)

	query := r.db.Model(&models.UserActivityLog{}).
		Select("telegram_id, COUNT(*) as count").
		Where("telegram_id IN ?", telegramIDs).
		Group("telegram_id")

	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at <= ?", to)
	}

	err := query.Find(&results).Error
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		counts[result.TelegramID] = result.Count
	}

	return counts, nil
}

// CreateBanLog створення нового запису про бан
func (r *PostgresRepository) CreateBanLog(log *models.UserBanLog) error {
	return r.db.Create(log).Error
}

// UpdateBet оновлення інформації про бан
func (r *PostgresRepository) UpdateBanLog(log *models.UserBanLog) error {
	return r.db.Save(log).Error
}

// GetActiveBanLog отримання активного бану по користувачу (якщо такий є)
func (r *PostgresRepository) GetActiveBanLog(userID uint) (models.UserBanLog, error) {
	var log models.UserBanLog
	err := r.db.Where("user_id = ? AND active = ?", userID, true).Order("id DESC").First(&log).Error
	if err != nil {
		return models.UserBanLog{}, err
	}
	return log, nil
}

// GetUserBanLogs історія блокувань користувача
func (r *PostgresRepository) GetUserBanLogs(userID uint, limit int) ([]models.UserBanLog, error) {
	var banLogsHistory []models.UserBanLog
	query := r.db.Where("user_id = ?", userID).Order("created_at desc")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&banLogsHistory).Error; err != nil {
		return nil, err
	}

	return banLogsHistory, nil
}

// UserUnban переведення всіх забанених користувачів час який підійшов в статус active
func (r *PostgresRepository) UserUnban() error {

	// Actions last hour
	current := time.Now()
	var userBan []uint

	if err := r.db.Model(&models.UserBanLog{}).Where("active = ? AND until_to >= ? AND until_to <= ?",
		true, current.Add(-3*time.Hour), current).Pluck("user_id", &userBan).Error; err != nil {
		return err
	}

	if len(userBan) == 0 {
		return nil
	}
	logger.Info.Println(userBan)

	if err := r.db.Model(&models.UserBanLog{}).Where("active = ? AND until_to >= ? AND until_to <= ?",
		true, current.Add(-3*time.Hour), current).UpdateColumn("active", false).Error; err != nil {
		return err
	}

	for i := range userBan {
		res, _ := r.GetActiveBanLog(userBan[i])
		if res.ID == 0 {
			if err := r.db.Model(&models.User{}).Where("id = ?", userBan[i]).UpdateColumn("status", config.UserStatusActive).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
