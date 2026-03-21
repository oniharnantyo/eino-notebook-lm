package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	geminiembedder "github.com/cloudwego/eino-ext/components/embedding/gemini"
	geminimodel "github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/callbacks"
	langfusecallback "github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/genai"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/chat"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/conversation"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/knowledge"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/notebook"
	responseusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/source"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/persistence"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/handlers"
	httproutes "github.com/oniharnantyo/eino-notebook/internal/interfaces/http/routes"
	"github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	pgvectoretriever "github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
)

var (
	servePort        int
	serveHost        string
	langfuseFlusher func()
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long: `Start the Eino Notebook HTTP server for managing notebook operations.
The server can be configured with custom host and port settings.`,
	Example: `  eino-notebook serve
  eino-notebook serve --port 9090
  eino-notebook serve --host 0.0.0.0 --port 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Validate configuration to ensure required fields are set
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}

		// Override with command-line flags if provided
		if cmd.Flags().Changed("host") {
			cfg.Server.Host = serveHost
		}
		if cmd.Flags().Changed("port") {
			cfg.Server.Port = servePort
		}

		addr := cfg.Server.GetServerAddress()

		// Initialize logger
		log := logger.New(logger.LogLevel(cfg.Log.Level), cfg.Log.Format)
		log.Info("Starting Eino Notebook server...",
			"address", addr,
			"log_level", cfg.Log.Level,
		)

		// Initialize Langfuse callback handler for observability
		if cfg.Langfuse.Enabled {
			langfuseHandler, flusher := langfusecallback.NewLangfuseHandler(&langfusecallback.Config{
				Host:             cfg.Langfuse.Host,
				PublicKey:        cfg.Langfuse.PublicKey,
				SecretKey:        cfg.Langfuse.SecretKey,
				SampleRate:       cfg.Langfuse.SampleRate,
				Release:          cfg.Langfuse.Release,
				Threads:          2,
				Timeout:          30 * time.Second,
				FlushAt:          15,
				FlushInterval:    500 * time.Millisecond,
				MaxTaskQueueSize: 100,
				MaxRetry:         3,
			})

			callbacks.AppendGlobalHandlers(langfuseHandler)
			langfuseFlusher = flusher

			log.Info("initialized", "langfuse", "enabled",
				"host", cfg.Langfuse.Host,
				"sample_rate", cfg.Langfuse.SampleRate)
		}

		// Initialize dependencies (Hexagonal Architecture - Dependency Injection)
		// Infrastructure Layer

		// Create database connection pool
		dbPool, err := pgxpool.New(ctx, cfg.Database.GetDSN())
		if err != nil {
			return fmt.Errorf("failed to create database pool: %w", err)
		}
		defer dbPool.Close()
		log.Info("initialized", "db_pool", "pgxpool")

		// Create pgvector indexer with default configuration
		// The pgvector package has built-in defaults for all fields
		pgvectorConfig := &pgvector.Config{
			Pool:                   dbPool,
			Dimension:              cfg.Gemini.Dimension,
			ReferenceIDColumn:      "source_id",
			AutoCreateTable:        false,
			DropBeforeCreate:       false,
			AutoCreateExtension:    false,
			CreateIndexIfNotExists: false,
		}
		vectorIndexer, err := pgvector.NewIndexer(ctx, pgvectorConfig)
		if err != nil {
			return fmt.Errorf("failed to create pgvector indexer: %w", err)
		}
		log.Info("initialized", "indexer", "pgvector",
			"dimension", cfg.Gemini.Dimension)

		notebookRepo := persistence.NewPostgresNotebookRepository(dbPool)
		log.Info("initialized", "repository", "PostgresNotebookRepository")

		knowledgeRepo := persistence.NewPostgresKnowledgeRepository(dbPool)
		log.Info("initialized", "repository", "PostgresKnowledgeRepository")

		sourceRepo := persistence.NewPostgresSourceRepository(dbPool)
		log.Info("initialized", "repository", "PostgresSourceRepository")

		conversationRepo := persistence.NewPostgresConversationRepository(dbPool)
		log.Info("initialized", "repository", "PostgresConversationRepository")

		// Create Gemini embedder for embeddings
		var geminiEmbedder *geminiembedder.Embedder
		var genaiClient *genai.Client
		if cfg.Gemini.APIKey != "" {
			// Create genai client
			genaiClient, err = genai.NewClient(ctx, &genai.ClientConfig{
				APIKey: cfg.Gemini.APIKey,
			})
			if err != nil {
				log.Warn("failed to create Gemini client", "error", err)
			} else {
				// Convert dimension to int32 for OutputDimensionality
				var outputDim *int32
				if cfg.Gemini.Dimension > 0 {
					dim := int32(cfg.Gemini.Dimension)
					outputDim = &dim
				}

				geminiEmbedder, err = geminiembedder.NewEmbedder(ctx, &geminiembedder.EmbeddingConfig{
					Client:               genaiClient,
					Model:                cfg.Gemini.EmbeddingModel,
					OutputDimensionality: outputDim,
				})
				if err != nil {
					log.Warn("failed to initialize Gemini embedder", "error", err)
				} else {
					log.Info("initialized", "embedder", "Gemini", "model", cfg.Gemini.EmbeddingModel, "dimension", cfg.Gemini.Dimension)
				}
			}
		}

		// Application Layer (Use Cases)
		notebookUseCase := notebook.NewNotebookUseCase(notebookRepo)
		log.Info("initialized", "usecase", "NotebookUseCase")

		// Create markdown document transformer
		docTransformer, err := markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
			Headers: map[string]string{
				"#":   "h1",
				"##":  "h2",
				"###": "h3",
			},
			TrimHeaders: false,
			IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
				return fmt.Sprintf("%s-chunk-%d", originalID, splitIndex)
			},
		})
		if err != nil {
			log.Warn("failed to create markdown transformer", "error", err)
			docTransformer = nil
		} else {
			log.Info("initialized", "transformer", "markdown-header-splitter")
		}

		knowledgeUseCase := knowledge.NewKnowledgeUseCase(knowledgeRepo, sourceRepo, vectorIndexer, geminiEmbedder, docTransformer)
		log.Info("initialized", "usecase", "KnowledgeUseCase")

		sourceUseCase := source.NewSourceUseCase(sourceRepo, notebookRepo)
		log.Info("initialized", "usecase", "SourceUseCase")

		conversationUseCase := conversation.NewConversationUseCase(conversationRepo)
		log.Info("initialized", "usecase", "ConversationUseCase")

		// Create pgvector retriever for RAG
		pgvectorRetriever, err := pgvectoretriever.NewRetriever(ctx, &pgvectoretriever.Config{
			Pool:                    dbPool,
			Dimension:               cfg.Gemini.Dimension,
			ReferenceIDColumn:       "source_id",
			AutoCreateBM25Extension: true,
			AutoCreateBM25Index:     true,
		})
		if err != nil {
			log.Warn("failed to create pgvector retriever", "error", err)
		} else {
			log.Info("initialized", "retriever", "pgvector")
		}

		// Create Gemini chat model
		var geminiChatModel *geminimodel.ChatModel
		if cfg.Gemini.APIKey != "" && genaiClient != nil {
			geminiChatModel, err = geminimodel.NewChatModel(ctx, &geminimodel.Config{
				Client: genaiClient,
				Model:  cfg.Gemini.ChatModel,
			})
			if err != nil {
				log.Warn("failed to initialize Gemini chat model", "error", err)
			} else {
				log.Info("initialized", "chat_model", "Gemini", "model", cfg.Gemini.ChatModel)
			}
		}

		// Create response use case with history management configuration
		var responseUseCase chat.ResponseUseCase
		if pgvectorRetriever != nil && geminiChatModel != nil && geminiEmbedder != nil {
			// Configure conversation history management
			historyConfig := &responseusecase.HistoryConfig{
				Strategy:             responseusecase.HistoryStrategySlidingWindow,
				MaxMessages:          10,   // Keep last 10 messages
				MaxTokens:            4000, // Max ~4000 tokens for history
				TokenEstimationRatio: 4,    // 1 token ≈ 4 chars
				SummarizeThreshold:   5,    // Summarize messages older than 5 turns
			}
			responseUseCase = responseusecase.NewResponseUseCase(notebookRepo, conversationRepo, pgvectorRetriever, geminiEmbedder, geminiChatModel, cfg.Gemini.ChatModel, historyConfig)
			log.Info("initialized", "usecase", "ResponseUseCase", "history_strategy", historyConfig.Strategy, "max_messages", historyConfig.MaxMessages)
		}

		// Initialize Kreuzberg document parser
		kreuzbergConfig := &kreuzberg.Config{
			ServiceURL:   cfg.Kreuzberg.ServiceURL,
			OutputFormat: cfg.Kreuzberg.OutputFormat,
			Timeout:      cfg.Kreuzberg.Timeout,
		}
		if cfg.Kreuzberg.OCR != nil {
			kreuzbergConfig.ExtractConfig = &kreuzberg.ExtractConfig{
				OCR: &kreuzberg.OCRConfig{
					Language: cfg.Kreuzberg.OCR.Language,
					Model:    cfg.Kreuzberg.OCR.Model,
				},
			}
		}

		// Create raw Kreuzberg parser for file extractor
		rawKreuzbergParser, err := kreuzberg.NewKreuzbergParser(context.Background(), kreuzbergConfig)
		if err != nil {
			log.Error("failed to initialize raw Kreuzberg parser", "error", err)
			panic("failed to initialize raw Kreuzberg parser: " + err.Error())
		}

		kreuzbergDocParser, err := document.NewKreuzbergDocumentParser(kreuzbergConfig)
		if err != nil {
			log.Error("failed to initialize Kreuzberg document parser", "error", err)
			panic("failed to initialize Kreuzberg document parser: " + err.Error())
		}
		log.Info("initialized", "parser", "KreuzbergDocumentParser", "service_url", cfg.Kreuzberg.ServiceURL)

		// Create document parser factory
		docParserFactory := document.NewDocumentParserFactory(kreuzbergDocParser)
		log.Info("initialized", "factory", "DocumentParserFactory")

		// Initialize content extractors following SOLID principles
		// Strategy Pattern: Different extractors for different content types
		fileExtractor := extractor.NewFileContentExtractor(rawKreuzbergParser, 100<<20)
		urlExtractor := extractor.NewURLContentExtractor(30 * time.Second)
		textExtractor := extractor.NewTextContentExtractor(1 << 20)

		// Factory Pattern: Create appropriate extractor based on content type
		contentExtractorFactory := extractor.NewContentExtractorFactory(
			fileExtractor,
			urlExtractor,
			textExtractor,
		)
		log.Info("initialized", "factory", "ContentExtractorFactory")

		// Interface Layer (HTTP Handlers)
		notebookHandler := handlers.NewNotebookHandler(notebookUseCase, log)
		sourceHandler := handlers.NewSourceHandler(sourceUseCase, log)
		knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeUseCase, sourceUseCase, notebookRepo, contentExtractorFactory, docParserFactory, log)
		var responseHandler *handlers.ResponseHandler
		if responseUseCase != nil {
			responseHandler = handlers.NewResponseHandler(responseUseCase, log)
			log.Info("initialized", "handler", "ResponseHandler")
		}

		conversationHandler := handlers.NewConversationHandler(conversationUseCase, log)
		log.Info("initialized", "handler", "ConversationHandler")

		// Setup routes
		router := mux.NewRouter()
		httproutes.Setup(router, notebookHandler, knowledgeHandler, sourceHandler, responseHandler, conversationHandler)
		log.Info("initialized", "router", "gorilla/mux")

		// Create HTTP server
		srv := &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  cfg.Server.IdleTimeout,
		}

		// Start server in goroutine
		go func() {
			log.Info("server listening", "address", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("server error", "error", err)
			}
		}()

		// Wait for interrupt signal
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Info("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Flush Langfuse events before shutdown
		if langfuseFlusher != nil {
			log.Info("flushing Langfuse events...")
			langfuseFlusher()
			log.Info("Langfuse events flushed")
		}

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("server forced to shutdown", "error", err)
			return err
		}

		log.Info("server stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Command-specific flags
	serveCmd.Flags().StringVarP(&serveHost, "host", "H", "localhost", "host to bind to")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")

	// Bind flags to viper (uppercase for .env compatibility)
	viper.BindPFlag("SERVER_HOST", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("SERVER_PORT", serveCmd.Flags().Lookup("port"))
}
