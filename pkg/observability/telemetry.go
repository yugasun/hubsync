package observability

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TelemetryManager handles telemetry for the application
type TelemetryManager struct {
	enabled        bool
	serviceName    string
	serviceVersion string
	environment    string
	hostInfo       *HostInfo
	startTime      time.Time
	correlationIDs map[string]string
}

// HostInfo contains information about the host environment
type HostInfo struct {
	Hostname     string
	OS           string
	Architecture string
	CPUCount     int
	GoVersion    string
}

// NewTelemetryManager creates a new telemetry manager
func NewTelemetryManager(enabled bool, serviceName, serviceVersion, environment string) *TelemetryManager {
	hostInfo := getHostInfo()

	return &TelemetryManager{
		enabled:        enabled,
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		environment:    environment,
		hostInfo:       hostInfo,
		startTime:      time.Now(),
		correlationIDs: make(map[string]string),
	}
}

// getHostInfo collects information about the host
func getHostInfo() *HostInfo {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return &HostInfo{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		CPUCount:     runtime.NumCPU(),
		GoVersion:    runtime.Version(),
	}
}

// Start records the start of an operation
func (t *TelemetryManager) Start(ctx context.Context, operationName string, correlationID string) context.Context {
	if !t.enabled {
		return ctx
	}

	// Store correlation ID
	t.correlationIDs[operationName] = correlationID

	// Add correlation ID to context
	ctx = context.WithValue(ctx, "correlation_id", correlationID)
	ctx = context.WithValue(ctx, "operation_name", operationName)
	ctx = context.WithValue(ctx, "operation_start", time.Now())

	// Log operation start
	log.Debug().
		Str("correlation_id", correlationID).
		Str("operation", operationName).
		Str("service", t.serviceName).
		Str("version", t.serviceVersion).
		Msg("Operation started")

	return ctx
}

// End records the end of an operation
func (t *TelemetryManager) End(ctx context.Context, err error) {
	if !t.enabled {
		return
	}

	// Extract context values
	correlationID, _ := ctx.Value("correlation_id").(string)
	operationName, _ := ctx.Value("operation_name").(string)
	startTime, _ := ctx.Value("operation_start").(time.Time)

	// Calculate duration
	duration := time.Since(startTime)

	if err != nil {
		log.Error().
			Str("correlation_id", correlationID).
			Str("operation", operationName).
			Dur("duration_ms", duration).
			Err(err).
			Msg("Operation failed")
	} else {
		log.Debug().
			Str("correlation_id", correlationID).
			Str("operation", operationName).
			Dur("duration_ms", duration).
			Msg("Operation completed successfully")
	}
}

// RecordEvent records a telemetry event
func (t *TelemetryManager) RecordEvent(ctx context.Context, eventName string, properties map[string]interface{}) {
	if !t.enabled {
		return
	}

	// Extract context values
	correlationID, _ := ctx.Value("correlation_id").(string)
	operationName, _ := ctx.Value("operation_name").(string)

	// Start with a logger containing common fields
	logger := log.Info().
		Str("event", eventName).
		Str("correlation_id", correlationID).
		Str("operation", operationName).
		Str("service", t.serviceName).
		Str("version", t.serviceVersion)

	// Add custom properties
	for k, v := range properties {
		addFieldToEvent(logger, k, v)
	}

	// Log the event
	logger.Msg("Telemetry event")
}

// RecordDependency records a dependency call
func (t *TelemetryManager) RecordDependency(ctx context.Context, dependencyType, dependencyName string, startTime time.Time, success bool, err error) {
	if !t.enabled {
		return
	}

	// Extract context values
	correlationID, _ := ctx.Value("correlation_id").(string)
	operationName, _ := ctx.Value("operation_name").(string)

	// Calculate duration
	duration := time.Since(startTime)

	// Log with appropriate level based on success
	if success {
		log.Debug().
			Str("dependency_type", dependencyType).
			Str("dependency_name", dependencyName).
			Str("correlation_id", correlationID).
			Str("operation", operationName).
			Dur("duration_ms", duration).
			Bool("success", success).
			Msg("Dependency call completed")
	} else {
		log.Error().
			Str("dependency_type", dependencyType).
			Str("dependency_name", dependencyName).
			Str("correlation_id", correlationID).
			Str("operation", operationName).
			Dur("duration_ms", duration).
			Bool("success", success).
			Err(err).
			Msg("Dependency call failed")
	}
}

// RecordMetric records a custom metric
func (t *TelemetryManager) RecordMetric(ctx context.Context, name string, value float64, properties map[string]interface{}) {
	if !t.enabled {
		return
	}

	// Extract context values
	correlationID, _ := ctx.Value("correlation_id").(string)

	// Build logger with common fields
	logger := log.Debug().
		Str("metric", name).
		Float64("value", value).
		Str("correlation_id", correlationID)

	// Add custom properties
	for k, v := range properties {
		addFieldToEvent(logger, k, v)
	}

	// Log the metric
	logger.Msg("Metric recorded")
}

// Helper function to add a field to the logger based on value type
func addFieldToLogger(logger zerolog.Context, key string, value interface{}) zerolog.Context {
	switch v := value.(type) {
	case string:
		return logger.Str(key, v)
	case int:
		return logger.Int(key, v)
	case int64:
		return logger.Int64(key, v)
	case float64:
		return logger.Float64(key, v)
	case bool:
		return logger.Bool(key, v)
	case time.Time:
		return logger.Time(key, v)
	case time.Duration:
		return logger.Dur(key, v)
	default:
		return logger.Interface(key, v)
	}
}

// Helper function to add a field to an event logger based on value type
func addFieldToEvent(e *zerolog.Event, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		e.Str(key, v)
	case int:
		e.Int(key, v)
	case int64:
		e.Int64(key, v)
	case float64:
		e.Float64(key, v)
	case bool:
		e.Bool(key, v)
	case time.Time:
		e.Time(key, v)
	case time.Duration:
		e.Dur(key, v)
	default:
		e.Interface(key, v)
	}
}

// GetCorrelationID gets a correlation ID for an operation
func (t *TelemetryManager) GetCorrelationID(operationName string) string {
	if id, exists := t.correlationIDs[operationName]; exists {
		return id
	}

	// Generate new correlation ID
	id := fmt.Sprintf("%s-%d", operationName, time.Now().UnixNano())
	t.correlationIDs[operationName] = id
	return id
}

// GetUptime returns the application uptime
func (t *TelemetryManager) GetUptime() time.Duration {
	return time.Since(t.startTime)
}

// LogApplicationInfo logs information about the application at startup
func (t *TelemetryManager) LogApplicationInfo() {
	if !t.enabled {
		return
	}

	log.Info().
		Str("service", t.serviceName).
		Str("version", t.serviceVersion).
		Str("environment", t.environment).
		Str("hostname", t.hostInfo.Hostname).
		Str("os", t.hostInfo.OS).
		Str("arch", t.hostInfo.Architecture).
		Int("cpu_count", t.hostInfo.CPUCount).
		Str("go_version", t.hostInfo.GoVersion).
		Msg("Application started")
}
