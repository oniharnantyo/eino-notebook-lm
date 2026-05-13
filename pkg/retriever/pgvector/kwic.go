/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pgvector

import (
	"sort"
	"strings"
)

// ExtractKeywordContexts extracts up to 5 snippets from the content that contain any of the keywords.
// Each snippet includes a window of characters around the matching keyword.
// Overlapping windows are merged. Case-insensitive matching is used.
func ExtractKeywordContexts(content string, keywords []string, window int) []string {
	if content == "" || len(keywords) == 0 {
		return nil
	}

	contentLower := strings.ToLower(content)
	var ranges [][2]int

	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		kwLower := strings.ToLower(kw)
		start := 0
		for {
			idx := strings.Index(contentLower[start:], kwLower)
			if idx == -1 {
				break
			}
			matchStart := start + idx
			matchEnd := matchStart + len(kw)

			// ±window char extraction
			s := matchStart - window
			if s < 0 {
				s = 0
			}
			e := matchEnd + window
			if e > len(content) {
				e = len(content)
			}

			ranges = append(ranges, [2]int{s, e})
			start = matchStart + 1 // Move past start of current match to find next occurrence
		}
	}

	if len(ranges) == 0 {
		return nil
	}

	// Sort ranges by start
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i][0] != ranges[j][0] {
			return ranges[i][0] < ranges[j][0]
		}
		return ranges[i][1] < ranges[j][1]
	})

	// Merge overlapping or adjacent ranges
	var merged [][2]int
	if len(ranges) > 0 {
		curr := ranges[0]
		for i := 1; i < len(ranges); i++ {
			if ranges[i][0] <= curr[1] { // Overlap or adjacent
				if ranges[i][1] > curr[1] {
					curr[1] = ranges[i][1]
				}
			} else {
				merged = append(merged, curr)
				curr = ranges[i]
			}
		}
		merged = append(merged, curr)
	}

	// Extract snippets and deduplicate
	snippetMap := make(map[string]struct{})
	var snippets []string
	for _, r := range merged {
		s := strings.TrimSpace(content[r[0]:r[1]])
		if s == "" {
			continue
		}
		if _, ok := snippetMap[s]; !ok {
			snippetMap[s] = struct{}{}
			snippets = append(snippets, s)
			if len(snippets) >= 5 {
				break
			}
		}
	}

	return snippets
}
