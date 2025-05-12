//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/registry"
	"github.com/yugasun/hubsync/pkg/sync"
)

// TestIntegration performs integration testing with real Docker operations
func TestIntegration(t *testing.T) {
	// Load environment variables from .env file if present
	_ = godotenv.Load("../../.env")

	// Skip if Docker credentials are not available
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if username == "" || password == "" {
		t.Skip("Skipping integration test: DOCKER_USERNAME or DOCKER_PASSWORD not set")
	}

	// Create a temporary directory for output
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.log")

	t.Run("Simple Image Sync", func(t *testing.T) {
		// Use a very small image for quick testing
		contentObj := map[string][]string{
			"hubsync": {"alpine:3.18"},
		}
		contentBytes, _ := json.Marshal(contentObj)
		content := string(contentBytes)

		// Create configuration
		cfg := &config.Config{
			Username:    username,
			Password:    password,
			Repository:  "", // Use Docker Hub
			Namespace:   username,
			Content:     content,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			LogLevel:    "info",
			Timeout:     5 * time.Minute,
		}

		// Create Docker client
		dockerConfig := docker.ClientConfig{
			Username:    username,
			Password:    password,
			Repository:  "",
			RetryCount:  3,
			RetryDelay:  5 * time.Second,
			PullTimeout: 5 * time.Minute,
			PushTimeout: 5 * time.Minute,
		}

		dockerClient, err := docker.NewClient(dockerConfig)
		require.NoError(t, err, "Failed to create Docker client")
		defer dockerClient.Close()

		// Create registry client
		registryConfig := registry.RegistryConfig{
			Provider: registry.DockerHub,
			Username: username,
			Password: password,
		}

		registryClient := registry.NewDockerHubRegistry(registryConfig)

		// Create syncer
		syncer := sync.NewSyncerV2(cfg, dockerClient, registryClient)

		// Run sync process
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		err = syncer.Run(ctx)
		require.NoError(t, err, "Sync process failed")

		// Check output file
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err, "Failed to read output file")

		assert.Contains(t, string(data), "docker pull")
		assert.Contains(t, string(data), username+"/alpine:3.18")
	})

	t.Run("Custom Image Name", func(t *testing.T) {
		// Use custom image naming
		contentObj := map[string][]string{
			"hubsync": {"nginx:1.25.0$custom-nginx"},
		}
		contentBytes, _ := json.Marshal(contentObj)
		content := string(contentBytes)

		// Create configuration
		cfg := &config.Config{
			Username:    username,
			Password:    password,
			Repository:  "", // Use Docker Hub
			Namespace:   username,
			Content:     content,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			LogLevel:    "info",
			Timeout:     5 * time.Minute,
		}

		// Create Docker client
		dockerConfig := docker.ClientConfig{
			Username:    username,
			Password:    password,
			Repository:  "",
			RetryCount:  3,
			RetryDelay:  5 * time.Second,
			PullTimeout: 5 * time.Minute,
			PushTimeout: 5 * time.Minute,
		}

		dockerClient, err := docker.NewClient(dockerConfig)
		require.NoError(t, err, "Failed to create Docker client")
		defer dockerClient.Close()

		// Create registry client
		registryConfig := registry.RegistryConfig{
			Provider: registry.DockerHub,
			Username: username,
			Password: password,
		}

		registryClient := registry.NewDockerHubRegistry(registryConfig)

		// Create syncer
		syncer := sync.NewSyncerV2(cfg, dockerClient, registryClient)

		// Run sync process
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		err = syncer.Run(ctx)
		require.NoError(t, err, "Sync process failed")

		// Check output file
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err, "Failed to read output file")

		outputContent := string(data)
		assert.Contains(t, outputContent, "docker pull")
		assert.True(t, strings.Contains(outputContent, "custom-nginx:1.25.0") ||
			strings.Contains(outputContent, username+"/custom-nginx:1.25.0"),
			"Output should contain the custom image name")
	})

	t.Run("Multiple Images", func(t *testing.T) {
		// Use multiple images
		contentObj := map[string][]string{
			"hubsync": {"alpine:3.18", "busybox:latest"},
		}
		contentBytes, _ := json.Marshal(contentObj)
		content := string(contentBytes)

		// Create configuration
		cfg := &config.Config{
			Username:    username,
			Password:    password,
			Repository:  "", // Use Docker Hub
			Namespace:   username,
			Content:     content,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 2, // Use concurrency for multiple images
			LogLevel:    "info",
			Timeout:     5 * time.Minute,
		}

		// Create Docker client
		dockerConfig := docker.ClientConfig{
			Username:    username,
			Password:    password,
			Repository:  "",
			RetryCount:  3,
			RetryDelay:  5 * time.Second,
			PullTimeout: 5 * time.Minute,
			PushTimeout: 5 * time.Minute,
		}

		dockerClient, err := docker.NewClient(dockerConfig)
		require.NoError(t, err, "Failed to create Docker client")
		defer dockerClient.Close()

		// Create registry client
		registryConfig := registry.RegistryConfig{
			Provider: registry.DockerHub,
			Username: username,
			Password: password,
		}

		registryClient := registry.NewDockerHubRegistry(registryConfig)

		// Create syncer
		syncer := sync.NewSyncerV2(cfg, dockerClient, registryClient)

		// Run sync process
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		err = syncer.Run(ctx)
		require.NoError(t, err, "Sync process failed")

		// Check output file
		data, err := os.ReadFile(outputPath)
		require.NoError(t, err, "Failed to read output file")

		outputContent := string(data)
		assert.Contains(t, outputContent, username+"/alpine:3.18")
		assert.Contains(t, outputContent, username+"/busybox:latest")
	})
}
