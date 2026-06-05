package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	stduuid "github.com/google/uuid"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
)

type ResponsesAPIFormatter struct{}

func NewResponsesAPIFormatter() *ResponsesAPIFormatter {
	return &ResponsesAPIFormatter{}
}

func (f *ResponsesAPIFormatter) WriteResponse(w io.Writer, stream *schema.StreamReader[*schema.Message], meta *StreamMeta) error {
	defer stream.Close()

	seqNum := 0
	messageID := fmt.Sprintf("msg_%s", stduuid.New().String())
	reasoningID := fmt.Sprintf("reason_%s", stduuid.New().String())

	var accumulatedText strings.Builder
	var accumulatedReasoning strings.Builder
	toolCallIDs := make(map[string]string)
	toolCallNames := make(map[string]string)
	accumulatedArguments := make(map[string]*strings.Builder)

	var lastUsage *dtos.Usage

	// Track state for different item types
	reasoningItemAdded := false
	textPartAdded := false
	reasoningPartAdded := false
	toolCallItemsAdded := make(map[string]bool)

	outputIndex := 0

	// 1. response.created

	// Define defaults
	falseVal := false
	defaultServiceTier := "default"
	zeroFloat := 0.0
	zeroInt := 0
	oneFloat := 1.0

	// 1. response.created
	err := f.sendEvent(w, &dtos.ResponseCreatedEvent{
		Type:           "response.created",
		SequenceNumber: seqNum,
		Response: &dtos.ResponseResource{
			ID:                meta.ResponseID,
			Object:            "response",
			CreatedAt:         meta.CreatedAt,
			Status:            "in_progress",
			Model:             meta.ModelName,
			Output:            []dtos.ItemField{},
			Tools:             []dtos.Tool{},
			ToolChoice:        dtos.ToolChoiceAuto,
			Truncation:        "disabled",
			ParallelToolCalls: true,
			Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
			Metadata:          meta.Metadata,
			Instructions:      meta.Instructions,
			MaxOutputTokens:   meta.MaxOutputTokens,
			Temperature:       meta.Temperature,
			MaxToolCalls:      meta.MaxToolCalls,
			PreviousResponseID: meta.PreviousResponseID,
			ConversationID:     meta.ConversationID,
			Background:         falseVal,
			Store:              &falseVal,
			ServiceTier:        &defaultServiceTier,
			FrequencyPenalty:   &zeroFloat,
			PresencePenalty:    &zeroFloat,
			TopLogprobs:        &zeroInt,
			TopP:               &oneFloat,
		},
	})
	if err != nil {
		return err
	}
	seqNum++

	// 2. response.in_progress
	err = f.sendEvent(w, &dtos.ResponseInProgressEvent{
		Type:           "response.in_progress",
		SequenceNumber: seqNum,
		Response: &dtos.ResponseResource{
			ID:                meta.ResponseID,
			Object:            "response",
			CreatedAt:         meta.CreatedAt,
			Status:            "in_progress",
			Model:             meta.ModelName,
			Output:            []dtos.ItemField{},
			Tools:             []dtos.Tool{},
			ToolChoice:        dtos.ToolChoiceAuto,
			Truncation:        "disabled",
			ParallelToolCalls: true,
			Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
			Metadata:          meta.Metadata,
			Instructions:      meta.Instructions,
			MaxOutputTokens:   meta.MaxOutputTokens,
			Temperature:       meta.Temperature,
			MaxToolCalls:      meta.MaxToolCalls,
			PreviousResponseID: meta.PreviousResponseID,
			ConversationID:     meta.ConversationID,
			Background:         falseVal,
			Store:              &falseVal,
			ServiceTier:        &defaultServiceTier,
			FrequencyPenalty:   &zeroFloat,
			PresencePenalty:    &zeroFloat,
			TopLogprobs:        &zeroInt,
			TopP:               &oneFloat,
		},
	})
	if err != nil {
		return err
	}
	seqNum++

	// 4. Loop through chunks
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			f.sendFailed(w, seqNum, meta.ResponseID, err)
			return err
		}

		// Capture usage if present
		if chunk.ResponseMeta != nil && chunk.ResponseMeta.Usage != nil {
			lastUsage = &dtos.Usage{
				InputTokens:  chunk.ResponseMeta.Usage.PromptTokens,
				OutputTokens: chunk.ResponseMeta.Usage.CompletionTokens,
				TotalTokens:  chunk.ResponseMeta.Usage.TotalTokens,
				InputTokensDetails: &dtos.InputTokensDetails{
					CachedTokens: chunk.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
				},
				OutputTokensDetails: &dtos.OutputTokensDetails{
					ReasoningTokens: chunk.ResponseMeta.Usage.CompletionTokensDetails.ReasoningTokens,
				},
			}
		}

		// Handle reasoning content
		if chunk.ReasoningContent != "" {
			// Add reasoning item if not already added
			if !reasoningItemAdded {
				err = f.sendEvent(w, &dtos.ResponseOutputItemAddedEvent{
					Type:           "response.output_item.added",
					SequenceNumber: seqNum,
					OutputIndex:    outputIndex,
					Item: &dtos.ReasoningItem{
						ID:     reasoningID,
						Type:   "reasoning",
						Status: "in_progress",
					},
				})
				if err != nil {
					return err
				}
				seqNum++
				reasoningItemAdded = true
			}

			// Add reasoning summary part if not already added
			if !reasoningPartAdded {
				err = f.sendEvent(w, &dtos.ReasoningSummaryPartAddedEvent{
					Type:           "response.reasoning_summary_part.added",
					SequenceNumber: seqNum,
					ItemID:         reasoningID,
					OutputIndex:    outputIndex,
					SummaryIndex:   0,
					Part: dtos.ReasoningSummary{
						Type: "summary_text",
						Text: "",
					},
				})
				if err != nil {
					return err
				}
				seqNum++
				reasoningPartAdded = true
			}

			// Send reasoning summary text delta
			accumulatedReasoning.WriteString(chunk.ReasoningContent)
			err = f.sendEvent(w, &dtos.ReasoningSummaryTextDeltaEvent{
				Type:           "response.reasoning_summary_text.delta",
				SequenceNumber: seqNum,
				ItemID:         reasoningID,
				OutputIndex:    outputIndex,
				SummaryIndex:   0,
				Delta:          chunk.ReasoningContent,
			})
			if err != nil {
				return err
			}
			seqNum++
		}

		// Handle tool calls
		if len(chunk.ToolCalls) > 0 {
			// Finalize reasoning item if it was added
			if reasoningItemAdded {
				err = f.finalizeReasoningItem(w, &seqNum, reasoningID, accumulatedReasoning.String(), outputIndex)
				if err != nil {
					return err
				}
				reasoningItemAdded = false
				reasoningPartAdded = false
				outputIndex++ // Distinct index for the next item
			}

			for _, tc := range chunk.ToolCalls {
				// Generate or retrieve tool call ID
				toolCallID := tc.ID
				if toolCallID == "" {
					// Check if we've already assigned an ID for this function call
					if existingID, exists := toolCallIDs[tc.Function.Name]; exists {
						toolCallID = existingID
					} else {
						toolCallID = fmt.Sprintf("call_%s", stduuid.New().String())
						toolCallIDs[tc.Function.Name] = toolCallID
						toolCallNames[toolCallID] = tc.Function.Name
					}
				}

				// If tool call is just starting (or we haven't seen it yet), add output item
				if !toolCallItemsAdded[toolCallID] {
					err = f.sendEvent(w, &dtos.ResponseOutputItemAddedEvent{
						Type:           "response.output_item.added",
						SequenceNumber: seqNum,
						OutputIndex:    outputIndex,
						Item: &dtos.FunctionCallItem{
							ID:        toolCallID,
							CallID:    toolCallID,
							Type:      "function_call",
							Status:    "in_progress",
							Name:      tc.Function.Name,
							Arguments: "",
						},					})
					if err != nil {
						return err
					}
					seqNum++
					toolCallItemsAdded[toolCallID] = true
					accumulatedArguments[toolCallID] = &strings.Builder{}
					// We don't increment outputIndex yet because deltas for this tool call share this index
				}

				// Send tool call arguments delta
				if tc.Function.Arguments != "" {
					accumulatedArguments[toolCallID].WriteString(tc.Function.Arguments)
					err = f.sendEvent(w, &dtos.ResponseFunctionCallArgumentsDeltaEvent{
						Type:           "response.function_call_arguments.delta",
						SequenceNumber: seqNum,
						ItemID:         toolCallID,
						CallID:         toolCallID,
						OutputIndex:    outputIndex,
						ContentIndex:   0,
						Delta:          tc.Function.Arguments,
					})
					if err != nil {
						return err
					}
					seqNum++
				}
			}
			continue
		}

		// Handle text content
		if chunk.Content != "" {
			// Finalize reasoning item if it was added and we're now getting text
			if reasoningItemAdded {
				err = f.finalizeReasoningItem(w, &seqNum, reasoningID, accumulatedReasoning.String(), outputIndex)
				if err != nil {
					return err
				}
				reasoningItemAdded = false
				reasoningPartAdded = false
				outputIndex++
			}

			// Add message item if not already added
			if !textPartAdded {
				// Add message item
				err = f.sendEvent(w, &dtos.ResponseOutputItemAddedEvent{
					Type:           "response.output_item.added",
					SequenceNumber: seqNum,
					OutputIndex:    outputIndex,
					Item: &dtos.Message{
						ID:      messageID,
						Type:    "message",
						Status:  "in_progress",
						Role:    "assistant",
						Content: []dtos.ContentPart{},
					},
				})
				if err != nil {
					return err
				}
				seqNum++

				// Add text content part
				err = f.sendEvent(w, &dtos.ResponseContentPartAddedEvent{
					Type:           "response.content_part.added",
					SequenceNumber: seqNum,
					ItemID:         messageID,
					OutputIndex:    outputIndex,
					ContentIndex:   0,
					Part: &dtos.OutputTextContent{
						Type:        "output_text",
						Text:        "",
						Annotations: []dtos.Annotation{},
					},
				})
				if err != nil {
					return err
				}
				seqNum++
				textPartAdded = true
			}

			accumulatedText.WriteString(chunk.Content)

			err = f.sendEvent(w, &dtos.ResponseOutputTextDeltaEvent{
				Type:           "response.output_text.delta",
				SequenceNumber: seqNum,
				ItemID:         messageID,
				OutputIndex:    outputIndex,
				ContentIndex:   0,
				Delta:          chunk.Content,
			})
			if err != nil {
				return err
			}
			seqNum++
		}

		// Handle multimodal content (images, audio, video, etc.)
		if len(chunk.AssistantGenMultiContent) > 0 {
			for _, part := range chunk.AssistantGenMultiContent {
				switch part.Type {
				case schema.ChatMessagePartTypeText:
					// Text is already handled via Content field
					continue
				case schema.ChatMessagePartTypeReasoning:
					// Reasoning is handled above via ReasoningContent field
					continue
				case schema.ChatMessagePartTypeImageURL, schema.ChatMessagePartTypeAudioURL, schema.ChatMessagePartTypeVideoURL:
					// Multimodal content - could be extended to send appropriate events
					// For now, we'll pass through as-is
					continue
				}
			}
		}
	}

	// Finalize reasoning item if still open (in case we ended with reasoning only)
	if reasoningItemAdded {
		err = f.finalizeReasoningItem(w, &seqNum, reasoningID, accumulatedReasoning.String(), outputIndex)
		if err != nil {
			return err
		}
		outputIndex++
	}

	// Finalize tool calls
	for name, toolCallID := range toolCallIDs {
		args := ""
		if b, ok := accumulatedArguments[toolCallID]; ok {
			args = b.String()
		}

		err = f.sendEvent(w, &dtos.ResponseFunctionCallArgumentsDoneEvent{
			Type:           "response.function_call_arguments.done",
			SequenceNumber: seqNum,
			ItemID:         toolCallID,
			CallID:         toolCallID,
			OutputIndex:    outputIndex,
			ContentIndex:   0,
			Arguments:      args,
		})
		if err != nil {
			return err
		}
		seqNum++

		err = f.sendEvent(w, &dtos.ResponseOutputItemDoneEvent{
			Type:           "response.output_item.done",
			SequenceNumber: seqNum,
			OutputIndex:    outputIndex,
			Item: &dtos.FunctionCallItem{
				ID:        toolCallID,
				CallID:    toolCallID,
				Type:      "function_call",
				Status:    "completed",
				Name:      name,
				Arguments: args,
			},
		})
		if err != nil {
			return err
		}
		seqNum++
		outputIndex++
	}

	// Finalize text content if it was added
	if textPartAdded {
		err = f.sendEvent(w, &dtos.ResponseOutputTextDoneEvent{
			Type:           "response.output_text.done",
			SequenceNumber: seqNum,
			ItemID:         messageID,
			OutputIndex:    outputIndex,
			ContentIndex:   0,
			Text:           accumulatedText.String(),
		})
		if err != nil {
			return err
		}
		seqNum++

		err = f.sendEvent(w, &dtos.ResponseContentPartDoneEvent{
			Type:           "response.content_part.done",
			SequenceNumber: seqNum,
			ItemID:         messageID,
			OutputIndex:    outputIndex,
			ContentIndex:   0,
			Part: &dtos.OutputTextContent{
				Type:        "output_text",
				Text:        accumulatedText.String(),
				Annotations: []dtos.Annotation{},
			},
		})
		if err != nil {
			return err
		}
		seqNum++

		// Finalize message item
		err = f.sendEvent(w, &dtos.ResponseOutputItemDoneEvent{
			Type:           "response.output_item.done",
			SequenceNumber: seqNum,
			OutputIndex:    outputIndex,
			Item: &dtos.Message{
				ID:      messageID,
				Type:    "message",
				Status:  "completed",
				Role:    "assistant",
				Content: buildContentParts(textPartAdded, accumulatedText.String()),
			},
		})
		if err != nil {
			return err
		}
		seqNum++
		outputIndex++
	}

	// Finalize response
	completedAt := time.Now().Unix()
	err = f.sendEvent(w, &dtos.ResponseCompletedEvent{
		Type:           "response.completed",
		SequenceNumber: seqNum,
		Response: &dtos.ResponseResource{
			ID:                meta.ResponseID,
			Object:            "response",
			CreatedAt:         meta.CreatedAt,
			CompletedAt:       &completedAt,
			Status:            "completed",
			Model:             meta.ModelName,
			Output:            buildOutputItems(messageID, textPartAdded, accumulatedText.String(), reasoningID, reasoningItemAdded, accumulatedReasoning.String(), toolCallIDs, accumulatedArguments),
			Tools:             []dtos.Tool{},
			ToolChoice:        dtos.ToolChoiceAuto,
			Truncation:        "disabled",
			ParallelToolCalls: true,
			Usage:             lastUsage,
			Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
			Metadata:          meta.Metadata,
			Instructions:      meta.Instructions,
			MaxOutputTokens:   meta.MaxOutputTokens,
			Temperature:       meta.Temperature,
			MaxToolCalls:      meta.MaxToolCalls,
			PreviousResponseID: meta.PreviousResponseID,
			ConversationID:     meta.ConversationID,
			Background:         falseVal,
			Store:              &falseVal,
			ServiceTier:        &defaultServiceTier,
			FrequencyPenalty:   &zeroFloat,
			PresencePenalty:    &zeroFloat,
			TopLogprobs:        &zeroInt,
			TopP:               &oneFloat,
		},
	})
	if err != nil {
		return err
	}

	// 5. data: [DONE]
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}

	return nil
}

// finalizeReasoningItem sends the done events for a reasoning item
func (f *ResponsesAPIFormatter) finalizeReasoningItem(w io.Writer, seqNum *int, reasoningID, reasoningText string, outputIndex int) error {
	// Send reasoning summary text done event
	err := f.sendEvent(w, &dtos.ReasoningSummaryTextDoneEvent{
		Type:           "response.reasoning_summary_text.done",
		SequenceNumber: *seqNum,
		ItemID:         reasoningID,
		OutputIndex:    outputIndex,
		SummaryIndex:   0,
		Text:           reasoningText,
	})
	if err != nil {
		return err
	}
	(*seqNum)++

	// Send reasoning summary part done event
	err = f.sendEvent(w, &dtos.ReasoningSummaryPartDoneEvent{
		Type:           "response.reasoning_summary_part.done",
		SequenceNumber: *seqNum,
		ItemID:         reasoningID,
		OutputIndex:    outputIndex,
		SummaryIndex:   0,
		Part: dtos.ReasoningSummary{
			Type: "summary_text",
			Text: reasoningText,
		},
	})
	if err != nil {
		return err
	}
	(*seqNum)++

	// Send reasoning item done event with summary
	err = f.sendEvent(w, &dtos.ResponseOutputItemDoneEvent{
		Type:           "response.output_item.done",
		SequenceNumber: *seqNum,
		OutputIndex:    outputIndex,
		Item: &dtos.ReasoningItem{
			ID:               reasoningID,
			EncryptedContent: nil,
			Type:             "reasoning",
			Status:           "completed",
			Summary: []dtos.ReasoningSummary{
				{
					Type: "summary_text",
					Text: reasoningText,
				},
			},
			Content: []dtos.ReasoningContent{},
		},
	})
	if err != nil {
		return err
	}
	(*seqNum)++

	return nil
}

// sendEvent sends a streaming event and flushes it immediately
func (f *ResponsesAPIFormatter) sendEvent(w io.Writer, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Extract event type for SSE event: line (spec requires event field matching JSON type)
	var eventType string
	switch e := event.(type) {
	case dtos.StreamingEvent:
		eventType = e.GetEventType()
	case map[string]any:
		if t, ok := e["type"].(string); ok {
			eventType = t
		}
	}

	if eventType != "" {
		_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(data))
	} else {
		_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	}
	if err != nil {
		return err
	}
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}
	return nil
}

// sendFailed sends a failure event
func (f *ResponsesAPIFormatter) sendFailed(w io.Writer, seqNum int, responseID string, err error) {
	_ = f.sendEvent(w, map[string]any{
		"type":            "response.failed",
		"sequence_number": seqNum,
		"response": map[string]any{
			"id":     responseID,
			"status": "failed",
			"error": map[string]string{
				"code":    "internal_error",
				"message": err.Error(),
				"type":    "internal_error",
			},
		},
	})
}

// buildContentParts constructs the final content parts array
func buildContentParts(hasText bool, text string) []dtos.ContentPart {
	var parts []dtos.ContentPart

	if hasText {
		parts = append(parts, &dtos.OutputTextContent{
			Type:        "output_text",
			Text:        text,
			Annotations: []dtos.Annotation{},
		})
	}

	return parts
}

// buildOutputItems constructs the final output items array for the response
func buildOutputItems(messageID string, hasText bool, text string, reasoningID string, hasReasoning bool, reasoningText string, toolCallIDs map[string]string, accumulatedArguments map[string]*strings.Builder) []dtos.ItemField {
	var items []dtos.ItemField

	// Add reasoning item first if it exists
	if hasReasoning {
		items = append(items, &dtos.ReasoningItem{
			ID:     reasoningID,
			Type:   "reasoning",
			Status: "completed",
			Summary: []dtos.ReasoningSummary{
				{
					Type: "summary_text",
					Text: reasoningText,
				},
			},
			Content: []dtos.ReasoningContent{
				{
					Type: "output_text",
					Text: reasoningText,
				},
			},
		})
	}

	// Add tool call items
	for name, toolCallID := range toolCallIDs {
		args := ""
		if b, ok := accumulatedArguments[toolCallID]; ok {
			args = b.String()
		}
		items = append(items, &dtos.FunctionCallItem{
			ID:        toolCallID,
			Type:      "function_call",
			Status:    "completed",
			Name:      name,
			Arguments: args,
		})
	}

	// Add message item
	items = append(items, &dtos.Message{
		ID:      messageID,
		Type:    "message",
		Status:  "completed",
		Role:    "assistant",
		Content: buildContentParts(hasText, text),
	})

	return items
}

// truncateForSummary truncates text to a maximum length for summary
func truncateForSummary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
