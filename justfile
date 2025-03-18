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
  go tool staticcheck ./...
  go tool govulncheck ./...
