package persistence

import (
	"encoding/json"
	"fmt"
	"strings"
)

// mapToJSON converts map to JSON
func mapToJSON(m map[string]any) []byte {
	if m == nil {
		return nil
	}

	data, _ := json.Marshal(m)
	return data
}

// metadataToJSON converts metadata map to JSON (alias for mapToJSON for compatibility)
func metadataToJSON(m map[string]any) []byte {
	return mapToJSON(m)
}

// float32VectorToString converts a float32 slice to pgvector string format
func float32VectorToString(vec []float32) any {
	if len(vec) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "%g", v)
	}
	sb.WriteString("]")
	return sb.String()
}

// stringToFloat32Vector converts pgvector string format to a float32 slice
func stringToFloat32Vector(s string) []float32 {
	if s == "" {
		return nil
	}

	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	vec := make([]float32, 0, len(parts))

	for _, p := range parts {
		var v float32
		fmt.Sscanf(strings.TrimSpace(p), "%f", &v)
		vec = append(vec, v)
	}

	return vec
}
