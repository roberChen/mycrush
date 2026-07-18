---
id: 011242f5530b26a261ed49156ee1a236
title: Build, Test, Lint, and Format Commands
keywords:
    - build
    - test
    - lint
    - format
    - gofumpt
    - golangci-lint
    - task
    - commands
    - golden files
    - VCR
summary: Build, test, lint, and format commands for the Crush project. Uses Taskfile, gofumpt, golangci-lint, and standard Go tooling.
created: 2026-07-10T19:23:27.562413833Z
updated: 2026-07-10T19:23:27.562413833Z
---

## Build Commands

```bash
# Build
go build .                          # or: task build
go run .                            # run directly

# Build with profiling
task dev                            # sets CRUSH_PROFILE=true

# Install
task install                        # go install with version ldflags
```

## Test Commands

```bash
# Run all tests with race detector
task test                           # go test -race -failfast ./...

# Run single test
go test ./internal/llm/prompt -run TestGetContextFromPaths

# Re-record VCR cassettes (LLM API recordings)
task test:record                    # rm -r internal/agent/testdata && go test -v ./internal/agent

# Update golden files
go test ./... -update
# Update specific package golden files
go test ./internal/tui/components/core -update
```

## Lint & Format

```bash
task fmt                            # gofumpt -w .
task lint                           # lint:log + golangci-lint run
task lint:fix                       # golangci-lint run --fix
task modernize                      # go run modernize analyzer
```

## Other Tasks

```bash
task schema                         # Generate JSON schema for config → schema.json
task hyper                          # Update embedded Hyper provider.json
task deps                           # Update Fantasy and Catwalk dependencies
task swag                           # Generate OpenAPI spec
```

## Code Style

- **Formatter:** gofumpt (stricter than gofmt)
- **Imports:** goimports, grouped (stdlib, external, internal)
- **Log messages:** Must start with capital letter (enforced by `task lint:log`)
- **Comments:** Own-line comments start with capital, end with period. Wrap at 78 columns.
- **JSON tags:** snake_case
- **File permissions:** Octal notation (0o755, 0o644)
- **Testing:** testify `require`, `t.Parallel()`, `t.SetEnv()`, `t.TempDir()`
