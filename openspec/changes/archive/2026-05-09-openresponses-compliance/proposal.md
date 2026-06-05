## Why

The `/v1/responses` endpoint is ~75% compliant with the OpenResponses spec. It handles the core streaming lifecycle, server-side tool calling, and history chaining — but is missing the terminal SSE event, token usage reporting, tool call visibility in the SSE stream, and reasoning item support needed for reasoning models (o1/o3, Claude extended thinking).

## What Changes

- Add `data: [DONE]\n\n` terminal SSE event after `response.completed`
- Wire up token usage (`input_tokens`, `output_tokens`, `total_tokens`) in streaming responses by capturing model usage metadata and including it in `StreamMeta` and the `response.completed` event
- Add `function_call` item type and streaming events so clients can observe when the retrieval agent uses tools (semantic_search, keyword_search, etc.) during generation
- Add `reasoning` item type, `summary_text` content type, and reasoning streaming events for models that expose chain-of-thought output

## Capabilities

### New Capabilities
- `openresponses-streaming`: Terminal event, token usage in SSE, and tool call visibility during streaming responses
- `openresponses-reasoning`: Reasoning item type and streaming events for chain-of-thought model output

### Modified Capabilities
_(none)_

## Impact

- `internal/interfaces/http/sse/formatter.go` — add `[DONE]` event, tool call events, reasoning events, usage population
- `internal/interfaces/http/sse/types.go` — extend `StreamMeta` with usage fields
- `internal/core/application/dtos/chat.go` — add `Reasoning`, `FunctionCall`, `SummaryTextContent` types and streaming events
- `internal/core/application/usecases/response/stages/agent_stage.go` — propagate tool call events and reasoning output from ADK runner
- `internal/core/application/usecases/response/response_usecase.go` — capture and pass usage metadata
