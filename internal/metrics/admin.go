package metrics

import "github.com/prometheus/client_golang/prometheus"

type AdminMetrics struct {
	// Одна метрика для примера
	WebRequests *prometheus.CounterVec
}

func NewAdminMetrics(registry *prometheus.Registry) *AdminMetrics {
	am := &AdminMetrics{
		WebRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "roulette_admin_web_requests_total",
				Help: "Total web requests to admin panel",
			},
			[]string{"method", "endpoint", "status"},
		),
	}

	registry.MustRegister(am.WebRequests)
	return am
}
