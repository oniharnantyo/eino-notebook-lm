## ADDED Requirements

### Requirement: Thinking tag parsing
The system SHALL parse `<think>` and `</think>` tags from model output streams, extracting the enclosed content as reasoning.

#### Scenario: Model streams thinking tags
- **WHEN** the model streams content containing `<think>` tags within the `Content` field
- **THEN** the system SHALL extract the text between `<think>` and `</think>` and treat it as reasoning content
- **AND** the extracted text SHALL NOT be included in the final `output_text` message content

## MODIFIED Requirements

### Requirement: Reasoning streaming events
When reasoning content is present, the SSE stream SHALL emit reasoning-specific lifecycle events.

#### Scenario: Reasoning content streamed
- **WHEN** the model streams reasoning content (either via `ReasoningContent` field or parsed from `<think>` tags)
- **THEN** the formatter SHALL emit `response.output_item.added` with a `reasoning` item in `in_progress` status
- **AND** emit `response.reasoning.delta` events for reasoning text chunks
- **AND** emit `response.reasoning.done` with the complete reasoning text
- **AND** emit `response.output_item.done` with the `reasoning` item in `completed` status
- **AND** all reasoning events SHALL use a distinct `output_index` from the subsequent message events