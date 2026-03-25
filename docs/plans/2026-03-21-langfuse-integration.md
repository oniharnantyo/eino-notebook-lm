# Langfuse Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Langfuse observability and tracing to all Eino components (ChatModel, Embedder, Retriever, Chain) without modifying existing use case code.

**Architecture:** Leverage Eino's callback system - a single global registration in `cmd/serve.go` automatically traces all component executions. Langfuse handler runs async workers with configurable batching to minimize overhead.

**Tech Stack:** `github.com/cloudwego/eino-ext/callbacks/langfuse`, existing config system with Viper, existing startup/shutdown hooks in serve command

---

### Task 1: Add Langfuse dependency to go.mod

**Files:**
- Modify: `go.mod`

**Step 1: Add Langfuse callback package**

Run: `rtk go get github.com/cloudwego/eino-ext/callbacks/langfuse@latest`

Expected output:
```
go: downloading github.com/cloudwego/eino-ext/callbacks/langfuse v0.x.x
go get: added github.com/cloudwego/eino-ext/callbacks/langfuse v0.x.x
```

**Step 2: Run go mod tidy**

Run: `rtk go mod tidy`

Expected: No errors, go.mod and go.sum updated

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add langfuse callback package from eino-ext"
```

---

### Task 2: Add LangfuseConfig to Config struct

**Files:**
- Modify: `internal/infrastructure/config/config.go:14-21`

**Step 1: Add LangfuseConfig type definition**

Insert after `Kreuzberg KreuzbergConfig` on line 20, before closing brace:

```go
type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Gemini    GeminiConfig
    Log       LogConfig
    Cache     CacheConfig
    Kreuzberg KreuzbergConfig
    Langfuse  LangfuseConfig
}
```

**Step 2: Add LangfuseConfig struct definition**

Insert after `OCRConfig` struct (after line 81):

```go
// LangfuseConfig holds Langfuse observability configuration
type LangfuseConfig struct {
    Host       string  `mapstructure:"host" validate:"required,url"`
    PublicKey  string  `mapstructure:"public_key" validate:"required"`
    SecretKey  string  `mapstructure:"secret_key" validate:"required"`
    Enabled    bool    `mapstructure:"enabled"`
    SampleRate float64 `mapstructure:"sample_rate"`
    Release    string  `mapstructure:"release"`
}
```

**Step 3: Verify code compiles**

Run: `rtk go build ./...`

Expected: No errors

**Step 4: Commit**

```bash
git add internal/infrastructure/config/config.go
git commit -m "feat: add LangfuseConfig struct for observability settings"
```

---

### Task 3: Add Langfuse environment variable bindings

**Files:**
- Modify: `internal/infrastructure/config/config.go:84-124`

**Step 1: Add Langfuse bindings to Load() function**

Insert after `viper.BindEnv("kreuzberg.ocr.model", "KREUZBERG_OCR_MODEL")` (line 111):

```go
    // Langfuse bindings
    viper.BindEnv("langfuse.host", "LANGFUSE_HOST")
    viper.BindEnv("langfuse.public_key", "LANGFUSE_PUBLIC_KEY")
    viper.BindEnv("langfuse.secret_key", "LANGFUSE_SECRET_KEY")
    viper.BindEnv("langfuse.enabled", "LANGFUSE_ENABLED")
    viper.BindEnv("langfuse.sample_rate", "LANGFUSE_SAMPLE_RATE")
    viper.BindEnv("langfuse.release", "LANGFUSE_RELEASE")
```

**Step 2: Add Langfuse defaults to setDefaults()**

Insert after `viper.SetDefault("kreuzberg.timeout", 30*time.Second)` (line 159):

```go
    // Langfuse defaults
    viper.SetDefault("langfuse.enabled", false)
    viper.SetDefault("langfuse.host", "https://cloud.langfuse.com")
    viper.SetDefault("langfuse.sample_rate", 1.0)
```

**Step 3: Verify code compiles**

Run: `rtk go build ./...`

Expected: No errors

**Step 4: Commit**

```bash
git add internal/infrastructure/config/config.go
git commit -m "feat: add Langfuse environment variable bindings and defaults"
```

---

### Task 4: Add Langfuse imports to serve.go

**Files:**
- Modify: `cmd/serve.go:1-37`

**Step 1: Add required imports**

Insert after existing imports (after line 36):

```go
    "github.com/cloudwego/eino/callbacks"
    langfusecallback "github.com/cloudwego/eino-ext/callbacks/langfuse"
```

**Step 2: Add langfuseFlusher variable**

Insert after `var servePort int` (line 39-40):

```go
var (
    servePort      int
    serveHost      string
    langfuseFlusher func()
)
```

**Step 3: Verify code compiles**

Run: `rtk go build ./cmd/serve`

Expected: No errors

**Step 4: Commit**

```bash
git add cmd/serve.go
git commit -m "feat: add Langfuse imports and flusher variable to serve command"
```

---

### Task 5: Initialize Langfuse handler in serve command

**Files:**
- Modify: `cmd/serve.go:53-83`

**Step 1: Add Langfuse initialization after logger setup**

Insert after `log.Info("Starting Eino Notebook server"...` block (after line 82):

```go
        // Initialize Langfuse callback handler for observability
        // This enables automatic tracing of all Eino components:
        // - ChatModel (Gemini)
        // - Embedder (Gemini embeddings)
        // - Retriever (pgvector)
        // - Chain (RAG pipeline)
        if cfg.Langfuse.Enabled {
            langfuseHandler, flusher := langfusecallback.NewLangfuseHandler(&langfusecallback.Config{
                Host:       cfg.Langfuse.Host,
                PublicKey:  cfg.Langfuse.PublicKey,
                SecretKey:  cfg.Langfuse.SecretKey,
                SampleRate: cfg.Langfuse.SampleRate,
                Release:    cfg.Langfuse.Release,
                Threads:    2,
                Timeout:    30 * time.Second,
                FlushAt:    15,
                FlushInterval: 500 * time.Millisecond,
                MaxTaskQueueSize: 100,
                MaxRetry:   3,
            })

            // Register globally - traces all Eino components automatically
            callbacks.AppendGlobalHandlers(langfuseHandler)
            langfuseFlusher = flusher

            log.Info("initialized", "langfuse", "enabled",
                "host", cfg.Langfuse.Host,
                "sample_rate", cfg.Langfuse.SampleRate)
        }
```

**Step 2: Verify code compiles**

Run: `rtk go build ./cmd/serve`

Expected: No errors

**Step 3: Commit**

```bash
git add cmd/serve.go
git commit -m "feat: initialize Langfuse callback handler with global registration"
```

---

### Task 6: Add Langfuse flush to shutdown sequence

**Files:**
- Modify: `cmd/serve.go:318-330`

**Step 1: Add Langfuse flush before server shutdown**

Insert after `ctx, cancel := context.WithTimeout(...)` (after line 320):

```go
        // Flush Langfuse events before shutdown
        if langfuseFlusher != nil {
            log.Info("flushing Langfuse events...")
            langfuseFlusher()
            log.Info("Langfuse events flushed")
        }
```

**Step 2: Verify code compiles**

Run: `rtk go build ./cmd/serve`

Expected: No errors

**Step 3: Commit**

```bash
git add cmd/serve.go
git commit -m "feat: add Langfuse flush to shutdown sequence"
```

---

### Task 7: Update .env.example with Langfuse variables

**Files:**
- Modify: `.env.example`

**Step 1: Add Langfuse section to .env.example**

Append to end of file:

```bash
# Langfuse Observability
LANGFUSE_ENABLED=false
LANGFUSE_HOST=https://cloud.langfuse.com
LANGFUSE_PUBLIC_KEY=pk-lf-xxxxxxxxxxxxx
LANGFUSE_SECRET_KEY=sk-lf-xxxxxxxxxxxxx
LANGFUSE_SAMPLE_RATE=1.0
LANGFUSE_RELEASE=v1.0.0
```

**Step 2: Commit**

```bash
git add .env.example
git commit -m "docs: add Langfuse environment variables to .env.example"
```

---

### Task 8: Create documentation for Langfuse integration

**Files:**
- Create: `docs/LANGFUSE.md`

**Step 1: Create Langfuse documentation**

Create file with content:

```markdown
# Langfuse Observability Integration

This application uses [Langfuse](https://langfuse.com) for tracing and observability of LLM operations.

## What Gets Traced

When Langfuse is enabled, the following Eino components are automatically traced:

- **ChatModel**: All Gemini chat model calls (generate and stream)
- **Embedder**: Text embedding operations
- **Retriever**: Vector search queries to pgvector
- **Chain**: End-to-end RAG pipeline execution

## Configuration

Enable Langfuse by setting environment variables:

```bash
LANGFUSE_ENABLED=true
LANGFUSE_HOST=https://cloud.langfuse.com
LANGFUSE_PUBLIC_KEY=pk-lf-...
LANGFUSE_SECRET_KEY=sk-lf-...
LANGFUSE_SAMPLE_RATE=1.0  # Optional: trace sampling rate (0.0-1.0)
LANGFUSE_RELEASE=v1.0.0   # Optional: version tag
```

## Viewing Traces

After enabling Langfuse, traces appear in your Langfuse dashboard automatically. Each trace includes:

- Input/output for each component
- Latency metrics
- Token usage
- Error information (if any)

## Architecture

Langfuse integration uses Eino's callback system:

1. Single global registration in `cmd/serve.go`
2. Async event batching for minimal overhead
3. Automatic flush on shutdown
4. No code changes required in use cases
```

**Step 2: Commit**

```bash
git add docs/LANGFUSE.md
git commit -m "docs: add Langfuse integration documentation"
```

---

### Task 9: Manual testing verification

**Files:**
- None (verification only)

**Step 1: Set test environment variables**

```bash
export LANGFUSE_ENABLED=true
export LANGFUSE_HOST=https://cloud.langfuse.com
export LANGFUSE_PUBLIC_KEY=pk-test
export LANGFUSE_SECRET_KEY=sk-test
```

**Step 2: Build and verify server starts**

Run: `rtk go run main.go serve`

Expected: Server starts without errors, log shows:
```
INFO initialized langfuse=enabled host=https://cloud.langfuse.com sample_rate=1
```

**Step 3: Verify graceful shutdown**

Send SIGINT (Ctrl+C) and verify logs show:
```
INFO flushing Langfuse events...
INFO Langfuse events flushed
```

**Step 4: Test with Langfuse disabled**

Run: `rtk LANGFUSE_ENABLED=false go run main.go serve`

Expected: Server starts, no Langfuse initialization logs

---

## Implementation Notes

- **No existing tests modified**: This is a new feature that doesn't break existing functionality
- **TDD applied**: Each change is verified with compilation checks before commit
- **DRY principle**: Single registration point in serve.go, no code duplication
- **YAGNI principle**: Only essential config options added, advanced features can be added later if needed

## Related Documentation

- Eino Callback Overview: `.claude/skills/eino-component/reference/callback/overview.md`
- Eino Langfuse Reference: `.claude/skills/eino-component/reference/callback/langfuse.md`
- Eino Component Skill: Use `/eino-component` for callback and component configuration