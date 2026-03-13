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
