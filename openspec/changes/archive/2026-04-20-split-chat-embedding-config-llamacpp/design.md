## Context

**Current State:**
- Configuration couples chat and embedding via provider-level configs (`GeminiConfig`, `OpenAIConfig`)
- Provider inference from model name prefixes (`gemini/model-name`, `openai/model-name`)
- Single shared client per provider used for both chat and embedding
- Environment variables: `GEMINI_API_KEY`, `CHAT_MODEL` (provider/model format), `EMBEDDING_MODEL`

**Constraints:**
- Must maintain Eino framework compatibility (embedding.Embedder, model.BaseChatModel interfaces)
- No external HTTP client dependencies beyond `net/http`
- Config validation via go-playground/validator
- Breaking change requires clear migration path

**Stakeholders:**
- Developers deploying with local llama.cpp instances
- Users needing different API keys for chat vs embedding (cost tracking, rate limits)
- Future: Multi-provider deployments (Gemini chat + OpenAI embeddings, etc.)

## Goals / Non-Goals

**Goals:**
- Independent configuration for chat and embedding (separate API keys, endpoints, models)
- Support llama.cpp as embedding provider via HTTP API
- Configurable prompt templates for different embedding models
- Factory pattern for provider-agnostic model/embedder creation
- Clean separation of concerns (use-case-centric over provider-centric)

**Non-Goals:**
- OpenAI chat/embedding support (future work, scaffolding only)
- Vision/multimodal embeddings (templates support it, implementation deferred)
- Runtime provider switching without restart
- Backward compatibility with old env var structure (breaking change accepted)

## Decisions

### 1. Use-Case-Centric Config Structure

**Decision:** Separate `ChatConfig` and `EmbeddingConfig` structs instead of nested provider configs.

```go
type ChatConfig struct {
    Provider string `mapstructure:"provider"`
    Model    string `mapstructure:"model"`
    APIKey   string `mapstructure:"api_key"`
    BaseURL  string `mapstructure:"base_url"`
}

type EmbeddingConfig struct {
    Provider       string `mapstructure:"provider"`
    Model          string `mapstructure:"model"`
    Dimension      int    `mapstructure:"dimension"`
    APIKey         string `mapstructure:"api_key"`
    BaseURL        string `mapstructure:"base_url"`
    PromptTemplate string `mapstructure:"prompt_template"`
}
```

**Rationale:**
- Aligns with mental model: "I need a chat model" vs "I need Gemini"
- Simpler validation (no cross-provider dependency checks)
- Easier to add new providers (single switch statement per factory)
- Clearer env var names (`CHAT_PROVIDER` vs `MODEL.ChatModel` parsing)

**Alternatives Considered:**
- *Provider-centric* (Option A from exploration): `Gemini.Chat`, `Gemini.Embedding`, `OpenAI.Chat`...
  - Rejected: More verbose, harder to find "what's my chat model?", requires nested structs

### 2. Factory Pattern for Provider Creation

**Decision:** Create `chat_factory.go` and `embedding_factory.go` with `CreateChatModel()` and `CreateEmbedder()` functions.

```go
func CreateChatModel(ctx context.Context, cfg *config.ChatConfig) (model.BaseChatModel, error)
func CreateEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (embedding.Embedder, error)
```

**Rationale:**
- Removes provider inference logic from `cmd/serve.go`
- Each factory handles provider-specific initialization
- Easy to add new providers (add case in switch, implement create function)
- Testable in isolation (mock config, verify correct provider created)

**Alternatives Considered:**
- *Keep InferProvider*: Rejected - couples config to model name format, requires parsing
- *Single factory with type parameter*: Rejected - Go generics would complicate, separate factories are clearer

### 3. llama.cpp HTTP Client Implementation

**Decision:** Implement llama.cpp embedder using `net/http` with batch support and configurable timeout.

**Request Format:**
```json
{
  "content": [
    {
      "prompt_string": "{{.Text}}",
      "multimodal_data": []
    }
  ]
}
```

**Response Format:**
```json
[
  {
    "index": 0,
    "embedding": [0.013, -0.031, ...]
  }
]
```

**Rationale:**
- No external dependencies (uses stdlib)
- Batch requests reduce HTTP overhead
- Index-based response matches llama.cpp's batch design
- Configurable timeout for local vs remote deployments

**Trade-offs:**
- No connection pooling by default (can add later if needed)
- No retry logic (llama.cpp typically local, adds complexity)

### 4. Go Template Syntax for Prompt Templates

**Decision:** Use Go `text/template` for prompt template variable substitution.

```go
type PromptTemplate struct {
    Name        string
    Description string
    Template    string  // Go template syntax
    Variables   []string // Required: {{.Text}}, {{.Media}}
}
```

**Rationale:**
- Stdlib, no new dependencies
- Familiar to Go developers
- Supports conditionals, loops (future extensibility)
- Easy to validate template syntax at startup

**Alternatives Considered:**
- *fmt.Sprintf*: Rejected - no extensibility, single variable only
- *Custom placeholder syntax*: Rejected - reinventing the wheel, error-prone

### 5. Built-in Template Registry

**Decision:** Pre-define templates in code (`default`, `with-instruction`, `vision`) loaded at startup.

```go
// pkg/embedding/templates/default.go
var DefaultTemplate = &PromptTemplate{
    Name:     "default",
    Template: `{"content": [{"prompt_string": "{{.Text}}", "multimodal_data": []}]}`,
    Variables: []string{".Text"},
}
```

**Rationale:**
- No external template files to manage
- Type-safe (compile-time checking)
- Easy to add new templates (add function, register in init)
- Fallback if user specifies invalid template name

**Alternatives Considered:**
- *File-based templates*: Rejected - deployment complexity, file I/O errors
- *User-provided templates*: Rejected - security risk (code injection), harder to validate

## Risks / Trade-offs

### Breaking Change: Environment Variables

**Risk:** Existing deployments will fail on upgrade due to env var changes.

**Mitigation:**
- Clear error message on validation failure listing required new vars
- Migration guide in proposal and README
- Version bump (e.g., v0.2.0 → v1.0.0) to signal breaking change

### Shared State Removal

**Risk:** Removing shared `geminiClient` means two separate Gemini clients (chat + embedding).

**Mitigation:**
- Gemini supports multiple clients from same API key
- Connection pooling handled by google/genai library internally
- No functional impact (separate clients are independent)

### llama.cpp API Compatibility

**Risk:** llama.cpp server API may change between versions.

**Mitigation:**
- Document tested llama.cpp version (e.g., llama.cpp b3662+)
- Graceful error on unexpected response format
- Configurable endpoint allows users to run specific version

### Template Validation

**Risk:** Invalid template syntax causes runtime panic.

**Mitigation:**
- Validate all templates at startup during config load
- Return clear error message on template parse failure
- Default template always available as fallback

## Migration Plan

### Phase 1: Prepare (Zero Downtime)
1. Create new config structs alongside old ones
2. Implement factories and llama.cpp embedder
3. Update `.env.example` with new structure
4. Add deprecation warnings for old env vars

### Phase 2: Migrate
1. Users update `.env` files with new env vars
2. Test with existing Gemini configuration
3. Optionally test with llama.cpp embeddings

### Phase 3: Deploy
1. Deploy new code with migrated configs
2. Verify embeddings and chat working
3. Monitor for any validation errors

### Phase 4: Cleanup (Post-Release)
1. Remove deprecated `GeminiConfig.ChatModel/EmbeddingModel` fields
2. Remove `InferProvider()` function
3. Remove shared client logic from `cmd/serve.go`

### Rollback Strategy
- Keep old config code in separate branch for quick revert
- If issues arise, users can revert to old env vars and rollback code
- Version tag before migration allows quick `git checkout`

## Open Questions

1. **Should we support custom prompt templates from files?**
   - Decision: Deferred to future enhancement (v1.1.0)
   - Current: Built-in templates sufficient for initial release

2. **Should llama.cpp API key be optional for local deployments?**
   - Decision: Yes, empty string = no Authorization header
   - Documented in config validation rules

3. **Should we add connection pooling for llama.cpp HTTP client?**
   - Decision: Deferred until performance testing shows need
   - Current: `http.DefaultClient` sufficient for local deployments

4. **What timeout should we use for llama.cpp requests?**
   - Decision: Configurable, default 30s
   - Local llama.cpp typically fast (<1s), remote may vary
