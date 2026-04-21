package source

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/image"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/knowledge"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/sentence"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SourceUseCase defines the interface for source business logic
type SourceUseCase interface {
	Create(ctx context.Context, req *dtos.CreateSourceRequest) (*dtos.SourceResponse, error)
	GetByID(ctx context.Context, id string) (*dtos.SourceResponse, error)
	GetStatus(ctx context.Context, id string) (*dtos.KnowledgeIngestionStatusResponse, error)
	List(ctx context.Context, req *dtos.ListSourcesRequest) (*dtos.ListSourcesResponse, error)
	Update(ctx context.Context, sourceID uuid.UUID, content string, size int) error
	UpdateStatus(ctx context.Context, sourceID uuid.UUID, status entities.SourceStatus, err error) error
	Delete(ctx context.Context, id string) error

	// IngestContent handles content extraction and knowledge creation
	// Supports both sync and async processing
	IngestContent(ctx context.Context, req *dtos.IngestContentRequest) (*dtos.IngestContentResponse, error)
}

// sourceUseCase implements SourceUseCase
type sourceUseCase struct {
	sourceRepo              repositories.SourceRepository
	notebookRepo            repositories.NotebookRepository
	contentExtractorFactory extractor.ContentExtractorFactory
	documentParserFactory   *document.DocumentParserFactory
	knowledgeUseCase        knowledge.KnowledgeUseCase
	sentenceUseCase         sentence.SentenceUseCase
	imageUseCase            image.ImageUseCase
	logger                  *logger.Logger
}

// NewSourceUseCase creates a new source use case
func NewSourceUseCase(
	sourceRepo repositories.SourceRepository,
	notebookRepo repositories.NotebookRepository,
	contentExtractorFactory extractor.ContentExtractorFactory,
	documentParserFactory *document.DocumentParserFactory,
	knowledgeUseCase knowledge.KnowledgeUseCase,
	sentenceUseCase sentence.SentenceUseCase,
	imageUseCase image.ImageUseCase,
	log *logger.Logger,
) SourceUseCase {
	return &sourceUseCase{
		sourceRepo:              sourceRepo,
		notebookRepo:            notebookRepo,
		contentExtractorFactory: contentExtractorFactory,
		documentParserFactory:   documentParserFactory,
		knowledgeUseCase:        knowledgeUseCase,
		sentenceUseCase:         sentenceUseCase,
		imageUseCase:            imageUseCase,
		logger:                  log,
	}
}

// Create creates a new source
func (uc *sourceUseCase) Create(ctx context.Context, req *dtos.CreateSourceRequest) (*dtos.SourceResponse, error) {
	// Verify notebook exists
	_, err := uc.notebookRepo.FindByID(ctx, req.NotebookID)
	if err != nil {
		return nil, errors.NewInternalError("failed to validate notebook", err)
	}

	// Check for duplicate URI if provided
	if req.URI != "" {
		existing, _ := uc.sourceRepo.GetByURI(ctx, req.NotebookID, req.URI)
		if existing != nil {
			return nil, errors.NewValidationError("source with this URI already exists")
		}
	}

	// Parse content type
	contentType := dtos.ParseContentType(req.ContentType)

	// Create source
	source, err := entities.NewSource(req.NotebookID, req.Title, req.URI, contentType)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create source: %v", err))
	}

	// Set metadata if provided
	if req.Metadata != nil {
		source.Metadata = req.Metadata
	}

	// Save to repository
	if err := uc.sourceRepo.Create(ctx, source); err != nil {
		return nil, errors.NewInternalError("failed to save source", err)
	}

	return dtos.ToSourceResponse(source), nil
}

// GetByID retrieves a source by ID
func (uc *sourceUseCase) GetByID(ctx context.Context, id string) (*dtos.SourceResponse, error) {
	// Parse ID
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid source ID")
	}

	// Find by ID
	source, err := uc.sourceRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return nil, errors.NewNotFoundError("source")
	}

	return dtos.ToSourceResponse(source), nil
}

// List retrieves a paginated list of sources for a notebook
func (uc *sourceUseCase) List(ctx context.Context, req *dtos.ListSourcesRequest) (*dtos.ListSourcesResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	offset := (req.Page - 1) * req.Limit

	// Build filter
	filter := repositories.SourceFilter{
		NotebookID: &req.NotebookID,
		Limit:      req.Limit,
		Offset:     offset,
		OrderBy:    "created_at",
	}

	if req.ContentType != "" {
		ct := dtos.ParseContentType(req.ContentType)
		filter.ContentType = &ct
	}

	// Get sources
	sources, total, err := uc.sourceRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewInternalError("failed to list sources", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListSourcesResponse{
		Sources:    dtos.ToSourceListResponses(sources),
		Total:      int64(total),
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates source content
func (uc *sourceUseCase) Update(ctx context.Context, sourceID uuid.UUID, content string, size int) error {
	source, err := uc.sourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return errors.NewNotFoundError("source")
	}

	source.SetContent(content, size)

	if err := uc.sourceRepo.Update(ctx, source); err != nil {
		return errors.NewInternalError("failed to update source", err)
	}

	return nil
}

// UpdateMetadata updates source metadata
func (uc *sourceUseCase) UpdateMetadata(ctx context.Context, sourceID uuid.UUID, metadata map[string]interface{}) error {
	source, err := uc.sourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return errors.NewNotFoundError("source")
	}

	// Merge metadata into source
	if source.Metadata == nil {
		source.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		source.Metadata[k] = v
	}

	if err := uc.sourceRepo.Update(ctx, source); err != nil {
		return errors.NewInternalError("failed to update source", err)
	}

	return nil
}

// GetStatus retrieves the status of a source for knowledge ingestion tracking
func (uc *sourceUseCase) GetStatus(ctx context.Context, id string) (*dtos.KnowledgeIngestionStatusResponse, error) {
	// Parse ID
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid source ID")
	}

	// Find by ID
	source, err := uc.sourceRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return nil, errors.NewNotFoundError("source")
	}

	return &dtos.KnowledgeIngestionStatusResponse{
		SourceID:  source.ID,
		Status:    string(source.Status),
		Error:     source.Error,
		UpdatedAt: source.UpdatedAt,
	}, nil
}

// UpdateStatus updates the status of a source
func (uc *sourceUseCase) UpdateStatus(ctx context.Context, sourceID uuid.UUID, status entities.SourceStatus, err error) error {
	source, repoErr := uc.sourceRepo.GetByID(ctx, sourceID)
	if repoErr != nil {
		return errors.NewInternalError("failed to get source", repoErr)
	}

	switch status {
	case entities.SourceStatusProcessing:
		source.MarkProcessing()
	case entities.SourceStatusCompleted:
		source.MarkCompleted()
	case entities.SourceStatusFailed:
		source.MarkFailed(err)
	}

	if repoErr := uc.sourceRepo.Update(ctx, source); repoErr != nil {
		return errors.NewInternalError("failed to update source", repoErr)
	}

	return nil
}

// Delete deletes a source by ID
func (uc *sourceUseCase) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid source ID")
	}

	return uc.sourceRepo.Delete(ctx, uid)
}

// IngestContent handles content extraction and knowledge creation
// Supports both sync and async processing
func (uc *sourceUseCase) IngestContent(ctx context.Context, req *dtos.IngestContentRequest) (*dtos.IngestContentResponse, error) {
	// Step 1: Determine MIME type for source creation
	mimeType := req.MIMEType
	if mimeType == "" {
		// Default MIME type based on content type
		switch req.ContentType {
		case usecases.ContentTypeFile:
			mimeType = entities.ContentTypePDF // Default to PDF, should be overridden by file detection
		case usecases.ContentTypeURL:
			mimeType = entities.ContentTypeWebsite
		case usecases.ContentTypeText:
			mimeType = entities.ContentTypeText
		}
	}

	// Step 2: Create source
	createSourceReq := &dtos.CreateSourceRequest{
		NotebookID:  req.NotebookID,
		Title:       req.Title,
		URI:         req.URI,
		ContentType: string(mimeType),
		Metadata:    req.Metadata,
	}

	sourceResp, err := uc.Create(ctx, createSourceReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	// Step 3: Build content source for extraction
	contentSource, err := uc.buildContentSource(req)
	if err != nil {
		// Mark source as failed
		uc.UpdateStatus(ctx, sourceResp.ID, entities.SourceStatusFailed, err)
		return nil, fmt.Errorf("failed to build content source: %w", err)
	}

	// Step 4: Get appropriate extractor
	contentExtractor, err := uc.contentExtractorFactory.GetExtractor(req.ContentType)
	if err != nil {
		uc.UpdateStatus(ctx, sourceResp.ID, entities.SourceStatusFailed, err)
		return nil, fmt.Errorf("unsupported content type: %w", err)
	}

	// Step 5: Determine source type
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = string(mimeType)
	}

	// Step 6: Handle async vs sync processing
	if req.Async {
		go uc.processAsync(context.WithoutCancel(ctx), sourceResp.ID, contentSource, contentExtractor, req.Title, sourceType, req.Metadata, req.SubIndexes)

		return &dtos.IngestContentResponse{
			SourceID:        sourceResp.ID,
			Status:          string(entities.SourceStatusPending),
			IsAsync:         true,
			StatusURL:       fmt.Sprintf("/api/v1/notebooks/%s/knowledges/status/%s", req.NotebookID, sourceResp.ID),
			StatusStreamURL: fmt.Sprintf("/api/v1/notebooks/%s/knowledges/status/%s/stream", req.NotebookID, sourceResp.ID),
		}, nil
	}

	// Sync processing
	if err := uc.processSync(ctx, sourceResp.ID, contentSource, contentExtractor, req.Title, sourceType, req.Metadata, req.SubIndexes); err != nil {
		return nil, err
	}

	// Get updated source for response
	source, err := uc.sourceRepo.GetByID(ctx, sourceResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated source: %w", err)
	}

	return &dtos.IngestContentResponse{
		SourceID:  sourceResp.ID,
		Status:    string(source.Status),
		Error:     source.Error,
		UpdatedAt: source.UpdatedAt,
		IsAsync:   false,
	}, nil
}

// buildContentSource builds a ContentSource from the ingestion request
func (uc *sourceUseCase) buildContentSource(req *dtos.IngestContentRequest) (usecases.ContentSource, error) {
	switch req.ContentType {
	case usecases.ContentTypeFile:
		if req.FileContent == nil {
			return usecases.ContentSource{}, fmt.Errorf("file content required for file content type")
		}
		// Read file content into memory to avoid "file already closed" errors
		content, err := io.ReadAll(req.FileContent)
		if err != nil {
			return usecases.ContentSource{}, fmt.Errorf("failed to read file content: %w", err)
		}
		return usecases.ContentSource{
			Type:     usecases.ContentTypeFile,
			Reader:   bytes.NewReader(content),
			Filename: req.Filename,
			Metadata: req.Metadata,
		}, nil

	case usecases.ContentTypeURL:
		if req.URI == "" {
			return usecases.ContentSource{}, fmt.Errorf("URI required for URL content type")
		}
		return usecases.ContentSource{
			Type:     usecases.ContentTypeURL,
			URL:      req.URI,
			Metadata: req.Metadata,
		}, nil

	case usecases.ContentTypeText:
		return usecases.ContentSource{
			Type:     usecases.ContentTypeText,
			Text:     req.TextContent,
			Metadata: req.Metadata,
		}, nil

	default:
		return usecases.ContentSource{}, fmt.Errorf("unsupported content type: %s", req.ContentType)
	}
}

// processAsync processes content extraction and knowledge creation asynchronously
func (uc *sourceUseCase) processAsync(
	ctx context.Context,
	sourceID uuid.UUID,
	contentSource usecases.ContentSource,
	extractor extractor.ContentExtractor,
	title string,
	sourceType string,
	metadata map[string]interface{},
	subIndexes []string,
) {
	defer func() {
		if r := recover(); r != nil {
			uc.logger.Error("Panic in async processing", "source_id", sourceID, "panic", r)
			_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, fmt.Errorf("panic: %v", r))
		}
	}()

	// 1. Update status to processing
	if err := uc.UpdateStatus(ctx, sourceID, entities.SourceStatusProcessing, nil); err != nil {
		uc.logger.Error("Failed to update source status to processing", "source_id", sourceID, "error", err)
		return
	}
	uc.logger.Info("Knowledge ingestion started", "source_id", sourceID)

	// 2. Extract content
	result, err := extractor.Extract(ctx, contentSource)
	if err != nil {
		uc.logger.Error("Failed to extract content", "source_id", sourceID, "error", err)
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return
	}
	uc.logger.Info("Content extracted successfully", "source_id", sourceID, "chunks", len(result.Chunks), "images", len(result.Images))

	sanitizedContent := uc.sanitizeContent(result.Content)

	// Merge metadata
	mergedMetadata := result.Metadata
	if mergedMetadata == nil {
		mergedMetadata = make(map[string]interface{})
	}
	if metadata != nil {
		for k, v := range metadata {
			mergedMetadata[k] = v
		}
	}
	if len(subIndexes) > 0 {
		mergedMetadata["sub_indexes"] = subIndexes
	}
	if title != "" {
		mergedMetadata["title"] = title
	}
	if sourceType != "" {
		mergedMetadata["source_type"] = sourceType
	}

	// 3. Update source with extracted content and metadata
	if err = uc.Update(ctx, sourceID, sanitizedContent, len(sanitizedContent)); err != nil {
		uc.logger.Error("Failed to update source content", "source_id", sourceID, "error", err)
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return
	}

	if err = uc.UpdateMetadata(ctx, sourceID, mergedMetadata); err != nil {
		uc.logger.Error("Failed to update source metadata", "source_id", sourceID, "error", err)
	}
	uc.logger.Info("Source updated with extracted content", "source_id", sourceID, "content_size", len(sanitizedContent))

	// 4. Save Knowledge chunks
	knowledges, err := uc.knowledgeUseCase.SaveChunks(ctx, sourceID, result.Chunks)
	if err != nil {
		uc.logger.Error("Failed to save knowledge chunks", "source_id", sourceID, "error", err)
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return
	}
	uc.logger.Info("Knowledge chunks saved", "source_id", sourceID, "chunk_count", len(knowledges))

	// 5. Process Sentences and Images in parallel
	uc.logger.Info("Processing sentences and images", "source_id", sourceID, "parallel", true)

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := uc.sentenceUseCase.ProcessKnowledgeChunks(ctx, sourceID, knowledges); err != nil {
			errChan <- fmt.Errorf("sentence processing error: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := uc.imageUseCase.ProcessImages(ctx, sourceID, result.Images); err != nil {
			errChan <- fmt.Errorf("image processing error: %w", err)
		}
	}()

	// Wait for processing to complete or error
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errChan:
		uc.logger.Error("Error in parallel processing", "source_id", sourceID, "error", err)
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return
	case <-done:
		uc.logger.Info("Sentences and images processed successfully", "source_id", sourceID)
	}

	// 6. Mark as completed
	if err = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusCompleted, nil); err != nil {
		uc.logger.Error("Failed to update source status to completed", "source_id", sourceID, "error", err)
	}
	uc.logger.Info("Knowledge ingestion completed successfully", "source_id", sourceID, "status", "completed")
}

// processSync processes content extraction and knowledge creation synchronously
func (uc *sourceUseCase) processSync(
	ctx context.Context,
	sourceID uuid.UUID,
	contentSource usecases.ContentSource,
	extractor extractor.ContentExtractor,
	title string,
	sourceType string,
	metadata map[string]interface{},
	subIndexes []string,
) error {
	// 1. Extract content
	uc.logger.Info("Knowledge ingestion started (sync)", "source_id", sourceID)
	result, err := extractor.Extract(ctx, contentSource)
	if err != nil {
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return fmt.Errorf("failed to extract content: %w", err)
	}
	uc.logger.Info("Content extracted successfully", "source_id", sourceID, "chunks", len(result.Chunks), "images", len(result.Images))

	sanitizedContent := uc.sanitizeContent(result.Content)

	// Merge metadata
	mergedMetadata := result.Metadata
	if mergedMetadata == nil {
		mergedMetadata = make(map[string]interface{})
	}
	if metadata != nil {
		for k, v := range metadata {
			mergedMetadata[k] = v
		}
	}
	if len(subIndexes) > 0 {
		mergedMetadata["sub_indexes"] = subIndexes
	}
	if title != "" {
		mergedMetadata["title"] = title
	}
	if sourceType != "" {
		mergedMetadata["source_type"] = sourceType
	}

	// 2. Update source with extracted content and metadata
	if err = uc.Update(ctx, sourceID, sanitizedContent, len(sanitizedContent)); err != nil {
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return fmt.Errorf("failed to update source content: %w", err)
	}

	if err = uc.UpdateMetadata(ctx, sourceID, mergedMetadata); err != nil {
		uc.logger.Error("Failed to update source metadata", "source_id", sourceID, "error", err)
	}
	uc.logger.Info("Source updated with extracted content", "source_id", sourceID, "content_size", len(sanitizedContent))

	// 3. Save Knowledge chunks
	knowledges, err := uc.knowledgeUseCase.SaveChunks(ctx, sourceID, result.Chunks)
	if err != nil {
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return fmt.Errorf("failed to save knowledge chunks: %w", err)
	}
	uc.logger.Info("Knowledge chunks saved", "source_id", sourceID, "chunk_count", len(knowledges))

	// 4. Process Sentences
	uc.logger.Info("Processing sentences", "source_id", sourceID)
	if err = uc.sentenceUseCase.ProcessKnowledgeChunks(ctx, sourceID, knowledges); err != nil {
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return fmt.Errorf("failed to process sentences: %w", err)
	}
	uc.logger.Info("Sentences processed successfully", "source_id", sourceID)

	// 5. Process Images
	uc.logger.Info("Processing images", "source_id", sourceID, "image_count", len(result.Images))
	if err = uc.imageUseCase.ProcessImages(ctx, sourceID, result.Images); err != nil {
		_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, err)
		return fmt.Errorf("failed to process images: %w", err)
	}
	uc.logger.Info("Images processed successfully", "source_id", sourceID)

	// 6. Mark as completed
	if err = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to update source status to completed: %w", err)
	}
	uc.logger.Info("Knowledge ingestion completed successfully", "source_id", sourceID, "status", "completed")
	return nil
}

// sanitizeContent removes null bytes (0x00) from content.
// PostgreSQL rejects null bytes in TEXT columns with error code 22021.
// Null bytes can come from external services like Kreuzberg (OCR artifacts, binary data).
func (uc *sourceUseCase) sanitizeContent(content string) string {
	// Fast path: check if null bytes exist
	if !strings.Contains(content, "\x00") {
		return content
	}
	// Remove all null bytes
	return strings.ReplaceAll(content, "\x00", "")
}
