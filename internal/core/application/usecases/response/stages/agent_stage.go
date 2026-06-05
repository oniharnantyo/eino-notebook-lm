package stages

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	agent "github.com/oniharnantyo/eino-notebook/internal/core/application/agent/retrieval"
	retrievalTools "github.com/oniharnantyo/eino-notebook/internal/core/application/agent/retrieval/tools"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type AgentStage struct {
	retrievalAgent *agent.RetrievalAgent
	sourceRepo     repositories.SourceRepository
	knowledgeRepo  repositories.KnowledgeRepository
}

func NewAgentStage(retrievalAgent *agent.RetrievalAgent, sourceRepo repositories.SourceRepository, knowledgeRepo repositories.KnowledgeRepository) *AgentStage {
	return &AgentStage{
		retrievalAgent: retrievalAgent,
		sourceRepo:     sourceRepo,
		knowledgeRepo:  knowledgeRepo,
	}
}

func (s *AgentStage) Execute(ctx context.Context, input *schema.Message, sourceIDs []uuid.UUID) (GenerationOutput, error) {
	catalog, err := agent.BuildCatalog(ctx, s.sourceRepo, sourceIDs)
	if err != nil {
		return GenerationOutput{}, fmt.Errorf("failed to build source catalog: %w", err)
	}

	listSourcesTool := retrievalTools.NewListSourcesTool(s.sourceRepo, sourceIDs)
	chunkReadTool := retrievalTools.NewChunkReadTool(s.knowledgeRepo, nil)

	ag, err := s.retrievalAgent.Invoke(ctx, listSourcesTool, chunkReadTool)
	if err != nil {
		return GenerationOutput{}, fmt.Errorf("failed to create agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           ag,
		EnableStreaming: true,
	})

	iter := runner.Run(ctx, []adk.Message{{Role: schema.User, Content: input.Content}}, adk.WithSessionValues(map[string]any{"catalog": catalog}))

	pr, pw := schema.Pipe[*schema.Message](10)

	go func() {
		defer pw.Close()
		parser := &thinkingParser{}
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				_ = pw.Send(nil, event.Err)
				return
			}

			if event.Output != nil && event.Output.MessageOutput != nil {
				mv := event.Output.MessageOutput

				if mv.IsStreaming && mv.MessageStream != nil {
					stream := mv.MessageStream
					defer stream.Close()

					for {
						chunk, err := stream.Recv()
						if err != nil {
							if err.Error() == "EOF" || err.Error() == "nil stream" {
								break
							}
							_ = pw.Send(nil, err)
							return
						}

						chunkJson, _ := json.Marshal(chunk)
						fmt.Println(string(chunkJson))

						if chunk == nil {
							continue
						}

						if chunk.Role == schema.Tool {
							continue
						}

						if len(chunk.ToolCalls) > 0 {
							_ = pw.Send(chunk, nil)
							continue
						}

						if err := processChunk(chunk, pw, parser); err != nil {
							_ = pw.Send(nil, err)
							return
						}
					}
				} else {
					_, msg, err := mapAgentEventToSSE(event)
					if err != nil {
						_ = pw.Send(nil, err)
						break
					}

					if msg != nil && msg.Role == schema.Tool {
						continue
					}

					if msg != nil && len(msg.ToolCalls) > 0 {
						_ = pw.Send(msg, nil)
						continue
					}

					if msg != nil && (msg.Content != "" || msg.ReasoningContent != "") {
						if msg.Content != "" {
							reasoning, content := parser.Process(msg.Content)
							if reasoning != "" {
								_ = pw.Send(&schema.Message{
									Role:             msg.Role,
									ReasoningContent: reasoning,
									Extra:            msg.Extra,
								}, nil)
							}
							if content != "" {
								_ = pw.Send(&schema.Message{
									Role:    msg.Role,
									Content: content,
									Extra:   msg.Extra,
								}, nil)
							}
						} else {
							_ = pw.Send(msg, nil)
						}
					}
				}
			}
		}
	}()

	return GenerationOutput{Stream: pr}, nil
}

func processChunk(chunk *schema.Message, pw *schema.StreamWriter[*schema.Message], parser *thinkingParser) error {
	if len(chunk.AssistantGenMultiContent) > 0 {
		return processMultimodalContent(chunk, pw, parser)
	}

	if chunk.ReasoningContent != "" {
		_ = pw.Send(&schema.Message{
			Role:             chunk.Role,
			ReasoningContent: chunk.ReasoningContent,
			Extra:            chunk.Extra,
		}, nil)
	}

	if chunk.Content != "" {
		reasoning, content := parser.Process(chunk.Content)
		if reasoning != "" {
			_ = pw.Send(&schema.Message{
				Role:             chunk.Role,
				ReasoningContent: reasoning,
				Extra:            chunk.Extra,
			}, nil)
		}
		if content != "" {
			_ = pw.Send(&schema.Message{
				Role:    chunk.Role,
				Content: content,
				Extra:   chunk.Extra,
			}, nil)
		}
		return nil
	}

	return nil
}

func processMultimodalContent(chunk *schema.Message, pw *schema.StreamWriter[*schema.Message], parser *thinkingParser) error {
	for _, part := range chunk.AssistantGenMultiContent {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			if part.Text != "" {
				reasoning, content := parser.Process(part.Text)
				if reasoning != "" {
					_ = pw.Send(&schema.Message{
						Role:             chunk.Role,
						ReasoningContent: reasoning,
						Extra:            chunk.Extra,
					}, nil)
				}
				if content != "" {
					_ = pw.Send(&schema.Message{
						Role:    chunk.Role,
						Content: content,
						Extra:   chunk.Extra,
					}, nil)
				}
			}

		case schema.ChatMessagePartTypeReasoning:
			continue

		case schema.ChatMessagePartTypeImageURL, schema.ChatMessagePartTypeAudioURL, schema.ChatMessagePartTypeVideoURL:
			_ = pw.Send(&schema.Message{
				Role:                     chunk.Role,
				AssistantGenMultiContent: []schema.MessageOutputPart{part},
			}, nil)

		default:
			_ = pw.Send(&schema.Message{
				Role:                     chunk.Role,
				AssistantGenMultiContent: []schema.MessageOutputPart{part},
			}, nil)
		}
	}
	return nil
}

type thinkingParser struct {
	inReasoning bool
	buffer      string
}

func (p *thinkingParser) Process(text string) (reasoning string, content string) {
	fullText := p.buffer + text
	p.buffer = ""

	for len(fullText) > 0 {
		if !p.inReasoning {
			// Handle <think> or <thinking> start tags
			// Support both to be flexible since some models output <think>
			tag := "<thinking>"
			idx := findStartTag(fullText, tag)
			if idx == -1 {
				tag = "<think>"
				idx = findStartTag(fullText, tag)
			}
			
			if idx == -1 {
				// Check for partial start tag at the end
				partialIdx := findPartialStartTag(fullText, "<thinking>")
				if partialIdx == -1 {
					partialIdx = findPartialStartTag(fullText, "<think>")
				}
				
				if partialIdx != -1 {
					content += fullText[:partialIdx]
					p.buffer = fullText[partialIdx:]
					return reasoning, content
				}
				content += fullText
				return reasoning, content
			}

			content += fullText[:idx]
			p.inReasoning = true
			fullText = fullText[idx+len(tag):]
		} else {
			// Handle </think> or </thinking> end tags
			tag := "</thinking>"
			idx := findStartTag(fullText, tag)
			if idx == -1 {
				tag = "</think>"
				idx = findStartTag(fullText, tag)
			}
			
			if idx == -1 {
				// Check for partial end tag at the end
				partialIdx := findPartialStartTag(fullText, "</thinking>")
				if partialIdx == -1 {
					partialIdx = findPartialStartTag(fullText, "</think>")
				}
				
				if partialIdx != -1 {
					reasoning += fullText[:partialIdx]
					p.buffer = fullText[partialIdx:]
					return reasoning, content
				}
				reasoning += fullText
				return reasoning, content
			}

			reasoning += fullText[:idx]
			p.inReasoning = false
			fullText = fullText[idx+len(tag):]
		}
	}

	return reasoning, content
}

func findStartTag(text, tag string) int {
	for i := 0; i <= len(text)-len(tag); i++ {
		if text[i:i+len(tag)] == tag {
			return i
		}
	}
	return -1
}

func findPartialStartTag(text, tag string) int {
	for i := 1; i < len(tag); i++ {
		suffixLen := i
		if len(text) < suffixLen {
			suffixLen = len(text)
		}
		if text[len(text)-suffixLen:] == tag[:suffixLen] {
			return len(text) - suffixLen
		}
	}
	return -1
}
