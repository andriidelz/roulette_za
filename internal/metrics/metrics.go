package metrics

import (
	"fmt"
	"net/http"
	"time"

	"roulette/internal/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ApplicationType represents different application types
type ApplicationType string

const (
	AppTypeBot     ApplicationType = "bot"
	AppTypeAdmin   ApplicationType = "admin"
	AppTypeRotator ApplicationType = "rotator"
)

// Metrics holds all application metrics
type Metrics struct {
	AppType ApplicationType

	// Basic application metrics (общие для всех)
	AppUptime prometheus.Gauge

	// Business logic metrics (инициализируются выборочно)
	Business *BusinessMetrics
	Bot      *BotMetrics     // только для бота
	Admin    *AdminMetrics   // только для админки
	Rotator  *RotatorMetrics // только для ротатора

	registry *prometheus.Registry
	server   *http.Server

	// Fields for uptime tracking
	startTime time.Time
}

// NewMetrics creates metrics based on application type
func NewMetrics(serviceName string, port int, appType ApplicationType) *Metrics {
	registry := prometheus.NewRegistry()

	// Register Go runtime metrics (всегда)
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Create basic metrics (общие для всех)
	appUptime := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "app_uptime_seconds",
		Help: "Application uptime in seconds",
		ConstLabels: prometheus.Labels{
			"service":  serviceName,
			"app_type": string(appType),
		},
	})
	registry.MustRegister(appUptime)

	m := &Metrics{
		AppType:   appType,
		AppUptime: appUptime,
		registry:  registry,
		startTime: time.Now(), // Initialize start time
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		},
	}

	// Initialize metrics based on application type
	switch appType {
	case AppTypeBot:
		m.Business = NewBusinessMetrics(registry, appType)
		m.Bot = NewBotMetrics(registry)
	case AppTypeAdmin:
		m.Business = NewBusinessMetrics(registry, appType)
		m.Admin = NewAdminMetrics(registry)
		m.Bot = NewBotMetrics(registry)
	case AppTypeRotator:
		m.Business = NewBusinessMetrics(registry, appType)
		m.Rotator = NewRotatorMetrics(registry)
	}

	// Start uptime tracking
	go m.trackUptime()

	return m
}

// Start starts the metrics server
func (m *Metrics) Start() error {
	logger.Info.Printf("Starting metrics server on %s", m.server.Addr)
	if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	return nil
}

// Stop gracefully stops the metrics server
func (m *Metrics) Stop() error {
	logger.Info.Println("Stopping metrics server...")
	return m.server.Close()
}

// trackUptime tracks application uptime
func (m *Metrics) trackUptime() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		uptime := time.Since(m.startTime).Seconds()
		m.AppUptime.Set(uptime)
	}
}

// GetRegistry returns the Prometheus registry
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}
