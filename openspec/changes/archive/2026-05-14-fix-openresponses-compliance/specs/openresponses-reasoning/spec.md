## MODIFIED Requirements

### Requirement: Reasoning streaming events
When reasoning content is present, the SSE stream SHALL emit reasoning-specific lifecycle events using the 3-layer architecture defined by the OpenAPI spec.

#### Scenario: Reasoning summary streamed
- **WHEN** the model streams reasoning content
- **THEN** the formatter SHALL emit `response.output_item.added` with a `reasoning` item (type "reasoning", status "in_progress", empty `summary` array)
- **AND** emit `response.reasoning_summary_part.added` with `summary_index: 0` and an empty `SummaryTextContent` part
- **AND** emit `response.reasoning_summary_text.delta` events with `summary_index` (not `content_index`) for each reasoning text chunk
- **AND** emit `response.reasoning_summary_text.done` with the complete summary text and `summary_index: 0`
- **AND** emit `response.reasoning_summary_part.done` with the finalized `SummaryTextContent` part
- **AND** emit `response.output_item.done` with the `reasoning` item in `completed` status containing `summary[]` with full text
- **AND** all reasoning events SHALL use a distinct `output_index` from the subsequent message events

#### Scenario: Model does not produce reasoning content
- **WHEN** the model returns a message with empty `ReasoningContent`
- **THEN** no `reasoning` item or reasoning events SHALL be emitted

### Requirement: Reasoning item in response output
The `response.completed` event SHALL include all reasoning items in its `output` array before the `message` item.

#### Scenario: Response with reasoning and message
- **WHEN** a response includes both reasoning content and a final message
- **THEN** the `response.completed` output array SHALL contain the `reasoning` item at `output[0]` with `summary[]` containing `SummaryTextContent` items
- **AND** the `message` item at `output[1]`
- **AND** the reasoning item SHALL include `encrypted_content` field (null when unencrypted)
