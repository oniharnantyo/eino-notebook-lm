package pgvector

import (
	"encoding/json"
	"fmt"
	"strings"
)

// vectorToString converts a vector to PostgreSQL vector format.
func vectorToString(embeddings [][]float64, index int) interface{} {
	if embeddings == nil || len(embeddings) <= index || embeddings[index] == nil {
		return nil
	}

	vec := embeddings[index]
	if len(vec) == 0 {
		return nil
	}

	// Convert to pgvector format: "[x1,x2,x3,...]"
	str := "["
	for i, v := range vec {
		if i > 0 {
			str += ","
		}
		str += fmt.Sprintf("%g", v)
	}
	str += "]"

	return str
}

// metadataToJSONB converts metadata map to JSONB.
func metadataToJSONB(metadata map[string]any) interface{} {
	if metadata == nil {
		return nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}

	return data
}

// subIndexesToArray merges option sub-indexes with document sub-indexes.
func subIndexesToArray(optionIndexes []string, docIndexes []string) interface{} {
	// Create a set to avoid duplicates
	indexSet := make(map[string]struct{})

	for _, idx := range optionIndexes {
		if idx != "" {
			indexSet[idx] = struct{}{}
		}
	}

	for _, idx := range docIndexes {
		if idx != "" {
			indexSet[idx] = struct{}{}
		}
	}

	if len(indexSet) == 0 {
		return nil
	}

	result := make([]string, 0, len(indexSet))
	for idx := range indexSet {
		result = append(result, idx)
	}

	return result
}

// extractReferenceID extracts reference_id from document metadata.
func extractReferenceID(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	if referenceID, ok := metadata["reference_id"].(string); ok {
		return referenceID
	}
	return ""
}

// joinQuoted joins string slices with a separator (no quoting, for SQL identifiers).
func joinQuoted(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, item := range items {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(item)
	}
	return sb.String()
}
