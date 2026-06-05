## ADDED Requirements

### Requirement: Terminal SSE event
The SSE stream SHALL emit `data: [DONE]\n\n` as the final event after `response.completed`.

#### Scenario: Stream completes successfully
- **WHEN** the formatter finishes emitting `response.completed`
- **THEN** the stream SHALL emit `data: [DONE]\n\n`
- **AND** the flusher SHALL flush immediately after

### Requirement: Token usage in streaming response
The `response.completed` event SHALL include a `usage` object with `input_tokens`, `output_tokens`, and `total_tokens` populated from the model's token usage metadata.

#### Scenario: Model returns usage metadata
- **WHEN** the model's final message contains `ResponseMeta.Usage` with `PromptTokens` and `CompletionTokens`
- **THEN** the `response.completed` event SHALL include `usage.input_tokens` equal to `PromptTokens`
- **AND** `usage.output_tokens` equal to `CompletionTokens`
- **AND** `usage.total_tokens` equal to `TotalTokens`
- **AND** `usage.output_tokens_details.reasoning_tokens` equal to `CompletionTokensDetails.ReasoningTokens` when non-zero

#### Scenario: Model does not return usage metadata
- **WHEN** the model's final message has nil `ResponseMeta.Usage`
- **THEN** the `response.completed` event SHALL omit the `usage` field

### Requirement: Tool call visibility in SSE stream
When the retrieval agent invokes a tool during generation, the SSE stream SHALL emit `function_call` items so the client can observe agent activity.

#### Scenario: Agent calls a tool
- **WHEN** the agent stage produces a message with `ToolCalls` containing one or more tool calls
- **THEN** the formatter SHALL emit a `response.output_item.added` event with item type `function_call` and status `in_progress`
- **AND** emit a `response.output_item.done` event with the same item having status `completed`
- **AND** each function_call item SHALL include `id`, `call_id`, `name`, `arguments`, and `status` fields

#### Scenario: Agent calls multiple tools in sequence
- **WHEN** the agent invokes tools sequentially during a single response
- **THEN** each tool call SHALL be emitted as a separate `function_call` item with its own `output_index`

#### Scenario: Agent makes no tool calls
- **WHEN** the agent produces a response without any tool calls
- **THEN** no `function_call` items SHALL be emitted
- **AND** the stream SHALL contain only the standard message lifecycle events

### Requirement: Complete response.in_progress event
The `response.in_progress` event SHALL include all required fields from the `response.created` event, not just `id` and `status`.

#### Scenario: Stream enters in_progress state
- **WHEN** the formatter emits the `response.in_progress` event
- **THEN** the event SHALL include `object: "response"`
- **AND** the event SHALL include `created_at` timestamp from the response metadata
- **AND** the event SHALL include `model` name
- **AND** the event SHALL include `truncation: "disabled"` or appropriate value
- **AND** the event SHALL include `parallel_tool_calls: true` or appropriate value
- **AND** the event SHALL include `text.format` object
- **AND** the event SHALL include `output: []` (empty array, not null)

**Rationale**: Clients may inspect the `response.in_progress` event to understand response configuration. Missing fields force clients to wait for `response.completed` or make incorrect assumptions.
