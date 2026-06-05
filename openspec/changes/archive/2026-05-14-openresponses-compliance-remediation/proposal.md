## Why

The current `/v1/responses` endpoint has several compliance gaps with the OpenResponses spec. It leaks model reasoning (e.g., `<think>` tags) into the final message output during multi-turn interactions, lacks the required terminal `[DONE]` SSE event, fails to include token usage metadata in the final completed event, and uses incorrect streaming event types for reasoning and tool calls. Resolving these issues is necessary for full spec compliance and proper client-side parsing.

## What Changes

- Add the `data: [DONE]\n\n` terminal event to the SSE stream.
- Populate token `usage` metadata in the `response.completed` event.
- Implement a thinking tag parser to extract `<think>` blocks from model outputs and prevent them from leaking into the `output_text` content.
- Add missing DTOs for `response.reasoning.delta`, `response.reasoning.done`, `response.function_call_arguments.delta`, and `response.function_call_arguments.done`.
- Update the SSE formatter and agent stage event mapper to emit the correct reasoning and function call events instead of generic text/content parts.

## Capabilities

### New Capabilities

*(none)*

### Modified Capabilities

- `openresponses-streaming`: Requires the terminal `[DONE]` event, token usage reporting, and specific `function_call_arguments` events.
- `openresponses-reasoning`: Requires proper thinking tag parsing to prevent tag leakage and the use of specific `reasoning.delta`/`done` events.

## Impact

- `internal/interfaces/http/sse/formatter.go`
- `internal/core/application/dtos/chat.go`
- `internal/core/application/usecases/response/stages/agent_stage.go`
- `internal/core/application/usecases/response/stages/event_mapper.go`