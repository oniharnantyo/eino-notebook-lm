# Capability: OpenResponses Reasoning

## Purpose
TBD: This capability enables support for reasoning (chain-of-thought) content in OpenResponses, including both streaming and non-streaming response formats.

## Requirements

### Requirement: Reasoning item type
The system SHALL support a `reasoning` item type that carries the model's chain-of-thought output, emitted before the final `message` item.

#### Scenario: Model produces reasoning content
- **WHEN** the model returns a message with non-empty `ReasoningContent`
- **THEN** the formatter SHALL emit a `reasoning` item before the `message` item
- **AND** the reasoning item SHALL include `id`, `type: "reasoning"`, `status: "completed"`, and `content` containing the reasoning text as `output_text`

#### Scenario: Model does not produce reasoning content
- **WHEN** the model returns a message with empty `ReasoningContent`
- **THEN** no `reasoning` item SHALL be emitted

### Requirement: Reasoning streaming events
When reasoning content is present, the SSE stream SHALL emit reasoning-specific lifecycle events.

#### Scenario: Reasoning content streamed
- **WHEN** the model streams reasoning content
- **THEN** the formatter SHALL emit `response.output_item.added` with a `reasoning` item in `in_progress` status
- **AND** emit `response.content_part.added` for the reasoning content
- **AND** emit `response.output_text.delta` events for reasoning text chunks
- **AND** emit `response.output_text.done` with the complete reasoning text
- **AND** emit `response.content_part.done`
- **AND** emit `response.output_item.done` with the `reasoning` item in `completed` status
- **AND** all reasoning events SHALL use a distinct `output_index` from the subsequent message events

### Requirement: Reasoning item in response output
The `response.completed` event SHALL include all reasoning items in its `output` array before the `message` item.

#### Scenario: Response with reasoning and message
- **WHEN** a response includes both reasoning content and a final message
- **THEN** the `response.completed` output array SHALL contain the `reasoning` item at `output[0]`
- **AND** the `message` item at `output[1]`
