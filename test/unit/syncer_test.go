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

// TestSyncerRun tests the full sync process
func TestSyncerRun(t *testing.T) {
	// Create a temporary directory for output files
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.log")

	t.Run("Successful Run", func(t *testing.T) {
		// Create a mock Docker client
		mockClient := mocks.NewMockDockerClient()

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

		// Create syncer with mock client
		syncer := sync.NewSyncer(cfg, mockClient)

		// Run the sync process
		ctx := context.Background()
		err := syncer.Run(ctx)

		// Validate results
		require.NoError(t, err)
		assert.Equal(t, 2, syncer.GetProcessedImageCount())
		assert.True(t, mockClient.PulledImages["nginx:latest"])
		assert.True(t, mockClient.PulledImages["alpine:3.18"])

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
}
