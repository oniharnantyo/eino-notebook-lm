package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

func BuildCatalog(ctx context.Context, repo repositories.SourceRepository, sourceIDs []uuid.UUID) (string, error) {
	if len(sourceIDs) == 0 {
		return "No sources available.", nil
	}

	sources, err := repo.ListSourceSummariesByID(ctx, sourceIDs)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sources for catalog: %w", err)
	}

	if len(sources) == 0 {
		return "No sources available.", nil
	}

	var sb strings.Builder
	sb.WriteString("Available Sources:\n")
	for _, s := range sources {
		sb.WriteString(fmt.Sprintf("- [%s] ID: %s, Title: %s\n", s.Status, s.ID.String(), s.Title))
	}
	return sb.String(), nil
}
