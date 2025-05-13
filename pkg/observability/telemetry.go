package observability

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// Custom context key types to avoid collisions
type contextKey string

const (
	correlationIDKey  contextKey = "correlation_id"
	operationNameKey  contextKey = "operation_name"
	operationStartKey contextKey = "operation_start"
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

	// Add operation details to context
	ctx = context.WithValue(ctx, correlationIDKey, correlationID)
	ctx = context.WithValue(ctx, operationNameKey, operationName)
	ctx = context.WithValue(ctx, operationStartKey, time.Now())

	// Log operation start
	log.Ctx(ctx).Debug().
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

	// Extract operation details from context
	correlationID, _ := ctx.Value(correlationIDKey).(string)
	operationName, _ := ctx.Value(operationNameKey).(string)
	startTime, _ := ctx.Value(operationStartKey).(time.Time)

	// Calculate duration
	duration := time.Since(startTime)

	// Create logger event
	event := log.Ctx(ctx).Info()
	if err != nil {
		event = log.Ctx(ctx).Error().Err(err)
	}

	// Add operation details
	event.
		Str("correlation_id", correlationID).
		Str("operation", operationName).
		Dur("duration", duration).
		Msg("Operation completed")
}

// RecordEvent records a telemetry event
func (t *TelemetryManager) RecordEvent(ctx context.Context, eventName string, properties map[string]interface{}) {
	if !t.enabled {
		return
	}

	// Extract context values
	correlationID, ok1 := ctx.Value(correlationIDKey).(string)
	operationName, ok2 := ctx.Value(operationNameKey).(string)

	if !ok1 || !ok2 {
		log.Warn().Str("event", eventName).Msg("Missing context values for telemetry event")
	}

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
	correlationID, ok1 := ctx.Value(correlationIDKey).(string)
	operationName, ok2 := ctx.Value(operationNameKey).(string)

	if !ok1 || !ok2 {
		log.Warn().
			Str("dependency_type", dependencyType).
			Str("dependency_name", dependencyName).
			Msg("Missing context values for dependency call")
	}

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
	correlationID, ok := ctx.Value(correlationIDKey).(string)
	if !ok {
		log.Warn().Str("metric", name).Msg("Missing correlation ID for metric")
		correlationID = "unknown"
	}

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

// LogCliCommand logs a CLI command execution
func (t *TelemetryManager) LogCliCommand(c *cli.Context) {
	if !t.enabled {
		return
	}

	// Convert arguments to string by joining them
	argsStr := strings.Join(c.Args().Slice(), " ")

	log.Info().
		Str("command", c.Command.Name).
		Str("args", argsStr).
		Msg("CLI command executed")
}

// GenerateID generates a unique identifier for operations
func GenerateID() string {
	return time.Now().Format("20060102-150405-000000")
}

// GetCorrelationID retrieves the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	id, _ := ctx.Value(correlationIDKey).(string)
	return id
}

// GetCorrelationIDKey returns the context key for correlation ID
func GetCorrelationIDKey() contextKey {
	return correlationIDKey
}

// GetOperationNameKey returns the context key for operation name
func GetOperationNameKey() contextKey {
	return operationNameKey
}

// GetOperationStartKey returns the context key for operation start time
func GetOperationStartKey() contextKey {
	return operationStartKey
}
