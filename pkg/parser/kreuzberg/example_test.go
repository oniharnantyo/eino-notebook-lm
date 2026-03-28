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

package kreuzberg_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
)

func ExampleKreuzbergParser_basic() {
	ctx := context.Background()

	// Create a new Kreuzberg parser
	parser, err := kreuzberg.NewKreuzbergParser(&kreuzberg.Config{
		ServiceURL:   "http://localhost:8000",
		OutputFormat: "markdown",
		Timeout:      30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Open a document file
	file, err := os.Open("document.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Parse the document
	docs, err := parser.Parse(ctx, file)
	if err != nil {
		log.Fatal(err)
	}

	// Use the extracted content
	for i, doc := range docs {
		fmt.Printf("Document %d:\n", i+1)
		fmt.Printf("Content: %s\n", doc.Content)
		fmt.Printf("Metadata: %v\n", doc.MetaData)
	}
}

func ExampleKreuzbergParser_withOCR() {
	ctx := context.Background()

	// Create parser with OCR configuration for scanned documents
	parser, err := kreuzberg.NewKreuzbergParser(&kreuzberg.Config{
		ServiceURL:   "http://localhost:8000",
		OutputFormat: "plain",
		ExtractConfig: &kreuzberg.ExtractConfig{
			OCR: &kreuzberg.OCRConfig{
				Language: "eng",
			},
			ForceOCR: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Parse a scanned document
	file, _ := os.Open("scanned.pdf")
	docs, err := parser.Parse(ctx, file)
	if err != nil {
		log.Fatal(err)
	}

	_ = docs
}

func ExampleKreuzbergParser_defaultConfig() {
	ctx := context.Background()

	// Create parser with default configuration (localhost:8000, plain text)
	parser, err := kreuzberg.NewKreuzbergParser(nil)
	if err != nil {
		log.Fatal(err)
	}

	file, _ := os.Open("document.docx")
	docs, err := parser.Parse(ctx, file)
	if err != nil {
		log.Fatal(err)
	}

	_ = docs
}