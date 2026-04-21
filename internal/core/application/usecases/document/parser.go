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

package document

import (
	"context"
	"io"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
)

// DocumentParser defines the interface for parsing documents
// Single Responsibility: Only handles document parsing
// Dependency Inversion: High-level modules depend on this abstraction
type DocumentParser interface {
	// Parse parses a document from io.Reader and returns schema documents
	Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error)

	// ParseFull parses the document and returns full Kreuzberg structured results
	ParseFull(ctx context.Context, reader io.Reader) ([]kreuzberg.KreuzbergExtractResponse, error)

	// IsAvailable checks if the parser is available
	IsAvailable(ctx context.Context) bool
}

// KreuzbergDocumentParser implements DocumentParser using Kreuzberg service
type KreuzbergDocumentParser struct {
	parser *kreuzberg.KreuzbergParser
}

// NewKreuzbergDocumentParser creates a new Kreuzberg document parser
func NewKreuzbergDocumentParser(cfg *kreuzberg.Config) (DocumentParser, error) {
	p, err := kreuzberg.NewKreuzbergParser(cfg)
	if err != nil {
		return nil, err
	}
	return &KreuzbergDocumentParser{parser: p}, nil
}

// Parse parses a document using Kreuzberg
func (p *KreuzbergDocumentParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	return p.parser.Parse(ctx, reader, opts...)
}

// ParseFull parses the document using Kreuzberg and returns full structured results
func (p *KreuzbergDocumentParser) ParseFull(ctx context.Context, reader io.Reader) ([]kreuzberg.KreuzbergExtractResponse, error) {
	return p.parser.ParseFull(ctx, reader)
}

// IsAvailable checks if the Kreuzberg parser is available
func (p *KreuzbergDocumentParser) IsAvailable(ctx context.Context) bool {
	// TODO: Implement health check for Kreuzberg service
	return true
}

// DocumentParserFactory creates document parsers based on configuration
type DocumentParserFactory struct {
	kreuzbergParser DocumentParser
}

// NewDocumentParserFactory creates a new document parser factory
func NewDocumentParserFactory(kreuzbergParser DocumentParser) *DocumentParserFactory {
	return &DocumentParserFactory{
		kreuzbergParser: kreuzbergParser,
	}
}

// GetParser returns the available document parser
func (f *DocumentParserFactory) GetParser(ctx context.Context) DocumentParser {
	return f.kreuzbergParser
}

// Parse parses a document using the available parser
func (f *DocumentParserFactory) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	return f.GetParser(ctx).Parse(ctx, reader, opts...)
}
