package strategies

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/errors"
)

// StandardStrategy implements a sequential synchronization strategy
type StandardStrategy struct {
	dockerClient docker.ClientInterface
}

// Ensure StandardStrategy implements SyncStrategy
var _ SyncStrategy = (*StandardStrategy)(nil)

// NewStandardStrategy creates a new standard (sequential) sync strategy
func NewStandardStrategy(dockerClient docker.ClientInterface) *StandardStrategy {
	return &StandardStrategy{
		dockerClient: dockerClient,
	}
}

// Name returns the name of the strategy
func (s *StandardStrategy) Name() string {
	return "standard"
}

// Execute performs sequential image synchronization
func (s *StandardStrategy) Execute(ctx context.Context, operations []*SyncOperation) ([]*SyncResult, error) {
	results := make([]*SyncResult, 0, len(operations))

	for _, op := range operations {
		// Check for context cancellation
		if ctx.Err() != nil {
			return results, errors.NewContextError(
				"sync",
				"context cancelled during sync execution",
				ctx.Err(),
			)
		}

		result := s.executeSingleOperation(ctx, op)
		results = append(results, result)

		// Log result
		if result.Success {
			log.Info().
				Str("source", op.Source.FullName).
				Str("target", op.Target.FullName).
				Int64("duration_ms", result.Duration).
				Int64("bytes", result.BytesMoved).
				Msg("Image sync successful")
		} else {
			log.Error().
				Str("source", op.Source.FullName).
				Str("target", op.Target.FullName).
				Err(result.Error).
				Msg("Image sync failed")
		}
	}

	return results, nil
}

// executeSingleOperation synchronizes a single image
func (s *StandardStrategy) executeSingleOperation(ctx context.Context, op *SyncOperation) *SyncResult {
	result := &SyncResult{
		Operation: op,
		StartTime: time.Now().Unix(),
		Success:   false,
	}

	defer func() {
		result.EndTime = time.Now().Unix()
		result.Duration = result.EndTime - result.StartTime
	}()

	// Add context to logs
	opLog := log.With().
		Str("source", op.Source.FullName).
		Str("target", op.Target.FullName).
		Bool("dry_run", op.DryRun).
		Logger()

	// If this is a dry run, just log and return success
	if op.DryRun {
		opLog.Info().Msg("Dry run: would sync image")
		result.Success = true
		result.DetailedLogs = append(result.DetailedLogs, fmt.Sprintf("Dry run: would sync from %s to %s",
			op.Source.FullName, op.Target.FullName))
		return result
	}

	// Step 1: Pull the source image
	opLog.Debug().Msg("Pulling source image")
	result.DetailedLogs = append(result.DetailedLogs, fmt.Sprintf("Pulling source image: %s", op.Source.FullName))

	if err := s.dockerClient.PullImage(ctx, op.Source.FullName); err != nil {
		opLog.Error().Err(err).Msg("Failed to pull source image")
		result.Error = errors.NewOperationError("sync", "failed to pull source image", err)
		result.DetailedLogs = append(result.DetailedLogs, fmt.Sprintf("Pull failed: %v", err))
		return result
	}

	// Step 2: Tag the image with the target name
	opLog.Debug().Msg("Tagging image")
	result.DetailedLogs = append(result.DetailedLogs,
		fmt.Sprintf("Tagging image from %s to %s", op.Source.FullName, op.Target.FullName))

	if err := s.dockerClient.TagImage(ctx, op.Source.FullName, op.Target.FullName); err != nil {
		opLog.Error().Err(err).Msg("Failed to tag image")
		result.Error = errors.NewOperationError("sync", "failed to tag image", err)
		result.DetailedLogs = append(result.DetailedLogs, fmt.Sprintf("Tag failed: %v", err))
		return result
	}

	// Step 3: Push the tagged image to the target registry
	opLog.Debug().Msg("Pushing target image")
	result.DetailedLogs = append(result.DetailedLogs, fmt.Sprintf("Pushing target image: %s", op.Target.FullName))

	if err := s.dockerClient.PushImage(ctx, op.Target.FullName); err != nil {
		opLog.Error().Err(err).Msg("Failed to push target image")
		result.Error = errors.NewOperationError("sync", "failed to push target image", err)
		result.DetailedLogs = append(result.DetailedLogs, fmt.Sprintf("Push failed: %v", err))
		return result
	}

	// Set success
	result.Success = true
	result.DetailedLogs = append(result.DetailedLogs, "Synchronization completed successfully")

	return result
}
