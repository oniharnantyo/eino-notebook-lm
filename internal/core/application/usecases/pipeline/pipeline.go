// Package pipeline provides a structured way to execute multiple stages of content ingestion.
// It allows for sequential execution of stages with progress reporting via channels.
package pipeline

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Stage interface represents a single step in the ingestion pipeline.
// Each stage takes a StageInput and returns a StageOutput.
type Stage interface {
	// Name returns the display name of the stage for progress reporting.
	Name() string
	// Execute performs the stage-specific logic.
	Execute(ctx context.Context, input StageInput) (StageOutput, error)
}

// StageInput contains data passed to a stage.
type StageInput struct {
	// SourceID is the unique identifier of the content source being processed.
	SourceID uuid.UUID
	// Data is the stage-specific input data.
	Data interface{}
}

// StageOutput contains data returned by a stage to be passed to the next stage.
type StageOutput struct {
	// Data is the stage-specific output data.
	Data interface{}
}

// Status represents the current state of a pipeline stage.
type Status string

const (
	// StatusInProgress indicates the stage is currently executing.
	StatusInProgress Status = "in_progress"
	// StatusCompleted indicates the stage has finished successfully.
	StatusCompleted Status = "completed"
	// StatusFailed indicates the stage has failed with an error.
	StatusFailed Status = "failed"
)

// Progress report sent via the progress channel during pipeline execution.
type Progress struct {
	// StageName is the name of the stage reported.
	StageName string
	// Status is the current status of the stage.
	Status Status
	// Error is non-nil if the status is StatusFailed.
	Error error
	// Metadata contains additional stage-specific information.
	Metadata map[string]interface{}
}

// IngestionPipeline orchestrates the execution of multiple stages.
//
// Example usage:
//
//	stages := []pipeline.Stage{
//	    pipeline.NewExtractionStage(extractor),
//	    pipeline.NewParsingStage(parser),
//	    pipeline.NewChunkingStage(1000),
//	}
//	p := pipeline.NewIngestionPipeline(stages, 1)
//
//	progressChan := p.Ingest(ctx, pipeline.StageInput{
//	    SourceID: sourceID,
//	    Data: contentSource,
//	})
//
//	for progress := range progressChan {
//	    fmt.Printf("Stage %s: %s\n", progress.StageName, progress.Status)
//	    if progress.Error != nil {
//	        log.Errorf("Error: %v", progress.Error)
//	    }
//	}
type IngestionPipeline struct {
	stages           []Stage
	parallelismLevel int
}

// NewIngestionPipeline creates a new IngestionPipeline with the given stages and parallelism level.
// parallelismLevel is currently used for internal stage optimizations (e.g. batch embedding).
func NewIngestionPipeline(stages []Stage, parallelismLevel int) *IngestionPipeline {
	return &IngestionPipeline{
		stages:           stages,
		parallelismLevel: parallelismLevel,
	}
}

// Ingest runs the pipeline stages sequentially.
// It returns a read-only channel that receives Progress updates.
// The channel is closed when the pipeline finishes or fails.
func (p *IngestionPipeline) Ingest(ctx context.Context, initialInput StageInput) <-chan Progress {
	progressChan := make(chan Progress)

	go func() {
		defer close(progressChan)

		currentInput := initialInput
		for _, stage := range p.stages {
			select {
			case <-ctx.Done():
				progressChan <- Progress{StageName: stage.Name(), Status: StatusFailed, Error: ctx.Err()}
				return
			default:
			}

			progressChan <- Progress{StageName: stage.Name(), Status: StatusInProgress}

			output, err := stage.Execute(ctx, currentInput)
			if err != nil {
				progressChan <- Progress{StageName: stage.Name(), Status: StatusFailed, Error: err}
				return
			}

			progressChan <- Progress{StageName: stage.Name(), Status: StatusCompleted}
			currentInput = StageInput{
				SourceID: initialInput.SourceID,
				Data:     output.Data,
			}
		}
	}()

	return progressChan
}
