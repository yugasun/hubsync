.PHONY: run test integration-test

help:
	go run main.go --help

run:
	sh run.sh $(image)

test:
	go test -v -short ./...


integration-test:
	go test -v -tags=integration ./...