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
	"github.com/yugasun/hubsync/internal/di"
	"github.com/yugasun/hubsync/pkg/errors"
)

// Run executes the main application logic
func Run(version string) error {
	log.Info().Str("version", version).Msg("Starting HubSync")

	// Parse configuration from command-line flags and environment variables
	cfg, err := config.ParseConfig()
	if err != nil {
		return errors.NewConfigError("app", "configuration error", err)
	}

	// Set version in config
	cfg.Version = version

	// Handle version flag
	if cfg.ShowVersion {
		fmt.Printf("HubSync version %s\n", version)
		return nil
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

	// Initialize dependency injection container
	container := di.GetInstance()
	if err := container.Initialize(cfg); err != nil {
		return errors.NewSystemError("app", "failed to initialize application container", err)
	}
	defer container.Cleanup()

	// Create syncer with timeout
	syncerCtx, syncerCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer syncerCancel()

	// Get syncer from container
	syncer := container.GetSyncer()

	// Run the sync operation
	log.Info().Msg("Starting image synchronization")
	startTime := time.Now()

	err = syncer.Run(syncerCtx)
	if err != nil {
		return errors.NewOperationError("app", "synchronization error", err)
	}

	log.Info().
		Dur("duration", time.Since(startTime)).
		Int("images_processed", syncer.GetProcessedImageCount()).
		Msg("Image synchronization completed successfully")

	return nil
}
