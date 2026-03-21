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

import "github.com/cloudwego/eino/components/retriever"

// RetrieveOptions are implementation-specific options for the Retrieve method.
type RetrieveOptions struct {
	// IncludeDistance includes the distance value in the document metadata.
	// Default: true
	IncludeDistance bool

	// IncludeVector includes the embedding vector in the document metadata.
	// Default: false
	IncludeVector bool

	// FilterSubIndexes filters results to only include documents with matching sub-indexes.
	// All specified sub-indexes must match (AND operation).
	// Default: nil (no filtering)
	FilterSubIndexes []string

	// FilterReferenceIDs filters results to only include documents with matching reference IDs.
	// Uses the dedicated reference_id column for efficient filtering.
	// Default: nil (no filtering)
	FilterReferenceIDs []string

	// FilterSourceTypes filters results to only include documents with matching source_type in metadata.
	// Uses parameterized queries for safe filtering: metadata->>'source_type' IN ($1, $2, ...)
	// Default: nil (no filtering)
	FilterSourceTypes []string

	// WhereClause adds a custom WHERE clause to the query.
	// Use this for advanced filtering.
	// Default: "" (no additional filtering)
	WhereClause string

	// RRFK is the K parameter for Reciprocal Rank Fusion (RRF).
	// Controls the weight of each ranking in the hybrid search.
	// Higher values give more weight to lower-ranked results.
	// Default: 60
	RRFK int

	// BM25Candidates is the number of top candidates to retrieve from BM25 search.
	// Default: 100
	BM25Candidates int

	// VectorCandidates is the number of top candidates to retrieve from vector search.
	// Default: 100
	VectorCandidates int

	// SkipRerank skips the reranking step when true.
	// When true, returns results directly from RRF without reranking.
	// Default: false
	SkipRerank bool
}

// WithIncludeDistance returns an option that includes distance in metadata.
func WithIncludeDistance(include bool) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.IncludeDistance = include
	})
}

// WithIncludeVector returns an option that includes the vector in metadata.
func WithIncludeVector(include bool) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.IncludeVector = include
	})
}

// WithFilterSubIndexes returns an option that filters by sub-indexes.
func WithFilterSubIndexes(subIndexes []string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.FilterSubIndexes = subIndexes
	})
}

// WithFilterReferenceIDs returns an option that filters by reference IDs.
// Uses the dedicated reference_id column for efficient filtering.
func WithFilterReferenceIDs(referenceIDs []string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.FilterReferenceIDs = referenceIDs
	})
}

// WithFilterSourceTypes returns an option that filters by source_type metadata.
// Uses parameterized queries for safe SQL filtering.
func WithFilterSourceTypes(sourceTypes []string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.FilterSourceTypes = sourceTypes
	})
}

// WithWhereClause returns an option that adds a custom WHERE clause.
func WithWhereClause(where string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.WhereClause = where
	})
}

// WithRRFK returns an option that sets the K parameter for Reciprocal Rank Fusion.
func WithRRFK(k int) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.RRFK = k
	})
}

// WithBM25Candidates returns an option that sets the number of BM25 candidates.
func WithBM25Candidates(candidates int) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.BM25Candidates = candidates
	})
}

// WithVectorCandidates returns an option that sets the number of vector candidates.
func WithVectorCandidates(candidates int) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.VectorCandidates = candidates
	})
}

// WithSkipRerank returns an option that skips the reranking step when true.
func WithSkipRerank(skip bool) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.SkipRerank = skip
	})
}

// getRetrieveOptions extracts the RetrieveOptions from the provided retriever options.
func getRetrieveOptions(opts ...retriever.Option) *RetrieveOptions {
	return retriever.GetImplSpecificOptions(&RetrieveOptions{
		IncludeDistance:  true,
		RRFK:             60,
		BM25Candidates:   100,
		VectorCandidates: 100,
		SkipRerank:       false,
	}, opts...)
}
