# Agent Instructions

Before pushing, always run lint and tests:

- `golangci-lint run`
- `go test ./...`

To make this the default behavior, enable the repo hook:

- `git config core.hooksPath .githooks`

Or use the convenience target:

- `make pre-push`
