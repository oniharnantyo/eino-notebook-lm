# Architecture Deepening Priority List

**Context**: Codebase analysis identified 5 major opportunities to convert shallow modules into deep, testable components. Each change is documented in `openspec/changes/` with proposal, specs, design, and tasks artifacts.

---

## Priority 1: Response API Deepening (HIGHEST)

**Status**: ✅ Artifacts complete | ⚠️ Implementation incomplete

**Problem**: `response_usecase.go` is 890 lines of untested orchestration coordinating 8+ services (agent factory, tool factory, retrievers, chat model, history manager, repositories).

**Solution**: Split into 4 focused stages (retrieval, tool preparation, generation, history) with clear interfaces and testable boundaries.

**Artifacts**: `openspec/changes/response-api-deepening/`
- `proposal.md` — Simplify 965-line usecase into focused modules
- `specs/response-orchestration/spec.md` — 8 requirements covering stages, retrieval, tools, generation, history, errors
- `design.md` — 4-stage pipeline architecture, 7 decisions, 4-phase migration plan
- `tasks.md` — 47 tasks (22 incomplete: stage implementations and orchestration replacement)

**Current State**: Stage interfaces created, but implementations are placeholders. The 890-line `response_usecase.go` still contains all the logic.

**Why Highest Priority**:
- Core feature (every request flows through this)
- Completely untested (no test file exists)
- Highest cognitive load (8+ dependencies to understand)
- Blocks ability to safely modify chat behavior

---

## Priority 2: Domain Entity Behavior (FOUNDATIONAL)

**Status**: ⚪ Not started

**Problem**: Domain entities (`internal/core/domain/entities/`) are anemic—data containers with no behavior. Business logic lives in usecases, not in the domain.

**Solution**: Move business logic from usecases into entities. For example, `Source.StatusTransition(from, to Status)` should validate state transitions.

**Impact**:
- `internal/core/domain/entities/knowledge.go`
- `internal/core/domain/entities/conversation.go`
- `internal/core/domain/entities/source.go`
- All usecase files (logic moves into entities)

**Why Foundational**:
- Enables long-term architectural health
- Addresses "anemic domain model" anti-pattern
- Makes usecases thinner and more focused
- Changes how business rules are organized throughout the codebase

---

## Priority 3: Unified Retrieval Seam (QUICK WIN)

**Status**: ✅ Artifacts complete (archived)

**Problem**: Three separate retriever types (`KnowledgesRetriever`, `SentencesRetriever`, `ImagesRetriever`) with nearly identical implementations—only difference is table name in SQL.

**Solution**: Create single `Retriever` module with `TableConfig` parameter. Existing retrievers become thin adapters.

**Artifacts**: `openspec/changes/archive/2026-05-03-unified-retrieval-seam/`

**Why Quick Win**:
- Clear duplication (3 identical implementations)
- Simple interface change
- Low risk, high value
- Enables simplification of Priority #1 (response API)

---

## Priority 4: Ingestion Pipeline (HIGH)

**Status**: ✅ Artifacts complete (archived)

**Problem**: Content ingestion requires coordinating 7+ services. `source/usecase.go` (627 lines) has duplicated `processAsync` and `processSync` methods with same 6-step pipeline.

**Solution**: Extract deep `IngestionPipeline` module with interface `Ingest(ctx, source) <-chan Progress`. Current usecases become adapters.

**Artifacts**: `openspec/changes/archive/2026-05-03-ingestion-pipeline/`

**Why High Priority**:
- Second-most critical feature after chat
- Duplicated logic (sync/async paths)
- Direct impact on data quality
- 627 lines of untested code

---

## Priority 5: Model Provider Seam (LOWEST)

**Status**: ⚪ Not started

**Problem**: Three separate factory modules (`chat_factory.go`, `embedding_factory.go`, `description_factory.go`) with identical structure. Type assertions (`pgRetriever, ok := uc.retriever.(*pgvector.Retriever)`) indicate abstraction leakage.

**Solution**: Create unified `ModelFactory` module with single interface: `CreateModel(modelType, config) (Model, error)`.

**Impact**:
- `pkg/model/chat_factory.go`
- `pkg/model/embedding_factory.go`
- `pkg/model/description_factory.go`
- Use cases with type assertions

**Why Lowest Priority**:
- Current factories work correctly
- Purely code organization
- No testability or correctness impact
- Adds no new behavior

---

## Recommended Implementation Sequence

```
3 → 2 → 1 → 4 → 5
```

1. **Unified Retrieval** (quick win, enables #1)
2. **Domain Entity** (foundational)
3. **Response API** (high impact, now easier)
4. **Ingestion Pipeline** (high impact)
5. **Model Provider** (cleanup)

**Rationale**: Quick wins first (#3) build momentum and enable larger changes. Foundation (#2) comes before high-impact features (#1, #4).

---

## Accessing Artifacts

Each change has 4 artifacts in `openspec/changes/<change-name>/`:

- `proposal.md` — Why and what (under 300 words)
- `specs/<capability>/spec.md` — Requirements with WHEN/THEN scenarios
- `design.md` — How (architecture, decisions, migration plan)
- `tasks.md` — Implementation checklist with checkboxes

**Archived changes**: Located in `openspec/changes/archive/YYYY-MM-DD-<change-name>/`

**To restore**: 
```bash
mv openspec/changes/archive/YYYY-MM-DD-<name> openspec/changes/<name>
```

**To continue work**: Use `/opsx:continue <change-name>` after restoring
