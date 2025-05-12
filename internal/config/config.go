package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
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
	LogLevel    string
	LogFile     string
	ShowVersion bool
}

// ParseConfig parses command-line flags and environment variables
func ParseConfig() (*Config, error) {
	// Load environment variables from .env file if present
	loadEnvFile()

	cfg := &Config{}

	// Define command-line flags with environment variable fallbacks
	pflag.StringVar(&cfg.Username, "username", getEnv("DOCKER_USERNAME", ""), "Docker registry username")
	pflag.StringVar(&cfg.Password, "password", getEnv("DOCKER_PASSWORD", ""), "Docker registry password")
	pflag.StringVar(&cfg.Repository, "repository", getEnv("DOCKER_REPOSITORY", ""), "Target repository address")
	pflag.StringVar(&cfg.Namespace, "namespace", getEnv("DOCKER_NAMESPACE", "yugasun"), "Target namespace")
	pflag.StringVar(&cfg.Content, "content", getEnv("CONTENT", ""), "JSON content with images to sync")
	pflag.IntVar(&cfg.MaxContent, "maxContent", getEnvInt("MAX_CONTENT", 10), "Maximum number of images to process")
	pflag.StringVar(&cfg.OutputPath, "outputPath", getEnv("OUTPUT_PATH", "output.log"), "Output file path")

	// Performance settings
	pflag.IntVar(&cfg.Concurrency, "concurrency", getEnvInt("CONCURRENCY", 3), "Maximum concurrent operations")
	pflag.DurationVar(&cfg.Timeout, "timeout", getEnvDuration("TIMEOUT", 10*time.Minute), "Operation timeout")
	pflag.IntVar(&cfg.RetryCount, "retryCount", getEnvInt("RETRY_COUNT", 3), "Number of retries for failed operations")
	pflag.DurationVar(&cfg.RetryDelay, "retryDelay", getEnvDuration("RETRY_DELAY", 2*time.Second), "Delay between retries")

	// Advanced settings
	pflag.StringVar(&cfg.LogLevel, "logLevel", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	pflag.StringVar(&cfg.LogFile, "logFile", getEnv("LOG_FILE", ""), "Log to file in addition to stdout")

	// Help and version
	help := pflag.BoolP("help", "h", false, "Show this help message")
	version := pflag.BoolP("version", "v", false, "Show version information")

	// Parse flags
	pflag.Parse()

	// Show help if requested
	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Store version flag
	cfg.ShowVersion = *version

	// Basic validation
	if cfg.Username == "" && !cfg.ShowVersion {
		return nil, fmt.Errorf("username is required")
	}

	if cfg.Password == "" && !cfg.ShowVersion {
		return nil, fmt.Errorf("password is required")
	}

	if cfg.Content == "" && !cfg.ShowVersion {
		return nil, fmt.Errorf("content is required")
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
		Msg("Configuration loaded")

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	if c.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

// Helper functions to get values from environment variables
// GetEnv gets an environment variable or returns a default value
func GetEnv(key, fallback string) string {
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
		filepath.Join(cwd, ".env"),          // .env in current directory
		filepath.Join(cwd, "..", ".env"),    // .env in parent directory (for subcommands)
		filepath.Join(cwd, "../..", ".env"), // .env two levels up (for deeper nesting)
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

	log.Debug().Msg("No .env file found in current directory or parent directories")
}

// LoadEnvFileForTesting exports the loadEnvFile function for testing purposes
func LoadEnvFileForTesting() {
	loadEnvFile()
}
