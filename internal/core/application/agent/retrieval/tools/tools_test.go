/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockDB is a mock implementation of the DB interface from pgvector.
type mockDB struct {
	mock.Mock
}

func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	mockArgs := m.Called(ctx, sql, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).(pgx.Rows), mockArgs.Error(1)
}

// mockRows is a mock implementation of pgx.Rows.
type mockRows struct {
	mock.Mock
}

func (m *mockRows) Close()                                   { m.Called() }
func (m *mockRows) Err() error                               { return m.Called().Error(0) }
func (m *mockRows) CommandTag() pgconn.CommandTag             { return m.Called().Get(0).(pgconn.CommandTag) }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return m.Called().Get(0).([]pgconn.FieldDescription) }
func (m *mockRows) Next() bool                               { return m.Called().Bool(0) }
func (m *mockRows) Scan(dest ...any) error                   { return m.Called(dest...).Error(0) }
func (m *mockRows) Values() ([]any, error)                   { args := m.Called(); return args.Get(0).([]any), args.Error(1) }
func (m *mockRows) RawValues() [][]byte                      { return m.Called().Get(0).([][]byte) }
func (m *mockRows) Conn() *pgx.Conn                          { return m.Called().Get(0).(*pgx.Conn) }

// mockEmbedder is a mock implementation of embedding.Embedder.
type mockEmbedder struct {
	mock.Mock
}

func (m *mockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	args := m.Called(ctx, texts, opts)
	return args.Get(0).([][]float64), args.Error(1)
}

// mockKnowledgeRepo is a mock implementation of repositories.KnowledgeRepository.
type mockKnowledgeRepo struct {
	mock.Mock
}

func (m *mockKnowledgeRepo) Save(ctx context.Context, k *entities.Knowledge) error {
	return m.Called(ctx, k).Error(0)
}

func (m *mockKnowledgeRepo) SaveBatch(ctx context.Context, ks []*entities.Knowledge) error {
	return m.Called(ctx, ks).Error(0)
}

func (m *mockKnowledgeRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.Knowledge, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entities.Knowledge), args.Error(1)
}

func (m *mockKnowledgeRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Knowledge, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Knowledge), args.Error(1)
}

func (m *mockKnowledgeRepo) GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Knowledge, error) {
	args := m.Called(ctx, sourceID)
	return args.Get(0).([]*entities.Knowledge), args.Error(1)
}

func (m *mockKnowledgeRepo) FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*entities.Knowledge), args.Error(1)
}

func (m *mockKnowledgeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockKnowledgeRepo) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	return m.Called(ctx, sourceID).Error(0)
}

func (m *mockKnowledgeRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockKnowledgeRepo) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockKnowledgeRepo) CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error) {
	args := m.Called(ctx, sourceID)
	return args.Int(0), args.Error(1)
}

func TestSemanticSearchTool(t *testing.T) {
	ctx := context.Background()
	mDB := new(mockDB)
	mRows := new(mockRows)
	mEmbedder := new(mockEmbedder)

	r, _ := pgvector.NewUnifiedRetriever(&pgvector.UnifiedConfig{
		Pool:      mDB,
		Dimension: 3,
	})

	st := NewSemanticSearchTool(r, mEmbedder, []string{"source1"})

	t.Run("successful semantic search", func(t *testing.T) {
		input := &SemanticSearchInput{Query: "test", TopK: 1}
		mEmbedder.On("EmbedStrings", ctx, []string{"test"}, mock.Anything).Return([][]float64{{0.1, 0.2, 0.3}}, nil).Once()

		mDB.On("Query", ctx, mock.MatchedBy(func(sql string) bool {
			return strings.Contains(sql, "SELECT") && strings.Contains(sql, "sentences")
		}), mock.Anything).Return(mRows, nil).Once()

		mRows.On("Next").Return(true).Once()
		mRows.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			s1 := "chunk1"
			*(args[0].(**string)) = &s1
			*(args[1].(*float64)) = 0.9 // similarity
			s2 := "snippet1"
			*(args[2].(**string)) = &s2
			s3 := "source1"
			*(args[3].(**string)) = &s3
		}).Return(nil).Once()
		mRows.On("Next").Return(false).Once()
		mRows.On("Close").Return().Once()
		mRows.On("Err").Return(nil).Once()

		inputJSON, _ := json.Marshal(input)
		resp, err := st.(tool.InvokableTool).InvokableRun(ctx, string(inputJSON))
		assert.NoError(t, err)

		var output SemanticSearchOutput
		err = json.Unmarshal([]byte(resp), &output)
		assert.NoError(t, err)
		assert.Len(t, output.Results, 1)
		assert.Equal(t, "chunk1", output.Results[0].ChunkID)
		assert.Equal(t, "snippet1", output.Results[0].Snippet)
		assert.Equal(t, 0.9, output.Results[0].Score)
	})
}

func TestKeywordSearchTool(t *testing.T) {
	ctx := context.Background()
	mDB := new(mockDB)
	mRows := new(mockRows)
	mRepo := new(mockKnowledgeRepo)

	r, _ := pgvector.NewUnifiedRetriever(&pgvector.UnifiedConfig{
		Pool:      mDB,
		Dimension: 3,
	})

	kt := NewKeywordSearchTool(r, mRepo, []string{"source1"})

	t.Run("successful keyword search with snippets", func(t *testing.T) {
		input := &KeywordSearchInput{Keywords: []string{"test"}, TopK: 1}
		
		mDB.On("Query", ctx, mock.MatchedBy(func(sql string) bool {
			return strings.Contains(sql, "SELECT") && strings.Contains(sql, "knowledges")
		}), mock.Anything).Return(mRows, nil).Once()

		id := uuid.New()
		mRows.On("Next").Return(true).Once()
		mRows.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			*(args[0].(*string)) = id.String()
			*(args[1].(*string)) = "some content"
			*(args[2].(*[]byte)) = nil
			*(args[3].(*float64)) = 0.5
		}).Return(nil).Once()
		mRows.On("Next").Return(false).Once()
		mRows.On("Close").Return().Once()
		mRows.On("Err").Return(nil).Once()

		mRepo.On("FindByIDs", ctx, []uuid.UUID{id}).Return([]*entities.Knowledge{
			{ID: id, Content: "This is a test content for snippet generation."},
		}, nil).Once()

		inputJSON, _ := json.Marshal(input)
		resp, err := kt.(tool.InvokableTool).InvokableRun(ctx, string(inputJSON))
		assert.NoError(t, err)

		var output KeywordSearchOutput
		err = json.Unmarshal([]byte(resp), &output)
		assert.NoError(t, err)
		assert.Len(t, output.Results, 1)
		assert.Equal(t, id.String(), output.Results[0].ChunkID)
		assert.NotEmpty(t, output.Results[0].Snippets)
		assert.Contains(t, output.Results[0].Snippets[0], "test")
	})
}

func TestImageSearchTool(t *testing.T) {
	ctx := context.Background()
	mDB := new(mockDB)
	mRows := new(mockRows)
	mEmbedder := new(mockEmbedder)

	r, _ := pgvector.NewUnifiedRetriever(&pgvector.UnifiedConfig{
		Pool:      mDB,
		Dimension: 3,
	})

	it := NewImageSearchTool(r, mEmbedder, []string{"source1"})

	t.Run("successful image search", func(t *testing.T) {
		input := &ImageSearchInput{Query: "cat", Limit: 1}
		mEmbedder.On("EmbedStrings", ctx, []string{"cat"}, mock.Anything).Return([][]float64{{0.1, 0.2, 0.3}}, nil).Once()

		mDB.On("Query", ctx, mock.MatchedBy(func(sql string) bool {
			return strings.Contains(sql, "SELECT") && strings.Contains(sql, "images")
		}), mock.Anything).Return(mRows, nil).Once()

		mRows.On("Next").Return(true).Once()
		mRows.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			*(args[0].(*string)) = "img1"
			*(args[1].(*string)) = "a cute cat"
			*(args[2].(*[]byte)) = []byte(`{"s3_key": "path/to/cat.jpg", "description": "a cute cat", "page_number": 1}`)
			*(args[3].(*float64)) = 0.1
		}).Return(nil).Once()
		mRows.On("Next").Return(false).Once()
		mRows.On("Close").Return().Once()
		mRows.On("Err").Return(nil).Once()

		inputJSON, _ := json.Marshal(input)
		resp, err := it.(tool.InvokableTool).InvokableRun(ctx, string(inputJSON))
		assert.NoError(t, err)

		var output ImageSearchOutput
		err = json.Unmarshal([]byte(resp), &output)
		assert.NoError(t, err)
		assert.Len(t, output.Results, 1)
		assert.Equal(t, "path/to/cat.jpg", output.Results[0].S3Key)
		assert.Equal(t, 1, output.Results[0].PageNumber)
		assert.Equal(t, 0.9, output.Results[0].Score)
	})
}
