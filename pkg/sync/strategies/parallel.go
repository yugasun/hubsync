package strategies

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/errors"
)

// ParallelStrategy implements a concurrent synchronization strategy
type ParallelStrategy struct {
	dockerClient docker.ClientInterface
	concurrency  int
}

// Ensure ParallelStrategy implements SyncStrategy
var _ SyncStrategy = (*ParallelStrategy)(nil)

// NewParallelStrategy creates a new parallel sync strategy
func NewParallelStrategy(dockerClient docker.ClientInterface, concurrency int) *ParallelStrategy {
	// Default to 4 concurrent operations if not specified
	if concurrency <= 0 {
		concurrency = 4
	}

	return &ParallelStrategy{
		dockerClient: dockerClient,
		concurrency:  concurrency,
	}
}

// Name returns the name of the strategy
func (s *ParallelStrategy) Name() string {
	return "parallel"
}

// Execute performs concurrent image synchronization
func (s *ParallelStrategy) Execute(ctx context.Context, operations []*SyncOperation) ([]*SyncResult, error) {
	// Create a channel to receive results from workers
	resultChan := make(chan *SyncResult, len(operations))

	// Create a worker pool with limited concurrency
	jobChan := make(chan *SyncOperation, len(operations))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Create a cancellable context for the worker pool
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	// Spawn worker goroutines
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()

			// Worker process
			for job := range jobChan {
				// Check if context was cancelled
				if workerCtx.Err() != nil {
					return
				}

				// Execute the synchronization operation
				workerLog := log.With().Int("worker", workerId).Logger()
				workerLog.Debug().
					Str("source", job.Source.FullName).
					Str("target", job.Target.FullName).
					Msg("Worker processing image sync")

				result := s.executeSingleOperation(workerCtx, job, workerId)
				resultChan <- result

				// Log detailed result
				if result.Success {
					workerLog.Debug().
						Str("source", job.Source.FullName).
						Str("target", job.Target.FullName).
						Int64("duration_ms", result.Duration).
						Msg("Worker completed image sync")
				} else {
					workerLog.Debug().
						Str("source", job.Source.FullName).
						Str("target", job.Target.FullName).
						Err(result.Error).
						Msg("Worker failed image sync")
				}
			}
		}(i)
	}

	// Start a goroutine to close the result channel when all workers finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Queue all jobs
	go func() {
		for _, op := range operations {
			select {
			case jobChan <- op:
				// Job queued successfully
			case <-ctx.Done():
				// Context cancelled, stop queueing jobs
				break
			}
		}
		close(jobChan)
	}()

	// Collect results
	var results []*SyncResult
	successCount := 0
	failureCount := 0

	for result := range resultChan {
		results = append(results, result)

		if result.Success {
			successCount++
			log.Info().
				Str("source", result.Operation.Source.FullName).
				Str("target", result.Operation.Target.FullName).
				Int64("duration_ms", result.Duration).
				Int64("bytes", result.BytesMoved).
				Int("success_count", successCount).
				Int("failure_count", failureCount).
				Msg("Image sync successful")
		} else {
			failureCount++
			log.Error().
				Str("source", result.Operation.Source.FullName).
				Str("target", result.Operation.Target.FullName).
				Err(result.Error).
				Int("success_count", successCount).
				Int("failure_count", failureCount).
				Msg("Image sync failed")
		}
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		return results, errors.NewContextError("sync", "context cancelled during parallel sync", ctx.Err())
	}

	log.Info().
		Int("concurrency", s.concurrency).
		Int("total", len(results)).
		Int("success", successCount).
		Int("failure", failureCount).
		Msg("Parallel sync completed")

	return results, nil
}

// executeSingleOperation synchronizes a single image
func (s *ParallelStrategy) executeSingleOperation(ctx context.Context, op *SyncOperation, workerId int) *SyncResult {
	result := &SyncResult{
		Operation: op,
		StartTime: time.Now().Unix(),
		Success:   false,
	}

	defer func() {
		result.EndTime = time.Now().Unix()
		result.Duration = result.EndTime - result.StartTime
	}()

	// Add worker ID to logs
	workerPrefix := fmt.Sprintf("Worker-%d: ", workerId)

	// If this is a dry run, just log and return success
	if op.DryRun {
		logMsg := fmt.Sprintf("Dry run: would sync from %s to %s", op.Source.FullName, op.Target.FullName)
		result.DetailedLogs = append(result.DetailedLogs, workerPrefix+logMsg)
		result.Success = true
		return result
	}

	// Step 1: Pull the source image
	result.DetailedLogs = append(result.DetailedLogs,
		workerPrefix+fmt.Sprintf("Pulling source image: %s", op.Source.FullName))

	if err := s.dockerClient.PullImage(ctx, op.Source.FullName); err != nil {
		result.Error = errors.NewOperationError(
			"sync",
			fmt.Sprintf("worker %d failed to pull source image", workerId),
			err,
		)
		result.DetailedLogs = append(result.DetailedLogs,
			workerPrefix+fmt.Sprintf("Pull failed: %v", err))
		return result
	}

	// Step 2: Tag the image with the target name
	result.DetailedLogs = append(result.DetailedLogs,
		workerPrefix+fmt.Sprintf("Tagging image from %s to %s", op.Source.FullName, op.Target.FullName))

	if err := s.dockerClient.TagImage(ctx, op.Source.FullName, op.Target.FullName); err != nil {
		result.Error = errors.NewOperationError(
			"sync",
			fmt.Sprintf("worker %d failed to tag image", workerId),
			err,
		)
		result.DetailedLogs = append(result.DetailedLogs,
			workerPrefix+fmt.Sprintf("Tag failed: %v", err))
		return result
	}

	// Step 3: Push the tagged image to the target registry
	result.DetailedLogs = append(result.DetailedLogs,
		workerPrefix+fmt.Sprintf("Pushing target image: %s", op.Target.FullName))

	if err := s.dockerClient.PushImage(ctx, op.Target.FullName); err != nil {
		result.Error = errors.NewOperationError(
			"sync",
			fmt.Sprintf("worker %d failed to push target image", workerId),
			err,
		)
		result.DetailedLogs = append(result.DetailedLogs,
			workerPrefix+fmt.Sprintf("Push failed: %v", err))
		return result
	}

	// Set success
	result.Success = true
	result.DetailedLogs = append(result.DetailedLogs,
		workerPrefix+"Synchronization completed successfully")

	return result
}
