package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/pkg/errors"
)

// ClientInterface defines the interface for Docker operations
type ClientInterface interface {
	PullImage(ctx context.Context, imageName string) error
	TagImage(ctx context.Context, source, target string) error
	PushImage(ctx context.Context, imageName string) error
	VerifyCredentials(ctx context.Context) error
	GetImageInfo(ctx context.Context, imageName string) (*ImageReference, error)
	Close() error
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)

// NewClient creates a new Docker client with the given configuration
func NewClient(cfg ClientConfig) (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if cfg.APIVersion != "" {
		opts = append(opts, client.WithVersion(cfg.APIVersion))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, errors.NewClientError("docker", "failed to create Docker client", err)
	}

	authConfig := registry.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		ServerAddress: cfg.Repository,
	}

	authStr, err := getAuthString(authConfig)
	if err != nil {
		return nil, errors.NewClientError("docker", "failed to create auth string", err)
	}

	dockerClient := &Client{
		DockerClient: cli,
		AuthConfig:   authConfig,
		AuthStr:      authStr,
		Config:       cfg,
	}

	return dockerClient, nil
}

// VerifyCredentials verifies Docker registry credentials
func (c *Client) VerifyCredentials(ctx context.Context) error {
	_, err := c.DockerClient.RegistryLogin(ctx, c.AuthConfig)
	if err != nil {
		return errors.NewAuthError("docker", "registry authentication failed", err)
	}
	return nil
}

// PerformWithRetry performs an operation with retry logic
func (c *Client) performWithRetry(ctx context.Context, imageName, operationType string, timeout time.Duration,
	operation func(opCtx context.Context) (io.ReadCloser, error),
) error {
	log.Debug().Str("image", imageName).Msgf("%sing image", operationType)

	var lastErr error
	for attempt := 0; attempt <= c.Config.RetryCount; attempt++ {
		if attempt > 0 {
			log.Debug().
				Int("attempt", attempt).
				Str("image", imageName).
				Msgf("Retrying %s operation", operationType)

			// Wait before retrying with exponential backoff
			// Fix for G115: integer overflow conversion int -> uint
			// Calculate backoff delay safely to avoid integer overflow
			var backoffDelay time.Duration
			if attempt > 30 {
				// Avoid overflow by capping at a maximum exponent value
				// 2^30 is already a large number for backoff calculation
				backoffDelay = time.Duration(1<<30) * c.Config.RetryDelay
			} else {
				backoffDelay = time.Duration(1<<attempt) * c.Config.RetryDelay / 2
			}

			select {
			case <-time.After(backoffDelay):
			case <-ctx.Done():
				return errors.NewContextError("docker", fmt.Sprintf("context cancelled during %s retry wait", operationType), ctx.Err())
			}
		}

		// Add timeout for the operation
		opCtx, cancel := context.WithTimeout(ctx, timeout)
		output, err := operation(opCtx)

		if err == nil {
			defer output.Close()
			// We need to read the output to completion or the operation may hang
			if _, err := io.Copy(io.Discard, output); err != nil {
				cancel()
				return errors.NewIOError("docker", fmt.Sprintf("error reading %s output", operationType), err)
			}
			cancel()
			return nil
		}

		cancel()
		lastErr = err
		log.Warn().Err(err).Str("image", imageName).Int("attempt", attempt+1).Msgf("%s attempt failed", operationType)
	}

	return errors.NewOperationError(
		"docker",
		fmt.Sprintf("failed to %s image after %d attempts", operationType, c.Config.RetryCount+1),
		lastErr,
	)
}

// PullImage pulls a Docker image with retry logic
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	return c.performWithRetry(ctx, imageName, "Pull", c.Config.PullTimeout, func(opCtx context.Context) (io.ReadCloser, error) {
		// DO NOT add auth config to the pull options
		return c.DockerClient.ImagePull(opCtx, imageName, image.PullOptions{})
	})
}

// TagImage tags a Docker image
func (c *Client) TagImage(ctx context.Context, source, target string) error {
	log.Debug().Str("source", source).Str("target", target).Msg("Tagging image")

	if err := c.DockerClient.ImageTag(ctx, source, target); err != nil {
		return errors.NewOperationError("docker", "failed to tag image", err)
	}
	return nil
}

// PushImage pushes a Docker image with retry logic
func (c *Client) PushImage(ctx context.Context, imageName string) error {
	return c.performWithRetry(ctx, imageName, "Push", c.Config.PushTimeout, func(opCtx context.Context) (io.ReadCloser, error) {
		return c.DockerClient.ImagePush(opCtx, imageName, image.PushOptions{
			RegistryAuth: c.AuthStr,
		})
	})
}

// GetImageInfo retrieves information about a Docker image
func (c *Client) GetImageInfo(ctx context.Context, imageName string) (*ImageReference, error) {
	// Implementation will parse the image name and return ImageReference
	// This is a placeholder that would be implemented with proper image reference parsing
	return &ImageReference{
		FullName: imageName,
	}, nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	if c.DockerClient != nil {
		err := c.DockerClient.Close()
		if err != nil {
			return fmt.Errorf("error closing Docker client: %w", err)
		}
	}
	return nil
}

// Helper function to create a base64 encoded auth string
func getAuthString(authConfig registry.AuthConfig) (string, error) {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}
