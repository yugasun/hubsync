package docker

import (
	"time"

	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// ClientConfig holds configuration for a Docker client
type ClientConfig struct {
	Username    string
	Password    string
	Repository  string
	RetryCount  int
	RetryDelay  time.Duration
	PullTimeout time.Duration
	PushTimeout time.Duration
	APIVersion  string
	TLSVerify   bool
	CertPath    string
}

// Client represents a Docker client with all necessary operations
type Client struct {
	DockerClient *client.Client
	AuthConfig   registry.AuthConfig
	AuthStr      string
	Config       ClientConfig
}

// DefaultClientConfig returns a default configuration for Docker clients
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		RetryCount:  3,
		RetryDelay:  2 * time.Second,
		PullTimeout: 5 * time.Minute,
		PushTimeout: 10 * time.Minute,
		APIVersion:  "",
		TLSVerify:   true,
	}
}

// ImageReference represents a Docker image reference
type ImageReference struct {
	Registry   string
	Repository string
	Name       string
	Tag        string
	Digest     string
	FullName   string
}

// SyncOperation represents a sync operation between source and target images
type SyncOperation struct {
	Source     ImageReference
	Target     ImageReference
	StartTime  time.Time
	EndTime    time.Time
	Status     string
	Error      error
	RetryCount int
	BytesMoved int64
	ChecksumOK bool
}
