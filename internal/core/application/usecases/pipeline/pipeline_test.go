package pipeline

import (
	"context"
	"errors"
	"testing"
)

type mockStage struct {
	name string
	fail bool
}

func (m *mockStage) Name() string {
	return m.name
}

func (m *mockStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	if m.fail {
		return StageOutput{}, errors.New("stage failed")
	}
	return StageOutput{Data: input.Data}, nil
}

func TestIngestionPipeline(t *testing.T) {
	stages := []Stage{
		&mockStage{name: "Stage1", fail: false},
		&mockStage{name: "Stage2", fail: false},
	}
	pipeline := NewIngestionPipeline(stages, 1)
	ctx := context.Background()

	progressChan := pipeline.Ingest(ctx, StageInput{Data: "initial"})

	var progress []Progress
	for p := range progressChan {
		progress = append(progress, p)
	}

	if len(progress) != 4 {
		t.Fatalf("expected 4 progress updates, got %d", len(progress))
	}
}

func TestIngestionPipeline_Failure(t *testing.T) {
	stages := []Stage{
		&mockStage{name: "Stage1", fail: false},
		&mockStage{name: "Stage2", fail: true},
	}
	pipeline := NewIngestionPipeline(stages, 1)
	ctx := context.Background()

	progressChan := pipeline.Ingest(ctx, StageInput{Data: "initial"})

	var progress []Progress
	for p := range progressChan {
		progress = append(progress, p)
	}

	if len(progress) != 4 {
		t.Fatalf("expected 4 progress updates, got %d", len(progress))
	}
	if progress[3].Status != StatusFailed {
		t.Errorf("expected failed status at index 3, got %s", progress[3].Status)
	}
}
