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
	UseCache                 bool            `json:"use_cache,omitempty"`
	EnableQualityProcessing  bool            `json:"enable_quality_processing,omitempty"`
	ForceOCR                 bool            `json:"force_ocr,omitempty"`
	DisableOCR               bool            `json:"disable_ocr,omitempty"`
	ExtractionTimeoutSecs    int             `json:"extraction_timeout_secs,omitempty"`
	MaxConcurrentExtractions int             `json:"max_concurrent_extractions,omitempty"`
	ResultFormat             string          `json:"result_format,omitempty"`
	OutputFormat             string          `json:"output_format,omitempty"`
	IncludeDocumentStructure bool            `json:"include_document_structure,omitempty"`
	CacheTTLSecs             int             `json:"cache_ttl_secs,omitempty"`
	MaxArchiveDepth          int             `json:"max_archive_depth,omitempty"`
	OCR                      *OCRConfig      `json:"ocr,omitempty"`
	ContentFilter            *ContentFilter  `json:"content_filter,omitempty"`
	Images                   *ImagesConfig   `json:"images,omitempty"`
	PDFOptions               *PDFOptions     `json:"pdf_options,omitempty"`
	TreeSitter               *TreeSitter     `json:"tree_sitter,omitempty"`
	SecurityLimits           *SecurityLimits `json:"security_limits,omitempty"`
	Pages                    *PagesConfig    `json:"pages,omitempty"`
	Chunking                 *Chunking       `json:"chunking,omitempty"`
}

// ContentFilter defines content filtering options.
type ContentFilter struct {
	StripRepeatingText bool `json:"strip_repeating_text,omitempty"`
}

// TreeSitter defines tree-sitter extraction options.
type TreeSitter struct {
	Enabled             bool    `json:"enabled,omitempty"`
	Language            *string `json:"language,omitempty"`
	ContentMode         string  `json:"content_mode,omitempty"`
	IncludeSyntaxColors bool    `json:"include_syntax_colors,omitempty"`
	CommentStyle        string  `json:"comment_style,omitempty"`
}

// SecurityLimits defines resource limits for extraction.
type SecurityLimits struct {
	MaxArchiveSize      int64 `json:"max_archive_size,omitempty"`
	MaxCompressionRatio int   `json:"max_compression_ratio,omitempty"`
	MaxFilesInArchive   int   `json:"max_files_in_archive,omitempty"`
	MaxNestingDepth     int   `json:"max_nesting_depth,omitempty"`
	MaxEntityLength     int   `json:"max_entity_length,omitempty"`
	MaxContentSize      int64 `json:"max_content_size,omitempty"`
	MaxIterations       int   `json:"max_iterations,omitempty"`
	MaxXMLDepth         int   `json:"max_xml_depth,omitempty"`
	MaxTableCells       int   `json:"max_table_cells,omitempty"`
}

// Chunking defines chunking strategy.
type Chunking struct {
	MaxCharacters         int          `json:"max_characters,omitempty"`
	Overlap               int          `json:"overlap,omitempty"`
	Trim                  bool         `json:"trim,omitempty"`
	ChunkerType           string       `json:"chunker_type,omitempty"`
	PrependHeadingContext bool         `json:"prepend_heading_context,omitempty"`
	Sizing                *ChunkSizing `json:"sizing,omitempty"`
}

// ChunkSizing defines how chunks are sized.
type ChunkSizing struct {
	Type string `json:"type,omitempty"`
}

// OCRConfig defines OCR-specific configuration.
type OCRConfig struct {
	Backend           string             `json:"backend,omitempty"`
	Language          string             `json:"language,omitempty"`
	AutoRotate        bool               `json:"auto_rotate,omitempty"`
	TesseractConfig   map[string]any     `json:"tesseract_config,omitempty"`
	PaddleOCRConfig   *PaddleOCRConfig   `json:"paddle_ocr_config,omitempty"`
	QualityThresholds *QualityThresholds `json:"quality_thresholds,omitempty"`
	Model             string             `json:"model,omitempty"`
}

// QualityThresholds defines quality check thresholds.
type QualityThresholds struct {
	MinTotalNonWhitespace       int     `json:"min_total_non_whitespace,omitempty"`
	MinNonWhitespacePerPage     int     `json:"min_non_whitespace_per_page,omitempty"`
	MinMeaningfulWordLen        int     `json:"min_meaningful_word_len,omitempty"`
	MinMeaningfulWords          int     `json:"min_meaningful_words,omitempty"`
	MinAlnumRatio               float64 `json:"min_alnum_ratio,omitempty"`
	MinGarbageChars             int     `json:"min_garbage_chars,omitempty"`
	MaxFragmentedWordRatio      float64 `json:"max_fragmented_word_ratio,omitempty"`
	CriticalFragmentedWordRatio float64 `json:"critical_fragmented_word_ratio,omitempty"`
	MinAvgWordLength            float64 `json:"min_avg_word_length,omitempty"`
	MinWordsForAvgLengthCheck   int     `json:"min_words_for_avg_length_check,omitempty"`
	MinConsecutiveRepeatRatio   float64 `json:"min_consecutive_repeat_ratio,omitempty"`
	MinWordsForRepeatCheck      int     `json:"min_words_for_repeat_check,omitempty"`
	SubstantiveMinChars         int     `json:"substantive_min_chars,omitempty"`
	NonTextMinChars             int     `json:"non_text_min_chars,omitempty"`
	AlnumWsRatioThreshold       float64 `json:"alnum_ws_ratio_threshold,omitempty"`
	PipelineMinQuality          float64 `json:"pipeline_min_quality,omitempty"`
}

// PaddleOCRConfig defines PaddleOCR-specific configuration.
type PaddleOCRConfig struct {
	Language             string  `json:"language,omitempty"`
	UseAngleCls          bool    `json:"use_angle_cls,omitempty"`
	EnableTableDetection bool    `json:"enable_table_detection,omitempty"`
	DetDBThresh          float64 `json:"det_db_thresh,omitempty"`
	DetDBBoxThresh       float64 `json:"det_db_box_thresh,omitempty"`
	DetDBUnclipRatio     float64 `json:"det_db_unclip_ratio,omitempty"`
	DetLimitSideLen      int     `json:"det_limit_side_len,omitempty"`
	RecBatchNum          int     `json:"rec_batch_num,omitempty"`
	Padding              int     `json:"padding,omitempty"`
	DropScore            float64 `json:"drop_score,omitempty"`
	ModelTier            string  `json:"model_tier,omitempty"`
}

// PDFOptions defines PDF-specific extraction options.
type PDFOptions struct {
	ExtractImages           bool             `json:"extract_images,omitempty"`
	ExtractMetadata         bool             `json:"extract_metadata,omitempty"`
	Hierarchy               *HierarchyConfig `json:"hierarchy,omitempty"`
	ExtractAnnotations      bool             `json:"extract_annotations,omitempty"`
	AllowSingleColumnTables bool             `json:"allow_single_column_tables,omitempty"`
}

// HierarchyConfig defines document hierarchy analysis options.
type HierarchyConfig struct {
	Enabled              bool    `json:"enabled,omitempty"`
	KClusters            int     `json:"k_clusters,omitempty"`
	IncludeBBox          bool    `json:"include_bbox,omitempty"`
	OCRCoverageThreshold float64 `json:"ocr_coverage_threshold,omitempty"`
}

// ImagesConfig defines image extraction options.
type ImagesConfig struct {
	ExtractImages      bool `json:"extract_images,omitempty"`
	TargetDPI          int  `json:"target_dpi,omitempty"`
	MaxImageDimension  int  `json:"max_image_dimension,omitempty"`
	InjectPlaceholders bool `json:"inject_placeholders,omitempty"`
	AutoAdjustDPI      bool `json:"auto_adjust_dpi,omitempty"`
	MinDPI             int  `json:"min_dpi,omitempty"`
	MaxDPI             int  `json:"max_dpi,omitempty"`
}

// PagesConfig defines page-level processing options.
type PagesConfig struct {
	ExtractPages      bool   `json:"extract_pages,omitempty"`
	InsertPageMarkers bool   `json:"insert_page_markers,omitempty"`
	MarkerFormat      string `json:"marker_format,omitempty"`
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
	Tables            []any                  `json:"tables"`
	DetectedLanguages []string               `json:"detected_languages"`
	Chunks            []KreuzbergChunk       `json:"chunks"`
	Images            []KreuzbergImage       `json:"images"`
	Elements          []KreuzbergElement     `json:"elements"`
	QualityScore      float64                `json:"quality_score"`
	Pages             []KreuzbergPage        `json:"pages"`
	URIs              []KreuzbergURI         `json:"uris"`
	Annotations       []any                  `json:"annotations"`
}

type KreuzbergChunk struct {
	Content   string             `json:"content"`
	ChunkType string             `json:"chunk_type"`
	Metadata  KreuzbergChunkMeta `json:"metadata"`
}

type KreuzbergChunkMeta struct {
	ChunkIndex     int            `json:"chunk_index"`
	TotalChunks    int            `json:"total_chunks"`
	ByteStart      int            `json:"byte_start"`
	ByteEnd        int            `json:"byte_end"`
	FirstPage      int            `json:"first_page"`
	LastPage       int            `json:"last_page"`
	HeadingContext map[string]any `json:"heading_context"`
}

type KreuzbergImage struct {
	Data       []byte             `json:"data"`
	Format     string             `json:"format"`
	Width      int                `json:"width"`
	Height     int                `json:"height"`
	PageNumber int                `json:"page_number"`
	OCRResult  KreuzbergOCRResult `json:"ocr_result"`
}

type KreuzbergOCRResult struct {
	Content     string                `json:"content"`
	MimeType    string                `json:"mime_type"`
	OCRElements []KreuzbergOCRElement `json:"ocr_elements"`
}

type KreuzbergOCRElement struct {
	Text       string                 `json:"text"`
	Geometry   KreuzbergOCRGeometry   `json:"geometry"`
	Confidence KreuzbergOCRConfidence `json:"confidence"`
	Level      string                 `json:"level"`
	PageNumber int                    `json:"page_number"`
}

type KreuzbergOCRGeometry struct {
	Type   string  `json:"type"`
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type KreuzbergOCRConfidence struct {
	Recognition float64 `json:"recognition"`
}

type KreuzbergElement struct {
	ElementID   string         `json:"element_id"`
	ElementType string         `json:"element_type"`
	Text        string         `json:"text"`
	Metadata    map[string]any `json:"metadata"`
}

type KreuzbergPage struct {
	PageNumber int            `json:"page_number"`
	Dimensions map[string]any `json:"dimensions"`
	Metadata   map[string]any `json:"metadata"`
}

type KreuzbergURI struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// ParseFull parses the document content and returns structured KreuzbergExtractResponse.
func (kp *KreuzbergParser) ParseFull(ctx context.Context, reader io.Reader) ([]KreuzbergExtractResponse, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("kreuzberg parser read all from reader failed: %w", err)
	}

	body, contentType, err := kp.buildMultipartBody(data)
	if err != nil {
		return nil, err
	}

	return kp.executeExtractRequest(ctx, body, contentType)
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
