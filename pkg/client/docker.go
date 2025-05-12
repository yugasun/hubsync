package client

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
)

// DockerClient encapsulates Docker operations with enhanced error handling and retry logic
type DockerClient struct {
	Client     *client.Client
	AuthStr    string
	Username   string
	Password   string
	Repository string
	RetryCount int
	RetryDelay time.Duration
}

// NewDockerClient creates a new Docker client with authentication
func NewDockerClient(username, password, repository string) (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	authStr, err := getAuthString(username, password, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth string: %w", err)
	}

	dockerClient := &DockerClient{
		Client:     cli,
		AuthStr:    authStr,
		Username:   username,
		Password:   password,
		Repository: repository,
		RetryCount: 3,
		RetryDelay: 2 * time.Second,
	}

	// Verify credentials
	if err := dockerClient.VerifyCredentials(context.Background()); err != nil {
		return nil, fmt.Errorf("Docker login failed: %w", err)
	}

	return dockerClient, nil
}

// VerifyCredentials verifies Docker registry credentials
func (d *DockerClient) VerifyCredentials(ctx context.Context) error {
	authConfig := registry.AuthConfig{
		Username:      d.Username,
		Password:      d.Password,
		ServerAddress: d.Repository,
	}

	_, err := d.Client.RegistryLogin(ctx, authConfig)
	if err != nil {
		return err
	}
	return nil
}

// PullImage pulls a Docker image with retry logic
func (d *DockerClient) PullImage(ctx context.Context, imageName string) error {
	log.Debug().Str("image", imageName).Msg("Pulling image")

	var lastErr error
	for attempt := 0; attempt <= d.RetryCount; attempt++ {
		if attempt > 0 {
			log.Debug().
				Int("attempt", attempt).
				Str("image", imageName).
				Msg("Retrying pull operation")

			// Wait before retrying
			select {
			case <-time.After(time.Duration(attempt) * d.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Add timeout for the pull operation
		pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		pullOut, err := d.Client.ImagePull(pullCtx, imageName, image.PullOptions{})
		if err == nil {
			defer pullOut.Close()
			// We need to read the output to completion or the pull may hang
			if _, err := io.Copy(io.Discard, pullOut); err != nil {
				cancel()
				return fmt.Errorf("error reading pull output: %w", err)
			}
			cancel()
			return nil
		}

		cancel()
		lastErr = err
		log.Warn().Err(err).Str("image", imageName).Int("attempt", attempt+1).Msg("Pull attempt failed")
	}

	return fmt.Errorf("failed to pull image after %d attempts: %w", d.RetryCount+1, lastErr)
}

// TagImage tags a Docker image
func (d *DockerClient) TagImage(ctx context.Context, source, target string) error {
	log.Debug().Str("source", source).Str("target", target).Msg("Tagging image")

	if err := d.Client.ImageTag(ctx, source, target); err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}
	return nil
}

// PushImage pushes a Docker image with retry logic
func (d *DockerClient) PushImage(ctx context.Context, imageName string) error {
	log.Debug().Str("image", imageName).Msg("Pushing image")

	var lastErr error
	for attempt := 0; attempt <= d.RetryCount; attempt++ {
		if attempt > 0 {
			log.Debug().
				Int("attempt", attempt).
				Str("image", imageName).
				Msg("Retrying push operation")

			// Wait before retrying
			select {
			case <-time.After(time.Duration(attempt) * d.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Add timeout for the push operation
		pushCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		pushOut, err := d.Client.ImagePush(pushCtx, imageName, image.PushOptions{
			RegistryAuth: d.AuthStr,
		})
		if err == nil {
			defer pushOut.Close()
			// We need to read the output to completion or the push may hang
			if _, err := io.Copy(io.Discard, pushOut); err != nil {
				cancel()
				return fmt.Errorf("error reading push output: %w", err)
			}
			cancel()
			return nil
		}

		cancel()
		lastErr = err
		log.Warn().Err(err).Str("image", imageName).Int("attempt", attempt+1).Msg("Push attempt failed")
	}

	return fmt.Errorf("failed to push image after %d attempts: %w", d.RetryCount+1, lastErr)
}

// Close closes the Docker client
func (d *DockerClient) Close() error {
	if d.Client != nil {
		return d.Client.Close()
	}
	return nil
}

// Helper function to create a base64 encoded auth string
func getAuthString(username, password, repository string) (string, error) {
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: repository,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}
