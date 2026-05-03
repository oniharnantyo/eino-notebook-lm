package source

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/image"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/knowledge"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/pipeline"
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
	IngestContent(ctx context.Context, req *dtos.IngestContentRequest) (*dtos.IngestContentResponse, <-chan pipeline.Progress, error)
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
	pipelineFactory         pipeline.PipelineFactory
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
	pipelineFactory pipeline.PipelineFactory,
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
		pipelineFactory:         pipelineFactory,
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
func (uc *sourceUseCase) IngestContent(ctx context.Context, req *dtos.IngestContentRequest) (*dtos.IngestContentResponse, <-chan pipeline.Progress, error) {
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
		return nil, nil, fmt.Errorf("failed to create source: %w", err)
	}

	// Step 3: Build content source for extraction
	contentSource, err := uc.buildContentSource(req)
	if err != nil {
		// Mark source as failed
		uc.UpdateStatus(ctx, sourceResp.ID, entities.SourceStatusFailed, err)
		return nil, nil, fmt.Errorf("failed to build content source: %w", err)
	}

	// Step 4: Get appropriate extractor
	contentExtractor, err := uc.contentExtractorFactory.GetExtractor(req.ContentType)
	if err != nil {
		uc.UpdateStatus(ctx, sourceResp.ID, entities.SourceStatusFailed, err)
		return nil, nil, fmt.Errorf("unsupported content type: %w", err)
	}

	// Step 5: Determine source type
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = string(mimeType)
	}

	// Step 6: Handle async vs sync processing
	if req.Async {
		progressChan := uc.processAsync(context.WithoutCancel(ctx), sourceResp.ID, contentSource, contentExtractor, string(mimeType))

		return &dtos.IngestContentResponse{
			SourceID:        sourceResp.ID,
			Status:          string(entities.SourceStatusPending),
			IsAsync:         true,
			StatusURL:       fmt.Sprintf("/api/v1/notebooks/%s/knowledges/status/%s", req.NotebookID, sourceResp.ID),
			StatusStreamURL: fmt.Sprintf("/api/v1/notebooks/%s/knowledges/status/%s/stream", req.NotebookID, sourceResp.ID),
		}, progressChan, nil
	}

	// Sync processing
	progressChan, err := uc.processSync(ctx, sourceResp.ID, contentSource, contentExtractor, string(mimeType))
	if err != nil {
		return nil, nil, err
	}

	// Consume progress channel for sync request
	for range progressChan {
		// Just wait for completion
	}

	// Get updated source for response
	source, err := uc.sourceRepo.GetByID(ctx, sourceResp.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get updated source: %w", err)
	}

	return &dtos.IngestContentResponse{
		SourceID:  sourceResp.ID,
		Status:    string(source.Status),
		Error:     source.Error,
		UpdatedAt: source.UpdatedAt,
		IsAsync:   false,
	}, nil, nil
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

// processAsync processes content extraction and knowledge creation asynchronously using the pipeline
func (uc *sourceUseCase) processAsync(
	ctx context.Context,
	sourceID uuid.UUID,
	contentSource usecases.ContentSource,
	extractor extractor.ContentExtractor,
	mimeType string,
) <-chan pipeline.Progress {
	pipe := uc.pipelineFactory.Create(extractor, mimeType)
	initialInput := pipeline.StageInput{
		SourceID: sourceID,
		Data:     contentSource,
	}

	progressChan := pipe.Ingest(ctx, initialInput)

	// Wrap progress channel to handle source status updates and logging
	wrappedChan := make(chan pipeline.Progress)
	go func() {
		defer close(wrappedChan)

		// 1. Update status to processing
		if err := uc.UpdateStatus(ctx, sourceID, entities.SourceStatusProcessing, nil); err != nil {
			uc.logger.Error("Failed to update source status to processing", "source_id", sourceID, "error", err)
		}

		for p := range progressChan {
			if p.Status == pipeline.StatusFailed {
				uc.logger.Error("Pipeline stage failed", "source_id", sourceID, "stage", p.StageName, "error", p.Error)
				_ = uc.UpdateStatus(ctx, sourceID, entities.SourceStatusFailed, p.Error)
			}
			wrappedChan <- p
		}
	}()

	return wrappedChan
}

// processSync processes content extraction and knowledge creation synchronously using the pipeline
func (uc *sourceUseCase) processSync(
	ctx context.Context,
	sourceID uuid.UUID,
	contentSource usecases.ContentSource,
	extractor extractor.ContentExtractor,
	mimeType string,
) (<-chan pipeline.Progress, error) {
	pipe := uc.pipelineFactory.Create(extractor, mimeType)
	initialInput := pipeline.StageInput{
		SourceID: sourceID,
		Data:     contentSource,
	}

	// 1. Update status to processing
	if err := uc.UpdateStatus(ctx, sourceID, entities.SourceStatusProcessing, nil); err != nil {
		uc.logger.Error("Failed to update source status to processing", "source_id", sourceID, "error", err)
	}

	return pipe.Ingest(ctx, initialInput), nil
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
