# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Build: `make build` - Builds the connector for current OS/arch
- Lint: `make lint` - Runs golangci-lint on the codebase
- Dependencies: `make update-deps` - Updates dependencies
- Add dependency: `make add-dep` - Adds a dependency and updates modules

## Run Commands
- Run connector: `./dist/darwin_arm64/baton-asana -h` to see options
- Use `baton` CLI to verify sync.c1z files: `baton -h` for capabilities

## Code Style
- Follow Go standard formatting with gofmt
- Error handling: Use errors.Join for multiple errors, wrap with %w
- Logging: Use zap logger from context (ctxzap.Extract(ctx))
- Naming: camelCase for variables, constants at package level
- Options pattern: Use WithX() functions for configuration
- Resource types prefixed with "resourceType"
- Max line length: 200 characters

## Project Structure
- cmd/ - Main executable code
- pkg/ - Core connector logic (client, models, helpers)
- Use functional options pattern for configuration
- Context should be passed and propagated through function calls
- Always close HTTP response bodies with defer