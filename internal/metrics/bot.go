package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type BotMetrics struct {
	// Telegram API метрики
	TelegramMessagesSent *prometheus.CounterVec
	TelegramErrors       *prometheus.CounterVec
	MessageQueueSize     prometheus.Gauge
	MessageQueueTime     *prometheus.HistogramVec

	// Игровые метрики
	ActiveUsers    prometheus.Gauge
	ActivePlayers  prometheus.Gauge
	BetsByType     *prometheus.CounterVec
	CommandLatency *prometheus.HistogramVec
	BetResults     *prometheus.CounterVec

	BanTriggered     *prometheus.CounterVec
	CaptchaTriggered *prometheus.CounterVec
	CaptchaBan       *prometheus.CounterVec
	CaptchaRefresh   *prometheus.CounterVec
	CaptchaPassed    *prometheus.CounterVec
}

func NewBotMetrics(registry *prometheus.Registry) *BotMetrics {
	bm := &BotMetrics{
		// Telegram API метрики
		TelegramMessagesSent: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "roulette_bot_telegram_messages_sent_total",
				Help: "Total Telegram messages sent by type",
			},
			[]string{"type"}, // text, photo, sticker
		),

		TelegramErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "roulette_bot_telegram_errors_total",
				Help: "Total Telegram API errors",
			},
			[]string{"error_type"}, // 429, timeout, other
		),

		MessageQueueSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "roulette_bot_message_queue_size",
				Help: "Current message queue size",
			},
		),

		MessageQueueTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "roulette_bot_message_queue_time_seconds",
				Help:    "Time messages spend in queue before sending",
				Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
			},
			[]string{"priority"}, // normal, high
		),

		ActiveUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "roulette_bot_active_users",
				Help: "Number of users currently",
			},
		),
		// Игровые метрики
		ActivePlayers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "roulette_bot_active_players",
				Help: "Number of players currently in game mode",
			},
		),

		BetsByType: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "roulette_bot_bets_total",
				Help: "Total bets placed by type",
			},
			[]string{"bet_type"}, // red, black, zero
		),

		CommandLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "roulette_bot_command_duration_seconds",
				Help:    "Command processing latency",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"command"}, // start, play, rating, etc
		),

		BetResults: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "roulette_bot_bet_results_total",
				Help: "Total bet results",
			},
			[]string{"result"}, // won, lost
		),

		BanTriggered: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ban_triggered_total",
				Help: "Total Ban",
			},
			[]string{"ban_reason"}, // country, age, manual
		),
		CaptchaTriggered: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_triggered_total",
				Help: "Total Captcha",
			},
			[]string{"captcha_reason"}, // captcha_user_activity, captcha_bet_points, captcha_bet_activity, captcha_bet_duplicate, manual
		),
		CaptchaBan: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_lockout_total",
				Help: "Total Captcha lockout",
			},
			[]string{"lockout_reason"}, // short, long
		),
		CaptchaRefresh: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_refresh_total",
				Help: "Total Captcha refresh",
			},
			[]string{"refresh_reason"}, // "refresh", "wrong", "stage"
		),
		CaptchaPassed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_passed_total",
				Help: "Total Captcha passed",
			},
			[]string{"passed_reason"}, //
		),
	}

	// Регистрируем все метрики
	registry.MustRegister(
		bm.TelegramMessagesSent,
		bm.TelegramErrors,
		bm.MessageQueueSize,
		bm.MessageQueueTime,
		bm.ActiveUsers,
		bm.ActivePlayers,
		bm.BetsByType,
		bm.CommandLatency,
		bm.BetResults,

		bm.BanTriggered,
		bm.CaptchaTriggered,
		bm.CaptchaBan,
		bm.CaptchaRefresh,
		bm.CaptchaPassed,
	)

	return bm
}

// Helper methods для удобства использования

// Telegram API метрики
func (bm *BotMetrics) RecordMessageSent(messageType string) {
	bm.TelegramMessagesSent.WithLabelValues(messageType).Inc()
}

func (bm *BotMetrics) RecordTelegramError(errorType string) {
	bm.TelegramErrors.WithLabelValues(errorType).Inc()
}

func (bm *BotMetrics) UpdateQueueSize(size float64) {
	bm.MessageQueueSize.Set(size)
}

func (bm *BotMetrics) RecordQueueTime(priority string, duration float64) {
	bm.MessageQueueTime.WithLabelValues(priority).Observe(duration)
}

func (bm *BotMetrics) SetActiveUsers(count float64) {
	bm.ActiveUsers.Set(count)
}

// Игровые метрики
func (bm *BotMetrics) SetActivePlayers(count float64) {
	bm.ActivePlayers.Set(count)
}

func (bm *BotMetrics) RecordBet(betType string) {
	bm.BetsByType.WithLabelValues(betType).Inc()
}

func (bm *BotMetrics) RecordCommandLatency(command string, duration float64) {
	bm.CommandLatency.WithLabelValues(command).Observe(duration)
}

func (bm *BotMetrics) RecordBetResult(result string) {
	bm.BetResults.WithLabelValues(result).Inc()
}

// Метрики капчі
func (bm *BotMetrics) RecordBanTriggered(reason string) {
	bm.BanTriggered.WithLabelValues(reason).Inc()
}
func (bm *BotMetrics) RecordCaptchaTriggered(reason string) {
	bm.CaptchaTriggered.WithLabelValues(reason).Inc()
}
func (bm *BotMetrics) RecordCaptchaBan(reason string) {
	bm.CaptchaBan.WithLabelValues(reason).Inc()
}
func (bm *BotMetrics) RecordCaptchaRefresh(reason string) {
	bm.CaptchaRefresh.WithLabelValues(reason).Inc()
}
func (bm *BotMetrics) RecordCaptchaPassed(reason string) {
	bm.CaptchaPassed.WithLabelValues(reason).Inc()
}
