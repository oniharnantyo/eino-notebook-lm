## Context

The `/v1/responses` endpoint streams responses through the Eino ADK runner. The runner already produces rich events including tool calls, reasoning content, and token usage — but the agent stage and SSE formatter strip everything down to plain text `Content`.

The Eino `schema.Message` struct already carries:
- `ToolCalls []ToolCall` — tool invocations during agent execution
- `ReasoningContent string` — chain-of-thought output from reasoning models
- `ResponseMeta.Usage *TokenUsage` — prompt/completion/reasoning token counts

The gap is in the **formatter layer**, not the model or agent layer.

## Goals / Non-Goals

**Goals:**
- Emit `data: [DONE]\n\n` after `response.completed`
- Surface tool call activity as `function_call` items in the SSE stream
- Surface reasoning content as `reasoning` items in the SSE stream
- Include token usage in the `response.completed` event

**Non-Goals:**
- Non-streaming (JSON) response mode
- Client-side tool execution (agentic loop where client feeds back `function_call_output`)
- Authorization middleware
- WebSocket transport or `/v1/responses/compact` endpoint

## Decisions

### 1. Formatter-driven enrichment (not pipeline restructuring)

The agent stage currently pipes all messages through a single `schema.Pipe[*schema.Message]` channel and only forwards `msg.Content`. Instead of restructuring the pipeline, we enrich what flows through the pipe.

**Approach**: Define a `StreamEvent` discriminated union that wraps the different event types (text delta, tool call, reasoning, usage). The agent stage emits `StreamEvent` values instead of raw messages. The formatter reacts to each event type.

**Why not raw messages**: The formatter needs to know *what* it's rendering (a tool call vs a text chunk vs reasoning) to emit the correct SSE event types. Raw `schema.Message` doesn't carry that discriminator — a single message can contain text + tool calls + reasoning simultaneously.

**Alternative considered**: Keep the pipe as `*schema.Message` and have the formatter inspect each message's fields. Rejected because the formatter would need complex branching logic and couldn't emit lifecycle events at the right times (e.g., `function_call.added` before the agent has finished generating arguments).

### 2. StreamEvent as an interface type

```go
type StreamEvent interface {
    EventType() string
}

type TextDeltaEvent struct { ... }    // Normal text chunk
type ToolCallEvent struct { ... }     // Agent invoked a tool
type ReasoningEvent struct { ... }    // Model reasoning output
type UsageEvent struct { ... }        // Token usage metadata
```

The agent stage emits these through a `schema.Pipe[StreamEvent]`. The formatter type-switches on each event.

### 3. Token usage from final message

Usage comes from `schema.Message.ResponseMeta.Usage` on the **last message** in the stream. The agent stage extracts it and emits a `UsageEvent` before closing the pipe.

### 4. Reasoning: passthrough, not generation

We don't generate reasoning content — we pass through what the model provides via `schema.Message.ReasoningContent`. If the model doesn't produce reasoning (e.g., standard Gemini), no reasoning events are emitted. This is model-dependent and requires no special configuration.

## Risks / Trade-offs

- **[Risk]**: Changing the pipe type from `*schema.Message` to `StreamEvent` touches `agent_stage.go`, `response_usecase.go`, and `formatter.go` simultaneously — all three must be updated in one commit.
  → **Mitigation**: Implement all changes in a single PR with a clear commit boundary.

- **[Risk]**: Some models may report usage mid-stream rather than on the final message. The Eino framework aggregates usage across chunks (see `schema/message.go:1891`), so the final accumulated message should have accurate totals.
  → **Mitigation**: Use the final message's `ResponseMeta.Usage`.

- **[Trade-off]**: Tool call visibility is read-only for the client. We're not implementing the client-side agentic loop where the client feeds back `function_call_output`. This keeps the server authoritative over tool execution, which matches the current RAG agent design.
