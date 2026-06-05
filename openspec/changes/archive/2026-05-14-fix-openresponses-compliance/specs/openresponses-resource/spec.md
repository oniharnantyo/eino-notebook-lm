## ADDED Requirements

### Requirement: ResponseResource field completeness
The `ResponseResource` struct emitted in `response.created`, `response.in_progress`, and `response.completed` events SHALL include all 27 fields defined in the OpenResponses OpenAPI spec.

#### Scenario: All required fields present in response.created
- **WHEN** the formatter emits a `response.created` event
- **THEN** the `response` object SHALL include `background` (false), `frequency_penalty` (0), `presence_penalty` (0), `instructions` (null), `max_output_tokens` (null), `max_tool_calls` (null), `store` (false), `service_tier` ("default"), `top_logprobs` (0), `top_p` (1), `temperature` (1), `reasoning` (null), `safety_identifier` (null), `prompt_cache_key` (null), `previous_response_id` (null), and `metadata` (empty object)

#### Scenario: Request fields forwarded to ResponseResource
- **WHEN** the request includes `instructions`, `temperature`, `max_output_tokens`, or `previous_response_id`
- **THEN** the `ResponseResource` SHALL reflect those values in the corresponding fields

### Requirement: FunctionCallItem includes call_id
The `FunctionCallItem` struct SHALL include a `call_id` field as required by the OpenAPI spec.

#### Scenario: Function call item emitted with call_id
- **WHEN** the formatter emits a `function_call` output item
- **THEN** the item SHALL include `call_id` populated with the tool call's unique identifier
- **AND** the `function_call_arguments.delta` and `function_call_arguments.done` events SHALL also include `call_id`

### Requirement: ReasoningBody includes encrypted_content field
The `ReasoningBody` struct SHALL include an `encrypted_content` field as defined in the OpenAPI spec.

#### Scenario: Reasoning item with encrypted_content
- **WHEN** the formatter emits a `reasoning` output item in `response.output_item.done`
- **THEN** the item SHALL include an `encrypted_content` field (may be null when no encryption is used)

#### Scenario: Reasoning item with summary content
- **WHEN** the formatter emits a `reasoning` output item
- **THEN** the item SHALL include `summary` array with `SummaryTextContent` items containing the full reasoning summary text
