# Streaming Events — Complete Catalog

## Table of Contents

1. [SSE Protocol Rules](#sse-protocol-rules)
2. [Response Lifecycle Events](#response-lifecycle-events)
3. [Output Item Events](#output-item-events)
4. [Content Part Events](#content-part-events)
5. [Text Output Events](#text-output-events)
6. [Refusal Events](#refusal-events)
7. [Reasoning Events](#reasoning-events)
8. [Function Call Events](#function-call-events)
9. [Error Events](#error-events)
10. [Event Ordering](#event-ordering)

---

## SSE Protocol Rules

- Content-Type: `text/event-stream`
- Each event: `data: <json>\n\n`
- Terminal event: `data: [DONE]\n\n`
- SSE `event` field MUST match JSON `type` field
- Every event includes `sequence_number` (monotonically increasing)

## Response Lifecycle Events

### response.created

Fired when the response object is first created.

```json
{
  "type": "response.created",
  "sequence_number": 0,
  "response": {
    "id": "resp_abc",
    "type": "response",
    "status": "in_progress",
    "model": "gpt-5.2",
    "created_at": 1704067200
  }
}
```

### response.queued

Fired when the response is queued (not yet processing). Optional — only when server has a queue.

```json
{
  "type": "response.queued",
  "sequence_number": 1,
  "response": {
    "id": "resp_abc",
    "status": "queued"
  }
}
```

### response.in_progress

Fired when the server begins processing the response.

```json
{
  "type": "response.in_progress",
  "sequence_number": 2,
  "response": {
    "id": "resp_abc",
    "status": "in_progress"
  }
}
```

### response.completed

Terminal event. Fired when the entire response is done, including all output items.

```json
{
  "type": "response.completed",
  "sequence_number": 50,
  "response": {
    "id": "resp_abc",
    "type": "response",
    "status": "completed",
    "model": "gpt-5.2",
    "output": [...],
    "usage": {
      "input_tokens": 45,
      "output_tokens": 120,
      "total_tokens": 165
    },
    "created_at": 1704067200,
    "completed_at": 1704067202
  }
}
```

### response.failed

Terminal event. Fired when the response fails.

```json
{
  "type": "response.failed",
  "sequence_number": 10,
  "response": {
    "id": "resp_abc",
    "status": "failed",
    "error": {
      "message": "Model overloaded",
      "type": "server_error",
      "code": "model_error"
    }
  }
}
```

### response.incomplete

Terminal event. Fired when the response stops without completing (token budget exhausted, etc.).

```json
{
  "type": "response.incomplete",
  "sequence_number": 30,
  "response": {
    "id": "resp_abc",
    "status": "incomplete",
    "incomplete_details": {
      "reason": "max_output_tokens_reached"
    }
  }
}
```

---

## Output Item Events

### response.output_item.added

A new output item has been created. The item starts with `status: in_progress`.

```json
{
  "type": "response.output_item.added",
  "sequence_number": 3,
  "output_index": 0,
  "item": {
    "id": "msg_001",
    "type": "message",
    "status": "in_progress",
    "role": "assistant",
    "content": []
  }
}
```

Fields:
- `output_index` — position in the output array
- `item` — the full item object (initially with empty content)

### response.output_item.done

An output item has been finalized. The item has a terminal status.

```json
{
  "type": "response.output_item.done",
  "sequence_number": 25,
  "output_index": 0,
  "item": {
    "id": "msg_001",
    "type": "message",
    "status": "completed",
    "role": "assistant",
    "content": [
      { "type": "output_text", "text": "Hello!" }
    ]
  }
}
```

---

## Content Part Events

### response.content_part.added

A new content part has been added to an item.

```json
{
  "type": "response.content_part.added",
  "sequence_number": 4,
  "item_id": "msg_001",
  "output_index": 0,
  "content_index": 0,
  "part": {
    "type": "output_text",
    "text": ""
  }
}
```

Fields:
- `item_id` — the parent item's ID
- `output_index` — position in output array
- `content_index` — position within the item's content array
- `part` — the content part (initially empty)

### response.content_part.done

A content part has been finalized.

```json
{
  "type": "response.content_part.done",
  "sequence_number": 24,
  "item_id": "msg_001",
  "output_index": 0,
  "content_index": 0,
  "part": {
    "type": "output_text",
    "text": "Hello! How can I help?"
  }
}
```

---

## Text Output Events

### response.output_text.delta

Incremental text chunk. This is the main streaming event for text content. May fire many times per content part.

```json
{
  "type": "response.output_text.delta",
  "sequence_number": 5,
  "item_id": "msg_001",
  "output_index": 0,
  "content_index": 0,
  "delta": "Hello"
}
```

Fields:
- `delta` — the text chunk (append to buffer)

### response.output_text.done

The complete text for a content part. Fired once after all deltas.

```json
{
  "type": "response.output_text.done",
  "sequence_number": 23,
  "item_id": "msg_001",
  "output_index": 0,
  "content_index": 0,
  "text": "Hello! How can I help?"
}
```

Fields:
- `text` — the complete assembled text

---

## Refusal Events

### response.refusal.delta

Incremental refusal text chunk.

```json
{
  "type": "response.refusal.delta",
  "sequence_number": 5,
  "item_id": "msg_001",
  "output_index": 0,
  "content_index": 0,
  "delta": "I cannot"
}
```

### response.refusal.done

Complete refusal text.

```json
{
  "type": "response.refusal.done",
  "sequence_number": 10,
  "item_id": "msg_001",
  "output_index": 0,
  "content_index": 0,
  "refusal": "I cannot assist with that request."
}
```

---

## Reasoning Events

### response.reasoning.delta

Incremental reasoning content chunk.

```json
{
  "type": "response.reasoning.delta",
  "sequence_number": 5,
  "item_id": "rs_001",
  "output_index": 0,
  "content_index": 0,
  "delta": "The user is asking about..."
}
```

### response.reasoning.done

Complete reasoning content.

```json
{
  "type": "response.reasoning.done",
  "sequence_number": 15,
  "item_id": "rs_001",
  "output_index": 0,
  "content_index": 0,
  "content": [
    { "type": "output_text", "text": "Full reasoning trace..." }
  ]
}
```

### response.reasoning_summary.delta

Incremental reasoning summary chunk.

```json
{
  "type": "response.reasoning_summary.delta",
  "sequence_number": 16,
  "item_id": "rs_001",
  "output_index": 0,
  "content_index": 0,
  "delta": "User requested weather info"
}
```

### response.reasoning_summary.done

Complete reasoning summary.

```json
{
  "type": "response.reasoning_summary.done",
  "sequence_number": 17,
  "item_id": "rs_001",
  "output_index": 0,
  "content_index": 0,
  "summary": [
    { "type": "summary_text", "text": "User requested weather info for Tokyo" }
  ]
}
```

---

## Function Call Events

### response.function_call_arguments.delta

Incremental arguments chunk (JSON string being built incrementally).

```json
{
  "type": "response.function_call_arguments.delta",
  "sequence_number": 5,
  "item_id": "fc_001",
  "output_index": 0,
  "call_id": "call_abc",
  "delta": "{\"locat"
}
```

Fields:
- `call_id` — links to the function_call item
- `delta` — partial JSON string chunk

### response.function_call_arguments.done

Complete function call arguments.

```json
{
  "type": "response.function_call_arguments.done",
  "sequence_number": 10,
  "item_id": "fc_001",
  "output_index": 0,
  "call_id": "call_abc",
  "arguments": "{\"location\":\"Tokyo\",\"unit\":\"celsius\"}"
}
```

Fields:
- `arguments` — the complete JSON argument string

---

## Error Events

### error

Fired when an error occurs during streaming.

```json
{
  "type": "error",
  "sequence_number": 5,
  "error": {
    "message": "Model temporarily unavailable",
    "type": "server_error",
    "code": "model_error"
  }
}
```

---

## Event Ordering

### Complete lifecycle for a text-only response:

```
1.  response.created
2.  response.in_progress
3.  response.output_item.added         (message item, status: in_progress)
4.  response.content_part.added        (empty output_text part)
5.  response.output_text.delta         (repeated N times)
6.  response.output_text.done          (complete text)
7.  response.content_part.done         (finalized part)
8.  response.output_item.done          (message item, status: completed)
9.  response.completed                 (full response, status: completed)
10. [DONE]
```

### Lifecycle for a function call response:

```
1.  response.created
2.  response.in_progress
3.  response.output_item.added         (function_call item)
4.  response.function_call_arguments.delta  (repeated N times)
5.  response.function_call_arguments.done   (complete arguments)
6.  response.output_item.done          (function_call item, status: completed)
7.  response.completed                 (full response)
8.  [DONE]
```

### Lifecycle for a multi-item response (function call + text):

```
1.  response.created
2.  response.in_progress
3.  response.output_item.added         (function_call item)
4.  response.function_call_arguments.delta  (repeated)
5.  response.function_call_arguments.done
6.  response.output_item.done          (function_call completed)
7.  response.output_item.added         (message item)
8.  response.content_part.added
9.  response.output_text.delta         (repeated)
10. response.output_text.done
11. response.content_part.done
12. response.output_item.done          (message completed)
13. response.completed
14. [DONE]
```

### Key ordering rules:

1. `response.created` is always first
2. `response.output_item.added` precedes all events for that item
3. `response.content_part.added` precedes all content events for that part
4. Delta events (`.delta`) always precede their completion event (`.done`)
5. `response.output_item.done` closes an item
6. `response.completed` or `response.failed` is always the last non-terminal event
7. `[DONE]` is the literal terminal string
