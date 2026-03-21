package unit_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/notebook"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/persistence"
)

func TestNotebookUseCase_Create(t *testing.T) {
	// Setup
	repo := persistence.NewInMemoryNotebookRepository()
	useCase := notebook.NewNotebookUseCase(repo)

	req := &dtos.CreateNotebookRequest{
		Title:       "Test Notebook",
		Description: "Test Description",
		Content:     "Test Content",
		Tags:        []string{"test", "unit"},
	}

	// Execute
	notebook, err := useCase.Create(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("failed to create notebook: %v", err)
	}
	if notebook == nil {
		t.Fatal("expected notebook to be returned")
	}
	if notebook.Title != req.Title {
		t.Errorf("expected title %s, got %s", req.Title, notebook.Title)
	}
}

func TestNotebookUseCase_GetByID(t *testing.T) {
	// Setup
	repo := persistence.NewInMemoryNotebookRepository()
	useCase := notebook.NewNotebookUseCase(repo)

	createReq := &dtos.CreateNotebookRequest{
		Title:       "Test Notebook",
		Description: "Test Description",
		Content:     "Test Content",
		Tags:        []string{"test"},
	}

	created, err := useCase.Create(context.Background(), createReq)
	if err != nil {
		t.Fatalf("failed to create notebook: %v", err)
	}

	// Execute
	found, err := useCase.GetByID(context.Background(), created.ID.String())

	// Assert
	if err != nil {
		t.Fatalf("failed to get notebook: %v", err)
	}
	if found == nil {
		t.Fatal("expected notebook to be found")
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, found.ID)
	}
}

func TestNotebookUseCase_List(t *testing.T) {
	// Setup
	repo := persistence.NewInMemoryNotebookRepository()
	useCase := notebook.NewNotebookUseCase(repo)

	// Create some notebooks
	for i := 1; i <= 3; i++ {
		req := &dtos.CreateNotebookRequest{
			Title:       fmt.Sprintf("Notebook %d", i),
			Description: "Test Description",
			Content:     "Test Content",
		}
		_, _ = useCase.Create(context.Background(), req)
	}

	// Execute
	listReq := &dtos.ListNotebooksRequest{
		Page:  1,
		Limit: 10,
	}

	result, err := useCase.List(context.Background(), listReq)

	// Assert
	if err != nil {
		t.Fatalf("failed to list notebooks: %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be returned")
	}
	if len(result.Notebooks) != 3 {
		t.Errorf("expected 3 notebooks, got %d", len(result.Notebooks))
	}
}
