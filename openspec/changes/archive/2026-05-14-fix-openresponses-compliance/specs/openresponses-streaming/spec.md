## MODIFIED Requirements

### Requirement: Tool call visibility in SSE stream
When the retrieval agent invokes a tool during generation, the SSE stream SHALL emit `function_call` items so the client can observe agent activity.

#### Scenario: Agent calls a tool
- **WHEN** the agent stage produces a message with `ToolCalls` containing one or more tool calls
- **THEN** the formatter SHALL emit a `response.output_item.added` event with item type `function_call` and status `in_progress`
- **AND** emit `response.function_call_arguments.delta` events for each arguments chunk with `call_id` field
- **AND** emit `response.function_call_arguments.done` with complete arguments and `call_id` field
- **AND** emit a `response.output_item.done` event with the same item having status `completed`
- **AND** each function_call item SHALL include `id`, `call_id`, `name`, `arguments`, and `status` fields

#### Scenario: Agent calls multiple tools in sequence
- **WHEN** the agent invokes tools sequentially during a single response
- **THEN** each tool call SHALL be emitted as a separate `function_call` item with its own `output_index`

#### Scenario: Agent makes no tool calls
- **WHEN** the agent produces a response without any tool calls
- **THEN** no `function_call` items SHALL be emitted
- **AND** the stream SHALL contain only the standard message lifecycle events
