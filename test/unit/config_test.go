package unit

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yugasun/hubsync/internal/config"
)

// TestConfigEnvironmentVars tests the environment variable handling in configuration
func TestConfigEnvironmentVars(t *testing.T) {
	// Save original environment and restore after test
	originalEnv := map[string]string{
		"DOCKER_USERNAME":   os.Getenv("DOCKER_USERNAME"),
		"DOCKER_PASSWORD":   os.Getenv("DOCKER_PASSWORD"),
		"DOCKER_REPOSITORY": os.Getenv("DOCKER_REPOSITORY"),
		"DOCKER_NAMESPACE":  os.Getenv("DOCKER_NAMESPACE"),
		"CONTENT":           os.Getenv("CONTENT"),
		"MAX_CONTENT":       os.Getenv("MAX_CONTENT"),
		"CONCURRENCY":       os.Getenv("CONCURRENCY"),
		"TIMEOUT":           os.Getenv("TIMEOUT"),
		"OUTPUT_PATH":       os.Getenv("OUTPUT_PATH"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	t.Run("Default Values", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("DOCKER_USERNAME")
		os.Unsetenv("DOCKER_PASSWORD")
		os.Unsetenv("DOCKER_REPOSITORY")
		os.Unsetenv("DOCKER_NAMESPACE")
		os.Unsetenv("CONTENT")
		os.Unsetenv("MAX_CONTENT")
		os.Unsetenv("CONCURRENCY")
		os.Unsetenv("TIMEOUT")
		os.Unsetenv("OUTPUT_PATH")

		// Test default values
		assert.Equal(t, "", config.GetEnv("DOCKER_USERNAME", ""))
		assert.Equal(t, "default", config.GetEnv("DOCKER_USERNAME", "default"))
		assert.Equal(t, 10, config.GetEnvInt("MAX_CONTENT", 10))
		assert.Equal(t, 5*time.Minute, config.GetEnvDuration("TIMEOUT", 5*time.Minute))
	})

	t.Run("Set Environment Variables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("DOCKER_USERNAME", "test-user")
		os.Setenv("DOCKER_PASSWORD", "test-pass")
		os.Setenv("DOCKER_REPOSITORY", "test-repo")
		os.Setenv("DOCKER_NAMESPACE", "test-ns")
		os.Setenv("CONTENT", `{"hubsync": ["nginx:latest"]}`)
		os.Setenv("MAX_CONTENT", "5")
		os.Setenv("CONCURRENCY", "3")
		os.Setenv("TIMEOUT", "2m")
		os.Setenv("OUTPUT_PATH", "test-output.log")

		// Test reading env variables
		assert.Equal(t, "test-user", config.GetEnv("DOCKER_USERNAME", "default"))
		assert.Equal(t, "test-pass", config.GetEnv("DOCKER_PASSWORD", "default"))
		assert.Equal(t, "test-repo", config.GetEnv("DOCKER_REPOSITORY", "default"))
		assert.Equal(t, "test-ns", config.GetEnv("DOCKER_NAMESPACE", "default"))
		assert.Equal(t, `{"hubsync": ["nginx:latest"]}`, config.GetEnv("CONTENT", "default"))
		assert.Equal(t, 5, config.GetEnvInt("MAX_CONTENT", 10))
		assert.Equal(t, 3, config.GetEnvInt("CONCURRENCY", 1))
		assert.Equal(t, 2*time.Minute, config.GetEnvDuration("TIMEOUT", 1*time.Minute))
		assert.Equal(t, "test-output.log", config.GetEnv("OUTPUT_PATH", "default.log"))
	})

	t.Run("Invalid Values", func(t *testing.T) {
		// Set invalid environment variables
		os.Setenv("MAX_CONTENT", "invalid")
		os.Setenv("TIMEOUT", "invalid")

		// Test handling of invalid values (should return defaults)
		assert.Equal(t, 10, config.GetEnvInt("MAX_CONTENT", 10))
		assert.Equal(t, 5*time.Minute, config.GetEnvDuration("TIMEOUT", 5*time.Minute))
	})
}

// TestConfigValidation tests the configuration validation
func TestConfigValidation(t *testing.T) {
	t.Run("Valid Configuration", func(t *testing.T) {
		cfg := &config.Config{
			Username:    "test-user",
			Password:    "test-pass",
			Repository:  "docker.io",
			Namespace:   "test-ns",
			Content:     `{"hubsync": ["nginx:latest"]}`,
			MaxContent:  10,
			OutputPath:  "output.log",
			LogLevel:    "info",
			Concurrency: 1,
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Missing Username", func(t *testing.T) {
		cfg := &config.Config{
			Password:   "test-pass",
			Repository: "docker.io",
			Namespace:  "test-ns",
			Content:    `{"hubsync": ["nginx:latest"]}`,
			MaxContent: 10,
			OutputPath: "output.log",
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username is required")
	})

	t.Run("Missing Password", func(t *testing.T) {
		cfg := &config.Config{
			Username:   "test-user",
			Repository: "docker.io",
			Namespace:  "test-ns",
			Content:    `{"hubsync": ["nginx:latest"]}`,
			MaxContent: 10,
			OutputPath: "output.log",
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("Missing Content", func(t *testing.T) {
		cfg := &config.Config{
			Username:   "test-user",
			Password:   "test-pass",
			Repository: "docker.io",
			Namespace:  "test-ns",
			MaxContent: 10,
			OutputPath: "output.log",
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is required")
	})
}
