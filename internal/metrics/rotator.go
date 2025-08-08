package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// RotatorMetrics contains rotator-specific metrics
type RotatorMetrics struct {
	// Одна метрика для примера
	RotationDuration prometheus.Histogram
}

// NewRotatorMetrics creates and registers rotator-specific metrics
func NewRotatorMetrics(registry *prometheus.Registry) *RotatorMetrics {
	rm := &RotatorMetrics{
		RotationDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "roulette_rotator_duration_seconds",
			Help:    "Time taken to complete hash rotation",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		}),
	}

	// Register metrics
	registry.MustRegister(rm.RotationDuration)

	return rm
}

// Helper methods for recording metrics
func (rm *RotatorMetrics) RecordRotation(duration float64) {
	rm.RotationDuration.Observe(duration)
}
