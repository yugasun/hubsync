//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestMainIntegration(t *testing.T) {
	// 加载 .env 文件
	_ = godotenv.Load(".env")

	// 构造测试用的 content
	content := map[string][]string{
		"hubsync": {"alpine:3.18"},
	}
	contentBytes, _ := json.Marshal(content)

	outputPath := "test_output.log"
	defer os.Remove(outputPath)

	cmd := exec.Command("go", "run", "main.go",
		"--content", string(contentBytes),
		"--outputPath", outputPath,
	)
	cmd.Env = append(os.Environ(),
		"DOCKER_USERNAME="+os.Getenv("DOCKER_USERNAME"),
		"DOCKER_PASSWORD="+os.Getenv("DOCKER_PASSWORD"),
		"DOCKER_REPOSITORY="+os.Getenv("DOCKER_REPOSITORY"),
		"DOCKER_NAMESPACE="+os.Getenv("DOCKER_NAMESPACE"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("main.go execution failed: %v\nOutput: %s", err, string(output))
	}

	// 检查输出文件内容
	data, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "docker pull"))
}
