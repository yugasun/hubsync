package unit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/sync"
	"github.com/yugasun/hubsync/test/mocks"
)

// TestSyncerProcessImage tests the image processing functionality
func TestSyncerProcessImage(t *testing.T) {
	// Create a temporary directory for output files
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.log")

	t.Run("Successful Image Processing", func(t *testing.T) {
		// Create a mock Docker client
		mockClient := mocks.NewMockDockerClient()

		// Create configuration
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock client
		syncer := &sync.Syncer{
			Config: cfg,
			Client: mockClient,
		}

		// Process a single image
		ctx := context.Background()
		source := "nginx:latest"

		// Execute the test
		source, target, err := syncer.ProcessImage(ctx, source)

		// Validate results
		require.NoError(t, err)
		assert.Equal(t, "nginx:latest", source)
		assert.Contains(t, target, "testns/nginx:latest")
		assert.True(t, mockClient.PulledImages[source])
		assert.Equal(t, target, mockClient.TaggedImages[source])
		assert.True(t, mockClient.PushedImages[target])
	})

	t.Run("Pull Failure", func(t *testing.T) {
		// Create a mock Docker client with pull error
		mockClient := mocks.NewMockDockerClient()
		mockClient.PullErrors["nginx:latest"] = fmt.Errorf("network error")

		// Create configuration
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock client
		syncer := &sync.Syncer{
			Config: cfg,
			Client: mockClient,
		}

		// Process a single image
		ctx := context.Background()
		source := "nginx:latest"

		// Execute the test
		_, _, err := syncer.ProcessImage(ctx, source)

		// Validate results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to pull image")
	})

	t.Run("Tag Failure", func(t *testing.T) {
		// Create a mock Docker client with tag error
		mockClient := mocks.NewMockDockerClient()
		mockClient.TagErrors["nginx:latest"] = fmt.Errorf("tag error")

		// Create configuration
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock client
		syncer := &sync.Syncer{
			Config: cfg,
			Client: mockClient,
		}

		// Process a single image
		ctx := context.Background()
		source := "nginx:latest"

		// Execute the test
		_, _, err := syncer.ProcessImage(ctx, source)

		// Validate results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to tag image")
	})

	t.Run("Push Failure", func(t *testing.T) {
		// Create a mock Docker client with push error
		mockClient := mocks.NewMockDockerClient()
		// We need to set up the scenario for the push error
		target := "docker.io/testns/nginx:latest"
		mockClient.PushErrors[target] = fmt.Errorf("push error")

		// Create configuration
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock client
		syncer := &sync.Syncer{
			Config: cfg,
			Client: mockClient,
		}

		// Process a single image
		ctx := context.Background()
		source := "nginx:latest"

		// Execute the test
		_, _, err := syncer.ProcessImage(ctx, source)

		// Validate results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to push image")
	})
}

// TestSyncerV2Run tests the full sync process with SyncerV2
func TestSyncerV2Run(t *testing.T) {
	// Create a temporary directory for output files
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.log")

	t.Run("Successful Run", func(t *testing.T) {
		// Create mock clients
		mockDockerClient := mocks.NewMockDockerClient()
		mockRegistryClient := mocks.NewMockRegistryClient()

		// Create configuration with multiple images
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest", "alpine:3.18"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 2,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock clients
		syncer := sync.NewSyncerV2(cfg, mockDockerClient, mockRegistryClient)

		// Run the sync process
		ctx := context.Background()
		err := syncer.Run(ctx)

		// Validate results
		require.NoError(t, err)
		assert.Equal(t, 2, syncer.GetProcessedImageCount())

		// Check output file
		_, err = os.Stat(outputPath)
		assert.NoError(t, err, "Output file should exist")

		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "docker pull")
		assert.Contains(t, content, "docker.io/testns/nginx:latest")
		assert.Contains(t, content, "docker.io/testns/alpine:3.18")
	})

	t.Run("Content Parse Error", func(t *testing.T) {
		// Create mock clients
		mockDockerClient := mocks.NewMockDockerClient()
		mockRegistryClient := mocks.NewMockRegistryClient()

		// Create configuration with invalid JSON
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `invalid json`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock clients
		syncer := sync.NewSyncerV2(cfg, mockDockerClient, mockRegistryClient)

		// Run the sync process
		ctx := context.Background()
		err := syncer.Run(ctx)

		// Validate results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse content")
	})

	t.Run("Too Many Images", func(t *testing.T) {
		// Create mock clients
		mockDockerClient := mocks.NewMockDockerClient()
		mockRegistryClient := mocks.NewMockRegistryClient()

		// Create configuration with too many images
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest", "alpine:3.18", "ubuntu:22.04"]}`,
			MaxContent:  2, // Limit is set to 2, but we have 3 images
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock clients
		syncer := sync.NewSyncerV2(cfg, mockDockerClient, mockRegistryClient)

		// Run the sync process
		ctx := context.Background()
		err := syncer.Run(ctx)

		// Validate results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many images")
	})

	t.Run("Image Sync Failure", func(t *testing.T) {
		// Create mock clients with error on push
		mockDockerClient := mocks.NewMockDockerClient()
		mockRegistryClient := mocks.NewMockRegistryClient()

		// Set up failure for docker operations
		mockDockerClient.PushErrors = map[string]error{
			"docker.io/testns/nginx:latest": fmt.Errorf("push error"),
		}

		// Create configuration
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock clients
		syncer := sync.NewSyncerV2(cfg, mockDockerClient, mockRegistryClient)

		// Run the sync process
		ctx := context.Background()
		err := syncer.Run(ctx)

		// In SyncerV2, errors during sync operations don't stop the Run function
		// They're recorded in the results and continue processing
		require.NoError(t, err)

		// Check output file for failed operations
		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "failed to sync")
	})
}

// TestSyncerCustomImageNames tests custom image naming with SyncerV2
func TestSyncerCustomImageNames(t *testing.T) {
	// Create a temporary directory for output files
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.log")

	t.Run("Custom Image Name", func(t *testing.T) {
		// Create mock clients
		mockDockerClient := mocks.NewMockDockerClient()
		mockRegistryClient := mocks.NewMockRegistryClient()

		// Create configuration with custom image name
		cfg := &config.Config{
			Username:    "testuser",
			Password:    "testpass",
			Repository:  "docker.io",
			Namespace:   "testns",
			Content:     `{"hubsync": ["nginx:latest$custom-nginx"]}`,
			MaxContent:  10,
			OutputPath:  outputPath,
			Concurrency: 1,
			Timeout:     10 * time.Second,
		}

		// Create syncer with mock clients
		syncer := sync.NewSyncerV2(cfg, mockDockerClient, mockRegistryClient)

		// Run the sync process
		ctx := context.Background()
		err := syncer.Run(ctx)

		// Validate results
		require.NoError(t, err)

		// Check output file
		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "docker pull")
		assert.Contains(t, content, "docker.io/testns/custom-nginx:latest")
	})
}
