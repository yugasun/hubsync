package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/client"
)

// OutputItem represents an item in the output
type OutputItem struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Repository string `json:"repository"`
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
}

// Syncer handles the synchronization of images
type Syncer struct {
	Config         *config.Config
	Client         client.DockerClientInterface
	Output         []OutputItem
	mutex          sync.Mutex
	processedCount int
}

// NewSyncer creates a new Syncer instance
func NewSyncer(config *config.Config, client client.DockerClientInterface) *Syncer {
	return &Syncer{
		Config: config,
		Client: client,
		Output: make([]OutputItem, 0),
	}
}

// Run processes all images with concurrency control
func (s *Syncer) Run(ctx context.Context) error {
	// Parse and validate the input content
	images, err := s.parseContent()
	if err != nil {
		return err
	}

	log.Info().Int("count", len(images)).Msg("Processing images")

	// Create semaphore to limit concurrency
	sem := semaphore.NewWeighted(int64(s.Config.Concurrency))
	g, ctx := errgroup.WithContext(ctx)

	// Process images with limited concurrency
	for _, imageName := range images {
		if imageName == "" {
			continue
		}

		// Make a copy for goroutine
		source := imageName

		// Process the image
		g.Go(func() error {
			// Acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				return fmt.Errorf("failed to acquire semaphore: %w", err)
			}
			defer sem.Release(1)

			// Process image - ignore target as we don't need it here
			source, _, err := s.ProcessImage(ctx, source)
			if err != nil {
				log.Error().Err(err).Str("source", source).Msg("Failed to process image")
				return err
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// Generate output file
	return s.generateOutput()
}

// ProcessImage processes a single image
func (s *Syncer) ProcessImage(ctx context.Context, source string) (string, string, error) {
	startTime := time.Now()
	log.Info().Str("image", source).Msg("Processing image")

	// Generate source and target names
	source, target := s.generateTargetName(source)

	// Pull image
	if err := s.Client.PullImage(ctx, source); err != nil {
		return source, target, fmt.Errorf("failed to pull image %s: %w", source, err)
	}

	// Tag image
	if err := s.Client.TagImage(ctx, source, target); err != nil {
		return source, target, fmt.Errorf("failed to tag image %s as %s: %w", source, target, err)
	}

	// Push image
	if err := s.Client.PushImage(ctx, target); err != nil {
		return source, target, fmt.Errorf("failed to push image %s: %w", target, err)
	}

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
	})
	s.processedCount++
	s.mutex.Unlock()

	log.Info().
		Str("source", source).
		Str("target", target).
		Dur("duration", duration).
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
	if s.Output[0].Repository != "" {
		loginCmd = fmt.Sprintf("# If your repository is private, please login first...\n# docker login %s --username={your username}\n\n", s.Output[0].Repository)
	}

	tmpl, err := template.New("pull_images").Parse(loginCmd + `{{- range . -}}
docker pull {{ .Target }}
{{ end -}}`)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(s.Config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Write to output file
	if err := tmpl.Execute(outputFile, s.Output); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Info().
		Str("path", s.Config.OutputPath).
		Int("count", len(s.Output)).
		Msg("Output file created successfully")

	return nil
}

// GetProcessedImageCount returns the number of successfully processed images
func (s *Syncer) GetProcessedImageCount() int {
	return s.processedCount
}
