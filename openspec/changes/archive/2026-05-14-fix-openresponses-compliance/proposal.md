## Why

The `/v1/responses` streaming implementation diverges from the OpenResponses OpenAPI spec (`openapi.json`) in multiple areas: missing reasoning summary streaming events, wrong event type names, missing `ResponseResource` fields, and incorrect reasoning lifecycle structure. Validating against the spec reveals 6 missing event types, 13 missing response fields, and a flattened reasoning architecture that conflates summary text with raw reasoning content.

## What Changes

- **BREAKING**: Rename `response.reasoning.delta/done` to `response.reasoning_summary_text.delta/done` (uses `summary_index` instead of `content_index`)
- **BREAKING**: Replace `response.content_part.added/done` for reasoning with `response.reasoning_summary_part.added/done`
- Add 6 missing streaming event types: `reasoning_summary_text.delta/done`, `reasoning_summary_part.added/done`, `reasoning_summary_text.delta/done`, `refusal.delta/done`
- Add 13 missing `ResponseResource` fields (`background`, `store`, `service_tier`, `reasoning`, etc.)
- Add `call_id` field to `FunctionCallItem`
- Add `encrypted_content` field to `ReasoningBody`
- Restructure formatter reasoning lifecycle into 3-layer architecture (item → summary part → summary text)
- Add `obfuscation` field to delta events

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `openresponses-streaming`: Add missing event types, add `obfuscation` field to delta events, fix `call_id` on function call arguments events
- `openresponses-reasoning`: Restructure reasoning lifecycle from flat `content_part` + `reasoning.delta` to 3-layer `reasoning_summary_part` + `reasoning_summary_text.delta`, add `encrypted_content` support
- `openresponses-resource`: New capability covering `ResponseResource` field completeness (13 missing fields)

## Impact

- `internal/core/application/dtos/chat.go` — DTO structs for events, items, and ResponseResource
- `internal/interfaces/http/sse/formatter.go` — Formatter logic for reasoning lifecycle
- `internal/interfaces/http/sse/types.go` — StreamMeta and supporting types
- `internal/interfaces/http/sse/formatter_test.go` — Existing tests need updating
- `internal/interfaces/http/handlers/response.go` — Handler that constructs StreamMeta
