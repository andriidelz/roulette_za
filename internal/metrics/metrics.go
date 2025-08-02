package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics представляет метрики приложения
type Metrics struct {
	registry      *prometheus.Registry
	server        *http.Server
	uptimeSeconds prometheus.Gauge
	startTime     time.Time
}

// NewMetrics создает новый сборщик метрик
func NewMetrics(serviceName string, port int) *Metrics {
	registry := prometheus.NewRegistry()

	// Регистрируем стандартные Go метрики (GC, горутины, память и т.д.)
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	// Создаем метрику времени работы
	uptimeSeconds := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "app_uptime_seconds",
		Help: "The uptime of the application in seconds",
	})
	registry.MustRegister(uptimeSeconds)

	// Создаем HTTP сервер
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	metrics := &Metrics{
		registry:      registry,
		server:        server,
		uptimeSeconds: uptimeSeconds,
		startTime:     time.Now(),
	}

	// Запускаем горутину для обновления метрики uptime
	go metrics.updateUptime()

	return metrics
}

// Start запускает сервер метрик
func (m *Metrics) Start() error {
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	fmt.Printf("Metrics server started on %s\n", m.server.Addr)
	return nil
}

// Stop останавливает сервер метрик
func (m *Metrics) Stop() error {
	return m.server.Close()
}

// updateUptime обновляет метрику времени работы каждую секунду
func (m *Metrics) updateUptime() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		uptime := time.Since(m.startTime).Seconds()
		m.uptimeSeconds.Set(uptime)
	}
}
