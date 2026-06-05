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

**Chunking Strategy**: We rely on ADK's native streaming chunks. When `runner.Query()` is called with `EnableStreaming: true`, each `iter.Next()` event contains a small chunk (a few tokens) directly from the LLM API. We emit `ReasoningEvent` immediately per chunk, without manual buffering or re-chunking.

**Exception**: We do minimal buffering for `` tag boundary parsing, since thinking tags can split across ADK chunks. Once content is extracted from within tags, we emit immediately.

**Known Issue**: The current thinking tag parser only handles literal `` tags but not HTML-encoded variants (`&lt;think&gt;`). This causes thinking content to leak into the message text when models use HTML encoding.

**Known Issue**: The `response.in_progress` event (formatter.go line 87-94) only includes `id` and `status` fields, missing `object`, `model`, `truncation`, `parallel_tool_calls`, and `text` fields. This violates the OpenResponses spec which requires these fields to be present.

## Risks / Trade-offs

- **[Risk]**: Changing the pipe type from `*schema.Message` to `StreamEvent` touches `agent_stage.go`, `response_usecase.go`, and `formatter.go` simultaneously — all three must be updated in one commit.
  → **Mitigation**: Implement all changes in a single PR with a clear commit boundary.

- **[Risk]**: Some models may report usage mid-stream rather than on the final message. The Eino framework aggregates usage across chunks (see `schema/message.go:1891`), so the final accumulated message should have accurate totals.
  → **Mitigation**: Use the final message's `ResponseMeta.Usage`.

- **[Trade-off]**: Tool call visibility is read-only for the client. We're not implementing the client-side agentic loop where the client feeds back `function_call_output`. This keeps the server authoritative over tool execution, which matches the current RAG agent design.

- **[Risk]**: Post-compliance testing revealed two issues: (1) `response.in_progress` event has incomplete fields, (2) thinking tag parser doesn't handle HTML-encoded tags. These were not caught by initial tests.
  → **Mitigation**: Add section 6 to tasks.md for these bug fixes. Add new requirements to specs for `response.in_progress` completeness and HTML-encoded tag handling.

**Additional Finding (Post-Archive)**: The thinking tag parser handles literal tags (``) and HTML entities (`&lt;think&gt;`) but NOT Unicode escape sequences (`

`). 

Some LLMs output Unicode escapes instead of HTML entities. This causes thinking tags to leak into message text even when HTML entity parsing is implemented. The fix requires adding Unicode escape sequences to the tag detection array or decoding Unicode escapes before parsing.

Example:
- HTML entity: `&lt;think&gt;` → Currently handled ✅
- Unicode escape: `

` → NOT handled ❌

The distinction is important because JSON encoding uses Unicode escapes by default, so many models output this format.

**Critical Finding (Post-Verification)**: The Unicode escape fix (section 7) is implemented but does NOT solve the actual tag leakage problem. Testing with stepfun-ai/step-3.5-flash revealed the root cause: **models that output closing `</think&gt;` tags WITHOUT opening `<think&gt;` tags**.

The model uses this pattern:
- Provides reasoning via `msg.ReasoningContent` (separate API field)
- Outputs `reasoning_text</think&gt;response_text` in `msg.Content` (no opening tag)
- The `</think&gt;` serves as a boundary marker, not a wrapping tag

The parser (`agent_stage.go:140-166`) only checks for closing tags when `inReasoning == true`, which is only set when an opening tag is found. Since no opening tag exists, the parser never enters reasoning mode, and closing tags pass through as regular text.

**Why the Unicode escape fix appeared to work in tests**: The test `TestAgentStage_Execute_UnicodeEncodedThinking` uses literal strings with backslashes (`\\u003c`), which matches the search pattern. But in real model output, Unicode escapes are decoded by the JSON parser into angle brackets BEFORE reaching the code. The literal tag search (`"think&gt;"`) should match, but the closing tag is only searched when in reasoning mode.

**Evidence**: response.log shows:
- 7 instances of `</think>` in message text deltas
- Reasoning item created from `msg.ReasoningContent` (correct)
- Closing tags leak into `response.output_text.delta` for the message item
- Output ordering wrong: message at output[0], reasoning at output[1]
