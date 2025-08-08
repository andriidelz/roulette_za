package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// BusinessMetrics contains business logic metrics available to specific app types
type BusinessMetrics struct {
	appType ApplicationType

	// Universal metrics (доступны всем)
	ErrorsTotal     *prometheus.CounterVec
	DatabaseQueries *prometheus.HistogramVec

	// Bot metrics
	UsersRegistered prometheus.Counter // bot создает, admin читает

	// Admin-only metrics (одна для примера)
	AdminActions *prometheus.CounterVec // только admin

	// Rotator-only metrics (одна для примера)
	RotationsTotal prometheus.Counter // только rotator
}

// NewBusinessMetrics creates business metrics based on app type
func NewBusinessMetrics(registry *prometheus.Registry, appType ApplicationType) *BusinessMetrics {
	bm := &BusinessMetrics{
		appType: appType,
	}

	// Universal metrics (всегда создаются)
	bm.ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "roulette_errors_total",
			Help:        "Total number of errors",
			ConstLabels: prometheus.Labels{"app_type": string(appType)},
		},
		[]string{"component", "type"},
	)
	registry.MustRegister(bm.ErrorsTotal)

	bm.DatabaseQueries = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "roulette_database_query_duration_seconds",
			Help:        "Database query duration",
			ConstLabels: prometheus.Labels{"app_type": string(appType)},
		},
		[]string{"operation", "table"},
	)
	registry.MustRegister(bm.DatabaseQueries)

	// App-specific metrics
	switch appType {
	case AppTypeBot:
		bm.initBotMetrics(registry)
	case AppTypeAdmin:
		bm.initAdminMetrics(registry)
	case AppTypeRotator:
		bm.initRotatorMetrics(registry)
	}

	return bm
}

// Initialize Bot-specific business metrics
func (bm *BusinessMetrics) initBotMetrics(registry *prometheus.Registry) {
	// User metrics (bot создает)
	bm.UsersRegistered = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "roulette_users_registered_total",
		Help:        "Total number of registered users",
		ConstLabels: prometheus.Labels{"app_type": "bot"},
	})
	registry.MustRegister(bm.UsersRegistered)
}

// Initialize Admin-specific business metrics
func (bm *BusinessMetrics) initAdminMetrics(registry *prometheus.Registry) {
	// Admin actions (одна для примера)
	bm.AdminActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "roulette_admin_actions_total",
			Help:        "Total admin actions",
			ConstLabels: prometheus.Labels{"app_type": "admin"},
		},
		[]string{"action", "user"},
	)
	registry.MustRegister(bm.AdminActions)
}

// Initialize Rotator-specific business metrics
func (bm *BusinessMetrics) initRotatorMetrics(registry *prometheus.Registry) {
	bm.RotationsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "roulette_rotations_total",
		Help:        "Total number of hash rotations",
		ConstLabels: prometheus.Labels{"app_type": "rotator"},
	})
	registry.MustRegister(bm.RotationsTotal)
}

// Helper methods that work only for specific app types
func (bm *BusinessMetrics) RecordUserRegistration() {
	if bm.appType == AppTypeBot && bm.UsersRegistered != nil {
		bm.UsersRegistered.Inc()
	}
}

func (bm *BusinessMetrics) RecordAdminAction(action, user string) {
	if bm.appType == AppTypeAdmin && bm.AdminActions != nil {
		bm.AdminActions.WithLabelValues(action, user).Inc()
	}
}

func (bm *BusinessMetrics) RecordRotation() {
	if bm.appType == AppTypeRotator && bm.RotationsTotal != nil {
		bm.RotationsTotal.Inc()
	}
}

// Universal methods (работают везде)
func (bm *BusinessMetrics) RecordError(component, errorType string) {
	bm.ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

func (bm *BusinessMetrics) RecordDatabaseQuery(operation, table string, duration float64) {
	bm.DatabaseQueries.WithLabelValues(operation, table).Observe(duration)
}
