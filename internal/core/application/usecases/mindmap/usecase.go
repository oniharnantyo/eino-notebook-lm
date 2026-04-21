package mindmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// MindmapUseCase defines the interface for mindmap generation business logic
type MindmapUseCase interface {
	Generate(ctx context.Context, req *dtos.TriggerMindmapRequest) (*dtos.TriggerMindmapResponse, error)
}

// mindmapUseCase implements MindmapUseCase
type mindmapUseCase struct {
	sourceRepo   repositories.SourceRepository
	artifactRepo repositories.ArtifactRepository
	chatModel    model.BaseChatModel
	logger       *logger.Logger
}

// NewMindmapUseCase creates a new mindmap use case
func NewMindmapUseCase(
	sourceRepo repositories.SourceRepository,
	artifactRepo repositories.ArtifactRepository,
	chatModel model.BaseChatModel,
	log *logger.Logger,
) MindmapUseCase {
	return &mindmapUseCase{
		sourceRepo:   sourceRepo,
		artifactRepo: artifactRepo,
		chatModel:    chatModel,
		logger:       log,
	}
}

// Generate triggers async mindmap generation from source content
func (uc *mindmapUseCase) Generate(ctx context.Context, req *dtos.TriggerMindmapRequest) (*dtos.TriggerMindmapResponse, error) {
	// Validate source IDs
	if len(req.SourceIDs) == 0 {
		return nil, errors.NewValidationError("at least one source ID is required")
	}

	// Fetch all sources to get their content
	var combinedContent string

	for _, sourceID := range req.SourceIDs {
		source, err := uc.sourceRepo.GetByID(ctx, sourceID)
		if err != nil {
			return nil, errors.NewInternalError("failed to find source", err)
		}
		if source == nil {
			return nil, errors.NewNotFoundError("source")
		}
		if source.Status != entities.SourceStatusCompleted {
			return nil, errors.NewValidationError("source must be completed before generating mindmap")
		}
		if source.Content == "" {
			return nil, errors.NewValidationError("source has no content")
		}

		combinedContent += fmt.Sprintf("## %s\n\n%s\n\n", source.Title, source.Content)
	}

	// Create artifact in pending state
	artifact, err := entities.NewArtifact(
		req.NotebookID,
		req.Title,
		entities.ArtifactTypeMindmap,
		entities.ArtifactFormatJSON,
	)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create artifact: %v", err))
	}

	// Set source IDs
	artifact.SetSourceIDs(req.SourceIDs)

	// Save artifact to repository
	if err := uc.artifactRepo.Create(ctx, artifact); err != nil {
		return nil, errors.NewInternalError("failed to save artifact", err)
	}

	// Spawn async generation
	go uc.generateAsync(context.WithoutCancel(ctx), artifact.ID, combinedContent)

	return &dtos.TriggerMindmapResponse{
		ArtifactID: artifact.ID,
		Status:     string(artifact.Status),
		CreatedAt:  artifact.CreatedAt,
	}, nil
}

// generateAsync performs async mindmap generation
func (uc *mindmapUseCase) generateAsync(ctx context.Context, artifactID uuid.UUID, content string) {
	defer func() {
		if r := recover(); r != nil {
			uc.logger.Error("Panic in mindmap generation", "artifact_id", artifactID, "panic", r)
			if err := uc.markFailed(ctx, artifactID, fmt.Errorf("panic: %v", r)); err != nil {
				uc.logger.Error("Failed to mark artifact as failed", "artifact_id", artifactID, "error", err)
			}
		}
	}()

	// Mark as processing
	if err := uc.markProcessing(ctx, artifactID); err != nil {
		uc.logger.Error("Failed to mark artifact as processing", "artifact_id", artifactID, "error", err)
		return
	}

	// Build prompt
	messages := uc.buildMindmapPrompt(content)

	// Generate mindmap via LLM
	result, err := uc.chatModel.Generate(ctx, messages)
	if err != nil {
		uc.logger.Error("Failed to generate mindmap", "artifact_id", artifactID, "error", err)
		if markErr := uc.markFailed(ctx, artifactID, err); markErr != nil {
			uc.logger.Error("Failed to mark artifact as failed", "artifact_id", artifactID, "error", markErr)
		}
		return
	}

	// Parse and validate JSON response
	mindmapData, err := uc.parseMindmapResponse(result.Content)
	if err != nil {
		uc.logger.Error("Failed to parse mindmap response", "artifact_id", artifactID, "error", err)
		if markErr := uc.markFailed(ctx, artifactID, err); markErr != nil {
			uc.logger.Error("Failed to mark artifact as failed", "artifact_id", artifactID, "error", markErr)
		}
		return
	}

	// Serialize to JSON
	jsonContent, err := json.MarshalIndent(mindmapData, "", "  ")
	if err != nil {
		uc.logger.Error("Failed to serialize mindmap", "artifact_id", artifactID, "error", err)
		if markErr := uc.markFailed(ctx, artifactID, err); markErr != nil {
			uc.logger.Error("Failed to mark artifact as failed", "artifact_id", artifactID, "error", markErr)
		}
		return
	}

	// Mark as completed
	if err := uc.markCompleted(ctx, artifactID, string(jsonContent)); err != nil {
		uc.logger.Error("Failed to mark artifact as completed", "artifact_id", artifactID, "error", err)
		return
	}

	uc.logger.Info("Mindmap generation completed", "artifact_id", artifactID)
}

// buildMindmapPrompt creates the prompt for mindmap generation
func (uc *mindmapUseCase) buildMindmapPrompt(content string) []*schema.Message {
	systemPrompt := `You are an expert at creating hierarchical mind maps from textual content.
Your task is to analyze the provided content and create a structured mind map that captures:
- Main topics and themes
- Subtopics and supporting details
- Relationships between concepts
- Key insights and takeaways

IMPORTANT: Return ONLY a valid JSON object. Do not include any explanatory text, markdown formatting, or code blocks.
The JSON must have this exact structure:
{
  "id": "unique-id",
  "label": "Root topic name",
  "summary": "Brief description of the root topic",
  "children": [
    {
      "id": "unique-id-1",
      "label": "First major topic",
      "summary": "Brief description",
      "children": [
        {
          "id": "unique-id-1-1",
          "label": "Subtopic",
          "summary": "Brief description",
          "children": []
        }
      ]
    }
  ]
}

Guidelines:
- Use concise, descriptive labels (max 5 words)
- Provide 1-2 sentence summaries for each node
- Create 2-4 levels of hierarchy
- Each node must have a unique id using format: level1-level2-level3
- Include only meaningful topics (avoid generic nodes like "Introduction" or "Conclusion")
- Balance the tree: avoid overly deep or broad branches`

	userPrompt := fmt.Sprintf("Create a mind map from the following content:\n\n%s", content)

	return []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}
}

// parseMindmapResponse parses and validates the LLM response
func (uc *mindmapUseCase) parseMindmapResponse(response string) (map[string]interface{}, error) {
	// Clean response - remove markdown code blocks if present
	cleaned := response
	if len(cleaned) > 0 {
		// Remove ```json and ``` markers
		if cleaned[0] == '`' {
			start := 0
			end := len(cleaned)
			if idx := findString(cleaned, "\n"); idx > 0 {
				start = idx + 1
			}
			if idx := findLastString(cleaned, "```"); idx > start {
				end = idx
			}
			cleaned = cleaned[start:end]
		}
	}

	// Parse JSON
	var mindmap map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &mindmap); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Validate structure
	if _, ok := mindmap["id"]; !ok {
		return nil, fmt.Errorf("missing required field: id")
	}
	if _, ok := mindmap["label"]; !ok {
		return nil, fmt.Errorf("missing required field: label")
	}
	if _, ok := mindmap["children"]; !ok {
		return nil, fmt.Errorf("missing required field: children")
	}

	return mindmap, nil
}

// markProcessing updates artifact status to processing
func (uc *mindmapUseCase) markProcessing(ctx context.Context, artifactID uuid.UUID) error {
	artifact, err := uc.artifactRepo.GetByID(ctx, artifactID)
	if err != nil {
		return err
	}
	if artifact == nil {
		return errors.NewNotFoundError("artifact")
	}

	artifact.MarkProcessing()
	return uc.artifactRepo.Update(ctx, artifact)
}

// markCompleted updates artifact with content and marks as completed
func (uc *mindmapUseCase) markCompleted(ctx context.Context, artifactID uuid.UUID, content string) error {
	artifact, err := uc.artifactRepo.GetByID(ctx, artifactID)
	if err != nil {
		return err
	}
	if artifact == nil {
		return errors.NewNotFoundError("artifact")
	}

	artifact.SetContent(content)
	artifact.MarkCompleted()
	return uc.artifactRepo.Update(ctx, artifact)
}

// markFailed marks artifact as failed with error message
func (uc *mindmapUseCase) markFailed(ctx context.Context, artifactID uuid.UUID, err error) error {
	artifact, repoErr := uc.artifactRepo.GetByID(ctx, artifactID)
	if repoErr != nil {
		return repoErr
	}
	if artifact == nil {
		return errors.NewNotFoundError("artifact")
	}

	artifact.MarkFailed(err)
	return uc.artifactRepo.Update(ctx, artifact)
}

// findString finds the first occurrence of a substring
func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// findLastString finds the last occurrence of a substring
func findLastString(s, substr string) int {
	idx := -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			idx = i
		}
	}
	return idx
}
