package registry

import (
	"context"
	"time"

	"github.com/yugasun/hubsync/pkg/docker"
)

// Provider represents the type of container registry
type Provider string

const (
	// DockerHub represents the Docker Hub registry
	DockerHub Provider = "dockerhub"

	// DockerIO represents the Docker.io registry
	DockerIO Provider = "docker.io"

	// ECR represents Amazon's Elastic Container Registry
	ECR Provider = "ecr"

	// GCR represents Google's Container Registry
	GCR Provider = "gcr"

	// ACR represents Azure's Container Registry
	ACR Provider = "acr"

	// Custom represents a custom registry
	Custom Provider = "custom"
)

// RegistryConfig holds configuration for connecting to a registry
type RegistryConfig struct {
	Provider        Provider
	URL             string
	Username        string
	Password        string
	AccessToken     string
	RefreshToken    string
	TokenExpiration time.Time
	Insecure        bool
	SkipVerify      bool
}

// RegistryInterface defines operations for interacting with a container registry
type RegistryInterface interface {
	// Auth authenticates with the registry
	Auth(ctx context.Context) error

	// ListImages lists available images
	ListImages(ctx context.Context, namespace string) ([]string, error)

	// GetImageTags gets all tags for an image
	GetImageTags(ctx context.Context, repository string) ([]string, error)

	// GetImageManifest gets the image manifest
	GetImageManifest(ctx context.Context, repository string, reference string) ([]byte, error)

	// ValidateImage checks if an image exists in the registry
	ValidateImage(ctx context.Context, imageRef *docker.ImageReference) (bool, error)

	// Close releases any resources associated with the registry
	Close() error
}
