package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/client"
	"github.com/yugasun/hubsync/pkg/sync"
)

// Run executes the main application logic
func Run(version string) error {
	log.Info().Str("version", version).Msg("Starting HubSync")

	// Parse configuration from command-line flags and environment variables
	cfg, err := config.ParseConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalCh
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	// Create Docker client
	dockerClient, err := client.NewDockerClient(cfg.Username, cfg.Password, cfg.Repository)
	if err != nil {
		return fmt.Errorf("failed to initialize Docker client: %w", err)
	}
	defer dockerClient.Close()

	// Create syncer with timeout
	syncerCtx, syncerCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer syncerCancel()

	syncer := sync.NewSyncer(cfg, dockerClient)

	// Run the sync operation
	log.Info().Msg("Starting image synchronization")
	startTime := time.Now()

	err = syncer.Run(syncerCtx)
	if err != nil {
		return fmt.Errorf("synchronization error: %w", err)
	}

	log.Info().
		Dur("duration", time.Since(startTime)).
		Int("images_processed", syncer.GetProcessedImageCount()).
		Msg("Image synchronization completed successfully")

	return nil
}
