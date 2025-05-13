package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/errors"
	"github.com/yugasun/hubsync/pkg/observability"
	"github.com/yugasun/hubsync/pkg/registry"
	"github.com/yugasun/hubsync/pkg/sync/strategies"
)

// SyncStatisticsV2 holds statistics for the synchronization process
type SyncStatisticsV2 struct {
	TotalImages     int           // Total number of images to sync
	Successful      int           // Number of successfully synced images
	Failed          int           // Number of failed sync operations
	Skipped         int           // Number of skipped images
	TotalDuration   time.Duration // Total duration of the sync process
	AverageDuration time.Duration // Average duration per successful operation
}

// SyncerV2 represents the enhanced syncer with dependency injection
type SyncerV2 struct {
	config           *config.Config
	dockerClient     docker.ClientInterface
	registryClient   registry.RegistryInterface
	strategyFactory  *strategies.StrategyFactory
	operations       []*strategies.SyncOperation
	results          []*strategies.SyncResult
	processedCount   int
	statistics       *SyncStatisticsV2
	telemetryEnabled bool
	correlationID    string
}

// NewSyncerV2 creates a new SyncerV2 instance with dependency injection
func NewSyncerV2(
	config *config.Config,
	dockerClient docker.ClientInterface,
	registryClient registry.RegistryInterface,
) *SyncerV2 {
	// Create a strategy factory
	strategyFactory := strategies.NewStrategyFactory(
		dockerClient,
		config.Concurrency,
		true, // validateDst
		config.Force,
		config.DryRun,
	)

	return &SyncerV2{
		config:           config,
		dockerClient:     dockerClient,
		registryClient:   registryClient,
		strategyFactory:  strategyFactory,
		operations:       make([]*strategies.SyncOperation, 0),
		results:          make([]*strategies.SyncResult, 0),
		statistics:       &SyncStatisticsV2{},
		telemetryEnabled: true,
		correlationID:    fmt.Sprintf("sync-%d", time.Now().UnixNano()),
	}
}

// Run executes the synchronization process
func (s *SyncerV2) Run(ctx context.Context) error {
	startTime := time.Now()

	// Initialize context if not already initialized
	if ctx == nil {
		ctx = context.Background()
	}

	// Add correlation ID to the context for tracing
	ctx = context.WithValue(ctx, observability.GetCorrelationIDKey(), s.correlationID)

	// Parse content to get image list
	images, err := s.parseContent()
	if err != nil {
		return errors.NewConfigError("sync", "failed to parse content", err)
	}

	// Initialize statistics
	s.statistics.TotalImages = len(images)

	log.Info().
		Int("total_images", len(images)).
		Int("concurrency", s.config.Concurrency).
		Str("correlation_id", s.correlationID).
		Str("start_time", startTime.Format(time.RFC3339)).
		Bool("dry_run", s.config.DryRun).
		Msg("Starting image synchronization")

	// Create sync operations
	s.operations = s.createSyncOperations(images)

	// Choose strategy based on configuration
	var strategy strategies.SyncStrategy
	if s.config.Concurrency > 1 {
		strategy = s.strategyFactory.CreateStrategy("parallel")
	} else {
		strategy = s.strategyFactory.CreateStrategy("standard")
	}

	log.Info().
		Str("strategy", strategy.Name()).
		Int("operations", len(s.operations)).
		Msg("Executing sync with strategy")

	// Execute sync operations using selected strategy
	results, err := strategy.Execute(ctx, s.operations)
	if err != nil {
		log.Error().Err(err).Msg("Error during synchronization execution")
		// Continue to process results even if there was an error
	}

	// Store results
	s.results = results

	// Calculate statistics
	s.calculateStatistics(results, time.Since(startTime))

	// Generate output file
	if err := s.generateOutput(); err != nil {
		log.Error().Err(err).Msg("Failed to generate output file")
		return errors.NewOperationError("sync", "failed to generate output file", err)
	}

	log.Info().
		Int("total", s.statistics.TotalImages).
		Int("successful", s.statistics.Successful).
		Int("failed", s.statistics.Failed).
		Int("skipped", s.statistics.Skipped).
		Dur("total_duration", s.statistics.TotalDuration).
		Str("correlation_id", s.correlationID).
		Msg("Image synchronization completed")

	return nil
}

// parseContent parses the JSON content to get the list of images
func (s *SyncerV2) parseContent() ([]string, error) {
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}

	if err := json.Unmarshal([]byte(s.config.Content), &hubMirrors); err != nil {
		return nil, err
	}

	if len(hubMirrors.Content) > s.config.MaxContent {
		return nil, fmt.Errorf("too many images in content: %d > %d",
			len(hubMirrors.Content), s.config.MaxContent)
	}

	return hubMirrors.Content, nil
}

// createSyncOperations converts image names to sync operations
func (s *SyncerV2) createSyncOperations(images []string) []*strategies.SyncOperation {
	operations := make([]*strategies.SyncOperation, 0, len(images))

	for _, imageName := range images {
		if imageName == "" {
			// Skip empty image names
			s.statistics.Skipped++
			continue
		}

		// Generate source and target image references
		sourceRef, targetRef := s.generateImageReferences(imageName)

		// Create sync operation
		operation := &strategies.SyncOperation{
			Source:      sourceRef,
			Target:      targetRef,
			ValidateDst: !s.config.Force, // Skip validation if force is enabled
			Force:       s.config.Force,
			DryRun:      s.config.DryRun,
		}

		operations = append(operations, operation)
	}

	return operations
}

// generateImageReferences converts an image name to source and target references
func (s *SyncerV2) generateImageReferences(image string) (*docker.ImageReference, *docker.ImageReference) {
	// Save the original input
	originalImage := image

	// Check for custom pattern
	hasCustomPattern := strings.Contains(image, "$")
	var customName string

	// Handle custom pattern
	if hasCustomPattern {
		parts := strings.Split(image, "$")
		image = parts[0]
		if len(parts) > 1 {
			customName = parts[1]
		}
	}

	// Ensure source has a tag
	if !strings.Contains(image, ":") {
		image = image + ":latest"
	}

	// Parse source image reference
	sourceParts := strings.Split(image, ":")
	sourceImage := sourceParts[0]
	sourceTag := "latest"
	if len(sourceParts) > 1 {
		sourceTag = sourceParts[1]
	}

	// Create source reference
	sourceRef := &docker.ImageReference{
		FullName: image,
		Name:     sourceImage,
		Tag:      sourceTag,
	}

	// Build target reference
	var targetFullName string

	// Check if original source has version tag
	isTaggedWithVersion := strings.Contains(originalImage, ":v") && hasCustomPattern

	// Handle custom naming with $ symbol
	if hasCustomPattern && customName != "" {
		// Use custom name with source tag
		targetFullName = customName
		if !strings.Contains(targetFullName, ":") {
			targetFullName = targetFullName + ":" + sourceTag
		}
	} else if isTaggedWithVersion {
		// Special case: when using version tags, keep original structure
		targetFullName = image
	} else {
		// Use source name
		targetFullName = image
	}

	// Add repository and namespace if needed
	if s.config.Repository == "" {
		// No repository specified, use Docker Hub format
		if !strings.HasPrefix(targetFullName, s.config.Namespace+"/") {
			// Extract image name without path
			imageName := targetFullName
			if strings.Contains(targetFullName, "/") {
				parts := strings.Split(targetFullName, "/")
				imageName = parts[len(parts)-1]
			}
			targetFullName = s.config.Namespace + "/" + imageName
		}
	} else {
		// Repository specified
		// Extract image name without path
		imageName := targetFullName
		if strings.Contains(targetFullName, "/") {
			parts := strings.Split(targetFullName, "/")
			imageName = parts[len(parts)-1]
		}
		targetFullName = s.config.Repository + "/" + s.config.Namespace + "/" + imageName
	}

	// Parse target image reference
	targetParts := strings.Split(targetFullName, ":")
	targetImage := targetParts[0]
	targetTag := "latest"
	if len(targetParts) > 1 {
		targetTag = targetParts[1]
	}

	// Create target reference
	targetRef := &docker.ImageReference{
		FullName:   targetFullName,
		Repository: s.config.Repository,
		Name:       targetImage,
		Tag:        targetTag,
	}

	return sourceRef, targetRef
}

// calculateStatistics computes statistics from sync results
func (s *SyncerV2) calculateStatistics(results []*strategies.SyncResult, totalDuration time.Duration) {
	s.statistics.TotalDuration = totalDuration

	for _, result := range results {
		if result.Success {
			s.statistics.Successful++
		} else {
			s.statistics.Failed++
		}

		// Update processed count
		s.processedCount++
	}

	// Calculate average duration if there were successful operations
	if s.statistics.Successful > 0 {
		s.statistics.AverageDuration = s.statistics.TotalDuration / time.Duration(s.statistics.Successful)
	}
}

// generateOutput creates the output file with sync results
func (s *SyncerV2) generateOutput() error {
	if len(s.results) == 0 {
		return fmt.Errorf("no sync results available")
	}

	// Create template for output
	var loginCmd string
	if s.config.Repository != "" {
		loginCmd = fmt.Sprintf("# If your repository is private, please login first...\n# docker login %s --username={your username}\n\n", s.config.Repository)
	}

	tmpl, err := template.New("pull_images").Parse(loginCmd +
		`# HubSync completed at {{ .Timestamp }}
# Summary: {{ .Stats.Successful }} successful, {{ .Stats.Failed }} failed, {{ .Stats.Skipped }} skipped
# Total duration: {{ .Stats.TotalDuration }}
# Correlation ID: {{ .CorrelationID }}

{{- range .Results -}}
{{- if .Success }}
docker pull {{ .Operation.Target.FullName }} # (from {{ .Operation.Source.FullName }} in {{ .Duration }}ms)
{{ end }}
{{- end -}}

{{ if gt .Stats.Failed 0 }}
# The following images failed to sync:
{{- range .Results -}}
{{- if not .Success }}
# {{ .Operation.Source.FullName }} -> {{ .Operation.Target.FullName }} ({{ .Error }})
{{ end }}
{{- end -}}
{{ end }}`)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(s.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Prepare data for template
	data := struct {
		Results       []*strategies.SyncResult
		Stats         *SyncStatisticsV2
		Timestamp     string
		CorrelationID string
	}{
		Results:       s.results,
		Stats:         s.statistics,
		Timestamp:     time.Now().Format(time.RFC3339),
		CorrelationID: s.correlationID,
	}

	// Write to output file
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Info().
		Str("path", s.config.OutputPath).
		Int("count", len(s.results)).
		Int("successful", s.statistics.Successful).
		Int("failed", s.statistics.Failed).
		Int("skipped", s.statistics.Skipped).
		Msg("Output file created successfully")

	return nil
}

// GetProcessedImageCount returns the number of successfully processed images
func (s *SyncerV2) GetProcessedImageCount() int {
	return s.statistics.Successful
}
