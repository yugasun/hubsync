package observability

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// MetricsManager handles metrics collection
type MetricsManager struct {
	registry   *prometheus.Registry
	namespace  string
	enabled    bool
	mutex      sync.Mutex
	server     *http.Server
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
}

// NewMetricsManager creates a new metrics manager
func NewMetricsManager(enabled bool, namespace string) *MetricsManager {
	registry := prometheus.NewRegistry()

	// Create default metrics for the service
	mm := &MetricsManager{
		registry:   registry,
		namespace:  namespace,
		enabled:    enabled,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
	}

	// Only initialize metrics if enabled
	if enabled {
		mm.initializeDefaultMetrics()
	}

	return mm
}

// initializeDefaultMetrics sets up default metrics for the service
func (m *MetricsManager) initializeDefaultMetrics() {
	// Create a counter for operations
	m.CreateCounter("operations_total", "Total count of operations", []string{"operation", "status"})

	// Create a counter for sync operations
	m.CreateCounter("sync_operations_total", "Total count of sync operations", []string{"source", "target", "status"})

	// Create a histogram for operation durations
	m.CreateHistogram(
		"operation_duration_seconds",
		"Duration of operations in seconds",
		[]string{"operation"},
		prometheus.ExponentialBuckets(0.001, 2, 15), // Buckets from 1ms to ~16s
	)

	// Create gauges for system metrics
	m.CreateGauge("system_memory_bytes", "Current system memory usage in bytes", []string{"type"})
	m.CreateGauge("goroutines_total", "Current number of goroutines", nil)

	// Create a summary for image sizes
	m.CreateSummary(
		"image_size_bytes",
		"Size of Docker images in bytes",
		[]string{"image"},
		map[float64]float64{
			0.5:  0.05,  // 50th percentile with 5% tolerance
			0.9:  0.01,  // 90th percentile with 1% tolerance
			0.99: 0.001, // 99th percentile with 0.1% tolerance
		},
	)
}

// StartServer starts the metrics HTTP server
func (m *MetricsManager) StartServer(addr string) error {
	if !m.enabled {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.server != nil {
		return fmt.Errorf("metrics server already running")
	}

	// Create a new HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	m.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
	}

	// Start the server in a goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("Starting metrics server")
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Metrics server error")
		}
	}()

	return nil
}

// StopServer stops the metrics HTTP server
func (m *MetricsManager) StopServer(ctx context.Context) error {
	if !m.enabled || m.server == nil {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Info().Msg("Stopping metrics server")
	if err := m.server.Shutdown(ctx); err != nil {
		return err
	}

	m.server = nil
	return nil
}

// CreateCounter creates a new counter metric
func (m *MetricsManager) CreateCounter(name, help string, labels []string) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.counters[name]; exists {
		return // Counter already exists
	}

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	// Register the counter
	err := m.registry.Register(counter)
	if err != nil {
		log.Warn().Err(err).Str("name", name).Msg("Failed to register counter metric")
		return
	}

	m.counters[name] = counter
}

// IncrementCounter increments a counter metric
func (m *MetricsManager) IncrementCounter(name string, value float64, labelValues ...string) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	counter, exists := m.counters[name]
	m.mutex.Unlock()

	if !exists {
		log.Warn().Str("name", name).Msg("Counter not found")
		return
	}

	counter.WithLabelValues(labelValues...).Add(value)
}

// CreateGauge creates a new gauge metric
func (m *MetricsManager) CreateGauge(name, help string, labels []string) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.gauges[name]; exists {
		return // Gauge already exists
	}

	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: m.namespace,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	// Register the gauge
	err := m.registry.Register(gauge)
	if err != nil {
		log.Warn().Err(err).Str("name", name).Msg("Failed to register gauge metric")
		return
	}

	m.gauges[name] = gauge
}

// SetGauge sets a gauge metric
func (m *MetricsManager) SetGauge(name string, value float64, labelValues ...string) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	gauge, exists := m.gauges[name]
	m.mutex.Unlock()

	if !exists {
		log.Warn().Str("name", name).Msg("Gauge not found")
		return
	}

	gauge.WithLabelValues(labelValues...).Set(value)
}

// CreateHistogram creates a new histogram metric
func (m *MetricsManager) CreateHistogram(name, help string, labels []string, buckets []float64) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.histograms[name]; exists {
		return // Histogram already exists
	}

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.namespace,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)

	// Register the histogram
	err := m.registry.Register(histogram)
	if err != nil {
		log.Warn().Err(err).Str("name", name).Msg("Failed to register histogram metric")
		return
	}

	m.histograms[name] = histogram
}

// ObserveHistogram adds an observation to a histogram
func (m *MetricsManager) ObserveHistogram(name string, value float64, labelValues ...string) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	histogram, exists := m.histograms[name]
	m.mutex.Unlock()

	if !exists {
		log.Warn().Str("name", name).Msg("Histogram not found")
		return
	}

	histogram.WithLabelValues(labelValues...).Observe(value)
}

// CreateSummary creates a new summary metric
func (m *MetricsManager) CreateSummary(name, help string, labels []string, objectives map[float64]float64) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.summaries[name]; exists {
		return // Summary already exists
	}

	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  m.namespace,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)

	// Register the summary
	err := m.registry.Register(summary)
	if err != nil {
		log.Warn().Err(err).Str("name", name).Msg("Failed to register summary metric")
		return
	}

	m.summaries[name] = summary
}

// ObserveSummary adds an observation to a summary
func (m *MetricsManager) ObserveSummary(name string, value float64, labelValues ...string) {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	summary, exists := m.summaries[name]
	m.mutex.Unlock()

	if !exists {
		log.Warn().Str("name", name).Msg("Summary not found")
		return
	}

	summary.WithLabelValues(labelValues...).Observe(value)
}

// RecordOperationDuration records the duration of an operation
func (m *MetricsManager) RecordOperationDuration(operationName string, duration time.Duration) {
	if !m.enabled {
		return
	}

	// Convert duration to seconds for the histogram
	durationSeconds := duration.Seconds()
	m.ObserveHistogram("operation_duration_seconds", durationSeconds, operationName)
}

// RecordSyncOperation records a sync operation
func (m *MetricsManager) RecordSyncOperation(source, target, status string) {
	if !m.enabled {
		return
	}

	m.IncrementCounter("sync_operations_total", 1, source, target, status)
}

// Close releases resources associated with metrics manager
func (m *MetricsManager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.StopServer(ctx)
}
