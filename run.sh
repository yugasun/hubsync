#!/bin/sh

source ./.env

image=$1
content='{ "hubsync": ["'"$image"'"] }'

go run main.go --username=$DOCKER_USERNAME --password=$DOCKER_TOKEN --content='{ "hubsync": ["'"$image"'"] }' --repository=$DOCKER_REPOSITORY
