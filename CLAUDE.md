# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Build: `go build -o twitchlinker`
- Run: `./twitchlinker` 
- Docker build: `docker build -t twitchlinker .`
- Docker run: `docker-compose up`

## Test Commands
- Run all tests: `go test ./...`
- Run package tests: `go test ./pkg/twitch`
- Run single test: `go test ./pkg/twitch -run TestFunctionName`
- Test with coverage: `go test -cover ./...`

## Code Style Guidelines
- Use `go fmt ./...` to format code before committing
- Package structure: main at root, components in `/pkg` directory
- Imports: standard lib → third-party → internal packages
- Naming: PascalCase for exported, camelCase for unexported
- Error handling: check all errors, use `fmt.Errorf` with `%w` for wrapping
- Configuration: use environment variables with validation
- Use consistent log patterns with appropriate levels
- Document functions with explanatory comments
- Use Go modules for dependency management

## Development Workflow
- Add tests for all packages
- Consider using golangci-lint for static analysis