package models

// UserActivityStats aggregated statistics per user for antifraud analysis
type UserActivityStats struct {
	TelegramID        int64   `json:"telegram_id"`
	Username          string  `json:"username,omitempty"`
	TotalActions      int64   `json:"total_actions"`
	UniqueActionTypes int64   `json:"unique_action_types"`
	FirstSeen         string  `json:"first_seen"`
	LastSeen          string  `json:"last_seen"`
	LastHourActions   int64   `json:"last_hour_actions"`
	Last24hActions    int64   `json:"last_24h_actions"`
	CaptchaCount      int64   `json:"captcha_count"`
	BetCount          int64   `json:"bet_count"`
	CommandCount      int64   `json:"command_count"`
	AvgInterval       float64 `json:"avg_interval_seconds"`
	MinInterval       float64 `json:"min_interval_seconds"`
	SuspicionScore    float64 `json:"suspicion_score"`
}

// ActionTimeSeries for timeline charts
type ActionTimeSeries struct {
	Timestamp   string `json:"timestamp"`
	ActionCount int64  `json:"action_count"`
	ActionType  string `json:"action_type,omitempty"`
}

// IntervalStats statistics about time intervals between actions
type IntervalStats struct {
	Interval  float64 `json:"interval_seconds"`
	Timestamp string  `json:"timestamp"`
}

// ActionTypeDistribution distribution of action types
type ActionTypeDistribution struct {
	ActionType string  `json:"action_type"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ActivityDashboardStats overall dashboard statistics
type ActivityDashboardStats struct {
	TotalUsers        int64   `json:"total_users"`
	TotalActions      int64   `json:"total_actions"`
	ActionsLastHour   int64   `json:"actions_last_hour"`
	ActionsLast24h    int64   `json:"actions_last_24h"`
	AvgActionsPerUser float64 `json:"avg_actions_per_user"`
	TopActionType     string  `json:"top_action_type"`
	SuspiciousUsers   int64   `json:"suspicious_users"`
}

// UserActivityDetail detailed user information with all data
type UserActivityDetail struct {
	Stats              *UserActivityStats          `json:"stats"`
	RecentActions      []UserActivityLog           `json:"recent_actions"`
	ActionDistribution []ActionTypeDistribution    `json:"action_distribution"`
	Timeline           []ActionTimeSeries          `json:"timeline"`
	Intervals          []IntervalStats             `json:"intervals"`
	SuspicionFactors   []string                    `json:"suspicion_factors"`
}
