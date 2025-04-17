# Help
help:
  just -l

# Build
build:
  go build -o mcp-language-server

# Install locally
install:
  go install

# Generate schema
generate:
  go generate ./...

# Run code audit checks
check:
  gofmt -l .
  test -z "$(gofmt -l .)"
  go tool staticcheck ./...
  go tool govulncheck ./...
  go tool errcheck ./...
  find . -path "./integrationtests/workspaces" -prune -o \
    -path "./integrationtests/test-output" -prune -o \
    -name "*.go" -print | xargs gopls check

# Run tests
test:
  go test ./...
