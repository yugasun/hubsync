package di

import (
	stdsync "sync"

	"github.com/yugasun/hubsync/internal/config"
	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/errors"
	"github.com/yugasun/hubsync/pkg/observability"
	"github.com/yugasun/hubsync/pkg/registry"
	"github.com/yugasun/hubsync/pkg/sync"
)

// Container is a dependency injection container that manages application services
type Container struct {
	config           *config.Config
	dockerClient     docker.ClientInterface
	registryClient   registry.RegistryInterface
	syncer           *sync.SyncerV2
	telemetryManager *observability.TelemetryManager
	metricsManager   *observability.MetricsManager
	mutex            stdsync.Mutex
	initialized      bool
}

var (
	// instance is the singleton instance of the DI container
	instance *Container
	// once ensures the singleton is initialized only once
	once stdsync.Once
)

// GetInstance returns the singleton instance of the DI container
func GetInstance() *Container {
	once.Do(func() {
		instance = &Container{
			initialized: false,
		}
	})
	return instance
}

// Initialize initializes the container with the given configuration
func (c *Container) Initialize(cfg *config.Config) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return nil // Already initialized
	}

	c.config = cfg

	// Initialize observability components first
	if err := c.initializeObservability(); err != nil {
		return err
	}

	// Initialize clients
	if err := c.initializeDockerClient(); err != nil {
		return err
	}

	c.initializeRegistryClient()

	// Initialize syncer
	c.syncer = sync.NewSyncerV2(c.config, c.dockerClient, c.registryClient)

	c.initialized = true

	// Log application info
	if c.telemetryManager != nil {
		c.telemetryManager.LogApplicationInfo()
	}

	return nil
}

// initializeObservability initializes telemetry and metrics
func (c *Container) initializeObservability() error {
	// Create telemetry manager
	c.telemetryManager = observability.NewTelemetryManager(
		c.config.TelemetryEnabled,
		"hubsync",
		c.config.Version,
		"production", // Could be configurable in the future
	)

	// Create metrics manager
	c.metricsManager = observability.NewMetricsManager(
		c.config.MetricsEnabled,
		"hubsync",
	)

	// Start metrics server if enabled
	if c.config.MetricsEnabled {
		if err := c.metricsManager.StartServer(":9090"); err != nil {
			return errors.NewSystemError("di", "failed to start metrics server", err)
		}
	}

	return nil
}

// initializeDockerClient initializes the Docker client
func (c *Container) initializeDockerClient() error {
	// Create Docker client configuration
	dockerConfig := docker.ClientConfig{
		Username:    c.config.Username,
		Password:    c.config.Password,
		Repository:  c.config.Repository,
		RetryCount:  c.config.RetryCount,
		RetryDelay:  c.config.RetryDelay,
		PullTimeout: c.config.Timeout,
		PushTimeout: c.config.Timeout,
	}

	// Create Docker client
	client, err := docker.NewClient(dockerConfig)
	if err != nil {
		return errors.NewClientError("di", "failed to initialize Docker client", err)
	}

	c.dockerClient = client
	return nil
}

// initializeRegistryClient initializes the registry client
func (c *Container) initializeRegistryClient() {
	// Create registry configuration
	registryConfig := registry.RegistryConfig{
		Provider:   registry.DockerHub, // Default to Docker Hub
		URL:        c.config.Repository,
		Username:   c.config.Username,
		Password:   c.config.Password,
		SkipVerify: false,
	}

	// Create appropriate registry client based on configuration
	// For now, only Docker Hub is supported
	// In the future, we can add more registry types based on URL or configuration
	registryClient := registry.NewDockerHubRegistry(registryConfig)

	c.registryClient = registryClient
}

func (c *Container) GetConfig() *config.Config {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.config
}

// GetDockerClient returns the Docker client
func (c *Container) GetDockerClient() docker.ClientInterface {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.dockerClient
}

// GetRegistryClient returns the registry client
func (c *Container) GetRegistryClient() registry.RegistryInterface {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.registryClient
}

// GetSyncer returns the syncer
func (c *Container) GetSyncer() *sync.SyncerV2 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.syncer
}

// GetTelemetryManager returns the telemetry manager
func (c *Container) GetTelemetryManager() *observability.TelemetryManager {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.telemetryManager
}

// GetMetricsManager returns the metrics manager
func (c *Container) GetMetricsManager() *observability.MetricsManager {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.metricsManager
}

// Cleanup releases all resources held by the container
func (c *Container) Cleanup() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var errs []error

	// Close metrics manager
	if c.metricsManager != nil {
		if err := c.metricsManager.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close Docker client
	if c.dockerClient != nil {
		if err := c.dockerClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close registry client
	if c.registryClient != nil {
		if err := c.registryClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Reset state
	c.dockerClient = nil
	c.registryClient = nil
	c.syncer = nil
	c.telemetryManager = nil
	c.metricsManager = nil
	c.initialized = false

	if len(errs) > 0 {
		return errors.NewOperationError("di", "failed to cleanup container", errs[0])
	}

	return nil
}
