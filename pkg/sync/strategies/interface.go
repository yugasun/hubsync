package strategies

import (
	"context"

	"github.com/yugasun/hubsync/pkg/docker"
)

// SyncStrategy defines the interface for different synchronization strategies
type SyncStrategy interface {
	// Execute performs the synchronization using the specific strategy
	Execute(ctx context.Context, operations []*SyncOperation) ([]*SyncResult, error)

	// Name returns the name of the strategy
	Name() string
}

// SyncOperation represents a single image synchronization operation
type SyncOperation struct {
	Source      *docker.ImageReference
	Target      *docker.ImageReference
	ValidateDst bool
	Force       bool
	DryRun      bool
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Operation    *SyncOperation
	Success      bool
	Error        error
	Duration     int64 // Duration in milliseconds
	ImageSize    int64 // Size in bytes
	BytesMoved   int64
	StartTime    int64 // Unix timestamp
	EndTime      int64 // Unix timestamp
	DetailedLogs []string
}

// StrategyFactory creates synchronization strategies
type StrategyFactory struct {
	dockerClient docker.ClientInterface
	concurrency  int
	validateDst  bool
	force        bool
	dryRun       bool
}

// NewStrategyFactory creates a new strategy factory
func NewStrategyFactory(
	dockerClient docker.ClientInterface,
	concurrency int,
	validateDst bool,
	force bool,
	dryRun bool,
) *StrategyFactory {
	return &StrategyFactory{
		dockerClient: dockerClient,
		concurrency:  concurrency,
		validateDst:  validateDst,
		force:        force,
		dryRun:       dryRun,
	}
}

// CreateStrategy creates a specific synchronization strategy
func (f *StrategyFactory) CreateStrategy(strategyName string) SyncStrategy {
	switch strategyName {
	case "parallel":
		return NewParallelStrategy(f.dockerClient, f.concurrency)
	default:
		return NewStandardStrategy(f.dockerClient)
	}
}
