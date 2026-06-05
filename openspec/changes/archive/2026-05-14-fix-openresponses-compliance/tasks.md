## 1. DTO Struct Updates

- [ ] 1.1 Add 13 missing fields to `ResponseResource` in `internal/core/application/dtos/chat.go`: `background`, `frequency_penalty`, `presence_penalty`, `instructions`, `max_output_tokens`, `max_tool_calls`, `store`, `service_tier`, `top_logprobs`, `previous_response_id`, `prompt_cache_key`, `reasoning`, `safety_identifier`
- [ ] 1.2 Add `call_id` field to `FunctionCallItem` struct
- [ ] 1.3 Add `encrypted_content` field to `ReasoningItem` struct
- [ ] 1.4 Add `SummaryTextContent` struct with `type: "summary_text"` and `text` fields
- [ ] 1.5 Add `ReasoningResource` struct (response-level reasoning with `effort` and `summary` fields)
- [ ] 1.6 Add `ReasoningSummaryPartAddedEvent` struct with `summary_index` and `Part` fields
- [ ] 1.7 Add `ReasoningSummaryPartDoneEvent` struct with `summary_index` and `Part` fields
- [ ] 1.8 Add `ReasoningSummaryTextDeltaEvent` struct with `summary_index` and `delta` fields (type: `response.reasoning_summary_text.delta`)
- [ ] 1.9 Add `ReasoningSummaryTextDoneEvent` struct with `summary_index` and `text` fields (type: `response.reasoning_summary_text.done`)
- [ ] 1.10 Add `call_id` field to `ResponseFunctionCallArgumentsDeltaEvent` and `ResponseFunctionCallArgumentsDoneEvent`

## 2. StreamMeta Update

- [ ] 2.1 Add request-derived fields to `StreamMeta` in `internal/interfaces/http/sse/types.go`: `Instructions`, `PreviousResponseID`, `MaxOutputTokens`, `Temperature`, `MaxToolCalls`, `Metadata`
- [ ] 2.2 Update `response_usecase.go` to populate new `StreamMeta` fields from the request

## 3. Formatter Reasoning Lifecycle Restructure

- [ ] 3.1 Replace `response.content_part.added` for reasoning with `response.reasoning_summary_part.added` (using `summary_index`)
- [ ] 3.2 Replace `response.reasoning.delta` with `response.reasoning_summary_text.delta` (using `summary_index` instead of `content_index`)
- [ ] 3.3 Replace `response.reasoning.done` with `response.reasoning_summary_text.done` (using `summary_index`)
- [ ] 3.4 Replace `response.content_part.done` for reasoning with `response.reasoning_summary_part.done` (using `summary_index`)
- [ ] 3.5 Update `finalizeReasoningItem` to use new event types and include `encrypted_content` (null) and full `summary[]`
- [ ] 3.6 Remove summary text truncation in `truncateForSummary` — emit full summary text per spec

## 4. Formatter ResponseResource Updates

- [ ] 4.1 Update `response.created` event to include all 27 `ResponseResource` fields with defaults
- [ ] 4.2 Update `response.in_progress` event to include all 27 `ResponseResource` fields with defaults
- [ ] 4.3 Update `response.completed` event to include all fields plus populated `usage`, `completed_at`, and `output`

## 5. Function Call Updates

- [ ] 5.1 Add `call_id` to `function_call` output item in `response.output_item.added`
- [ ] 5.2 Add `call_id` to `response.function_call_arguments.delta` events
- [ ] 5.3 Add `call_id` to `response.function_call_arguments.done` events
- [ ] 5.4 Add `call_id` to `function_call` output item in `response.output_item.done`

## 6. Tests

- [ ] 6.1 Update `formatter_test.go` — fix existing reasoning event assertions to use new event types
- [ ] 6.2 Add test for `reasoning_summary_part.added/done` events with correct `summary_index`
- [ ] 6.3 Add test for `reasoning_summary_text.delta/done` events
- [ ] 6.4 Add test for all 27 `ResponseResource` fields in `response.created`
- [ ] 6.5 Add test for `call_id` on function call items and delta/done events
- [ ] 6.6 Run `make test` to verify all tests pass
