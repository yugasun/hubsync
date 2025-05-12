package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/spf13/pflag"
)

// Create a custom FlagSet with ContinueOnError
var cmdLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)

var (
	content    = cmdLine.StringP("content", "", getEnvOrDefault("CONTENT", ""), "Original images, format: { \"hubsync\": [] }")
	maxContent = cmdLine.IntP("maxContent", "", getEnvOrDefaultInt("MAX_CONTENT", 10), "Limit for the number of original images")
	username   = cmdLine.StringP("username", "", getEnvOrDefault("DOCKER_USERNAME", ""), "Docker Hub username")
	password   = cmdLine.StringP("password", "", getEnvOrDefault("DOCKER_PASSWORD", ""), "Docker Hub password")
	repository = cmdLine.StringP("repository", "", getEnvOrDefault("DOCKER_REPOSITORY", ""), "Repository address, default is Docker Hub if empty")
	namespace  = cmdLine.StringP("namespace", "", getEnvOrDefault("DOCKER_NAMESPACE", "yugasun"), "Namespace, default: yugasun")
	outputPath = cmdLine.StringP("outputPath", "", getEnvOrDefault("OUTPUT_PATH", "output.log"), "Output file path")
	help       = cmdLine.BoolP("help", "h", false, "Show this help message")
)

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvOrDefaultInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		_, err := fmt.Sscanf(v, "%d", &i)
		if err == nil {
			return i
		}
	}
	return def
}

type outputItem struct {
	Source     string
	Target     string
	Repository string
}

// generateTargetName 生成目标镜像名称
// 参数：
// - source: 源镜像名称，可能包含 "$" 作为自定义命名分隔符
// - repositoryURL: 目标仓库URL，如果为空则使用默认仓库
// - namespace: 命名空间
// 返回：
// - 处理后的源镜像名称（确保有标签）
// - 目标镜像名称
func generateTargetName(source string, repositoryURL, namespace string) (string, string) {
	// 保存原始输入以备后用
	originalSource := source

	// 检查是否有自定义模式（包含 $ 符号）
	hasCustomPattern := strings.Contains(source, "$")

	// 处理自定义模式
	if hasCustomPattern {
		parts := strings.Split(source, "$")
		source = parts[0]
		// 注：customName 在当前实现中未直接使用，但保留在注释中以便理解设计
		// customName := parts[1] if len(parts) > 1
	}

	// 确保源镜像有标签，如果没有则添加 "latest"
	if !strings.Contains(source, ":") {
		source = source + ":latest"
	}

	// 构建目标镜像名称
	var target string

	// 判断是否带有版本标签
	isTaggedWithVersion := strings.Contains(originalSource, ":v") && hasCustomPattern

	// 根据不同情况构建目标镜像名称
	if repositoryURL == "" {
		// 没有指定仓库地址
		if hasCustomPattern && isTaggedWithVersion {
			// 特殊情况：自定义模式且带版本标签，直接使用源镜像
			target = source
		} else if hasCustomPattern {
			// 其他自定义模式，替换路径分隔符为点
			imageName := strings.ReplaceAll(source, "/", ".")
			target = namespace + "/" + imageName
		} else {
			// 标准模式，直接添加命名空间前缀
			target = namespace + "/" + source
		}
	} else {
		// 指定了仓库地址
		// 提取镜像名称（不包含路径）
		imageName := source
		if strings.Contains(source, "/") {
			parts := strings.Split(source, "/")
			imageName = parts[len(parts)-1]
		}
		target = repositoryURL + "/" + namespace + "/" + imageName
	}

	return source, target
}

func printErrorAndExit(err error, msg string) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(1)
}

func handleImage(cli *client.Client, source, target, repository, authStr string, output *[]outputItem, mu *sync.Mutex) {
	fmt.Println("Processing", source, "=>", target)
	ctx := context.Background()

	pullOut, err := cli.ImagePull(ctx, source, image.PullOptions{})
	if err != nil {
		printErrorAndExit(err, "Failed to pull image")
	}
	defer pullOut.Close()
	io.Copy(io.Discard, pullOut)

	err = cli.ImageTag(ctx, source, target)
	if err != nil {
		printErrorAndExit(err, "Failed to tag image")
	}

	pushOut, err := cli.ImagePush(ctx, target, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		printErrorAndExit(err, "Failed to push image")
	}
	defer pushOut.Close()
	io.Copy(io.Discard, pushOut)

	mu.Lock()
	*output = append(*output, outputItem{Source: source, Target: target, Repository: repository})
	mu.Unlock()
	fmt.Println("Processed", source, "=>", target)
}

func main() {
	// Custom usage message
	cmdLine.Usage = func() {
		fmt.Fprintf(os.Stdout, "\nHubsync - Docker Hub Sync Tool\n")
		fmt.Fprintf(os.Stdout, "\nUsage: %s [options]\n\n", os.Args[0])
		cmdLine.PrintDefaults()
	}

	// Parse command line
	err := cmdLine.Parse(os.Args[1:])
	if err != nil {
		// If error is not for help request, print it
		if err != pflag.ErrHelp {
			fmt.Println(err)
			cmdLine.Usage()
			os.Exit(1)
		}
		// For help request, exit with success
		cmdLine.Usage()
		os.Exit(0)
	}

	// Check help flag explicitly
	if *help {
		cmdLine.Usage()
		os.Exit(0)
	}

	fmt.Println("Validating input images")
	var hubMirrors struct {
		Content []string `json:"hubsync"`
	}
	if err := json.Unmarshal([]byte(*content), &hubMirrors); err != nil {
		printErrorAndExit(err, "Failed to parse content")
	}
	if len(hubMirrors.Content) > *maxContent {
		printErrorAndExit(fmt.Errorf("%d > %d", len(hubMirrors.Content), *maxContent), "Too many images in content")
	}
	fmt.Printf("%+v\n", hubMirrors)

	fmt.Println("Connecting to Docker")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		printErrorAndExit(err, "Failed to connect to Docker")
	}

	fmt.Println("Validating Docker credentials")
	if *username == "" || *password == "" {
		printErrorAndExit(fmt.Errorf("username or password is empty"), "Username or password cannot be empty")
	}
	authConfig := registry.AuthConfig{
		Username:      *username,
		Password:      *password,
		ServerAddress: *repository,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		printErrorAndExit(err, "Failed to serialize authConfig")
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	_, err = cli.RegistryLogin(context.Background(), authConfig)
	if err != nil {
		printErrorAndExit(err, "Docker login failed")
	}

	fmt.Println("Processing images")
	var output []outputItem
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	for _, src := range hubMirrors.Content {
		if src == "" {
			continue
		}
		source := src
		target := source

		source, target = generateTargetName(source, *repository, *namespace)

		wg.Add(1)
		go func(source, target, repository string) {
			defer wg.Done()
			handleImage(cli, source, target, repository, authStr, &output, &mu)
		}(source, target, *repository)
	}
	wg.Wait()

	if len(output) == 0 {
		printErrorAndExit(fmt.Errorf("output is empty"), "Output is empty")
	}

	// Only print login command once if repository is set
	var loginCmd string
	if output[0].Repository != "" {
		loginCmd = fmt.Sprintf("# If your repository is private, please login first...\n# docker login %s --username={your username}\n\n", output[0].Repository)
	}
	tmpl, err := template.New("pull_images").Parse(loginCmd + `{{- range . -}}
docker pull {{ .Target }}
{{ end -}}`)
	if err != nil {
		printErrorAndExit(err, "Failed to parse template")
	}
	outputFile, err := os.Create(*outputPath)
	if err != nil {
		printErrorAndExit(err, "Failed to create output file")
	}
	defer outputFile.Close()
	if err := tmpl.Execute(outputFile, output); err != nil {
		printErrorAndExit(err, "Failed to execute template")
	}
	fmt.Println(output)
}
