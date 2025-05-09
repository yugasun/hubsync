.PHONY: run test integration-test

run:
	sh run.sh $(image)

test:
	go test -v -short ./...

integration-test:
	go test -v -tags=integration ./...