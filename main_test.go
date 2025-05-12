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

func TestGenerateTargetName(t *testing.T) {
	testCases := []struct {
		name           string
		source         string
		repositoryURL  string
		namespace      string
		expectedSource string
		expectedTarget string
	}{
		{
			name:           "Standard input without $ and with tag",
			source:         "nginx:1.19",
			repositoryURL:  "",
			namespace:      "yugasun",
			expectedSource: "nginx:1.19",
			expectedTarget: "yugasun/nginx:1.19",
		},
		{
			name:           "Standard input without $ and without tag",
			source:         "nginx",
			repositoryURL:  "",
			namespace:      "yugasun",
			expectedSource: "nginx:latest",
			expectedTarget: "yugasun/nginx:latest",
		},
		{
			name:           "Custom input with $ symbol",
			source:         "yugasun/alpine$alpine",
			repositoryURL:  "",
			namespace:      "yugasun",
			expectedSource: "yugasun/alpine:latest",
			expectedTarget: "yugasun/yugasun.alpine:latest",
		},
		{
			name:           "Custom input with $ symbol and tag",
			source:         "yugasun/alpine:v1$alpine",
			repositoryURL:  "",
			namespace:      "yugasun",
			expectedSource: "yugasun/alpine:v1",
			expectedTarget: "yugasun/alpine:v1",
		},
		{
			name:           "With repository address",
			source:         "nginx:1.19",
			repositoryURL:  "registry.cn-beijing.aliyuncs.com",
			namespace:      "yugasun",
			expectedSource: "nginx:1.19",
			expectedTarget: "registry.cn-beijing.aliyuncs.com/yugasun/nginx:1.19",
		},
		{
			name:           "With repository address and $ symbol",
			source:         "yugasun/alpine$alpine",
			repositoryURL:  "registry.cn-beijing.aliyuncs.com",
			namespace:      "yugasun",
			expectedSource: "yugasun/alpine:latest",
			expectedTarget: "registry.cn-beijing.aliyuncs.com/yugasun/alpine:latest",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotSource, gotTarget := generateTargetName(tc.source, tc.repositoryURL, tc.namespace)
			if gotSource != tc.expectedSource {
				t.Errorf("generateTargetName() source image error = %v, expected %v", gotSource, tc.expectedSource)
			}
			if gotTarget != tc.expectedTarget {
				t.Errorf("generateTargetName() target image error = %v, expected %v", gotTarget, tc.expectedTarget)
			}
		})
	}
}
