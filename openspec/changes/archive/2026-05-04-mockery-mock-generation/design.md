## Context

Hand-written testify mocks are duplicated across test files. `MockSourceRepository` is defined identically in 3 files, `MockChatModel` in 2. When repository interfaces change (methods added/renamed), each copy must be updated manually. This is ~70 lines of boilerplate per mock per file, and the problem grows with each new repository.

The project uses `testify/mock` for all mocking. mockery generates exactly this format.

## Goals / Non-Goals

**Goals:**
- Eliminate mock duplication across test files
- Ensure mocks stay in sync with interfaces automatically
- Establish a single command (`make mocks`) to regenerate all mocks
- Generate mocks for all 9 domain repository interfaces and 2 external model interfaces

**Non-Goals:**
- Changing test assertions or test structure
- Mocking interfaces that don't yet have hand-written mocks (add those later as needed)
- Using mocking frameworks other than testify/mock

## Decisions

### Output location: `internal/mocks/` with subdirectories

**Decision:** Generate mocks into `internal/mocks/repositories/` for domain interfaces and `internal/mocks/models/` for external interfaces.

**Rationale:**
- Respects Clean Architecture — mocks are test infrastructure, not domain concerns
- Mirrors the domain package structure (`repositories/` maps to domain interfaces)
- Single location for all mocks, easy to import
- Scales cleanly as new interfaces are added

**Alternatives considered:**
1. **Flat `internal/mocks/`** — Simpler but doesn't separate domain from external mocks. Gets cluttered at scale.
2. **Co-located with interfaces** — Pollutes domain layer with test concerns. Violates Clean Architecture.
3. **Package-level mocks/ subdirectories** — Scattered across codebase, harder to generate in one command.

### Configuration: `.mockery.yaml` at repo root

**Decision:** Use a single `.mockery.yaml` configuration file at the repository root.

**Rationale:**
- Single source of truth for all mock generation config
- Standard mockery convention
- Easy to add new interfaces by appending to the config

### Generation trigger: Makefile target only

**Decision:** Use `make mocks` as the generation trigger. No `go:generate` directives.

**Rationale:**
- Matches existing project pattern (all commands via Makefile)
- `go generate ./...` would run during `go test` which is unnecessary overhead
- Makefile target is explicit and intentional
- mockery binary only needed during development, not CI (committed generated files)

**Alternatives considered:**
1. **`go:generate` directives** — Would run during `go test` unnecessarily. More scattered configuration.
2. **Both** — Redundant. One clear path is better.

### Commit generated files

**Decision:** Commit generated mock files to the repository.

**Rationale:**
- No mockery dependency in CI — tests just work
- Git diff shows what changed when interfaces change
- Standard Go practice for generated code
- mockery doesn't need to be installed on every developer machine

### Scope: All repository + model interfaces

**Decision:** Generate mocks for all 9 repository interfaces and 2 model interfaces.

**Interfaces to mock:**

| Package | Interface | Output |
|---------|-----------|--------|
| `repositories` | `SourceRepository` | `internal/mocks/repositories/source_repository.go` |
| `repositories` | `KnowledgeRepository` | `internal/mocks/repositories/knowledge_repository.go` |
| `repositories` | `NotebookRepository` | `internal/mocks/repositories/notebook_repository.go` |
| `repositories` | `ConversationRepository` | `internal/mocks/repositories/conversation_repository.go` |
| `repositories` | `ArtifactRepository` | `internal/mocks/repositories/artifact_repository.go` |
| `repositories` | `SentenceRepository` | `internal/mocks/repositories/sentence_repository.go` |
| `repositories` | `ImageRepository` | `internal/mocks/repositories/image_repository.go` |
| `repositories` | `UserRepository` | `internal/mocks/repositories/user_repository.go` |
| `repositories` | `CacheRepository` | `internal/mocks/repositories/cache_repository.go` |
| `model` (Eino) | `ToolCallingChatModel` | `internal/mocks/models/tool_calling_chat_model.go` |

**Rationale:** Generate all at once to establish the pattern, even if some mocks aren't used yet. Unused mocks don't hurt and will be ready when needed.

## Risks / Trade-offs

### Risk: External interface changes break generated mocks

**Impact:** When Eino updates `ToolCallingChatModel`, `make mocks` must be re-run.

**Mitigation:** Generated files are committed — breakage is caught at compile time. Just re-run `make mocks`.

### Trade-off: Mockery as dev dependency

**Impact:** Developers need mockery installed to regenerate mocks.

**Mitigation:** Mockery is only needed when interfaces change. Generated files are committed, so most developers never need it. Document in Makefile help text.

### Trade-off: One more generated code layer

**Impact:** Test failures may require looking at generated mock code.

**Mitigation:** Generated code is straightforward testify/mock — same pattern as hand-written, just auto-generated. Add `// Code generated by mockery. DO NOT EDIT.` header.

## Migration Plan

1. Install mockery, create `.mockery.yaml`
2. Add `make mocks` to Makefile
3. Generate all mocks
4. Update test imports to use generated mocks
5. Remove inline mock definitions
6. Run tests to verify