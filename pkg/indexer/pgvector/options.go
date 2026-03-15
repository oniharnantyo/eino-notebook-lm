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

import "github.com/cloudwego/eino/components/indexer"

// StoreOptions are implementation-specific options for the Store method.
type StoreOptions struct {
	// SkipExisting skips documents that already exist (by ID).
	SkipExisting bool

	// Upsert performs an upsert operation (insert or update) instead of failing on conflict.
	Upsert bool
}

// WithSkipExisting returns an option that skips documents that already exist.
func WithSkipExisting(skip bool) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *StoreOptions) {
		opts.SkipExisting = skip
	})
}

// WithUpsert returns an option that enables upsert mode.
func WithUpsert(upsert bool) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *StoreOptions) {
		opts.Upsert = upsert
	})
}

// getStoreOptions extracts the StoreOptions from the provided indexer options.
func getStoreOptions(opts ...indexer.Option) *StoreOptions {
	return indexer.GetImplSpecificOptions(new(StoreOptions), opts...)
}

// SearchOptions are implementation-specific options for similarity search.
type SearchOptions struct {
	// K is the number of results to return. If 0, uses config.DefaultK.
	K int

	// EF is the HNSW search parameter. If 0, uses config.HNSWM.
	EF int

	// IncludeDistance includes the distance/score in the document metadata.
	IncludeDistance bool

	// SubIndexes filters results by sub-indexes.
	SubIndexes []string

	// WhereClause adds a custom WHERE clause for filtering.
	WhereClause string
}

// WithK returns an option that sets the number of results to return.
func WithK(k int) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *SearchOptions) {
		opts.K = k
	})
}

// WithSearchEF returns an option that sets the HNSW search parameter.
func WithSearchEF(ef int) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *SearchOptions) {
		opts.EF = ef
	})
}

// WithIncludeDistance returns an option that includes distance in results.
func WithIncludeDistance(include bool) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *SearchOptions) {
		opts.IncludeDistance = include
	})
}

// WithSubIndexes returns an option that filters by sub-indexes.
func WithSubIndexes(indexes ...string) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *SearchOptions) {
		opts.SubIndexes = indexes
	})
}

// WithWhereClause returns an option that adds a custom WHERE clause.
func WithWhereClause(clause string) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *SearchOptions) {
		opts.WhereClause = clause
	})
}

// getSearchOptions extracts the SearchOptions from the provided indexer options.
func getSearchOptions(opts ...indexer.Option) *SearchOptions {
	return indexer.GetImplSpecificOptions(new(SearchOptions), opts...)
}
