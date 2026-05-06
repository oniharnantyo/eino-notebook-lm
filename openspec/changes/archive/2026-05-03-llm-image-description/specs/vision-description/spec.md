# Vision Description Capability

## Purpose

Generate rich, contextual descriptions of images using multimodal LLMs to serve as the primary searchable text representation for image retrieval.

## ADDED Requirements

### Requirement: VisionDescriber interface
The system SHALL provide a `VisionDescriber` interface that accepts image bytes, MIME type, and optional OCR text, and returns a textual description of the image.

#### Scenario: Describe an image with OCR context
- **WHEN** `Describe` is called with image bytes, MIME type "image/png", and OCR text "Shanghai Artificial Intelligence Laboratory"
- **THEN** the system SHALL return a description that includes factual observations about the image and contextual interpretation of its purpose

#### Scenario: Describe an image without OCR context
- **WHEN** `Describe` is called with image bytes, MIME type "image/jpeg", and empty OCR text
- **THEN** the system SHALL return a description based solely on visual analysis of the image

#### Scenario: Description generation failure
- **WHEN** the LLM provider returns an error (timeout, rate limit, invalid response)
- **THEN** `Describe` SHALL return an error and the caller MUST handle it as a failure

### Requirement: Multi-provider support via factory
The system SHALL support multiple vision description providers (Gemini, LlamaCPP) selected via configuration using a factory pattern.

#### Scenario: Create Gemini provider
- **WHEN** configuration specifies `VISION_DESCRIPTION_PROVIDER=gemini` with a valid API key and model
- **THEN** the factory SHALL return a Gemini-backed VisionDescriber

#### Scenario: Create LlamaCPP provider
- **WHEN** configuration specifies `VISION_DESCRIPTION_PROVIDER=llamacpp` with a valid base URL
- **THEN** the factory SHALL return a LlamaCPP-backed VisionDescriber using its `/v1/chat/completions` endpoint with base64-encoded image in the message content

#### Scenario: Unsupported provider
- **WHEN** configuration specifies a provider not in the supported list
- **THEN** the factory SHALL return an error indicating the unsupported provider

### Requirement: Description uses OCR as grounding context
The system SHALL pass OCR text to the LLM as grounding context when generating descriptions, to reduce hallucination and improve factual accuracy.

#### Scenario: LLM receives OCR text as reference
- **WHEN** generating a description for an image with OCR text "Revenue: $4.2M | Growth: 23%"
- **THEN** the prompt to the LLM SHALL include the OCR text as reference material alongside the image data

### Requirement: Configuration for vision description
The system SHALL support environment variables for vision description: `VISION_DESCRIPTION_PROVIDER`, `VISION_DESCRIPTION_MODEL`, `VISION_DESCRIPTION_API_KEY`, `VISION_DESCRIPTION_BASE_URL`.

#### Scenario: Load configuration from environment
- **WHEN** the application starts with `VISION_DESCRIPTION_PROVIDER=gemini`, `VISION_DESCRIPTION_MODEL=gemini-2.0-flash-exp`, and `VISION_DESCRIPTION_API_KEY=abc123`
- **THEN** the system SHALL initialize the Gemini VisionDescriber with the specified model and API key