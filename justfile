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
  test -z "$(gofmt -s -l .)"
  go tool staticcheck ./...
  go tool govulncheck ./...
  go tool errcheck ./...
  find . -name "*.go" | xargs gopls check

# Run tests
test:
  go test ./...
