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

package kreuzberg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
)

// Config is the configuration for Kreuzberg parser.
type Config struct {
	// ServiceURL is the URL of the Kreuzberg service
	ServiceURL string

	// OutputFormat is the format for extracted text: plain, markdown, djot, or html
	OutputFormat string

	// ToPages indicates whether to return each page as a separate document
	ToPages bool

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// ExtractConfig is optional configuration overrides for Kreuzberg
	ExtractConfig *ExtractConfig
}

// ExtractConfig defines the extraction configuration options for Kreuzberg.
type ExtractConfig struct {
	// OCR configuration for optical character recognition
	OCR *OCRConfig `json:"ocr,omitempty"`

	// ForceOCR forces OCR even for text-based documents
	ForceOCR bool `json:"force_ocr,omitempty"`

	// Table extraction mode
	TableExtractionMode string `json:"table_extraction_mode,omitempty"`
}

// OCRConfig defines OCR-specific configuration.
type OCRConfig struct {
	// Language specifies the OCR language (e.g., "eng", "deu", "fra")
	Language string `json:"language,omitempty"`

	// Model specifies the OCR model to use
	Model string `json:"model,omitempty"`
}

// KreuzbergParser extracts text from documents using the Kreuzberg HTTP API.
// It supports multiple file formats including PDF, Office docs, and images with OCR.
type KreuzbergParser struct {
	client        *http.Client
	serviceURL    string
	outputFormat  string
	toPages       bool
	extractConfig *ExtractConfig
}

// NewKreuzbergParser creates a new Kreuzberg parser.
func NewKreuzbergParser(ctx context.Context, config *Config) (*KreuzbergParser, error) {
	if config == nil {
		config = &Config{}
	}

	serviceURL := "http://localhost:8000"
	if config.ServiceURL != "" {
		serviceURL = config.ServiceURL
	}

	outputFormat := "plain"
	if config.OutputFormat != "" {
		outputFormat = config.OutputFormat
	}

	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	return &KreuzbergParser{
		client: &http.Client{
			Timeout: timeout,
		},
		serviceURL:    serviceURL,
		outputFormat:  outputFormat,
		toPages:       config.ToPages,
		extractConfig: config.ExtractConfig,
	}, nil
}

// KreuzbergExtractResponse represents a single extraction result from Kreuzberg.
type KreuzbergExtractResponse struct {
	Content           string                 `json:"content"`
	MimeType          string                 `json:"mime_type"`
	Metadata          map[string]interface{} `json:"metadata"`
	Tables            []interface{}          `json:"tables"`
	DetectedLanguages []string               `json:"detected_languages"`
	Chunks            interface{}            `json:"chunks"`
	Images            interface{}            `json:"images"`
}

// Parse parses the document content from io.Reader using Kreuzberg service.
func (kp *KreuzbergParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
	commonOpts := parser.GetCommonOptions(nil, opts...)

	specificOpts := parser.GetImplSpecificOptions(&options{
		toPages: &kp.toPages,
	}, opts...)

	// Read the content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("kreuzberg parser read all from reader failed: %w", err)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file
	part, err := writer.CreateFormFile("files", "document")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content to form
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add output format field
	if err := writer.WriteField("output_format", kp.outputFormat); err != nil {
		return nil, fmt.Errorf("failed to write output_format: %w", err)
	}

	// Add config field if provided
	if kp.extractConfig != nil {
		configJSON, err := json.Marshal(kp.extractConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal extract config: %w", err)
		}
		if err := writer.WriteField("config", string(configJSON)); err != nil {
			return nil, fmt.Errorf("failed to write config: %w", err)
		}
	}

	// Close writer to finalize form
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/extract", kp.serviceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	resp, err := kp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kreuzberg service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var results []KreuzbergExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no extraction results returned from kreuzberg")
	}

	// Merge common opts metadata with extraction metadata
	metadata := make(map[string]interface{})
	for k, v := range commonOpts.ExtraMeta {
		metadata[k] = v
	}

	// Process each result
	for _, result := range results {
		// Add result-specific metadata
		resultMeta := make(map[string]interface{})
		for k, v := range metadata {
			resultMeta[k] = v
		}
		resultMeta["mime_type"] = result.MimeType
		resultMeta["detected_languages"] = result.DetectedLanguages
		if len(result.Metadata) > 0 {
			for k, v := range result.Metadata {
				resultMeta[k] = v
			}
		}
		resultMeta["extractor"] = "kreuzberg"
		resultMeta["output_format"] = kp.outputFormat

		toPages := specificOpts.toPages != nil && *specificOpts.toPages

		// TODO: Handle toPages when Kreuzberg supports page-level extraction
		// For now, we return the full content as a single document
		if toPages {
			// If pages are requested, we'll need to handle it differently
			// Kreuzberg returns the full content, so we'd need to split it
			// For now, just return as single document
		}

		docs = append(docs, &schema.Document{
			Content:  result.Content,
			MetaData: resultMeta,
		})
	}

	return docs, nil
}

// options holds the implementation-specific options for Kreuzberg parser.
type options struct {
	toPages *bool
}
