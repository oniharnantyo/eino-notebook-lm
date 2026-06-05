## ADDED Requirements

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
- **AND** emit `response.output_text.delta` events for reasoning text chunks **as received from the model's streaming API**
- **AND** emit `response.output_text.done` with the complete reasoning text
- **AND** emit `response.content_part.done`
- **AND** emit `response.output_item.done` with the `reasoning` item in `completed` status
- **AND** all reasoning events SHALL use a distinct `output_index` from the subsequent message events

**Note**: The system relies on the ADK runner's native streaming chunks. Each `iter.Next()` event contains a small chunk (a few tokens) from the LLM API. We emit delta events immediately per chunk, without artificial buffering or re-chunking.

### Requirement: Reasoning item in response output
The `response.completed` event SHALL include all reasoning items in its `output` array before the `message` item.

#### Scenario: Response with reasoning and message
- **WHEN** a response includes both reasoning content and a final message
- **THEN** the `response.completed` output array SHALL contain the `reasoning` item at `output[0]`
- **AND** the `message` item at `output[1]`

### Requirement: HTML-encoded thinking tag handling
The system SHALL handle both literal thinking tags (``) and HTML-encoded tags (`&lt;think&gt;` and `&lt;/think&gt;`) when parsing reasoning content from model output.

#### Scenario: Model uses HTML-encoded thinking tags
- **WHEN** the model outputs thinking content wrapped in `&lt;think&gt;` and `&lt;/think&gt;` tags
- **THEN** the parser SHALL recognize and extract the thinking content
- **AND** the thinking tags SHALL NOT appear in the final message text
- **AND** the extracted content SHALL be emitted as `ReasoningEvent`

#### Scenario: Model uses literal thinking tags
- **WHEN** the model outputs thinking content wrapped in `<think>` and `</think>` tags
- **THEN** the parser SHALL recognize and extract the thinking content
- **AND** the thinking tags SHALL NOT appear in the final message text
- **AND** the extracted content SHALL be emitted as `ReasoningEvent`

**Rationale**: Some models HTML-encode special characters in their output. The parser must handle both literal and encoded forms to prevent tag leakage into the message text, which would confuse users and break the semantic separation between reasoning and response content.

#### Scenario: Model uses Unicode escape encoded tags
- **WHEN** the model outputs thinking content wrapped in `<think>` and `</think>` tags (Unicode escape sequences)
- **THEN** the parser SHALL recognize and extract the thinking content
- **AND** the thinking tags SHALL NOT appear in the final message text
- **AND** the extracted content SHALL be emitted as `ReasoningEvent`

**Note**: The distinction between HTML entities (`&lt;think&gt;`) and Unicode escapes (`<think>`) is important. Some LLMs output Unicode escapes instead of HTML entities. The parser must handle both formats to prevent tag leakage.

#### Scenario: Model uses standalone closing tags without opening tags
- **WHEN** the model provides reasoning via `msg.ReasoningContent` field AND outputs `reasoning</think>response` in `msg.Content` without `<think>` opening tags
- **THEN** the parser SHALL recognize the standalone `</think>` closing tag when NOT in reasoning mode
- **AND** the parser SHALL strip the closing tag AND all preceding content from the message text (since reasoning was already captured via `ReasoningContent`)
- **AND** the parser SHALL only emit text that appears AFTER the closing tag as `TextDeltaEvent`
- **AND** the thinking tags SHALL NOT appear in the final message text

**Rationale**: Some models (e.g., stepfun-ai/step-3.5-flash) provide reasoning through a dedicated API field (`ReasoningContent`) while also including the reasoning content in the main `Content` field with `</think>` as a boundary marker. The parser must handle this pattern to prevent tag leakage, even though no opening `<think>` tag is present in the content stream.

**Note**: This is distinct from the wrapped tag pattern (``). The parser must support BOTH patterns:
1. **Wrapped**: `<think>reasoning</think>text` — parser enters reasoning mode on opening tag, exits on closing tag
2. **Standalone closing**: `reasoning</think>text` — parser detects closing tag even without opening tag, strips everything before it
