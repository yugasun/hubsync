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

// PullImage pulls a Docker image with retry logic
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	log.Debug().Str("image", imageName).Msg("Pulling image")

	var lastErr error
	for attempt := 0; attempt <= c.Config.RetryCount; attempt++ {
		if attempt > 0 {
			log.Debug().
				Int("attempt", attempt).
				Str("image", imageName).
				Msg("Retrying pull operation")

			// Wait before retrying with exponential backoff
			backoffDelay := time.Duration(1<<uint(attempt-1)) * c.Config.RetryDelay
			select {
			case <-time.After(backoffDelay):
			case <-ctx.Done():
				return errors.NewContextError("docker", "context cancelled during pull retry wait", ctx.Err())
			}
		}

		// Add timeout for the pull operation
		pullCtx, cancel := context.WithTimeout(ctx, c.Config.PullTimeout)
		pullOut, err := c.DockerClient.ImagePull(pullCtx, imageName, image.PullOptions{
			RegistryAuth: c.AuthStr,
		})

		if err == nil {
			defer pullOut.Close()
			// We need to read the output to completion or the pull may hang
			if _, err := io.Copy(io.Discard, pullOut); err != nil {
				cancel()
				return errors.NewIOError("docker", "error reading pull output", err)
			}
			cancel()
			return nil
		}

		cancel()
		lastErr = err
		log.Warn().Err(err).Str("image", imageName).Int("attempt", attempt+1).Msg("Pull attempt failed")
	}

	return errors.NewOperationError(
		"docker",
		fmt.Sprintf("failed to pull image after %d attempts", c.Config.RetryCount+1),
		lastErr,
	)
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
	log.Debug().Str("image", imageName).Msg("Pushing image")

	var lastErr error
	for attempt := 0; attempt <= c.Config.RetryCount; attempt++ {
		if attempt > 0 {
			log.Debug().
				Int("attempt", attempt).
				Str("image", imageName).
				Msg("Retrying push operation")

			// Wait before retrying with exponential backoff
			backoffDelay := time.Duration(1<<uint(attempt-1)) * c.Config.RetryDelay
			select {
			case <-time.After(backoffDelay):
			case <-ctx.Done():
				return errors.NewContextError("docker", "context cancelled during push retry wait", ctx.Err())
			}
		}

		// Add timeout for the push operation
		pushCtx, cancel := context.WithTimeout(ctx, c.Config.PushTimeout)
		pushOut, err := c.DockerClient.ImagePush(pushCtx, imageName, image.PushOptions{
			RegistryAuth: c.AuthStr,
		})

		if err == nil {
			defer pushOut.Close()
			// We need to read the output to completion or the push may hang
			if _, err := io.Copy(io.Discard, pushOut); err != nil {
				cancel()
				return errors.NewIOError("docker", "error reading push output", err)
			}
			cancel()
			return nil
		}

		cancel()
		lastErr = err
		log.Warn().Err(err).Str("image", imageName).Int("attempt", attempt+1).Msg("Push attempt failed")
	}

	return errors.NewOperationError(
		"docker",
		fmt.Sprintf("failed to push image after %d attempts", c.Config.RetryCount+1),
		lastErr,
	)
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
		return c.DockerClient.Close()
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
