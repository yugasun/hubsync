package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/pkg/docker"
	"github.com/yugasun/hubsync/pkg/errors"
)

const (
	dockerHubBaseURL  = "https://hub.docker.com/v2"
	dockerHubAuthURL  = "https://auth.docker.io/token"
	dockerRegistryAPI = "https://registry-1.docker.io/v2"
)

// DockerHubRegistry implements the RegistryInterface for Docker Hub
type DockerHubRegistry struct {
	config       RegistryConfig
	client       *http.Client
	authToken    string
	tokenExpires time.Time
}

// Ensure DockerHubRegistry implements RegistryInterface
var _ RegistryInterface = (*DockerHubRegistry)(nil)

// NewDockerHubRegistry creates a new Docker Hub registry client
func NewDockerHubRegistry(config RegistryConfig) *DockerHubRegistry {
	return &DockerHubRegistry{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Auth authenticates with Docker Hub
func (r *DockerHubRegistry) Auth(ctx context.Context) error {
	// Return if we have a valid token
	if r.authToken != "" && time.Now().Before(r.tokenExpires) {
		return nil
	}

	var authResp struct {
		Token       string    `json:"token"`
		AccessToken string    `json:"access_token"`
		ExpiresIn   int       `json:"expires_in"`
		IssuedAt    time.Time `json:"issued_at"`
	}

	// First try direct login with username/password
	if r.config.Username != "" && r.config.Password != "" {
		reqBody := map[string]string{
			"username": r.config.Username,
			"password": r.config.Password,
		}

		reqJSON, err := json.Marshal(reqBody)
		if err != nil {
			return errors.NewOperationError("registry", "failed to marshal auth request", err)
		}

		req, err := http.NewRequestWithContext(
			ctx,
			"POST",
			fmt.Sprintf("%s/users/login", dockerHubBaseURL),
			strings.NewReader(string(reqJSON)),
		)
		if err != nil {
			return errors.NewOperationError("registry", "failed to create auth request", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := r.client.Do(req)
		if err != nil {
			return errors.NewOperationError("registry", "failed to execute auth request", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return errors.NewAuthError(
				"registry",
				fmt.Sprintf("authentication failed with status %d: %s", resp.StatusCode, string(body)),
				nil,
			)
		}

		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return errors.NewOperationError("registry", "failed to decode auth response", err)
		}

		if authResp.Token == "" && authResp.AccessToken != "" {
			authResp.Token = authResp.AccessToken
		}

		// Set token and expiration
		r.authToken = authResp.Token
		r.tokenExpires = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

		return nil
	}

	// If no username/password, try anonymous token
	scope := "repository:library/hello-world:pull"
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s?service=registry.docker.io&scope=%s", dockerHubAuthURL, url.QueryEscape(scope)),
		nil,
	)
	if err != nil {
		return errors.NewOperationError("registry", "failed to create anonymous auth request", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.NewOperationError("registry", "failed to execute anonymous auth request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.NewAuthError(
			"registry",
			fmt.Sprintf("anonymous authentication failed with status %d", resp.StatusCode),
			nil,
		)
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return errors.NewOperationError("registry", "failed to decode anonymous auth response", err)
	}

	// Set token and expiration (default to 300 seconds if not provided)
	r.authToken = authResp.Token
	expiresIn := 300
	if authResp.ExpiresIn > 0 {
		expiresIn = authResp.ExpiresIn
	}
	r.tokenExpires = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return nil
}

// makeAuthenticatedRequest makes an authenticated request to Docker Hub
func (r *DockerHubRegistry) makeAuthenticatedRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	// Ensure we have a valid auth token
	if err := r.Auth(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.NewOperationError("registry", "failed to create request", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.authToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return r.client.Do(req)
}

// ListImages lists available images in the given namespace
func (r *DockerHubRegistry) ListImages(ctx context.Context, namespace string) ([]string, error) {
	if namespace == "" {
		namespace = "library"
	}

	url := fmt.Sprintf("%s/repositories/%s/", dockerHubBaseURL, namespace)
	resp, err := r.makeAuthenticatedRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewOperationError(
			"registry",
			fmt.Sprintf("failed to list images with status %d: %s", resp.StatusCode, string(body)),
			nil,
		)
	}

	var response struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewOperationError("registry", "failed to decode image list response", err)
	}

	images := make([]string, 0, len(response.Results))
	for _, result := range response.Results {
		images = append(images, result.Name)
	}

	return images, nil
}

// GetImageTags gets all tags for an image in the registry
func (r *DockerHubRegistry) GetImageTags(ctx context.Context, repository string) ([]string, error) {
	// Split repository into namespace and image name
	parts := strings.Split(repository, "/")
	var namespace, imageName string

	if len(parts) == 1 {
		namespace = "library"
		imageName = parts[0]
	} else {
		namespace = parts[0]
		imageName = strings.Join(parts[1:], "/")
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/tags", dockerHubBaseURL, namespace, imageName)
	resp, err := r.makeAuthenticatedRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewOperationError(
			"registry",
			fmt.Sprintf("failed to get image tags with status %d: %s", resp.StatusCode, string(body)),
			nil,
		)
	}

	var response struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.NewOperationError("registry", "failed to decode image tags response", err)
	}

	tags := make([]string, 0, len(response.Results))
	for _, result := range response.Results {
		tags = append(tags, result.Name)
	}

	return tags, nil
}

// GetImageManifest gets the manifest for an image
func (r *DockerHubRegistry) GetImageManifest(ctx context.Context, repository string, reference string) ([]byte, error) {
	// Handle library namespace
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	url := fmt.Sprintf("%s/%s/manifests/%s", dockerRegistryAPI, repository, reference)

	// First ensure authentication
	if err := r.Auth(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewOperationError("registry", "failed to create manifest request", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.authToken)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, errors.NewOperationError("registry", "failed to execute manifest request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewOperationError(
			"registry",
			fmt.Sprintf("failed to get manifest with status %d: %s", resp.StatusCode, string(body)),
			nil,
		)
	}

	return io.ReadAll(resp.Body)
}

// ValidateImage checks if an image exists in the registry
func (r *DockerHubRegistry) ValidateImage(ctx context.Context, imageRef *docker.ImageReference) (bool, error) {
	repository := imageRef.Repository
	if repository == "" {
		repository = "library/" + imageRef.Name
	} else {
		repository = repository + "/" + imageRef.Name
	}

	reference := imageRef.Tag
	if reference == "" {
		reference = "latest"
	}

	log.Debug().
		Str("repository", repository).
		Str("reference", reference).
		Msg("Validating image existence")

	_, err := r.GetImageManifest(ctx, repository, reference)
	if err != nil {
		// Check if it's a 404 error
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Close releases resources associated with the registry client
func (r *DockerHubRegistry) Close() error {
	r.client.CloseIdleConnections()
	return nil
}
