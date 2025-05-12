package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yugasun/hubsync/internal/config"
)

// TestEnvFileLoading tests the automatic loading of environment variables from .env file
func TestEnvFileLoading(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// List of environment variables to manage
	envVars := []string{
		"DOCKER_USERNAME",
		"DOCKER_PASSWORD",
		"CUSTOM_TEST_VAR",
		"SHARED_VAR",
		"PARENT_ONLY",
		"CHILD_ONLY",
	}

	// Save current environment variables
	originalEnv := make(map[string]string)
	for _, name := range envVars {
		originalEnv[name] = os.Getenv(name)
	}

	// Cleanup environment after test
	defer func() {
		for name, value := range originalEnv {
			if value == "" {
				os.Unsetenv(name)
			} else {
				os.Setenv(name, value)
			}
		}
	}()

	// Helper to reset environment variables before each test
	resetEnv := func() {
		for _, name := range envVars {
			os.Unsetenv(name)
		}
	}

	t.Run("Loads variables from .env file", func(t *testing.T) {
		resetEnv()

		// Create a .env file in the temporary directory
		envFilePath := filepath.Join(tmpDir, ".env")
		envContent := `DOCKER_USERNAME=testuser
DOCKER_PASSWORD=testpassword
CUSTOM_TEST_VAR=testvalue`

		err := os.WriteFile(envFilePath, []byte(envContent), 0o644)
		require.NoError(t, err, "Failed to write .env file")

		// Remember current directory
		currentDir, err := os.Getwd()
		require.NoError(t, err, "Failed to get current directory")

		// Change to the temporary directory for the test
		err = os.Chdir(tmpDir)
		require.NoError(t, err, "Failed to change to temp directory")

		// Restore the original directory when test finishes
		defer func() {
			err := os.Chdir(currentDir)
			require.NoError(t, err, "Failed to restore original directory")
		}()

		// Test the env loading through the exported function
		config.LoadEnvFileForTesting()

		// Check that environment variables were loaded
		assert.Equal(t, "testuser", os.Getenv("DOCKER_USERNAME"), "DOCKER_USERNAME should be loaded from .env file")
		assert.Equal(t, "testpassword", os.Getenv("DOCKER_PASSWORD"), "DOCKER_PASSWORD should be loaded from .env file")
		assert.Equal(t, "testvalue", os.Getenv("CUSTOM_TEST_VAR"), "CUSTOM_TEST_VAR should be loaded from .env file")
	})

	t.Run("Loads variables from parent directory", func(t *testing.T) {
		resetEnv()

		// Create a nested directory structure
		parentDir := filepath.Join(tmpDir, "parent")
		childDir := filepath.Join(parentDir, "child")

		err := os.MkdirAll(childDir, 0o755)
		require.NoError(t, err, "Failed to create nested directory structure")

		// Create a .env file in the parent directory
		envFilePath := filepath.Join(parentDir, ".env")
		envContent := `DOCKER_USERNAME=parentuser
DOCKER_PASSWORD=parentpass
CUSTOM_TEST_VAR=parentvalue`

		err = os.WriteFile(envFilePath, []byte(envContent), 0o644)
		require.NoError(t, err, "Failed to write .env file in parent directory")

		// Make sure there's no .env file in the child directory or test root
		// that could interfere with the test
		os.Remove(filepath.Join(childDir, ".env"))
		os.Remove(filepath.Join(tmpDir, ".env"))

		// Remember current directory
		currentDir, err := os.Getwd()
		require.NoError(t, err, "Failed to get current directory")

		// Change to the child directory for the test
		err = os.Chdir(childDir)
		require.NoError(t, err, "Failed to change to child directory")

		// Restore the original directory when test finishes
		defer func() {
			err := os.Chdir(currentDir)
			require.NoError(t, err, "Failed to restore original directory")
		}()

		// Test the env loading through the exported function
		config.LoadEnvFileForTesting()

		// Check that environment variables were loaded from parent directory
		assert.Equal(t, "parentuser", os.Getenv("DOCKER_USERNAME"), "DOCKER_USERNAME should be loaded from parent .env file")
		assert.Equal(t, "parentpass", os.Getenv("DOCKER_PASSWORD"), "DOCKER_PASSWORD should be loaded from parent .env file")
		assert.Equal(t, "parentvalue", os.Getenv("CUSTOM_TEST_VAR"), "CUSTOM_TEST_VAR should be loaded from parent .env file")
	})

	t.Run("Child .env takes precedence over parent", func(t *testing.T) {
		resetEnv()

		// Create a nested directory structure
		parentDir := filepath.Join(tmpDir, "precedence-parent")
		childDir := filepath.Join(parentDir, "precedence-child")

		err := os.MkdirAll(childDir, 0o755)
		require.NoError(t, err, "Failed to create nested directory structure")

		// Create a .env file in the parent directory
		parentEnvPath := filepath.Join(parentDir, ".env")
		parentEnvContent := `DOCKER_USERNAME=parentuser
DOCKER_PASSWORD=parentpass
SHARED_VAR=parent-value
PARENT_ONLY=parent-specific`

		err = os.WriteFile(parentEnvPath, []byte(parentEnvContent), 0o644)
		require.NoError(t, err, "Failed to write parent .env file")

		// Create a .env file in the child directory
		childEnvPath := filepath.Join(childDir, ".env")
		childEnvContent := `DOCKER_USERNAME=childuser
SHARED_VAR=child-value
CHILD_ONLY=child-specific`

		err = os.WriteFile(childEnvPath, []byte(childEnvContent), 0o644)
		require.NoError(t, err, "Failed to write child .env file")

		// Make sure there's no .env file in the test root that could interfere
		os.Remove(filepath.Join(tmpDir, ".env"))

		// Remember current directory
		currentDir, err := os.Getwd()
		require.NoError(t, err, "Failed to get current directory")

		// Change to the child directory for the test
		err = os.Chdir(childDir)
		require.NoError(t, err, "Failed to change to child directory")

		// Restore the original directory when test finishes
		defer func() {
			err := os.Chdir(currentDir)
			require.NoError(t, err, "Failed to restore original directory")
		}()

		// Test the env loading through the exported function
		config.LoadEnvFileForTesting()

		// Check that environment variables were loaded with correct precedence
		assert.Equal(t, "childuser", os.Getenv("DOCKER_USERNAME"), "Child directory .env file should take precedence")
		assert.Equal(t, "child-value", os.Getenv("SHARED_VAR"), "Child directory value should override parent directory value")
		assert.Equal(t, "child-specific", os.Getenv("CHILD_ONLY"), "Child-specific values should be loaded")
		// Parent-only values won't be loaded since we just check the current directory in our implementation
		// If we wanted to merge parent and child values, we would need to modify the loadEnvFile function
		assert.Equal(t, "", os.Getenv("PARENT_ONLY"), "Parent-only values should not be loaded when child .env exists")
	})

	t.Run("No .env file", func(t *testing.T) {
		resetEnv()

		// Create an empty directory
		emptyDir := filepath.Join(tmpDir, "empty")

		err := os.MkdirAll(emptyDir, 0o755)
		require.NoError(t, err, "Failed to create empty directory")

		// Make sure there's no .env file anywhere that could interfere
		os.Remove(filepath.Join(emptyDir, ".env"))
		os.Remove(filepath.Join(tmpDir, ".env"))
		os.Remove(filepath.Join(emptyDir, "..", ".env"))

		// Remember current directory
		currentDir, err := os.Getwd()
		require.NoError(t, err, "Failed to get current directory")

		// Change to the empty directory for the test
		err = os.Chdir(emptyDir)
		require.NoError(t, err, "Failed to change to empty directory")

		// Restore the original directory when test finishes
		defer func() {
			err := os.Chdir(currentDir)
			require.NoError(t, err, "Failed to restore original directory")
		}()

		// Explicitly set test values to check they don't get overwritten
		os.Setenv("DOCKER_USERNAME", "manual-user")
		os.Setenv("DOCKER_PASSWORD", "manual-pass")

		// Test the env loading through the exported function
		config.LoadEnvFileForTesting()

		// Check that existing environment variables remain unchanged
		assert.Equal(t, "manual-user", os.Getenv("DOCKER_USERNAME"), "Environment variables should not be changed when no .env file exists")
		assert.Equal(t, "manual-pass", os.Getenv("DOCKER_PASSWORD"), "Environment variables should not be changed when no .env file exists")
	})
}
