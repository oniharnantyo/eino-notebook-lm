## Context

The `/v1/responses` endpoint streams SSE events through `ResponsesAPIFormatter` in `internal/interfaces/http/sse/formatter.go`. Validation against the OpenResponses OpenAPI spec (`openapi.json`) and Ollama's reference implementation (`responses.txt`) reveals three categories of non-compliance: missing event types, missing response fields, and incorrect reasoning lifecycle structure.

Current state: The formatter emits reasoning content using `content_part.added/done` + `reasoning.delta/done` events. The spec requires a distinct 3-layer architecture with `reasoning_summary_part.added/done` + `reasoning_summary_text.delta/done` for summaries, and separate `reasoning.delta/done` for raw content.

## Goals / Non-Goals

**Goals:**
- Full compliance with the OpenResponses OpenAPI spec for all currently supported features
- Correct 3-layer reasoning event lifecycle
- Complete `ResponseResource` fields in lifecycle events (`created`, `in_progress`, `completed`)
- Correct item schemas (`FunctionCallItem.call_id`, `ReasoningBody.encrypted_content`)

**Non-Goals:**
- Non-streaming response mode (not yet implemented)
- `response.queued` / `response.incomplete` events (no queueing infrastructure)
- `response.refusal.delta/done` events (no refusal detection)
- `response.output_text.annotation.added` events (no annotation infrastructure)
- `obfuscation` field on delta events (no encryption layer)
- `POST /v1/responses/compact` endpoint
- WebSocket transport

## Decisions

### Decision 1: Follow OpenAPI spec, not Ollama's output

The `openapi.json` is the canonical spec. Ollama is a reference but has its own deviations (e.g., including `encrypted_content` as plaintext). We validate against the OpenAPI schema.

**Alternative**: Follow Ollama's exact wire format. Rejected because Ollama is one implementation among many — the spec is the contract.

### Decision 2: Restructure reasoning into 3-layer lifecycle

Replace current flat reasoning lifecycle with the spec's layered approach:

```
Current (flat):
  output_item.added → content_part.added → reasoning.delta → reasoning.done → content_part.done → output_item.done

Spec (3-layer):
  output_item.added
    → reasoning_summary_part.added
      → reasoning_summary_text.delta (×N)
      → reasoning_summary_text.done
    → reasoning_summary_part.done
    → reasoning.delta (×N, raw content, optional)
    → reasoning.done (optional)
  → output_item.done
```

**Rationale**: The spec separates user-visible summary from raw thinking trace. Even if we only populate one layer today, the event structure must match.

**Alternative**: Keep flat structure and just rename events. Rejected because `content_part` events have different semantics from `reasoning_summary_part` events.

### Decision 3: Populate `ResponseResource` defaults from request + constants

Add all 27 required fields to `ResponseResource`. Fields not present in the request (e.g., `background`, `store`) default to spec-standard values. Pass relevant request fields (`instructions`, `previous_response_id`, `max_output_tokens`) through `StreamMeta`.

**Alternative**: Only add fields we actively use. Rejected because clients may validate against the OpenAPI schema and fail on missing required fields.

### Decision 4: Keep `ReasoningSummary` content in `ReasoningBody` instead of `encrypted_content`

The spec allows `content[]` with `SummaryTextContent` type for the summary, and `encrypted_content` for raw trace. Since we don't encrypt, we'll populate `summary[]` with full summary text and omit `encrypted_content`.

## Risks / Trade-offs

- **[Breaking change]** → Clients consuming `response.reasoning.delta` will break. Mitigate by noting this is a spec-alignment fix — the old event names were non-compliant.
- **[Increased payload size]** → 13 additional `ResponseResource` fields add ~200 bytes per lifecycle event. Acceptable for SSE streams.
- **[Spec may evolve]** → The OpenResponses spec is still evolving. Mitigate by keeping DTOs close to the OpenAPI schema for easy future updates.
