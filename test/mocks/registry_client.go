package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/registry"
)

// MockRegistryClient provides a mock implementation for testing registry operations
type MockRegistryClient struct {
	mu                sync.Mutex
	AuthError         error
	ExistingImages    map[string]bool
	ExistingTags      map[string][]string
	ImageManifests    map[string][]byte
	ValidationResults map[string]bool
}

// Ensure MockRegistryClient implements registry.RegistryInterface
var _ registry.RegistryInterface = (*MockRegistryClient)(nil)

// NewMockRegistryClient creates a new instance of MockRegistryClient
func NewMockRegistryClient() *MockRegistryClient {
	return &MockRegistryClient{
		ExistingImages:    make(map[string]bool),
		ExistingTags:      make(map[string][]string),
		ImageManifests:    make(map[string][]byte),
		ValidationResults: make(map[string]bool),
	}
}

// Auth mocks registry authentication
func (m *MockRegistryClient) Auth(ctx context.Context) error {
	return m.AuthError
}

// ListImages mocks listing images in a registry namespace
func (m *MockRegistryClient) ListImages(ctx context.Context, namespace string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var images []string
	for image, exists := range m.ExistingImages {
		if exists {
			images = append(images, image)
		}
	}
	return images, nil
}

// GetImageTags mocks retrieving tags for an image
func (m *MockRegistryClient) GetImageTags(ctx context.Context, repository string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tags, exists := m.ExistingTags[repository]; exists {
		return tags, nil
	}
	return []string{}, nil
}

// GetImageManifest mocks retrieving an image manifest
func (m *MockRegistryClient) GetImageManifest(ctx context.Context, repository string, reference string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", repository, reference)
	if manifest, exists := m.ImageManifests[key]; exists {
		return manifest, nil
	}
	return nil, fmt.Errorf("manifest for %s not found", key)
}

// ValidateImage mocks checking if an image exists in the registry
func (m *MockRegistryClient) ValidateImage(ctx context.Context, imageRef *docker.ImageReference) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := imageRef.FullName
	if result, exists := m.ValidationResults[key]; exists {
		return result, nil
	}
	// Default to false if not specifically set
	return false, nil
}

// Close mocks closing the registry client
func (m *MockRegistryClient) Close() error {
	return nil
}
