package unit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUnmarshalHubMirrors tests the JSON parsing for the hubsync format
func TestUnmarshalHubMirrors(t *testing.T) {
	jsonStr := `{"hubsync": ["nginx:latest", "alpine:3.18"]}`
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}
	err := json.Unmarshal([]byte(jsonStr), &hubMirrors)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(hubMirrors.Content))
	assert.Equal(t, "nginx:latest", hubMirrors.Content[0])
	assert.Equal(t, "alpine:3.18", hubMirrors.Content[1])
}

// TestInvalidJSON tests handling of invalid JSON content
func TestInvalidJSON(t *testing.T) {
	invalidJSON := `{"hubsync": ["nginx:latest", "alpine:3.18"`
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}
	err := json.Unmarshal([]byte(invalidJSON), &hubMirrors)
	assert.Error(t, err)
}

// TestEmptyJSON tests handling of empty JSON content
func TestEmptyJSON(t *testing.T) {
	emptyJSON := `{"hubsync": []}`
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}
	err := json.Unmarshal([]byte(emptyJSON), &hubMirrors)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(hubMirrors.Content))
}

// TestComplexJSON tests handling of more complex JSON content
func TestComplexJSON(t *testing.T) {
	complexJSON := `{
		"hubsync": [
			"nginx:latest",
			"alpine:3.18",
			"ubuntu:22.04$myubuntu",
			"k8s.gcr.io/kube-apiserver:v1.23.0"
		]
	}`
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}
	err := json.Unmarshal([]byte(complexJSON), &hubMirrors)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(hubMirrors.Content))
	assert.Equal(t, "nginx:latest", hubMirrors.Content[0])
	assert.Equal(t, "alpine:3.18", hubMirrors.Content[1])
	assert.Equal(t, "ubuntu:22.04$myubuntu", hubMirrors.Content[2])
	assert.Equal(t, "k8s.gcr.io/kube-apiserver:v1.23.0", hubMirrors.Content[3])
}
