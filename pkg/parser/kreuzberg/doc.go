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

// Package kreuzberg provides a document parser that uses the Kreuzberg HTTP API
// to extract text content from various document formats.
//
// Kreuzberg is a document extraction service that supports:
// - PDF files
// - Office documents (DOCX, XLSX, PPTX)
// - Images with OCR support
// - Plain text files
//
// # Service Setup
//
// Start the Kreuzberg service (default port 8000):
//
//	kreuzberg serve
//
// For custom configuration:
//
//	kreuzberg serve --port 8080
//
// # Usage
//
//	import (
//	    "context"
//	    "os"
//	    "github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
//	)
//
//	// Create parser with default configuration
//	parser, _ := kreuzberg.NewKreuzbergParser(ctx, &kreuzberg.Config{
//	    ServiceURL:   "http://localhost:8000",
//	    OutputFormat: "markdown",
//	    Timeout:      30 * time.Second,
//	})
//
//	// Parse a document
//	file, _ := os.Open("document.pdf")
//	docs, _ := parser.Parse(ctx, file)
//
// # Output Formats
//
// The parser supports multiple output formats:
// - "plain": Plain text (default)
// - "markdown": Markdown format
// - "djot": Djot format
// - "html": HTML format
//
// Use the OutputFormat config option to specify:
//
//	parser, _ := kreuzberg.NewKreuzbergParser(ctx, &kreuzberg.Config{
//	    OutputFormat: "markdown",
//	})
//
// # OCR Configuration
//
// For scanned documents or images, you can configure OCR:
//
//	parser, _ := kreuzberg.NewKreuzbergParser(ctx, &kreuzberg.Config{
//	    ExtractConfig: &kreuzberg.ExtractConfig{
//	        OCR: &kreuzberg.OCRConfig{
//	            Language: "eng",
//	        },
//	        ForceOCR: true,
//	    },
//	})
package kreuzberg
