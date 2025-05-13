package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/docker"
)

// OutputItem represents an item in the output
type OutputItem struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Repository string `json:"repository"`
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Status     string `json:"status"` // Success, Failed, or Skipped
	ErrorMsg   string `json:"error,omitempty"`
}

// SyncStatistics holds statistics about the sync operation
type SyncStatistics struct {
	TotalImages     int           `json:"total_images"`
	Successful      int           `json:"successful"`
	Failed          int           `json:"failed"`
	Skipped         int           `json:"skipped"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
}

// Syncer handles the synchronization of images
type Syncer struct {
	Config         *config.Config
	Client         docker.ClientInterface
	Output         []OutputItem
	mutex          sync.Mutex
	processedCount int32
	totalCount     int32
	startTime      time.Time
	stats          SyncStatistics
}

// NewSyncer creates a new Syncer instance
func NewSyncer(config *config.Config, client docker.ClientInterface) *Syncer {
	return &Syncer{
		Config: config,
		Client: client,
		Output: make([]OutputItem, 0),
	}
}

// Run processes all images with concurrency control
func (s *Syncer) Run(ctx context.Context) error {
	s.startTime = time.Now()

	// Parse and validate the input content
	images, err := s.parseContent()
	if err != nil {
		return err
	}

	// Initialize statistics
	s.stats.TotalImages = len(images)

	// Fix for G115: integer overflow conversion int -> int32
	// Safely convert len(images) to int32, capping at max int32 if needed
	imageCount := len(images)
	if imageCount > (1<<31 - 1) {
		log.Warn().Msg("Number of images exceeds maximum value for int32, count may be inaccurate")
		atomic.StoreInt32(&s.totalCount, (1<<31)-1)
	} else {
		atomic.StoreInt32(&s.totalCount, int32(imageCount))
	}

	log.Info().
		Int("total_images", len(images)).
		Int("concurrency", s.Config.Concurrency).
		Str("start_time", s.startTime.Format(time.RFC3339)).
		Msg("Starting image synchronization")

	// Create semaphore to limit concurrency
	sem := semaphore.NewWeighted(int64(s.Config.Concurrency))
	g, ctx := errgroup.WithContext(ctx)

	// Process images with limited concurrency
	for idx, imageName := range images {
		if imageName == "" {
			atomic.AddInt32(&s.processedCount, 1)
			s.stats.Skipped++

			log.Warn().
				Int("index", idx+1).
				Int("total", len(images)).
				Float64("progress_pct", float64(idx+1)*100/float64(len(images))).
				Msg("Empty image name skipped")

			continue
		}

		// Make a copy for goroutine
		source := imageName
		imageIndex := idx

		// Process the image
		g.Go(func() error {
			// Acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Error().Err(err).Str("source", source).Msg("Failed to acquire semaphore")
				return fmt.Errorf("failed to acquire semaphore: %w", err)
			}
			defer sem.Release(1)

			// Log start of processing with progress indicator
			log.Info().
				Int("index", imageIndex+1).
				Int("total", len(images)).
				Float64("progress_pct", float64(imageIndex+1)*100/float64(len(images))).
				Str("image", source).
				Msg("Starting image processing")

			// Process image with retries
			source, target, err := s.ProcessImageWithRetry(ctx, source)

			// Update progress counter regardless of success/failure
			newCount := atomic.AddInt32(&s.processedCount, 1)

			// Log completion status with progress
			if err != nil {
				log.Error().
					Err(err).
					Str("source", source).
					Int("processed", int(newCount)).
					Int("total", len(images)).
					Float64("progress_pct", float64(newCount)*100/float64(len(images))).
					Msg("Image processing failed")
				return err
			}

			log.Info().
				Str("source", source).
				Str("target", target).
				Int("processed", int(newCount)).
				Int("total", len(images)).
				Float64("progress_pct", float64(newCount)*100/float64(len(images))).
				Msg("Image processing completed")

			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		// Still generate output even if some images failed
		log.Error().Err(err).Msg("Some images failed to process")
	}

	// Calculate final statistics
	s.stats.TotalDuration = time.Since(s.startTime)
	if s.stats.Successful > 0 {
		s.stats.AverageDuration = s.stats.TotalDuration / time.Duration(s.stats.Successful)
	}

	// Log completion summary
	log.Info().
		Int("total", s.stats.TotalImages).
		Int("successful", s.stats.Successful).
		Int("failed", s.stats.Failed).
		Int("skipped", s.stats.Skipped).
		Dur("total_duration", s.stats.TotalDuration).
		Msg("Image synchronization completed")

	// Generate output file
	return s.generateOutput()
}

// ProcessImageWithRetry processes a single image with retry logic
func (s *Syncer) ProcessImageWithRetry(ctx context.Context, source string) (string, string, error) {
	var lastErr error
	startTime := time.Now()

	// Default target to empty string until we generate it
	target := ""

	// Try to process the image with retries
	for attempt := 1; attempt <= s.Config.RetryCount+1; attempt++ {
		if attempt > 1 {
			log.Warn().
				Str("source", source).
				Int("attempt", attempt).
				Int("max_attempts", s.Config.RetryCount+1).
				Dur("retry_delay", s.Config.RetryDelay).
				Msg("Retrying image processing")

			// Wait before retry
			select {
			case <-ctx.Done():
				return source, target, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(s.Config.RetryDelay):
				// Continue with retry
			}
		}

		// Process the image
		source, target, lastErr = s.ProcessImage(ctx, source)
		if lastErr == nil {
			// Success, no more retries needed
			return source, target, nil
		}

		log.Error().
			Err(lastErr).
			Str("source", source).
			Int("attempt", attempt).
			Int("max_attempts", s.Config.RetryCount+1).
			Msg("Image processing attempt failed")
	}

	// All retries failed, record the failure
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	s.mutex.Lock()
	s.Output = append(s.Output, OutputItem{
		Source:     source,
		Target:     target,
		Repository: s.Config.Repository,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   duration,
		Status:     "Failed",
		ErrorMsg:   lastErr.Error(),
	})
	s.stats.Failed++
	s.mutex.Unlock()

	return source, target, lastErr
}

// processImage handles the core image processing logic
func (s *Syncer) ProcessImage(ctx context.Context, source string) (string, string, error) {
	startTime := time.Now()

	// Generate source and target names
	log.Debug().Str("image", source).Msg("Generating target image name")
	source, target := s.generateTargetName(source)
	log.Info().
		Str("source", source).
		Str("target", target).
		Msg("Image mapping generated")

	// Pull image
	log.Info().Str("image", source).Msg("Pulling image")
	pullStart := time.Now()
	if err := s.Client.PullImage(ctx, source); err != nil {
		return source, target, fmt.Errorf("failed to pull image %s: %w", source, err)
	}
	log.Info().
		Str("image", source).
		Dur("duration", time.Since(pullStart)).
		Msg("Image pull completed")

	// Tag image
	log.Info().
		Str("source", source).
		Str("target", target).
		Msg("Tagging image")
	tagStart := time.Now()
	if err := s.Client.TagImage(ctx, source, target); err != nil {
		return source, target, fmt.Errorf("failed to tag image %s as %s: %w", source, target, err)
	}
	log.Info().
		Str("source", source).
		Str("target", target).
		Dur("duration", time.Since(tagStart)).
		Msg("Image tag completed")

	// Push image
	log.Info().Str("image", target).Msg("Pushing image")
	pushStart := time.Now()
	if err := s.Client.PushImage(ctx, target); err != nil {
		return source, target, fmt.Errorf("failed to push image %s: %w", target, err)
	}
	log.Info().
		Str("image", target).
		Dur("duration", time.Since(pushStart)).
		Msg("Image push completed")

	// Record output
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	s.mutex.Lock()
	s.Output = append(s.Output, OutputItem{
		Source:     source,
		Target:     target,
		Repository: s.Config.Repository,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   duration,
		Status:     "Success",
	})
	s.stats.Successful++
	s.mutex.Unlock()

	log.Info().
		Str("source", source).
		Str("target", target).
		Dur("pull_duration", time.Since(pullStart)-time.Since(tagStart)).
		Dur("tag_duration", time.Since(tagStart)-time.Since(pushStart)).
		Dur("push_duration", time.Since(pushStart)).
		Dur("total_duration", duration).
		Msg("Image processed successfully")

	return source, target, nil
}

// generateTargetName generates the target image name
func (s *Syncer) generateTargetName(source string) (string, string) {
	// Save the original input
	originalSource := source

	// Check for custom pattern
	hasCustomPattern := strings.Contains(source, "$")
	var customName string

	// Handle custom pattern
	if hasCustomPattern {
		parts := strings.Split(source, "$")
		source = parts[0]
		if len(parts) > 1 {
			customName = parts[1]
		}
	}

	// Ensure source has a tag
	if !strings.Contains(source, ":") {
		source = source + ":latest"
	}

	// Build target name
	var target string

	// Extract tag from source
	var imageTag string
	imageNameParts := strings.Split(source, ":")
	if len(imageNameParts) > 1 {
		imageTag = imageNameParts[1]
	} else {
		imageTag = "latest"
	}

	// Check if original source has version tag
	isTaggedWithVersion := strings.Contains(originalSource, ":v") && hasCustomPattern

	// Handle custom naming with $ symbol
	if hasCustomPattern && customName != "" {
		// Use custom name with source tag
		target = customName
		if !strings.Contains(target, ":") {
			target = target + ":" + imageTag
		}
	} else if isTaggedWithVersion {
		// Special case: when using version tags, keep original structure
		target = source
	} else {
		// Use source name
		target = source
	}

	// Add repository and namespace if needed
	if s.Config.Repository == "" {
		// No repository specified, use Docker Hub format
		if !strings.HasPrefix(target, s.Config.Namespace+"/") {
			// Extract image name without path
			imageName := target
			if strings.Contains(target, "/") {
				parts := strings.Split(target, "/")
				imageName = parts[len(parts)-1]
			}
			target = s.Config.Namespace + "/" + imageName
		}
	} else {
		// Repository specified
		// Extract image name without path
		imageName := target
		if strings.Contains(target, "/") {
			parts := strings.Split(target, "/")
			imageName = parts[len(parts)-1]
		}
		target = s.Config.Repository + "/" + s.Config.Namespace + "/" + imageName
	}

	return source, target
}

// parseContent parses the JSON content
func (s *Syncer) parseContent() ([]string, error) {
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}

	if err := json.Unmarshal([]byte(s.Config.Content), &hubMirrors); err != nil {
		return nil, fmt.Errorf("failed to parse content: %w", err)
	}

	if len(hubMirrors.Content) > s.Config.MaxContent {
		return nil, fmt.Errorf("too many images in content: %d > %d", len(hubMirrors.Content), s.Config.MaxContent)
	}

	return hubMirrors.Content, nil
}

// generateOutput generates the output file
func (s *Syncer) generateOutput() error {
	if len(s.Output) == 0 {
		return fmt.Errorf("output is empty")
	}

	// Create template for output
	var loginCmd string
	if len(s.Output) > 0 && s.Output[0].Repository != "" {
		loginCmd = fmt.Sprintf("# If your repository is private, please login first...\n# docker login %s --username={your username}\n\n", s.Output[0].Repository)
	}

	tmpl, err := template.New("pull_images").Parse(loginCmd +
		`# HubSync completed at {{ .Timestamp }}
# Summary: {{ .Stats.Successful }} successful, {{ .Stats.Failed }} failed, {{ .Stats.Skipped }} skipped
# Total duration: {{ .Stats.TotalDuration }}

{{- range .Output -}}
{{- if eq .Status "Success" }}
docker pull {{ .Target }} # (from {{ .Source }} in {{ .Duration }})
{{ end }}
{{- end -}}

{{ if gt .Stats.Failed 0 }}
# The following images failed to sync:
{{- range .Output -}}
{{- if eq .Status "Failed" }}
# {{ .Source }} -> {{ .Target }} ({{ .ErrorMsg }})
{{ end }}
{{- end -}}
{{ end }}`)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(s.Config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Prepare data for template
	data := struct {
		Output    []OutputItem
		Stats     SyncStatistics
		Timestamp string
	}{
		Output:    s.Output,
		Stats:     s.stats,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Write to output file
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Info().
		Str("path", s.Config.OutputPath).
		Int("count", len(s.Output)).
		Int("successful", s.stats.Successful).
		Int("failed", s.stats.Failed).
		Int("skipped", s.stats.Skipped).
		Msg("Output file created successfully")

	return nil
}

// GetProcessedImageCount returns the number of successfully processed images
func (s *Syncer) GetProcessedImageCount() int {
	return s.stats.Successful
}
