package client

import (
	"context"
)

// DockerClientInterface defines the interface for Docker operations
type DockerClientInterface interface {
	PullImage(ctx context.Context, imageName string) error
	TagImage(ctx context.Context, source, target string) error
	PushImage(ctx context.Context, imageName string) error
	VerifyCredentials(ctx context.Context) error
	Close() error
}

// Ensure DockerClient implements DockerClientInterface
var _ DockerClientInterface = (*DockerClient)(nil)
