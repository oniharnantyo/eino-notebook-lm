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
	"math"
	"testing"
)

func TestMergeByRRF_SingleList(t *testing.T) {
	tests := []struct {
		name     string
		k        int
		results  []RankedDocument
		expected map[string]float64
	}{
		{
			name: "single document at rank 1",
			k:    60,
			results: []RankedDocument{
				{ID: "doc1", Rank: 1},
			},
			expected: map[string]float64{
				"doc1": 1.0 / 61.0,
			},
		},
		{
			name: "multiple documents",
			k:    60,
			results: []RankedDocument{
				{ID: "doc1", Rank: 1},
				{ID: "doc2", Rank: 2},
				{ID: "doc3", Rank: 3},
			},
			expected: map[string]float64{
				"doc1": 1.0 / 61.0,
				"doc2": 1.0 / 62.0,
				"doc3": 1.0 / 63.0,
			},
		},
		{
			name: "custom k value",
			k:    10,
			results: []RankedDocument{
				{ID: "doc1", Rank: 1},
				{ID: "doc2", Rank: 5},
			},
			expected: map[string]float64{
				"doc1": 1.0 / 11.0,
				"doc2": 1.0 / 15.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeByRRF(tt.k, tt.results)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
			}

			for id, expectedScore := range tt.expected {
				actualScore, ok := result[id]
				if !ok {
					t.Errorf("missing document ID: %s", id)
					continue
				}
				if math.Abs(actualScore-expectedScore) > 1e-9 {
					t.Errorf("document %s: expected score %f, got %f", id, expectedScore, actualScore)
				}
			}
		})
	}
}

func TestMergeByRRF_MultipleLists(t *testing.T) {
	tests := []struct {
		name     string
		k        int
		lists    [][]RankedDocument
		expected map[string]float64
	}{
		{
			name: "two lists with overlapping documents",
			k:    60,
			lists: [][]RankedDocument{
				{
					{ID: "doc1", Rank: 1},
					{ID: "doc2", Rank: 2},
				},
				{
					{ID: "doc2", Rank: 1},
					{ID: "doc3", Rank: 2},
				},
			},
			expected: map[string]float64{
				"doc1": 1.0 / 61.0,          // rank 1 in list 1 only
				"doc2": 1.0/62.0 + 1.0/61.0, // rank 2 in list 1, rank 1 in list 2
				"doc3": 1.0 / 62.0,          // rank 2 in list 2 only
			},
		},
		{
			name: "three lists with different overlaps",
			k:    100,
			lists: [][]RankedDocument{
				{
					{ID: "doc1", Rank: 1},
					{ID: "doc2", Rank: 2},
					{ID: "doc3", Rank: 3},
				},
				{
					{ID: "doc2", Rank: 1},
					{ID: "doc3", Rank: 2},
					{ID: "doc4", Rank: 3},
				},
				{
					{ID: "doc1", Rank: 1},
					{ID: "doc3", Rank: 3},
					{ID: "doc4", Rank: 5},
				},
			},
			expected: map[string]float64{
				"doc1": 1.0/101.0 + 1.0/101.0,             // rank 1 in lists 1 and 3
				"doc2": 1.0/102.0 + 1.0/101.0,             // rank 2 in list 1, rank 1 in list 2
				"doc3": 1.0/103.0 + 1.0/102.0 + 1.0/103.0, // rank 3 in list 1, rank 2 in list 2, rank 3 in list 3
				"doc4": 1.0/103.0 + 1.0/105.0,             // rank 3 in list 2, rank 5 in list 3
			},
		},
		{
			name: "empty list included",
			k:    60,
			lists: [][]RankedDocument{
				{
					{ID: "doc1", Rank: 1},
				},
				{},
			},
			expected: map[string]float64{
				"doc1": 1.0 / 61.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeByRRF(tt.k, tt.lists...)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
			}

			for id, expectedScore := range tt.expected {
				actualScore, ok := result[id]
				if !ok {
					t.Errorf("missing document ID: %s", id)
					continue
				}
				if math.Abs(actualScore-expectedScore) > 1e-9 {
					t.Errorf("document %s: expected score %f, got %f", id, expectedScore, actualScore)
				}
			}
		})
	}
}

func TestMergeByRRF_EdgeCases(t *testing.T) {
	t.Run("no result lists", func(t *testing.T) {
		result := MergeByRRF(60)

		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("all empty result lists", func(t *testing.T) {
		result := MergeByRRF(60, []RankedDocument{}, []RankedDocument{})

		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("documents with rank 0 or negative (should be skipped)", func(t *testing.T) {
		result := MergeByRRF(60,
			[]RankedDocument{
				{ID: "doc1", Rank: 0},
				{ID: "doc2", Rank: -1},
				{ID: "doc3", Rank: 1},
			},
		)

		if len(result) != 1 {
			t.Errorf("expected 1 result, got %d", len(result))
		}

		if _, ok := result["doc3"]; !ok {
			t.Error("expected doc3 to be present")
		}
	})

	t.Run("k value of 0", func(t *testing.T) {
		result := MergeByRRF(0,
			[]RankedDocument{
				{ID: "doc1", Rank: 1},
				{ID: "doc2", Rank: 2},
			},
		)

		expected := map[string]float64{
			"doc1": 1.0 / 1.0,
			"doc2": 1.0 / 2.0,
		}

		for id, expectedScore := range expected {
			actualScore, ok := result[id]
			if !ok {
				t.Errorf("missing document ID: %s", id)
				continue
			}
			if math.Abs(actualScore-expectedScore) > 1e-9 {
				t.Errorf("document %s: expected score %f, got %f", id, expectedScore, actualScore)
			}
		}
	})
}

func TestSortByScore(t *testing.T) {
	scores := map[string]float64{
		"doc1": 0.5,
		"doc2": 0.9,
		"doc3": 0.3,
		"doc4": 0.9,
	}

	result := SortByScore(scores)

	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	expectedOrder := []string{"doc2", "doc4", "doc1", "doc3"}
	for i, id := range expectedOrder {
		if result[i].ID != id {
			t.Errorf("position %d: expected ID %s, got %s", i, id, result[i].ID)
		}
	}

	// Verify scores are correct
	if result[0].Score != 0.9 || result[1].Score != 0.9 {
		t.Error("top scores should be 0.9")
	}
	if result[2].Score != 0.5 {
		t.Error("third score should be 0.5")
	}
	if result[3].Score != 0.3 {
		t.Error("fourth score should be 0.3")
	}
}

func TestSortByScore_TieBreaking(t *testing.T) {
	// Test that ties are broken by ID (alphabetically)
	scores := map[string]float64{
		"zebra":  0.5,
		"apple":  0.5,
		"banana": 0.5,
	}

	result := SortByScore(scores)

	expectedOrder := []string{"apple", "banana", "zebra"}
	for i, id := range expectedOrder {
		if result[i].ID != id {
			t.Errorf("position %d: expected ID %s, got %s", i, id, result[i].ID)
		}
	}
}

func TestTopN(t *testing.T) {
	ranked := []RankedDocumentWithScore{
		{ID: "doc1", Score: 0.9},
		{ID: "doc2", Score: 0.8},
		{ID: "doc3", Score: 0.7},
		{ID: "doc4", Score: 0.6},
		{ID: "doc5", Score: 0.5},
	}

	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{"top 2", 2, 2},
		{"top 5", 5, 5},
		{"top 10 (more than available)", 10, 5},
		{"top 0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TopN(ranked, tt.n)

			if len(result) != tt.expected {
				t.Errorf("expected %d results, got %d", tt.expected, len(result))
			}

			if tt.expected > 0 && tt.expected <= len(ranked) {
				for i := 0; i < tt.expected; i++ {
					if result[i].ID != ranked[i].ID {
						t.Errorf("position %d: expected ID %s, got %s", i, ranked[i].ID, result[i].ID)
					}
				}
			}
		})
	}
}

func TestNormalizeScores(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]float64
		expected map[string]float64
	}{
		{
			name: "normal scores",
			input: map[string]float64{
				"doc1": 1.0,
				"doc2": 0.5,
				"doc3": 0.25,
			},
			expected: map[string]float64{
				"doc1": 1.0,
				"doc2": 0.5,
				"doc3": 0.25,
			},
		},
		{
			name: "scores requiring normalization",
			input: map[string]float64{
				"doc1": 0.0328,
				"doc2": 0.0164,
				"doc3": 0.0109,
			},
			expected: map[string]float64{
				"doc1": 1.0,
				"doc2": 0.5,
				"doc3": 0.332,
			},
		},
		{
			name:     "empty map",
			input:    map[string]float64{},
			expected: map[string]float64{},
		},
		{
			name: "all zero scores",
			input: map[string]float64{
				"doc1": 0.0,
				"doc2": 0.0,
			},
			expected: map[string]float64{
				"doc1": 0.0,
				"doc2": 0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeScores(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
			}

			for id, expectedScore := range tt.expected {
				actualScore, ok := result[id]
				if !ok {
					t.Errorf("missing document ID: %s", id)
					continue
				}
				tolerance := 0.01
				if math.Abs(actualScore-expectedScore) > tolerance {
					t.Errorf("document %s: expected score %f (±%f), got %f", id, expectedScore, tolerance, actualScore)
				}
			}
		})
	}
}

func TestDefaultK(t *testing.T) {
	if DefaultK != 60 {
		t.Errorf("expected DefaultK to be 60, got %d", DefaultK)
	}
}

func ExampleMergeByRRF() {
	// Simulate vector search results
	vectorResults := []RankedDocument{
		{ID: "doc1", Rank: 1},
		{ID: "doc2", Rank: 2},
		{ID: "doc3", Rank: 3},
	}

	// Simulate BM25 keyword search results
	bm25Results := []RankedDocument{
		{ID: "doc2", Rank: 1},
		{ID: "doc4", Rank: 2},
		{ID: "doc5", Rank: 3},
	}

	// Merge using RRF
	scores := MergeByRRF(DefaultK, vectorResults, bm25Results)

	// Sort by score
	ranked := SortByScore(scores)

	// Get top 3 results
	topResults := TopN(ranked, 3)

	// doc2 ranks highest because it appears in both result lists
	_ = topResults
}
