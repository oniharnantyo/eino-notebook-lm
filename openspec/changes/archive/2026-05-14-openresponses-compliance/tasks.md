## 1. StreamEvent Types

- [x] 1.1 Define `StreamEvent` interface and concrete event types (`TextDeltaEvent`, `ToolCallEvent`, `ReasoningEvent`, `UsageEvent`) in `internal/interfaces/http/sse/types.go`
- [x] 1.2 Add `FunctionCall` item type implementing `ItemField` in `internal/core/application/dtos/chat.go`
- [x] 1.3 Add `Reasoning` item type implementing `ItemField` and `SummaryTextContent` implementing `ContentPart` in `internal/core/application/dtos/chat.go`
- [x] 1.4 Add streaming event DTOs for function_call lifecycle (`response.function_call_arguments.delta`, etc.) in `internal/core/application/dtos/chat.go`
- [x] 1.5 Add streaming event DTOs for reasoning lifecycle in `internal/core/application/dtos/chat.go`

## 2. Agent Stage Refactor

- [x] 2.1 Change `AgentStage.Execute` to emit `StreamEvent` values through the pipe instead of raw `*schema.Message`, detecting tool calls from `msg.ToolCalls` and reasoning from `msg.ReasoningContent`
- [x] 2.2 Extract token usage from the final message's `ResponseMeta.Usage` and emit a `UsageEvent` before closing the pipe
- [x] 2.3 Update `GenerationOutput` to use `*schema.StreamReader[StreamEvent]` instead of `*schema.StreamReader[*schema.Message]`
- [x] 2.4 **Ensure no manual chunking**: Emit `StreamEvent` immediately per ADK chunk without artificial buffering. Only minimal buffering for `` tag boundary parsing.

## 3. Response UseCase Wiring

- [x] 3.1 Update `response_usecase.go` to pass `StreamEvent` reader to the formatter and carry usage metadata in `StreamMeta`
- [x] 3.2 Update `HistorySavingReader` to work with the new `StreamEvent` stream, accumulating text from `TextDeltaEvent` for history saving

## 4. Formatter Enrichment

- [x] 4.1 Add `data: [DONE]\n\n` terminal event after `response.completed` in `formatter.go`
- [x] 4.2 Refactor `WriteResponse` to type-switch on `StreamEvent` and dispatch to handler methods per event type
- [x] 4.3 Implement tool call SSE emission: emit `response.output_item.added` (function_call, in_progress) → `response.output_item.done` (function_call, completed) for each `ToolCallEvent`
- [x] 4.4 Implement reasoning SSE emission: emit full lifecycle (output_item.added → content_part.added → output_text.delta → output_text.done → content_part.done → output_item.done) for `ReasoningEvent`
- [x] 4.5 Populate `usage` field in `response.completed` from `UsageEvent` data

## 5. Tests

- [x] 5.1 Add unit test for `data: [DONE]` terminal event in `formatter_test.go`
- [x] 5.2 Add unit test for tool call SSE event emission
- [x] 5.3 Add unit test for reasoning SSE event emission
- [x] 5.4 Add unit test for usage metadata in `response.completed`
- [x] 5.5 Verify existing handler test still passes after type changes
- [x] 5.6 **Verify no artificial chunking**: Add test or manual verification that `ReasoningEvent` is emitted immediately per ADK chunk, not buffered to 50-character chunks

## 6. Bug Fixes (Post-Compliance Review)

- [x] 6.1 **Fix `response.in_progress` event fields**: Update `formatter.go` line 87-94 to include all fields from `response.created` (object, model, truncation, parallel_tool_calls, text)
- [x] 6.2 **Fix thinking tag parser for HTML entities**: Update `agent_stage.go` to handle HTML-encoded thinking tags (`</think>` and `</think>`) in addition to literal tags
- [x] 6.3 **Prevent thinking tag leakage**: Ensure thinking tags and their content are completely extracted and routed to `ReasoningEvent`, preventing `</think>` from appearing in message text
- [x] 6.4 Add test for HTML-encoded thinking tag parsing
- [x] 6.5 Add test for `response.in_progress` event completeness

## 7. Additional Fix (Post-Archive Review)

- [x] 7.1 **Fix Unicode escape thinking tag parsing**: Update `agent_stage.go` to handle Unicode escape sequences (`

` and `

`) in addition to literal tags and HTML entities
- [x] 7.2 Add test for Unicode escape thinking tag parsing

**Note**: Unicode escapes are different from HTML entities. The current parser handles `&lt;think&gt;` but not `

`. Models may output either format.

## 8. Standalone Closing Tag Issue (Post-Verification Review)

- [x] 8.1 **Fix standalone closing tag parsing**: Update `agent_stage.go` to handle `</think&gt;` closing tags when NOT in reasoning mode. Some models (e.g., step-3.5-flash) output `reasoning</think&gt;text` without `<think&gt;` opening tags, relying on `msg.ReasoningContent` for the reasoning content. The parser must strip standalone closing tags and preceding content from message text.
- [x] 8.2 **Fix response.in_progress null arrays**: Update `formatter.go` line 87-99 to initialize `Output: []dtos.ItemField{}` and `Tools: []dtos.Tool{}` instead of leaving them nil (which JSON-encodes as `null`).
- [x] 8.3 **Fix response.in_progress missing fields**: Add `ToolChoice` and `Text` fields to `response.in_progress` event to match `response.created`.
- [x] 8.4 Add test for standalone closing tag parsing
- [x] 8.5 Add test for response.in_progress completeness (null → empty arrays, missing fields)

**Note**: This is distinct from the Unicode escape fix (section 7). The parser already handles Unicode escapes (they decode to angle brackets which match literal tag search). The real issue is models that output closing tags WITHOUT opening tags — the parser never enters reasoning mode, so closing tags are never detected and leak into message text.

**Evidence**: response.log from step-3.5-flash shows 7 instances of `</think>` in `response.output_text.delta` events (lines 17, 33, 43, 53, 63, 73, 83). The model provides reasoning via `msg.ReasoningContent` but also includes `reasoning</think&gt;text` in `msg.Content` without opening tags.
