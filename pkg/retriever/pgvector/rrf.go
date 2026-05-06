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
	"sort"
)

// DefaultK is the default constant used in RRF scoring.
// A value of 60 is commonly used in information retrieval research.
const DefaultK = 60

// RankedDocument represents a document with its rank from a retrieval method.
type RankedDocument struct {
	// ID is the unique identifier for the document.
	ID string
	// Rank is the position of the document in the ranked list (1-indexed).
	Rank int
}

// MergeByRRF merges multiple ranked result lists using Reciprocal Rank Fusion (RRF).
//
// The RRF algorithm combines rankings from different retrieval methods (e.g., vector search
// and BM25) by computing a fusion score for each document based on its ranks across all lists.
//
// The RRF formula is: score(d) = Σ (1 / (k + rank(d)))
//
// Where:
//   - d is a document
//   - k is a constant (default 60) that dampens the contribution of high ranks
//   - rank(d) is the position of document d in a ranked list (1-indexed)
//   - The sum is taken over all ranked lists where document d appears
//
// Documents that appear in multiple lists receive higher scores, and documents
// that rank higher in each list contribute more to the score.
//
// # Parameters
//
//	resultLists - Variable number of ranked document lists from different retrieval methods
//	k - The constant parameter in the RRF formula (use DefaultK for standard behavior)
//
// # Returns
//
// A map of document IDs to their RRF scores. Higher scores indicate better relevance.
//
// # Example
//
//	vectorResults := []RankedDocument{
//	    {ID: "doc1", Rank: 1},
//	    {ID: "doc2", Rank: 2},
//	}
//	bm25Results := []RankedDocument{
//	    {ID: "doc2", Rank: 1},
//	    {ID: "doc3", Rank: 2},
//	}
//	scores := MergeByRRF(DefaultK, vectorResults, bm25Results)
//	// scores["doc1"] = 1/(60+1) ≈ 0.0164 (only in vector results at rank 1)
//	// scores["doc2"] = 1/(60+2) + 1/(60+1) ≈ 0.0323 (rank 2 in vector, rank 1 in BM25)
//	// scores["doc3"] = 1/(60+2) ≈ 0.0161 (only in BM25 results at rank 2)
func MergeByRRF(k int, resultLists ...[]RankedDocument) map[string]float64 {
	scores := make(map[string]float64)

	for _, results := range resultLists {
		for _, doc := range results {
			if doc.Rank < 1 {
				continue
			}
			rrfScore := 1.0 / float64(k+doc.Rank)
			scores[doc.ID] += rrfScore
		}
	}

	return scores
}

// RankedDocumentWithScore represents a document with its RRF fusion score.
type RankedDocumentWithScore struct {
	ID    string
	Score float64
}

// SortByScore sorts ranked documents by their RRF scores in descending order.
func SortByScore(scores map[string]float64) []RankedDocumentWithScore {
	result := make([]RankedDocumentWithScore, 0, len(scores))

	for id, score := range scores {
		result = append(result, RankedDocumentWithScore{
			ID:    id,
			Score: score,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		return result[i].ID < result[j].ID
	})

	return result
}

// TopN returns the top N documents from a ranked list by score.
func TopN(ranked []RankedDocumentWithScore, n int) []RankedDocumentWithScore {
	if n >= len(ranked) {
		return ranked
	}
	return ranked[:n]
}

// NormalizeScores normalizes RRF scores to the range [0, 1] based on the maximum score.
// This is useful when you need to compare RRF scores across different queries.
func NormalizeScores(scores map[string]float64) map[string]float64 {
	if len(scores) == 0 {
		return scores
	}

	maxScore := math.Inf(-1)
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
	}

	if maxScore == 0 {
		return scores
	}

	normalized := make(map[string]float64, len(scores))
	for id, score := range scores {
		normalized[id] = score / maxScore
	}

	return normalized
}
