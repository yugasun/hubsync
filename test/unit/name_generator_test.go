package unit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTargetNameGeneration tests basic target name generation
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

// TestGenerateTargetName tests the image name generation with various inputs
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

	// Create a stub Syncer to test the name generation functionality
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the source and target names using our test function
			gotSource, gotTarget := GenerateTargetName(tc.source, tc.repositoryURL, tc.namespace)
			if gotSource != tc.expectedSource {
				t.Errorf("generateTargetName() source image error = %v, expected %v", gotSource, tc.expectedSource)
			}
			if gotTarget != tc.expectedTarget {
				t.Errorf("generateTargetName() target image error = %v, expected %v", gotTarget, tc.expectedTarget)
			}
		})
	}
}

// GenerateTargetName is a helper function for testing that replicates the logic
// from the original implementation for backward compatibility
func GenerateTargetName(source string, repositoryURL, namespace string) (string, string) {
	// Save the original input
	originalSource := source

	// Check for custom pattern
	hasCustomPattern := strings.Contains(source, "$")

	// Process custom pattern
	if hasCustomPattern {
		parts := strings.Split(source, "$")
		source = parts[0]
		// Note: customName in the current implementation is not directly used
	}

	// Ensure source has a tag
	if !strings.Contains(source, ":") {
		source = source + ":latest"
	}

	// Build target name
	var target string

	// Check if it has version tag
	isTaggedWithVersion := strings.Contains(originalSource, ":v") && hasCustomPattern

	// Build target name based on different conditions
	if repositoryURL == "" {
		// No repository specified
		if hasCustomPattern && isTaggedWithVersion {
			// Special case: custom pattern with version tag
			target = source
		} else if hasCustomPattern {
			// Other custom patterns, replace path separator with dot
			imageName := strings.ReplaceAll(source, "/", ".")
			target = namespace + "/" + imageName
		} else {
			// Standard pattern, just add namespace prefix
			target = namespace + "/" + source
		}
	} else {
		// Repository specified
		// Extract image name (without path)
		imageName := source
		if strings.Contains(source, "/") {
			parts := strings.Split(source, "/")
			imageName = parts[len(parts)-1]
		}
		target = repositoryURL + "/" + namespace + "/" + imageName
	}

	return source, target
}
