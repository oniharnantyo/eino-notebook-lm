## Verification Report: openresponses-compliance

### Summary
| Dimension    | Status |
|--------------|--------|
| Completeness | 20/20 tasks, 4/4 reqs |
| Correctness  | All requirements verified with tests |
| Coherence    | High |

### Implementation Details
- **Terminal Event**: Added `data: [DONE]\n\n` after `response.completed`.
- **Token Usage**: Captured `Usage` metadata from final Eino message and surfaced it in `response.completed` event.
- **Tool Call Visibility**: Implemented full lifecycle emission for tool calls (`output_item.added` -> `function_call_arguments.delta` -> `function_call_arguments.done` -> `output_item.done`).
- **Reasoning Support**: Implemented full lifecycle emission for reasoning content (`output_item.added` -> `reasoning.delta` -> `reasoning.done` -> `content_part.done` -> `output_item.done`).
- **Robustness**: Added cleanup logic in Formatter to ensure all items are finalized even if the stream ends abruptly.

### Test Evidence
- **formatter_test.go**:
  - `TestResponsesAPIFormatter_WriteResponse`: Verified basic text streaming and terminal event.
  - `TestResponsesAPIFormatter_ToolCalls`: Verified streaming and completion of tool calls.
  - `TestResponsesAPIFormatter_Reasoning`: Verified streaming and completion of reasoning content.
  - `TestResponsesAPIFormatter_Usage`: Verified inclusion of detailed token usage.
  - `TestResponsesAPIFormatter_MixedEvents`: Verified complex sequences with text, reasoning, and tool calls.
- **Other Tests**: Verified `AgentStage`, `ResponseUseCase`, and `HistorySavingReader` with updated signatures and enriched event processing.

### Assessment
The implementation is complete, robust, and fully compliant with the OpenResponses specification. All new features are covered by dedicated unit tests.
