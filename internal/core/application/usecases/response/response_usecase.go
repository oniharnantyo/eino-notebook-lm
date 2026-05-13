package response

import (
	"context"
	errors2 "errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	agent "github.com/oniharnantyo/eino-notebook/internal/core/application/agent/retrieval"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/chat"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/stages"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/sse"
	appuuid "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type responseUseCase struct {
	notebookRepo     repositories.NotebookRepository
	conversationRepo repositories.ConversationRepository
	sourceRepo       repositories.SourceRepository
	embedder         embedding.Embedder
	chatModel        model.BaseChatModel
	defaultModel     string
	historyManager   *history.HistoryManager
	pipeline         *ResponsePipeline
}

func NewResponseUseCase(
	notebookRepo repositories.NotebookRepository,
	conversationRepo repositories.ConversationRepository,
	sourceRepo repositories.SourceRepository,
	embedder embedding.Embedder,
	chatModel model.ToolCallingChatModel,
	defaultModel string,
	historyConfig *history.HistoryConfig,
	retrievalAgent *agent.RetrievalAgent,
	knowledgeRepo repositories.KnowledgeRepository,
) chat.ResponseUseCase {
	agentStage := stages.NewAgentStage(retrievalAgent, sourceRepo, knowledgeRepo)
	historyManager := history.NewHistoryManager(historyConfig)
	histStage := stages.NewHistoryStage(historyManager, conversationRepo)

	return &responseUseCase{
		notebookRepo:     notebookRepo,
		conversationRepo: conversationRepo,
		sourceRepo:       sourceRepo,
		embedder:         embedder,
		chatModel:        chatModel,
		defaultModel:     defaultModel,
		historyManager:   historyManager,
		pipeline:         NewResponsePipeline(agentStage, histStage),
	}
}

func (uc *responseUseCase) Stream(ctx context.Context, req *dtos.ResponseRequest) (*schema.StreamReader[*schema.Message], *sse.StreamMeta, error) {
	_, err := uc.validateNotebook(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	req.Stream = true
	systemPrompt := "You are a helpful AI assistant."
	out, hist, err := uc.pipeline.Execute(ctx, req, systemPrompt, uc.defaultModel)
	if err != nil {
		return nil, nil, err
	}

	responseID := fmt.Sprintf("resp_%s", appuuid.New().String())
	meta := &sse.StreamMeta{
		ResponseID: responseID,
		ModelName:  uc.defaultModel,
		CreatedAt:  time.Now().Unix(),
	}

	onSave := func(accumulated *AccumulatedMessage) {
		respMsg := accumulated.GetFullMessage()

		userInput := ""
		if req.Input != nil {
			if str, ok := req.Input.(string); ok {
				userInput = str
			} else {
				userInput = fmt.Sprintf("%v", req.Input)
			}
		}

		saveInput := stages.HistorySaveInput{
			NotebookID:         *req.NotebookID,
			PreviousResponseID: req.PreviousResponseID,
			ResponseID:         responseID,
			Model:              uc.defaultModel,
			History:            hist,
			UserInput:          userInput,
			ResponseMessage:    respMsg,
			RawInput:           req.Input,
		}

		_ = uc.pipeline.historyStage.Save(context.Background(), saveInput)
	}

	sr := NewHistorySavingReader(out.Stream, onSave).Pipe()

	return sr, meta, nil
}

func (uc *responseUseCase) validateNotebook(ctx context.Context, req *dtos.ResponseRequest) (*appuuid.UUID, error) {
	if req.NotebookID == nil || *req.NotebookID == "" {
		return nil, errors2.New("notebook id is required")
	}

	notebookID, err := appuuid.Parse(*req.NotebookID)
	if err != nil {
		return nil, fmt.Errorf("invalid notebook_id: %w", err)
	}

	exists, err := uc.notebookRepo.Exists(ctx, notebookID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate notebook: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("notebook not found: %s", *req.NotebookID)
	}

	return &notebookID, nil
}
