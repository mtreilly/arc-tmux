.PHONY: test lint pre-push

test:
	go test ./...

lint:
	golangci-lint run

pre-push: lint test
