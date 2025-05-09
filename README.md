# hubsync

A tool for accelerating the download of images from foreign registries such as gcr.io, k8s.gcr.io, quay.io, ghcr.io, etc., using docker.io or other mirror services.

> To avoid duplicate requests and make efficient use of resources, please search the issues to see if the image has already been mirrored.

## Getting Started

### Option 1: Submit via GitHub Issue

- **Requirement:** Strictly follow the [template](https://github.com/yugasun/hubsync/issues/2) when submitting.
- **Limit:** Up to 11 image addresses per submission.
- **Note:** Docker accounts have daily pull limits. Please use responsibly.

### Option 2: Use GitHub Actions

1. **Bind your DockerHub account:**  
   Go to `Settings` → `Secrets` → `Actions` and add two secrets:

   - `DOCKER_USERNAME` (your Docker username)
   - `DOCKER_PASSWORD` (your Docker password)

2. **Enable Issues:**  
   In `Settings` → `Options` → `Features`, enable the `Issues` feature.

3. **Add Labels:**  
   In `Issues` → `Labels`, add the following labels: `hubsync`, `success`, `failure`.

### Option 3: Run Locally

1. **Clone the repository:**

   ```shell
   git clone https://github.com/yugasun/hubsync
   cd hubsync
   ```

2. **Install dependencies:**

   ```shell
   go install
   ```

3. **Run the sync:**

   ```shell
   go run main.go --username=xxxxxx --password=xxxxxx --content='{ "hubsync": ["hello-world:latest"] }'
   ```

   **To use a custom image registry:**

   ```shell
   go run main.go --username=xxxxxx --password=xxxxxx --repository=registry.cn-hangzhou.aliyuncs.com/xxxxxx --content='{ "hubsync": ["hello-world:latest"] }'
   ```

## License

MIT @yugasun
