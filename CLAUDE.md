# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test Commands

- Build: `just build` or `go build -o mcp-language-server`
- Install locally: `just install` or `go install`
- Generate schema: `just generate` or `go generate ./...`
- Code audit: `just check` or `go tool staticcheck ./... && go tool govulncheck ./... && go tool errcheck ./...`
- Run tests: `go test ./...`
- Run single test: `go test -run TestName ./path/to/package`

## Code Style Guidelines

- Follow standard Go conventions (gofmt)
- Error handling: Return errors with context using `fmt.Errorf("failed to X: %w", err)`
- Tool functions return both result and error
- Context should be first parameter for functions that need it
- Types should have proper documentation comments
- Config validation in separate functions
- Proper resource cleanup in shutdown handlers

## Behaviour

- Don't make assumptions. Ask the user clarifying questions.
- Ask the user before making changes and only do one thing at a time. Do not dive in and make additional optimizations without asking first.
- After completing a task, run `go fmt` and `go tool staticcheck`
- When finishing a task, run tests and ask the user to confirm that it works
- Do not update documentation until finished and the user has confirmed that things work
- Use `any` instead of `interface{}`

## Notes about codebase

- Most of the `internal/protocol` package is auto generated based on the LSP spec. Do not make edits to it. The files are large, so use grep to search them instead of reading the whole file if possible.
- Types and methods related to the LSP spec are auto generated and should be used instead of making own types.
- The exception is the `protocol/interfaces.go`` file. It contains interfaces that account for the fact that some methods may have multiple return types
- Check for existing helpers and types before making them.
- This repo is for a Model Context Provider (MCP) server. It runs a Language Server specified by the user and communicates with it over stdio. It exposes tools to interact with it via the MCP protocol.
- Integration tests are in the `integrationtests/` folder and these should be used for development. This is the main important test suite.
- Moving forwards, add unit tests next to the relevant code.

