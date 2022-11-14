#!/bin/env/bash sh

go run main.go --username= <DockerHubUsername >--password= <DockerHubPassword >--content='{ "hubsync": ["helloworld:latest"] }'
