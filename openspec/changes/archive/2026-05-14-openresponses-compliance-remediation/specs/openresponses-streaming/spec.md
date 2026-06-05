## MODIFIED Requirements

### Requirement: Tool call visibility in SSE stream
When the retrieval agent invokes a tool during generation, the SSE stream SHALL emit `function_call_arguments` events so the client can observe agent activity.

#### Scenario: Agent calls a tool
- **WHEN** the agent stage produces a message with `ToolCalls` containing one or more tool calls
- **THEN** the formatter SHALL emit a `response.output_item.added` event with item type `function_call` and status `in_progress`
- **AND** emit `response.function_call_arguments.delta` events for argument chunks
- **AND** emit a `response.function_call_arguments.done` event when arguments are complete
- **AND** emit a `response.output_item.done` event with the same item having status `completed`

#### Scenario: Agent calls multiple tools in sequence
- **WHEN** the agent invokes tools sequentially during a single response
- **THEN** each tool call SHALL be emitted as a separate `function_call` item with its own `output_index`

#### Scenario: Agent makes no tool calls
- **WHEN** the agent produces a response without any tool calls
- **THEN** no `function_call` items SHALL be emitted
- **AND** the stream SHALL contain only the standard message lifecycle events