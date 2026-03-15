# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Structure

This is a Go-based notebook application using Clean Architecture with DDD principles:
- `internal/core/domain` - Domain entities and repository interfaces
- `internal/core/application` - Use cases and DTOs
- `internal/infrastructure` - External dependencies (persistence, config)
- `internal/interfaces` - HTTP handlers and routes
- `pkg/` - Shared packages (indexer, parser, uuid, logger)

## Coding Conventions

### Dependency Injection

**All dependencies MUST be initialized via constructors - never nil.**

When a usecase/handler receives dependencies through its constructor:
- DO NOT check `if uc.xxx != nil` or `if h.yyy != nil`
- Assume all dependencies are non-nil and ready to use
- If a dependency might be optional, design it explicitly (e.g., interface with no-op implementation)

Example:
```go
// WRONG
func (uc *usecase) DoSomething() error {
    if uc.repo != nil {  // Unnecessary!
        return uc.repo.Save()
    }
}

// CORRECT
func (uc *usecase) DoSomething() error {
    return uc.repo.Save()  // Dependency is always injected
}
```

### Error Handling

- Use `internal/core/domain/errors` for domain-specific errors
- Wrap errors with context using `fmt.Errorf`
- Return validation errors for invalid input, internal errors for system failures

### Entity Mapping

- Use mappers in `internal/core/application/mappers` for entity <-> DTO conversions
- Keep entities pure (no HTTP concerns)

## Build Commands

- `make build` - Build the application
- `make run` - Run the server
- `make test` - Run tests
- `make lint` - Run linters
