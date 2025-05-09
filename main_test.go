//go:build !integration
// +build !integration

package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputItemStruct(t *testing.T) {
	item := outputItem{Source: "src", Target: "tgt", Repository: "repo"}
	assert.Equal(t, "src", item.Source)
	assert.Equal(t, "tgt", item.Target)
	assert.Equal(t, "repo", item.Repository)
}

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

func TestTargetNameGeneration(t *testing.T) {
	source := "nginx:latest"
	namespace := "testns"
	repository := ""
	target := namespace + "/" + source
	assert.Equal(t, "testns/nginx:latest", target)

	repository = "docker.io"
	target = repository + "/" + namespace + "/" + source
	assert.Equal(t, "docker.io/testns/nginx:latest", target)
}
