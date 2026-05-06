package sse

import (
	"encoding/json"
	"fmt"
	"io"
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
	var accumulatedText strings.Builder

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
			ID:     meta.ResponseID,
			Status: "in_progress",
		},
	})
	if err != nil {
		return err
	}
	seqNum++

	// 3. response.output_item.added
	err = f.sendEvent(w, &dtos.ResponseOutputItemAddedEvent{
		Type:           "response.output_item.added",
		SequenceNumber: seqNum,
		OutputIndex:    0,
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

	// 4. response.content_part.added
	err = f.sendEvent(w, &dtos.ResponseContentPartAddedEvent{
		Type:           "response.content_part.added",
		SequenceNumber: seqNum,
		ItemID:         messageID,
		OutputIndex:    0,
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

	// 5. Loop response.output_text.delta
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			f.sendFailed(w, seqNum, meta.ResponseID, err)
			return err
		}

		accumulatedText.WriteString(chunk.Content)

		err = f.sendEvent(w, &dtos.ResponseOutputTextDeltaEvent{
			Type:           "response.output_text.delta",
			SequenceNumber: seqNum,
			ItemID:         messageID,
			OutputIndex:    0,
			ContentIndex:   0,
			Delta:          chunk.Content,
		})
		if err != nil {
			return err
		}
		seqNum++
	}

	// 6. response.output_text.done
	err = f.sendEvent(w, &dtos.ResponseOutputTextDoneEvent{
		Type:           "response.output_text.done",
		SequenceNumber: seqNum,
		ItemID:         messageID,
		OutputIndex:    0,
		ContentIndex:   0,
		Text:           accumulatedText.String(),
	})
	if err != nil {
		return err
	}
	seqNum++

	// 7. response.content_part.done
	err = f.sendEvent(w, &dtos.ResponseContentPartDoneEvent{
		Type:           "response.content_part.done",
		SequenceNumber: seqNum,
		ItemID:         messageID,
		OutputIndex:    0,
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

	// 8. response.output_item.done
	err = f.sendEvent(w, &dtos.ResponseOutputItemDoneEvent{
		Type:           "response.output_item.done",
		SequenceNumber: seqNum,
		OutputIndex:    0,
		Item: &dtos.Message{
			ID:     messageID,
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			Content: []dtos.ContentPart{
				&dtos.OutputTextContent{
					Type:        "output_text",
					Text:        accumulatedText.String(),
					Annotations: []dtos.Annotation{},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	seqNum++

	// 9. response.completed
	completedAt := time.Now().Unix()
	err = f.sendEvent(w, &dtos.ResponseCompletedEvent{
		Type:           "response.completed",
		SequenceNumber: seqNum,
		Response: &dtos.ResponseResource{
			ID:          meta.ResponseID,
			Object:      "response",
			CreatedAt:   meta.CreatedAt,
			CompletedAt: &completedAt,
			Status:      "completed",
			Model:       meta.ModelName,
			Output: []dtos.ItemField{
				&dtos.Message{
					ID:     messageID,
					Type:   "message",
					Status: "completed",
					Role:   "assistant",
					Content: []dtos.ContentPart{
						&dtos.OutputTextContent{
							Type:        "output_text",
							Text:        accumulatedText.String(),
							Annotations: []dtos.Annotation{},
						},
					},
				},
			},
			Tools:             []dtos.Tool{},
			ToolChoice:        dtos.ToolChoiceAuto,
			Truncation:        "disabled",
			ParallelToolCalls: true,
			Text:              &dtos.TextField{Format: &dtos.TextFormatParam{Type: "text"}},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (f *ResponsesAPIFormatter) sendEvent(w io.Writer, event dtos.StreamingEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	return err
}

func (f *ResponsesAPIFormatter) sendFailed(w io.Writer, seqNum int, responseID string, err error) {
	_ = f.sendEvent(w, &dtos.ResponseFailedEvent{
		Type:           "response.failed",
		SequenceNumber: seqNum,
		Response: &dtos.ResponseResource{
			ID:     responseID,
			Status: "failed",
			Error:  &dtos.Error{Code: "internal_error", Message: err.Error()},
		},
	})
}
