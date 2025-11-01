package service

import (
	"fmt"
	"time"

	"roulette/internal/models"
)

// GetActivityDashboardData retrieves all dashboard statistics
func (s *ServiceImpl) GetActivityDashboardData() (*models.ActivityDashboardStats, error) {
	return s.repo.GetActivityDashboardStats()
}

// GetTopSuspiciousUsers retrieves users sorted by suspicion score
func (s *ServiceImpl) GetTopSuspiciousUsers(limit int, timeFrom, timeTo *time.Time, minActions int) ([]models.UserActivityStats, error) {
	if limit <= 0 {
		limit = 50
	}
	if minActions <= 0 {
		minActions = 10
	}
	return s.repo.GetTopActivityUsers(limit, timeFrom, timeTo, minActions)
}

// GetUserActivityDetail retrieves comprehensive user information
func (s *ServiceImpl) GetUserActivityDetail(telegramID int64, timeFrom, timeTo *time.Time) (*models.UserActivityDetail, error) {
	detail := &models.UserActivityDetail{}

	// Get basic stats
	stats, err := s.repo.GetUserActivityStats(telegramID, timeFrom, timeTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	detail.Stats = stats

	// Get recent actions
	recentActions, err := s.repo.GetUserActivityLogsForAnalyzer(telegramID, 100, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get recent actions: %w", err)
	}
	detail.RecentActions = recentActions

	// Get action distribution
	distribution, err := s.repo.GetUserActionDistribution(telegramID, timeFrom, timeTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get action distribution: %w", err)
	}
	detail.ActionDistribution = distribution

	// Get timeline
	timeline, err := s.repo.GetUserActivityTimeline(telegramID, "hour", timeFrom, timeTo, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}
	detail.Timeline = timeline

	// Get intervals
	intervals, err := s.repo.GetUserActivityIntervals(telegramID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get intervals: %w", err)
	}
	detail.Intervals = intervals

	// Analyze suspicion factors
	detail.SuspicionFactors = s.analyzeSuspicionFactors(stats)

	return detail, nil
}

// analyzeSuspicionFactors generates human-readable suspicion factors
func (s *ServiceImpl) analyzeSuspicionFactors(stats *models.UserActivityStats) []string {
	factors := []string{}

	if stats.LastHourActions > 100 {
		factors = append(factors, fmt.Sprintf("🔴 High activity: %d actions in last hour", stats.LastHourActions))
	}

	if stats.AvgInterval > 0 && stats.AvgInterval < 2 {
		factors = append(factors, fmt.Sprintf("🔴 Very fast actions: avg %.2f seconds between actions", stats.AvgInterval))
	}

	if stats.MinInterval > 0 && stats.MinInterval < 0.5 {
		factors = append(factors, fmt.Sprintf("🔴 Instant actions detected: min %.2f seconds", stats.MinInterval))
	}

	if stats.TotalActions > 0 {
		captchaRate := float64(stats.CaptchaCount) / float64(stats.TotalActions) * 100
		if captchaRate > 30 {
			factors = append(factors, fmt.Sprintf("🟡 High captcha rate: %.1f%% (%d captchas)", captchaRate, stats.CaptchaCount))
		}
	}

	if stats.UniqueActionTypes < 3 {
		factors = append(factors, fmt.Sprintf("🟡 Limited action variety: only %d unique action types", stats.UniqueActionTypes))
	}

	if stats.BetCount > 0 && stats.TotalActions > 0 {
		betRate := float64(stats.BetCount) / float64(stats.TotalActions) * 100
		if betRate > 70 {
			factors = append(factors, fmt.Sprintf("🟡 Betting focused: %.1f%% are bets", betRate))
		}
	}

	// Check for repeating patterns
	if stats.AvgInterval > 0 && stats.AvgInterval < 10 && stats.TotalActions > 50 {
		factors = append(factors, "🟠 Possible automated pattern detected")
	}

	if len(factors) == 0 {
		factors = append(factors, "✅ No suspicious patterns detected")
	}

	return factors
}

// GetUserActivityTimeline retrieves timeline with filters
func (s *ServiceImpl) GetUserActivityTimeline(telegramID int64, interval string, timeFrom, timeTo *time.Time, actionType string) ([]models.ActionTimeSeries, error) {
	if interval == "" {
		interval = "hour"
	}
	return s.repo.GetUserActivityTimeline(telegramID, interval, timeFrom, timeTo, actionType)
}

// GetAllActivityActionTypes retrieves all available action types
func (s *ServiceImpl) GetAllActivityActionTypes() ([]string, error) {
	return s.repo.GetAllActionTypes()
}

// GetOverallActivityTimeline retrieves overall activity timeline
func (s *ServiceImpl) GetOverallActivityTimeline(interval string, timeFrom, timeTo *time.Time, limit int) ([]models.ActionTimeSeries, error) {
	if interval == "" {
		interval = "hour"
	}
	if limit <= 0 {
		limit = 48
	}
	return s.repo.GetOverallActivityTimeline(interval, timeFrom, timeTo, limit)
}

func (s *ServiceImpl) GetTopActionTypes(limit int) ([]models.ActionTypeDistribution, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.GetTopActionTypes(limit)
}
