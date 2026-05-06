## Why

Hand-written mocks are duplicated across test files — `MockSourceRepository` is copy-pasted in 3 files, `MockChatModel` in 2. When interfaces change, every copy must be updated manually. This is error-prone boilerplate that grows with each new repository.

## What Changes

- Add `mockery` as a dev dependency for generating testify-compatible mock implementations
- Configure `.mockery.yaml` to generate mocks for all 9 repository interfaces and 2 external model interfaces
- Generate mocks into `internal/mocks/repositories/` and `internal/mocks/models/`
- Add `make mocks` Makefile target for regeneration
- Replace all hand-written mocks with generated ones
- Delete inline mock definitions from test files

## Capabilities

### New Capabilities
- `mock-generation`: Automated mock generation using mockery for all domain and external interfaces

### Modified Capabilities
_(none — this is a test infrastructure change, no spec-level behavior changes)_

## Impact

**New Files:**
- `.mockery.yaml` — Mockery configuration
- `internal/mocks/repositories/` — Generated repository mocks (9 files)
- `internal/mocks/models/` — Generated model mocks (2 files)

**Modified Files:**
- `Makefile` — Add `mocks` target
- All test files currently containing hand-written mocks (3-4 files)

**Deleted Code:**
- Inline `MockSourceRepository` definitions (3 copies)
- Inline `MockChatModel` / `MockToolCallingChatModel` definitions (2 copies)