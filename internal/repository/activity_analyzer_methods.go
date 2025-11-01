package repository

import (
	"fmt"
	"time"

	"roulette/internal/models"
)

// GetActivityDashboardStats retrieves overall activity statistics
func (r *PostgresRepository) GetActivityDashboardStats() (*models.ActivityDashboardStats, error) {
	stats := &models.ActivityDashboardStats{}

	// Total unique users
	if err := r.db.Model(&models.UserActivityLog{}).
		Distinct("telegram_id").
		Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	// Total actions
	if err := r.db.Model(&models.UserActivityLog{}).
		Count(&stats.TotalActions).Error; err != nil {
		return nil, err
	}

	// Actions last hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if err := r.db.Model(&models.UserActivityLog{}).
		Where("created_at >= ?", oneHourAgo).
		Count(&stats.ActionsLastHour).Error; err != nil {
		return nil, err
	}

	// Actions last 24h
	dayAgo := time.Now().Add(-24 * time.Hour)
	if err := r.db.Model(&models.UserActivityLog{}).
		Where("created_at >= ?", dayAgo).
		Count(&stats.ActionsLast24h).Error; err != nil {
		return nil, err
	}

	// Average actions per user
	if stats.TotalUsers > 0 {
		stats.AvgActionsPerUser = float64(stats.TotalActions) / float64(stats.TotalUsers)
	}

	// Top action type
	var topAction struct {
		ActionType string
		Count      int64
	}
	if err := r.db.Model(&models.UserActivityLog{}).
		Select("action_type, COUNT(*) as count").
		Group("action_type").
		Order("count DESC").
		Limit(1).
		Scan(&topAction).Error; err == nil {
		stats.TopActionType = topAction.ActionType
	}

	// Count suspicious users
	var suspiciousCount int64
	subQuery := r.db.Model(&models.UserActivityLog{}).
		Select("telegram_id, COUNT(*) as cnt").
		Where("created_at >= ?", oneHourAgo).
		Group("telegram_id").
		Having("COUNT(*) > ?", 100)

	if err := r.db.Table("(?) as sub", subQuery).Count(&suspiciousCount).Error; err == nil {
		stats.SuspiciousUsers = suspiciousCount
	}

	return stats, nil
}

// GetTopActivityUsers retrieves top users by action count and suspicion score
func (r *PostgresRepository) GetTopActivityUsers(limit int, timeFrom, timeTo *time.Time, minActions int) ([]models.UserActivityStats, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if timeFrom != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *timeFrom)
		argIndex++
	}

	if timeTo != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *timeTo)
		argIndex++
	}

	// IMPORTANT: We need to count last_hour/last_24h from ALL user data, not just filtered
	query := fmt.Sprintf(`
		WITH user_actions AS (
			SELECT
				telegram_id,
				COUNT(*) as total_actions,
				COUNT(DISTINCT action_type) as unique_action_types,
				TO_CHAR(MIN(created_at), 'YYYY-MM-DD HH24:MI:SS') as first_seen,
				TO_CHAR(MAX(created_at), 'YYYY-MM-DD HH24:MI:SS') as last_seen,
				COUNT(CASE WHEN action_type LIKE 'callback_captcha%%' THEN 1 END) as captcha_count,
				COUNT(CASE WHEN action_type LIKE 'callback_bet_%%' THEN 1 END) as bet_count,
				COUNT(CASE WHEN action_type LIKE 'command_%%' THEN 1 END) as command_count
			FROM user_activity_logs
			%s
			GROUP BY telegram_id
			HAVING COUNT(*) >= $%d
		),
		user_recent_activity AS (
			SELECT
				telegram_id,
				COUNT(CASE WHEN created_at >= NOW() - INTERVAL '1 hour' THEN 1 END) as last_hour_actions,
				COUNT(CASE WHEN created_at >= NOW() - INTERVAL '24 hours' THEN 1 END) as last_24h_actions
			FROM user_activity_logs
			WHERE telegram_id IN (SELECT telegram_id FROM user_actions)
			GROUP BY telegram_id
		),
		intervals_calc AS (
			SELECT
				telegram_id,
				created_at,
				created_at - LAG(created_at) OVER (PARTITION BY telegram_id ORDER BY created_at) as time_diff
			FROM user_activity_logs
			%s
		),
		user_intervals AS (
			SELECT
				telegram_id,
				AVG(EXTRACT(EPOCH FROM time_diff)) as avg_interval,
				MIN(EXTRACT(EPOCH FROM time_diff)) as min_interval
			FROM intervals_calc
			WHERE time_diff IS NOT NULL
			GROUP BY telegram_id
		)
		SELECT
			ua.telegram_id,
			ua.total_actions,
			ua.unique_action_types,
			ua.first_seen,
			ua.last_seen,
			COALESCE(ura.last_hour_actions, 0) as last_hour_actions,
			COALESCE(ura.last_24h_actions, 0) as last_24h_actions,
			ua.captcha_count,
			ua.bet_count,
			ua.command_count,
			COALESCE(ui.avg_interval, 0) as avg_interval,
			COALESCE(ui.min_interval, 0) as min_interval,
			(
				CASE WHEN COALESCE(ura.last_hour_actions, 0) > 100 THEN 30 ELSE 0 END +
				CASE WHEN COALESCE(ui.avg_interval, 999) < 2 THEN 25 ELSE 0 END +
				CASE WHEN COALESCE(ui.min_interval, 999) < 0.5 THEN 20 ELSE 0 END +
				CASE WHEN ua.captcha_count > ua.total_actions * 0.3 THEN 15 ELSE 0 END +
				CASE WHEN ua.unique_action_types < 3 THEN 10 ELSE 0 END
			) as suspicion_score
		FROM user_actions ua
		LEFT JOIN user_recent_activity ura ON ua.telegram_id = ura.telegram_id
		LEFT JOIN user_intervals ui ON ua.telegram_id = ui.telegram_id
		ORDER BY suspicion_score DESC, ua.total_actions DESC
		LIMIT $%d`, whereClause, argIndex, whereClause, argIndex+1)

	args = append(args, minActions, limit)

	var users []models.UserActivityStats
	if err := r.db.Raw(query, args...).Scan(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserActivityStats retrieves detailed statistics for a specific user
func (r *PostgresRepository) GetUserActivityStats(telegramID int64, timeFrom, timeTo *time.Time) (*models.UserActivityStats, error) {
	whereClause := "WHERE telegram_id = $1"
	args := []interface{}{telegramID}
	argIndex := 2

	if timeFrom != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *timeFrom)
		argIndex++
	}

	if timeTo != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *timeTo)
		argIndex++
	}

	query := fmt.Sprintf(`
		WITH user_data AS (
			SELECT
				telegram_id,
				COUNT(*) as total_actions,
				COUNT(DISTINCT action_type) as unique_action_types,
				TO_CHAR(MIN(created_at), 'YYYY-MM-DD HH24:MI:SS') as first_seen,
				TO_CHAR(MAX(created_at), 'YYYY-MM-DD HH24:MI:SS') as last_seen,
				COUNT(CASE WHEN created_at >= NOW() - INTERVAL '1 hour' THEN 1 END) as last_hour_actions,
				COUNT(CASE WHEN created_at >= NOW() - INTERVAL '24 hours' THEN 1 END) as last_24h_actions,
				COUNT(CASE WHEN action_type LIKE 'callback_captcha%%' THEN 1 END) as captcha_count,
				COUNT(CASE WHEN action_type LIKE 'callback_bet_%%' THEN 1 END) as bet_count,
				COUNT(CASE WHEN action_type LIKE 'command_%%' THEN 1 END) as command_count
			FROM user_activity_logs
			%s
			GROUP BY telegram_id
		),
		intervals_calc AS (
			SELECT
				created_at,
				created_at - LAG(created_at) OVER (ORDER BY created_at) as time_diff
			FROM user_activity_logs
			%s
		),
		intervals AS (
			SELECT
				AVG(EXTRACT(EPOCH FROM time_diff)) as avg_interval,
				MIN(EXTRACT(EPOCH FROM time_diff)) as min_interval
			FROM intervals_calc
			WHERE time_diff IS NOT NULL
		)
		SELECT
			ud.telegram_id,
			ud.total_actions,
			ud.unique_action_types,
			ud.first_seen,
			ud.last_seen,
			ud.last_hour_actions,
			ud.last_24h_actions,
			ud.captcha_count,
			ud.bet_count,
			ud.command_count,
			COALESCE(i.avg_interval, 0) as avg_interval,
			COALESCE(i.min_interval, 0) as min_interval,
			(
				CASE WHEN ud.last_hour_actions > 100 THEN 30 ELSE 0 END +
				CASE WHEN COALESCE(i.avg_interval, 999) < 2 THEN 25 ELSE 0 END +
				CASE WHEN COALESCE(i.min_interval, 999) < 0.5 THEN 20 ELSE 0 END +
				CASE WHEN ud.captcha_count > ud.total_actions * 0.3 THEN 15 ELSE 0 END +
				CASE WHEN ud.unique_action_types < 3 THEN 10 ELSE 0 END
			) as suspicion_score
		FROM user_data ud
		CROSS JOIN intervals i`, whereClause, whereClause)

	var stats models.UserActivityStats
	if err := r.db.Raw(query, args...).Scan(&stats).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetUserActivityLogsForAnalyzer retrieves recent actions for a user
func (r *PostgresRepository) GetUserActivityLogsForAnalyzer(telegramID int64, limit int, actionType string) ([]models.UserActivityLog, error) {
	query := r.db.Where("telegram_id = ?", telegramID)

	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	var actions []models.UserActivityLog
	err := query.Order("created_at DESC").Limit(limit).Find(&actions).Error
	return actions, err
}

// GetUserActivityTimeline retrieves action counts over time
func (r *PostgresRepository) GetUserActivityTimeline(telegramID int64, interval string, timeFrom, timeTo *time.Time, actionType string) ([]models.ActionTimeSeries, error) {
	query := `
		SELECT
			TO_CHAR(date_trunc($1, created_at), 'YYYY-MM-DD HH24:MI:SS') as timestamp,
			COUNT(*) as action_count
		FROM user_activity_logs
		WHERE telegram_id = $2
	`

	args := []interface{}{interval, telegramID}
	argIndex := 3

	if timeFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *timeFrom)
		argIndex++
	}

	if timeTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *timeTo)
		argIndex++
	}

	if actionType != "" {
		query += fmt.Sprintf(" AND action_type = $%d", argIndex)
		args = append(args, actionType)
	}

	query += " GROUP BY date_trunc($1, created_at) ORDER BY date_trunc($1, created_at) ASC"

	var timeline []models.ActionTimeSeries
	if err := r.db.Raw(query, args...).Scan(&timeline).Error; err != nil {
		return nil, err
	}

	return timeline, nil
}

// GetUserActionDistribution retrieves distribution of action types
func (r *PostgresRepository) GetUserActionDistribution(telegramID int64, timeFrom, timeTo *time.Time) ([]models.ActionTypeDistribution, error) {
	query := `
		WITH action_counts AS (
			SELECT
				action_type,
				COUNT(*) as count
			FROM user_activity_logs
			WHERE telegram_id = $1
	`

	args := []interface{}{telegramID}
	argIndex := 2

	if timeFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *timeFrom)
		argIndex++
	}

	if timeTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *timeTo)
		argIndex++
	}

	query += `
			GROUP BY action_type
		),
		total_count AS (
			SELECT SUM(count) as total FROM action_counts
		)
		SELECT
			ac.action_type,
			ac.count,
			ROUND((ac.count::numeric / tc.total * 100)::numeric, 2) as percentage
		FROM action_counts ac
		CROSS JOIN total_count tc
		ORDER BY ac.count DESC
	`

	var distribution []models.ActionTypeDistribution
	if err := r.db.Raw(query, args...).Scan(&distribution).Error; err != nil {
		return nil, err
	}

	return distribution, nil
}

// GetUserActivityIntervals retrieves time intervals between actions
func (r *PostgresRepository) GetUserActivityIntervals(telegramID int64, limit int) ([]models.IntervalStats, error) {
	query := `
		WITH intervals_calc AS (
			SELECT
				created_at,
				created_at - LAG(created_at) OVER (ORDER BY created_at) as time_diff
			FROM user_activity_logs
			WHERE telegram_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		)
		SELECT
			TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS') as timestamp,
			COALESCE(EXTRACT(EPOCH FROM time_diff), 0) as interval_seconds
		FROM intervals_calc
		WHERE time_diff IS NOT NULL
		ORDER BY created_at DESC
	`

	var intervals []models.IntervalStats
	if err := r.db.Raw(query, telegramID, limit).Scan(&intervals).Error; err != nil {
		return nil, err
	}

	return intervals, nil
}

// GetAllActionTypes retrieves all unique action types
func (r *PostgresRepository) GetAllActionTypes() ([]string, error) {
	var actionTypes []string
	err := r.db.Model(&models.UserActivityLog{}).
		Distinct("action_type").
		Order("action_type ASC").
		Pluck("action_type", &actionTypes).Error
	return actionTypes, err
}

// GetOverallActivityTimeline retrieves overall activity timeline
func (r *PostgresRepository) GetOverallActivityTimeline(interval string, timeFrom, timeTo *time.Time, limit int) ([]models.ActionTimeSeries, error) {
	query := `
		SELECT
			TO_CHAR(date_trunc($1, created_at), 'YYYY-MM-DD HH24:MI:SS') as timestamp,
			COUNT(*) as action_count
		FROM user_activity_logs
		WHERE 1=1
	`

	args := []interface{}{interval}
	argIndex := 2

	if timeFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *timeFrom)
		argIndex++
	}

	if timeTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *timeTo)
		argIndex++
	}

	query += fmt.Sprintf(" GROUP BY date_trunc($1, created_at) ORDER BY date_trunc($1, created_at) DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	var timeline []models.ActionTimeSeries
	if err := r.db.Raw(query, args...).Scan(&timeline).Error; err != nil {
		return nil, err
	}

	return timeline, nil
}

// GetTopActionTypes retrieves most frequent action types
func (r *PostgresRepository) GetTopActionTypes(limit int) ([]models.ActionTypeDistribution, error) {
	query := `
		WITH action_counts AS (
			SELECT
				action_type,
				COUNT(*) as count
			FROM user_activity_logs
			GROUP BY action_type
		),
		total_count AS (
			SELECT SUM(count) as total FROM action_counts
		)
		SELECT
			ac.action_type,
			ac.count,
			ROUND((ac.count::numeric / tc.total * 100)::numeric, 2) as percentage
		FROM action_counts ac
		CROSS JOIN total_count tc
		ORDER BY ac.count DESC
		LIMIT $1
	`

	var distribution []models.ActionTypeDistribution
	if err := r.db.Raw(query, limit).Scan(&distribution).Error; err != nil {
		return nil, err
	}

	return distribution, nil
}
