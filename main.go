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
		if strings.Contains(source, "$") {
			str1 := strings.Split(source, "$")
			repositoryParts := strings.Split(str1[0], ":")
			target = str1[1] + ":" + repositoryParts[len(repositoryParts)-1]
			source = str1[0]
		}
		if *repository == "" {
			target = *namespace + "/" + strings.ReplaceAll(target, "/", ".")
		} else {
			target = *repository + "/" + *namespace + "/" + strings.ReplaceAll(target, "/", ".")
		}
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
		loginCmd = fmt.Sprintf("# If your repository is private, please login first...\n# docker login %s --username={your username}\n", output[0].Repository)
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
