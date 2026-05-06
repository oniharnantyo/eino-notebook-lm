# Capability: Keyword-in-Context (KWIC) Search

## Purpose
The Keyword-in-Context (KWIC) Search capability provides utility functions for extracting contextual snippets around keyword matches within a larger body of text. This ensures that users and agents can see exactly how search terms are used in the source material without reading entire documents.

## Requirements

### Requirement: Keyword-in-context snippet extraction
The system SHALL provide a `ExtractKeywordContexts(content string, keywords []string, window int) []string` function that extracts contextual snippets around keyword matches in a text. The function MUST perform case-insensitive matching, extract ±`window` characters around each keyword occurrence, merge overlapping windows, and deduplicate identical snippets. The function SHALL cap results at 5 snippets per content.

#### Scenario: Single keyword match
- **WHEN** `ExtractKeywordContexts` is called with content "Alexander Graham Bell invented the telephone in 1876", keywords ["telephone"], window 80
- **THEN** the result contains one snippet wrapping "telephone" with up to 80 chars of context on each side
- **AND** the snippet is prefixed and suffixed with "..." if the window extends beyond the content boundaries

#### Scenario: Multiple keywords with overlapping windows
- **WHEN** `ExtractKeywordContexts` is called with content where "telephone" and "invented" occur within 160 chars of each other
- **THEN** overlapping windows SHALL be merged into a single snippet
- **AND** the merged snippet covers both keyword matches

#### Scenario: Multiple non-overlapping keyword matches
- **WHEN** `ExtractKeywordContexts` is called with keywords ["telephone", "born"] and both appear in the content but are far apart
- **THEN** the result contains separate snippets for each keyword context
- **AND** each snippet is independently wrapped with "..."

#### Scenario: Multi-word keyword phrase
- **WHEN** `ExtractKeywordContexts` is called with keywords ["Alexander Graham Bell"]
- **THEN** the function matches the full phrase "Alexander Graham Bell" as a unit
- **AND** extracts ±80 chars around the entire phrase

#### Scenario: Keyword not found in content
- **WHEN** `ExtractKeywordContexts` is called with a keyword that does not appear in the content
- **THEN** the function returns no snippets for that keyword
- **AND** continues processing other keywords

#### Scenario: Snippet count capped at 5
- **WHEN** a keyword appears more than 5 times in the content with non-overlapping windows
- **THEN** the function returns at most 5 snippets
- **AND** the 5 snippets are the first 5 occurrences

#### Scenario: Case-insensitive matching
- **WHEN** `ExtractKeywordContexts` is called with keywords ["telephone"] and content contains "Telephone" (capitalized)
- **THEN** the function SHALL match regardless of case
- **AND** the returned snippet preserves the original casing from the content
