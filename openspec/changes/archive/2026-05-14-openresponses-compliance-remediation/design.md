## Context

The `/v1/responses` endpoint implements the OpenResponses specification but currently fails compliance checks in three main areas:
1. Multi-turn reasoning leaks `<think>` tags into the output message because the formatter lacks a parser for these tags.
2. The SSE stream ends without the mandatory `data: [DONE]\n\n` terminal event.
3. Token usage metadata is missing from the `response.completed` event.
4. Reasoning and tool calls use generic text/content event types instead of the specific `response.reasoning.delta`/`done` and `response.function_call_arguments.delta`/`done`.

## Goals / Non-Goals

**Goals:**
- Add `[DONE]` terminal event at the end of the SSE stream.
- Forward token usage metrics from the underlying model to the `response.completed` event.
- Add `<think>` tag parsing in the agent stage/formatter to capture reasoning chunks properly and prevent them from leaking into normal message content.
- Update DTOs and the SSE formatter to use the correct reasoning and function call event schemas.

**Non-Goals:**
- Rewriting the entire SSE formatter structure. We will amend the current state-tracking logic.
- Supporting models that output reasoning in formats other than `<think>` tags or the `ReasoningContent` property.

## Decisions

1. **Thinking Tag Parser**: We will implement the parser within the `AgentStage`'s stream processing loop or `event_mapper` to extract text within `<think>` and `</think>` bounds. If text is extracted here, it will be mapped into a schema that the `ResponsesAPIFormatter` interprets as reasoning content. Since chunks can split tags, the parser must hold a small buffer to detect tag boundaries properly.
2. **DTO Additions**: We will introduce `ResponseReasoningDeltaEvent`, `ResponseReasoningDoneEvent`, `ResponseFunctionCallArgumentsDeltaEvent`, and `ResponseFunctionCallArgumentsDoneEvent` to `dtos/chat.go` and implement `GetEventType()` on them.
3. **SSE Formatter Updates**: The `WriteResponse` function in `internal/interfaces/http/sse/formatter.go` will be updated to:
   - Handle reasoning items using the new reasoning DTOs instead of `ResponseOutputTextDeltaEvent`.
   - Handle function calls using the new function call DTOs instead of generic `ResponseContentPartAddedEvent`.
   - Write `data: [DONE]\n\n` immediately before closing the stream.
   - Include `meta.Usage` in the `response.completed` event payload.

## Risks / Trade-offs

- **[Risk]** Chunked streaming might split `<think>` tags (e.g., `<thi` and `nk>`), causing the parser to miss them.
  - **Mitigation**: The thinking tag parser must maintain a buffer to detect partial tags across chunk boundaries.
- **[Risk]** Models might use HTML encoded tags like `&lt;think&gt;` or Unicode escapes.
  - **Mitigation**: The parser should be robust enough to handle the most common variations or decode entities before matching.