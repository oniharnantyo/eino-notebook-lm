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

	// WhereClause adds a custom WHERE clause to the query.
	// Use this for advanced filtering.
	// Default: "" (no additional filtering)
	WhereClause string
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

// WithWhereClause returns an option that adds a custom WHERE clause.
func WithWhereClause(where string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RetrieveOptions) {
		opts.WhereClause = where
	})
}

// getRetrieveOptions extracts the RetrieveOptions from the provided retriever options.
func getRetrieveOptions(opts ...retriever.Option) *RetrieveOptions {
	return retriever.GetImplSpecificOptions(&RetrieveOptions{
		IncludeDistance: true,
	}, opts...)
}
