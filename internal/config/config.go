package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/yugasun/hubsync/pkg/errors"
)

// Config represents the application configuration
type Config struct {
	// Essential settings
	Username   string
	Password   string
	Repository string
	Namespace  string
	Content    string
	MaxContent int
	OutputPath string

	// Performance settings
	Concurrency int
	Timeout     time.Duration
	RetryCount  int
	RetryDelay  time.Duration

	// Advanced settings
	LogLevel       string
	LogFile        string
	ShowVersion    bool
	Force          bool
	DryRun         bool
	Profile        string
	ConfigFilePath string
	Version        string

	// Telemetry settings
	TelemetryEnabled bool
	MetricsEnabled   bool
}

// ConfigProfile represents a named set of configuration options
type ConfigProfile struct {
	Name        string
	Description string
	Settings    map[string]interface{}
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Namespace:        "yugasun",
		MaxContent:       10,
		OutputPath:       "output.log",
		Concurrency:      3,
		Timeout:          10 * time.Minute,
		RetryCount:       3,
		RetryDelay:       2 * time.Second,
		LogLevel:         "info",
		Force:            false,
		DryRun:           false,
		Profile:          "default",
		TelemetryEnabled: true,
		MetricsEnabled:   false,
	}
}

// ParseConfig parses command-line flags, environment variables, and config files
func ParseConfig() (*Config, error) {
	// Start with default config
	cfg := DefaultConfig()

	// Load environment variables from .env file if present
	loadEnvFile()

	// Initialize viper for hierarchical configuration
	v := viper.New()
	v.SetConfigName("hubsync")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.hubsync")
	v.AddConfigPath("/etc/hubsync")

	// Define command-line flags with environment variable fallbacks
	pflag.StringVar(&cfg.Username, "username", getEnv("DOCKER_USERNAME", cfg.Username), "Docker registry username")
	pflag.StringVar(&cfg.Password, "password", getEnv("DOCKER_PASSWORD", cfg.Password), "Docker registry password")
	pflag.StringVar(&cfg.Repository, "repository", getEnv("DOCKER_REPOSITORY", cfg.Repository), "Target repository address")
	pflag.StringVar(&cfg.Namespace, "namespace", getEnv("DOCKER_NAMESPACE", cfg.Namespace), "Target namespace")
	pflag.StringVar(&cfg.Content, "content", getEnv("CONTENT", cfg.Content), "JSON content with images to sync")
	pflag.IntVar(&cfg.MaxContent, "max-content", getEnvInt("MAX_CONTENT", cfg.MaxContent), "Maximum number of images to process")
	pflag.StringVar(&cfg.OutputPath, "output", getEnv("OUTPUT_PATH", cfg.OutputPath), "Output file path")

	// Performance settings
	pflag.IntVar(&cfg.Concurrency, "concurrency", getEnvInt("CONCURRENCY", cfg.Concurrency), "Maximum concurrent operations")
	pflag.DurationVar(&cfg.Timeout, "timeout", getEnvDuration("TIMEOUT", cfg.Timeout), "Operation timeout")
	pflag.IntVar(&cfg.RetryCount, "retry-count", getEnvInt("RETRY_COUNT", cfg.RetryCount), "Number of retries for failed operations")
	pflag.DurationVar(&cfg.RetryDelay, "retry-delay", getEnvDuration("RETRY_DELAY", cfg.RetryDelay), "Delay between retries")

	// Advanced settings
	pflag.StringVar(&cfg.LogLevel, "log-level", getEnv("LOG_LEVEL", cfg.LogLevel), "Log level (debug, info, warn, error)")
	pflag.StringVar(&cfg.LogFile, "log-file", getEnv("LOG_FILE", cfg.LogFile), "Log to file in addition to stdout")
	pflag.BoolVar(&cfg.Force, "force", getBoolEnv("FORCE", cfg.Force), "Force synchronization even if target exists")
	pflag.BoolVar(&cfg.DryRun, "dry-run", getBoolEnv("DRY_RUN", cfg.DryRun), "Show what would be done without actually performing operations")
	pflag.StringVar(&cfg.Profile, "profile", getEnv("PROFILE", cfg.Profile), "Configuration profile to use")
	pflag.StringVar(&cfg.ConfigFilePath, "config", getEnv("CONFIG_FILE", ""), "Path to configuration file")

	// Telemetry settings
	pflag.BoolVar(&cfg.TelemetryEnabled, "telemetry", getBoolEnv("TELEMETRY_ENABLED", cfg.TelemetryEnabled), "Enable telemetry collection")
	pflag.BoolVar(&cfg.MetricsEnabled, "metrics", getBoolEnv("METRICS_ENABLED", cfg.MetricsEnabled), "Enable metrics collection")

	// Help and version
	var (
		help    = pflag.BoolP("help", "h", false, "Show this help message")
		version = pflag.BoolP("version", "v", false, "Show version information")
	)

	// Parse flags
	pflag.Parse()

	// Bind pflags to viper
	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return nil, errors.NewConfigError("config", "failed to bind flags to viper", err)
	}

	// Show help if requested
	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Store version flag
	cfg.ShowVersion = *version

	// Try to load configuration file
	if cfg.ConfigFilePath != "" {
		v.SetConfigFile(cfg.ConfigFilePath)
	}

	// Read in config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, errors.NewConfigError("config", "error reading config file", err)
		}
		// Config file not found is normal, continue with other sources
		log.Debug().Msg("No configuration file found, using command-line and environment values")
	} else {
		log.Debug().Str("file", v.ConfigFileUsed()).Msg("Configuration loaded from file")

		// Apply profile if specified and found
		if cfg.Profile != "default" && v.IsSet("profiles") {
			profiles := v.GetStringMap("profiles")
			if profileSettings, ok := profiles[cfg.Profile]; ok {
				// Convert map to JSON and back to apply profile settings
				jsonBytes, err := json.Marshal(profileSettings)
				if err != nil {
					return nil, errors.NewConfigError("config", "error marshaling profile settings", err)
				}

				if err := json.Unmarshal(jsonBytes, cfg); err != nil {
					return nil, errors.NewConfigError("config", "error applying profile settings", err)
				}

				log.Debug().Str("profile", cfg.Profile).Msg("Applied configuration profile")
			} else {
				log.Warn().Str("profile", cfg.Profile).Msg("Requested profile not found in configuration")
			}
		}

		// Map all remaining config settings from viper to struct
		if err := v.Unmarshal(cfg); err != nil {
			return nil, errors.NewConfigError("config", "error unmarshaling config", err)
		}
	}

	// Validate required fields based on mode
	if !cfg.ShowVersion {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}

	// Log the configuration (omitting sensitive fields)
	log.Debug().
		Str("username", cfg.Username).
		Str("repository", cfg.Repository).
		Str("namespace", cfg.Namespace).
		Int("maxContent", cfg.MaxContent).
		Int("concurrency", cfg.Concurrency).
		Dur("timeout", cfg.Timeout).
		Str("outputPath", cfg.OutputPath).
		Bool("force", cfg.Force).
		Bool("dryRun", cfg.DryRun).
		Str("profile", cfg.Profile).
		Str("logLevel", cfg.LogLevel).
		Bool("telemetryEnabled", cfg.TelemetryEnabled).
		Bool("metricsEnabled", cfg.MetricsEnabled).
		Msg("Configuration loaded")

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Username == "" {
		return errors.NewValidationError("config", "username is required", nil)
	}
	if c.Password == "" {
		return errors.NewValidationError("config", "password is required", nil)
	}
	if c.Content == "" {
		return errors.NewValidationError("config", "content is required", nil)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	if _, valid := validLogLevels[c.LogLevel]; !valid {
		return errors.NewValidationError(
			"config",
			fmt.Sprintf("invalid log level: %s (must be one of: debug, info, warn, error, fatal)", c.LogLevel),
			nil,
		)
	}

	// Validate performance settings
	if c.Concurrency < 1 {
		return errors.NewValidationError(
			"config",
			fmt.Sprintf("invalid concurrency: %d (must be >= 1)", c.Concurrency),
			nil,
		)
	}

	if c.RetryCount < 0 {
		return errors.NewValidationError(
			"config",
			fmt.Sprintf("invalid retry count: %d (must be >= 0)", c.RetryCount),
			nil,
		)
	}

	return nil
}

// LoadFromFile loads configuration from a file
func (c *Config) LoadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.NewIOError("config", "failed to read configuration file", err)
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, c); err != nil {
			return errors.NewConfigError("config", "failed to parse JSON configuration", err)
		}
	case ".yml", ".yaml":
		v := viper.New()
		v.SetConfigType("yaml")
		if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
			return errors.NewConfigError("config", "failed to parse YAML configuration", err)
		}
		if err := v.Unmarshal(c); err != nil {
			return errors.NewConfigError("config", "failed to apply YAML configuration", err)
		}
	default:
		return errors.NewConfigError("config", fmt.Sprintf("unsupported config file format: %s", ext), nil)
	}

	return nil
}

// SaveToFile saves configuration to a file
func (c *Config) SaveToFile(filePath string) error {
	// Check file extension
	ext := filepath.Ext(filePath)
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(c, "", "  ")
		if err != nil {
			return errors.NewOperationError("config", "failed to marshal configuration to JSON", err)
		}
	case ".yml", ".yaml":
		v := viper.New()
		for key, value := range structToMap(c) {
			v.Set(key, value)
		}
		v.SetConfigType("yaml")

		// Create a temporary file for viper to write to
		tmpFile := filePath + ".tmp"
		if err := v.WriteConfigAs(tmpFile); err != nil {
			return errors.NewOperationError("config", "failed to write YAML configuration", err)
		}

		// Read the file back
		data, err = os.ReadFile(tmpFile)
		if err != nil {
			return errors.NewOperationError("config", "failed to read temporary YAML configuration", err)
		}

		// Clean up
		os.Remove(tmpFile)
	default:
		return errors.NewConfigError("config", fmt.Sprintf("unsupported config file format: %s", ext), nil)
	}

	return os.WriteFile(filePath, data, 0o600)
}

// Helper to convert struct to map
func structToMap(obj interface{}) map[string]interface{} {
	// Convert to JSON
	jsonData, err := json.Marshal(obj)
	if err != nil {
		// This shouldn't fail since we're converting from a valid struct
		// But log and return empty map just in case
		log.Warn().Err(err).Msg("Failed to marshal struct to JSON")
		return make(map[string]interface{})
	}

	// Convert JSON to map
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		// This shouldn't fail since we're converting from a valid JSON
		// But log a warning just in case
		log.Warn().Err(err).Msg("Failed to unmarshal struct to map")
	}

	return result
}

// Helper functions to get values from environment variables
// GetEnv gets an environment variable or returns a default value
func GetEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// GetEnvInt gets an integer environment variable or returns a default value
func GetEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return fallback
}

// GetEnvDuration gets a duration environment variable or returns a default value
func GetEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if result, err := time.ParseDuration(value); err == nil {
			return result
		}
	}
	return fallback
}

// GetBoolEnv gets a boolean environment variable or returns a default value
func GetBoolEnv(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if value == "true" || value == "1" || value == "yes" {
			return true
		}
		if value == "false" || value == "0" || value == "no" {
			return false
		}
	}
	return fallback
}

// Unexported versions for internal use
func getEnv(key, fallback string) string {
	return GetEnv(key, fallback)
}

func getEnvInt(key string, fallback int) int {
	return GetEnvInt(key, fallback)
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	return GetEnvDuration(key, fallback)
}

func getBoolEnv(key string, fallback bool) bool {
	return GetBoolEnv(key, fallback)
}

// loadEnvFile attempts to load environment variables from .env file in the current directory
func loadEnvFile() {
	// Try to get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get current working directory")
		return
	}

	// Define possible .env file locations
	envFiles := []string{
		filepath.Join(cwd, ".env"),                           // .env in current directory
		filepath.Join(cwd, "..", ".env"),                     // .env in parent directory (for subcommands)
		filepath.Join(cwd, "../..", ".env"),                  // .env two levels up (for deeper nesting)
		filepath.Join(os.Getenv("HOME"), ".hubsync", ".env"), // .env in user's home config directory
	}

	// Try loading from each location
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			// Found a .env file, try to load it
			err := godotenv.Load(envFile)
			if err != nil {
				log.Debug().Err(err).Str("file", envFile).Msg("Failed to load .env file")
			} else {
				log.Debug().Str("file", envFile).Msg("Loaded environment variables from .env file")
				return
			}
		}
	}

	log.Debug().Msg("No .env file found in standard locations")
}

// LoadEnvFileForTesting exports the loadEnvFile function for testing purposes
func LoadEnvFileForTesting() {
	loadEnvFile()
}
