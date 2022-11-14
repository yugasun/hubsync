#!/bin/sh

source ./.env

go run main.go --username=$DOCKER_USERNAME --password=$DOCKER_TOKEN --content='{ "hubsync": ["hello-world:latest"] }'
