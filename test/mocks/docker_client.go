package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/yugasun/hubsync/pkg/docker"
)

// MockDockerClient provides a mock implementation for testing Docker client operations
type MockDockerClient struct {
	mu              sync.Mutex
	PulledImages    map[string]bool
	TaggedImages    map[string]string
	PushedImages    map[string]bool
	PullErrors      map[string]error
	TagErrors       map[string]error
	PushErrors      map[string]error
	CredentialError error
}

// Ensure MockDockerClient implements docker.ClientInterface
var _ docker.ClientInterface = (*MockDockerClient)(nil)

// NewMockDockerClient creates a new instance of MockDockerClient
func NewMockDockerClient() *MockDockerClient {
	return &MockDockerClient{
		PulledImages: make(map[string]bool),
		TaggedImages: make(map[string]string),
		PushedImages: make(map[string]bool),
		PullErrors:   make(map[string]error),
		TagErrors:    make(map[string]error),
		PushErrors:   make(map[string]error),
	}
}

// PullImage mocks the Docker image pull operation
func (m *MockDockerClient) PullImage(ctx context.Context, imageName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for error scenario
	if err, exists := m.PullErrors[imageName]; exists && err != nil {
		return err
	}

	// Record successful pull
	m.PulledImages[imageName] = true
	return nil
}

// TagImage mocks the Docker image tag operation
func (m *MockDockerClient) TagImage(ctx context.Context, source, target string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for error scenario
	if err, exists := m.TagErrors[source]; exists && err != nil {
		return err
	}

	// Check if source exists
	if !m.PulledImages[source] {
		return fmt.Errorf("source image %s not found", source)
	}

	// Record successful tag
	m.TaggedImages[source] = target
	return nil
}

// PushImage mocks the Docker image push operation
func (m *MockDockerClient) PushImage(ctx context.Context, imageName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for error scenario
	if err, exists := m.PushErrors[imageName]; exists && err != nil {
		return err
	}

	// Check if image exists (was tagged)
	var exists bool
	for _, tagged := range m.TaggedImages {
		if tagged == imageName {
			exists = true
			break
		}
	}

	if !exists {
		return fmt.Errorf("target image %s not found", imageName)
	}

	// Record successful push
	m.PushedImages[imageName] = true
	return nil
}

// VerifyCredentials mocks Docker registry authentication
func (m *MockDockerClient) VerifyCredentials(ctx context.Context) error {
	return m.CredentialError
}

// GetImageInfo implements the new method required by docker.ClientInterface
func (m *MockDockerClient) GetImageInfo(ctx context.Context, imageName string) (*docker.ImageReference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse the image name to extract parts
	// For mock purposes, we'll just return a simple ImageReference
	var tag string
	name := imageName

	// Extract tag if present
	for i := len(imageName) - 1; i >= 0; i-- {
		if imageName[i] == ':' {
			name = imageName[:i]
			tag = imageName[i+1:]
			break
		}
	}

	// If no tag was found, use "latest"
	if tag == "" {
		tag = "latest"
	}

	return &docker.ImageReference{
		FullName: imageName,
		Name:     name,
		Tag:      tag,
	}, nil
}

// Close mocks the closing of resources
func (m *MockDockerClient) Close() error {
	return nil
}
