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

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// ExtractConfig is optional configuration overrides for Kreuzberg
	ExtractConfig *ExtractConfig
}

// ExtractConfig defines the extraction configuration options for Kreuzberg.
type ExtractConfig struct {
	// Enable quality processing for better extraction results
	EnableQualityProcessing bool `json:"enable_quality_processing,omitempty"`

	// ForceOCR forces OCR even for text-based documents
	ForceOCR bool `json:"force_ocr,omitempty"`

	// ResultFormat specifies how to structure results (e.g., "element_based")
	ResultFormat string `json:"result_format,omitempty"`

	// OutputFormat is the format for extracted text: plain, markdown, djot, or html
	OutputFormat string `json:"output_format,omitempty"`

	// IncludeDocumentStructure includes structural elements in output
	IncludeDocumentStructure bool `json:"include_document_structure,omitempty"`

	// OCR configuration for optical character recognition
	OCR *OCRConfig `json:"ocr,omitempty"`

	// PDFOptions for PDF-specific extraction settings
	PDFOptions *PDFOptions `json:"pdf_options,omitempty"`

	// Images options for image extraction
	Images *ImagesConfig `json:"images,omitempty"`

	// Pages options for page-level processing
	Pages *PagesConfig `json:"pages,omitempty"`

	// Layout options for layout analysis
	Layout *LayoutConfig `json:"layout,omitempty"`

	// Table extraction mode
	TableExtractionMode string `json:"table_extraction_mode,omitempty"`
}

// OCRConfig defines OCR-specific configuration.
type OCRConfig struct {
	// Backend specifies the OCR backend (e.g., "paddleocr", "tesseract")
	Backend string `json:"backend,omitempty"`

	// Language specifies the OCR language (e.g., "eng", "deu", "fra")
	Language string `json:"language,omitempty"`

	// PaddleOCRConfig for PaddleOCR-specific settings
	PaddleOCRConfig *PaddleOCRConfig `json:"paddle_ocr_config,omitempty"`

	// Model specifies the OCR model to use (for non-PaddleOCR backends)
	Model string `json:"model,omitempty"`
}

// PaddleOCRConfig defines PaddleOCR-specific configuration.
type PaddleOCRConfig struct {
	// ModelTier specifies the model tier (e.g., "mobile", "server")
	ModelTier string `json:"model_tier,omitempty"`

	// Padding specifies padding around text regions
	Padding int `json:"padding,omitempty"`
}

// PDFOptions defines PDF-specific extraction options.
type PDFOptions struct {
	// ExtractImages indicates whether to extract images from PDFs
	ExtractImages bool `json:"extract_images,omitempty"`

	// ExtractMetadata indicates whether to extract PDF metadata
	ExtractMetadata bool `json:"extract_metadata,omitempty"`

	// Hierarchy options for document hierarchy analysis
	Hierarchy *HierarchyConfig `json:"hierarchy,omitempty"`
}

// HierarchyConfig defines document hierarchy analysis options.
type HierarchyConfig struct {
	// Enabled enables hierarchy analysis
	Enabled bool `json:"enabled,omitempty"`

	// KClusters specifies the number of clusters for hierarchy detection
	KClusters int `json:"k_clusters,omitempty"`

	// IncludeBBox includes bounding box information in output
	IncludeBBox bool `json:"include_bbox,omitempty"`

	// OCRCoverageThreshold is the threshold for OCR coverage (0.0-1.0)
	OCRCoverageThreshold float64 `json:"ocr_coverage_threshold,omitempty"`
}

// ImagesConfig defines image extraction options.
type ImagesConfig struct {
	// ExtractImages indicates whether to extract images
	ExtractImages bool `json:"extract_images,omitempty"`

	// InjectPlaceholders injects placeholders where images were located
	InjectPlaceholders bool `json:"inject_placeholders,omitempty"`
}

// PagesConfig defines page-level processing options.
type PagesConfig struct {
	// ExtractPages enables page-level extraction
	ExtractPages bool `json:"extract_pages,omitempty"`

	// InsertPageMarkers inserts markers between pages
	InsertPageMarkers bool `json:"insert_page_markers,omitempty"`

	// MarkerFormat is the format string for page markers
	MarkerFormat string `json:"marker_format,omitempty"`
}

// LayoutConfig defines layout analysis options.
type LayoutConfig struct {
	// Preset specifies the layout analysis preset (e.g., "accurate", "fast")
	Preset string `json:"preset,omitempty"`

	// ApplyHeuristics applies heuristic rules to layout analysis
	ApplyHeuristics bool `json:"apply_heuristics,omitempty"`
}

// KreuzbergParser extracts text from documents using the Kreuzberg HTTP API.
// It supports multiple file formats including PDF, Office docs, and images with OCR.
type KreuzbergParser struct {
	client        *http.Client
	serviceURL    string
	outputFormat  string
	extractConfig *ExtractConfig
}

// NewKreuzbergParser creates a new Kreuzberg parser.
func NewKreuzbergParser(config *Config) (*KreuzbergParser, error) {
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

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("kreuzberg parser read all from reader failed: %w", err)
	}

	body, contentType, err := kp.buildMultipartBody(data)
	if err != nil {
		return nil, err
	}

	results, err := kp.executeExtractRequest(ctx, body, contentType)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		resultMeta := kp.buildResultMeta(result, commonOpts.ExtraMeta)
		docs = append(docs, &schema.Document{
			Content:  result.Content,
			MetaData: resultMeta,
		})
	}

	return docs, nil
}

// buildMultipartBody creates the multipart form data for the Kreuzberg API.
func (kp *KreuzbergParser) buildMultipartBody(data []byte) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("files", "document")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(data); err != nil {
		return nil, "", fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := writer.WriteField("output_format", kp.outputFormat); err != nil {
		return nil, "", fmt.Errorf("failed to write output_format: %w", err)
	}

	if kp.extractConfig != nil {
		configJSON, err := json.Marshal(kp.extractConfig)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal extract config: %w", err)
		}
		if err := writer.WriteField("config", string(configJSON)); err != nil {
			return nil, "", fmt.Errorf("failed to write config: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}

// executeExtractRequest sends the extract request to Kreuzberg and returns results.
func (kp *KreuzbergParser) executeExtractRequest(ctx context.Context, body *bytes.Buffer, contentType string) ([]KreuzbergExtractResponse, error) {
	url := fmt.Sprintf("%s/extract", kp.serviceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := kp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kreuzberg service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var results []KreuzbergExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no extraction results returned from kreuzberg")
	}

	return results, nil
}

// buildResultMeta creates metadata for an extraction result.
func (kp *KreuzbergParser) buildResultMeta(result KreuzbergExtractResponse, extraMeta map[string]any) map[string]interface{} {
	meta := make(map[string]interface{})
	for k, v := range extraMeta {
		meta[k] = v
	}
	for k, v := range result.Metadata {
		meta[k] = v
	}
	return meta
}
