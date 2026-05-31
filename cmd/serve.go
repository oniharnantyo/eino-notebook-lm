package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	langfusecallback "github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/callbacks"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	agent "github.com/oniharnantyo/eino-notebook/internal/core/application/agent/retrieval"
	retrievalTools "github.com/oniharnantyo/eino-notebook/internal/core/application/agent/retrieval/tools"
	artifactusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/artifact"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/chat"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/conversation"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/image"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/knowledge"
	mindmapusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/mindmap"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/notebook"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/pipeline"
	responseusecase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/sentence"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/source"
	"github.com/oniharnantyo/eino-notebook/internal/adk/middleware"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/persistence"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/storage"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/handlers"
	httproutes "github.com/oniharnantyo/eino-notebook/internal/interfaces/http/routes"
	"github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/model"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	pgvectoretriever "github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
)

var (
	servePort       int
	serveHost       string
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
		logger.SetDefault(log) // Set default slog logger for packages using slog directly
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

		// Initialize S3 storage
		s3Storage, err := storage.NewS3Storage(cfg.S3)
		if err != nil {
			return fmt.Errorf("failed to create S3 storage: %w", err)
		}
		if err := s3Storage.EnsureBucket(ctx); err != nil {
			log.Warn("failed to ensure S3 bucket exists", "error", err)
		}
		log.Info("initialized", "storage", "S3", "bucket", cfg.S3.Bucket)

		// Create pgvector indexers for different content types (for storage/indexing)
		// Sentence indexer
		sentenceIndexerConfig := &pgvector.Config{
			Pool:                   dbPool,
			TableName:              "sentences",
			Dimension:              cfg.Embedding.Dimension,
			ReferenceIDColumn:      "knowledge_id", // Sentences reference knowledge chunks
			AutoCreateTable:        false,
			CreateIndexIfNotExists: false,
		}
		sentenceIndexer, err := pgvector.NewIndexer(ctx, sentenceIndexerConfig)
		if err != nil {
			return fmt.Errorf("failed to create sentence indexer: %w", err)
		}
		log.Info("initialized", "indexer", "sentence-pgvector")
		_ = sentenceIndexer // Repositories use dbPool directly for now, but indexer is initialized as requested

		// Image indexer
		imageIndexerConfig := &pgvector.Config{
			Pool:                   dbPool,
			TableName:              "images",
			Dimension:              cfg.Embedding.Dimension,
			ReferenceIDColumn:      "source_id", // Images reference sources directly
			AutoCreateTable:        false,
			CreateIndexIfNotExists: false,
		}
		imageIndexer, err := pgvector.NewIndexer(ctx, imageIndexerConfig)
		if err != nil {
			return fmt.Errorf("failed to create image indexer: %w", err)
		}
		log.Info("initialized", "indexer", "image-pgvector")
		_ = imageIndexer

		notebookRepo := persistence.NewPostgresNotebookRepository(dbPool)
		log.Info("initialized", "repository", "PostgresNotebookRepository")

		knowledgeRepo := persistence.NewPostgresKnowledgeRepository(dbPool)
		log.Info("initialized", "repository", "PostgresKnowledgeRepository")

		sentenceRepo := persistence.NewPostgresSentenceRepository(dbPool)
		log.Info("initialized", "repository", "PostgresSentenceRepository")

		imageRepo := persistence.NewPostgresImageRepository(dbPool)
		log.Info("initialized", "repository", "PostgresImageRepository")

		sourceRepo := persistence.NewPostgresSourceRepository(dbPool)
		log.Info("initialized", "repository", "PostgresSourceRepository")

		conversationRepo := persistence.NewPostgresConversationRepository(dbPool)
		log.Info("initialized", "repository", "PostgresConversationRepository")

		artifactRepo := persistence.NewPostgresArtifactRepository(dbPool)
		log.Info("initialized", "repository", "PostgresArtifactRepository")

		// Initialize embedder using factory
		embedder, err := model.CreateEmbedder(ctx, &cfg.Embedding)
		if err != nil {
			log.Warn("failed to initialize embedder", "error", err)
		} else {
			log.Info("initialized", "embedder", cfg.Embedding.Provider, "model", cfg.Embedding.Model, "dimension", cfg.Embedding.Dimension)
		}

		// Initialize vision embedder for image processing
		visionEmbedder, err := model.CreateVisionEmbedder(ctx, &cfg.Embedding)
		if err != nil {
			log.Warn("failed to initialize vision embedder", "error", err)
		} else {
			log.Info("initialized", "vision_embedder", cfg.Embedding.Provider, "model", cfg.Embedding.Model, "dimension", cfg.Embedding.Dimension)
		}

		// Initialize vision describer for image description generation
		visionDescriber, err := model.CreateVisionDescriber(ctx, &cfg.Chat)
		if err != nil {
			log.Warn("failed to initialize vision describer", "error", err)
		} else {
			log.Info("initialized", "vision_describer", cfg.Chat.Provider, "model", cfg.Chat.Model)
		}

		// Application Layer (Use Cases)
		notebookUseCase := notebook.NewNotebookUseCase(notebookRepo)
		log.Info("initialized", "usecase", "NotebookUseCase")

		// Create a transformer specifically for sentences (Shard 10.2)
		sentenceTransformer, err := recursive.NewSplitter(ctx, &recursive.Config{
			ChunkSize:   200,
			OverlapSize: 20,
			Separators:  []string{"\n\n", "\n", ". ", " ", ""},
		})
		if err != nil {
			log.Warn("failed to create sentence splitter", "error", err)
		} else {
			log.Info("initialized", "transformer", "sentence-splitter",
				"chunk_size", 200, "overlap_size", 20)
		}

		knowledgeUseCase := knowledge.NewKnowledgeUseCase(knowledgeRepo)
		log.Info("initialized", "usecase", "KnowledgeUseCase")

		sentenceUseCase := sentence.NewSentenceUseCase(sentenceRepo, sentenceTransformer, embedder)
		log.Info("initialized", "usecase", "SentenceUseCase")

		imageUseCase := image.NewImageUseCase(imageRepo, s3Storage, visionEmbedder, visionDescriber, log)
		log.Info("initialized", "usecase", "ImageUseCase")

		conversationUseCase := conversation.NewConversationUseCase(conversationRepo)
		log.Info("initialized", "usecase", "ConversationUseCase")

		// Initialize specific retrievers for agent tools
		sentencesRetriever, err := pgvectoretriever.NewSentencesRetriever(dbPool, cfg.Embedding.Dimension, embedder)
		if err != nil {
			log.Warn("failed to create sentences retriever", "error", err)
		} else {
			log.Info("initialized", "retriever", "sentences")
		}

		imagesRetriever, err := pgvectoretriever.NewImagesRetriever(dbPool, cfg.Embedding.Dimension)
		if err != nil {
			log.Warn("failed to create images retriever", "error", err)
		} else {
			log.Info("initialized", "retriever", "images")
		}

		knowledgesRetriever, err := pgvectoretriever.NewKnowledgesRetriever(dbPool, cfg.Embedding.Dimension)
		if err != nil {
			log.Warn("failed to create knowledges retriever", "error", err)
		} else {
			log.Info("initialized", "retriever", "knowledges")
		}

		// Initialize chat model using factory
		chatModel, err := model.CreateToolCallingChatModel(ctx, &cfg.Chat)
		if err != nil {
			log.Warn("failed to initialize chat model", "error", err)
		} else {
			log.Info("initialized", "chat_model", cfg.Chat.Provider, "model", cfg.Chat.Model)
		}

		// Create response use case with history management configuration
		var responseUseCase chat.ResponseUseCase
		if sentencesRetriever != nil && chatModel != nil && embedder != nil && knowledgesRetriever != nil && imagesRetriever != nil {
			// Create static tools (reusable across requests)
			keywordSearchTool := retrievalTools.NewKeywordSearchTool(knowledgesRetriever)
			semanticSearchTool := retrievalTools.NewSemanticSearchTool(sentencesRetriever)
			imageSearchTool := retrievalTools.NewImageSearchTool(imagesRetriever)

			// Create retrieval agent with static tools
			retrievalAgent := agent.NewRetrievalAgent(chatModel, keywordSearchTool, semanticSearchTool, imageSearchTool)
			log.Info("initialized", "agent", "RetrievalAgent", "static_tools", 3)

			// Create conversation memory middleware
			conversationMemoryMiddleware := middleware.NewConversationMemory(conversationRepo, log)
			retrievalAgent.WithMiddlewares(conversationMemoryMiddleware)
			log.Info("initialized", "middleware", "ConversationMemory")

			// Configure conversation history management
			historyConfig := &history.HistoryConfig{
				Strategy:             history.HistoryStrategySlidingWindow,
				MaxMessages:          10,   // Keep last 10 messages
				MaxTokens:            4000, // Max ~4000 tokens for history
				TokenEstimationRatio: 4,    // 1 token ≈ 4 chars
				SummarizeThreshold:   5,    // Summarize messages older than 5 turns
			}
			responseUseCase = responseusecase.NewResponseUseCase(notebookRepo, conversationRepo, sourceRepo, embedder, chatModel, cfg.Chat.Model, historyConfig, retrievalAgent, knowledgeRepo)
			log.Info("initialized", "usecase", "ResponseUseCase", "history_strategy", historyConfig.Strategy, "max_messages", historyConfig.MaxMessages)
		}

		// Initialize Kreuzberg document parser with the provided detailed configuration
		kreuzbergConfig := &kreuzberg.Config{
			ServiceURL:   cfg.Kreuzberg.ServiceURL,
			OutputFormat: cfg.Kreuzberg.OutputFormat,
			Timeout:      cfg.Kreuzberg.Timeout,
			ExtractConfig: &kreuzberg.ExtractConfig{
				UseCache:                 true,
				EnableQualityProcessing:  true,
				ForceOCR:                 false,
				DisableOCR:               false,
				ExtractionTimeoutSecs:    300,
				MaxConcurrentExtractions: 4,
				ResultFormat:             "element_based",
				OutputFormat:             "markdown",
				IncludeDocumentStructure: false,
				CacheTTLSecs:             3600,
				MaxArchiveDepth:          4,
				OCR: &kreuzberg.OCRConfig{
					Backend:    "paddleocr",
					Language:   "eng",
					AutoRotate: true,
					PaddleOCRConfig: &kreuzberg.PaddleOCRConfig{
						Language:             "en",
						UseAngleCls:          true,
						EnableTableDetection: true,
						DetDBThresh:          0.3,
						DetDBBoxThresh:       0.6,
						DetDBUnclipRatio:     1.5,
						DetLimitSideLen:      960,
						RecBatchNum:          6,
						Padding:              0,
						DropScore:            0.5,
						ModelTier:            "mobile",
					},
					QualityThresholds: &kreuzberg.QualityThresholds{
						MinTotalNonWhitespace:       10,
						MinNonWhitespacePerPage:     5,
						MinMeaningfulWordLen:        3,
						MinMeaningfulWords:          3,
						MinAlnumRatio:               0.25,
						MinGarbageChars:             100,
						MaxFragmentedWordRatio:      0.5,
						CriticalFragmentedWordRatio: 0.8,
						MinAvgWordLength:            3.5,
						MinWordsForAvgLengthCheck:   10,
						MinConsecutiveRepeatRatio:   0.2,
						MinWordsForRepeatCheck:      20,
						SubstantiveMinChars:         100,
						NonTextMinChars:             50,
						AlnumWsRatioThreshold:       0.1,
						PipelineMinQuality:          0.7,
					},
				},
				ContentFilter: &kreuzberg.ContentFilter{
					StripRepeatingText: true,
				},
				Images: &kreuzberg.ImagesConfig{
					ExtractImages:      true,
					TargetDPI:          72,
					MaxImageDimension:  512,
					InjectPlaceholders: true,
					AutoAdjustDPI:      true,
					MinDPI:             72,
					MaxDPI:             150,
				},
				PDFOptions: &kreuzberg.PDFOptions{
					ExtractImages:           true,
					ExtractMetadata:         true,
					ExtractAnnotations:      true,
					AllowSingleColumnTables: true,
					Hierarchy: &kreuzberg.HierarchyConfig{
						Enabled:              false,
						KClusters:            3,
						IncludeBBox:          false,
						OCRCoverageThreshold: 0.1,
					},
				},
				TreeSitter: &kreuzberg.TreeSitter{
					Enabled:             true,
					ContentMode:         "full",
					IncludeSyntaxColors: true,
					CommentStyle:        "docstring",
				},
				SecurityLimits: &kreuzberg.SecurityLimits{
					MaxArchiveSize:      104857600,
					MaxCompressionRatio: 100,
					MaxFilesInArchive:   1000,
					MaxNestingDepth:     10,
					MaxEntityLength:     65536,
					MaxContentSize:      10485760,
					MaxIterations:       1000,
					MaxXMLDepth:         100,
					MaxTableCells:       10000,
				},
				Pages: &kreuzberg.PagesConfig{
					ExtractPages: false,
				},
				Chunking: &kreuzberg.Chunking{
					MaxCharacters:         1000,
					Overlap:               0,
					Trim:                  true,
					ChunkerType:           "markdown",
					PrependHeadingContext: false,
					Sizing: &kreuzberg.ChunkSizing{
						Type: "characters",
					},
				},
			},
		}

		// rawKreuzbergParser kept for fallback (currently unused, Docling is default)
		_ = kreuzbergConfig

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
		fileExtractor := extractor.NewFileContentExtractor(kreuzbergDocParser, 100<<20)
		urlExtractor := extractor.NewURLContentExtractor(30 * time.Second)
		textExtractor := extractor.NewTextContentExtractor(1 << 20)

		// Factory Pattern: Create appropriate extractor based on content type
		contentExtractorFactory := extractor.NewContentExtractorFactory(
			fileExtractor,
			urlExtractor,
			textExtractor,
		)
		log.Info("initialized", "factory", "ContentExtractorFactory")

		// Create pipeline factory for ingestion pipelines
		pipelineFactory := pipeline.NewPipelineFactory(
			sourceRepo,
			knowledgeRepo,
			sentenceRepo,
			imageRepo,
			docParserFactory,
			embedder,
			s3Storage,
			visionDescriber,
			log,
		)
		log.Info("initialized", "factory", "PipelineFactory")

		// Create source usecase with content extraction dependencies
		sourceUseCase := source.NewSourceUseCase(sourceRepo, notebookRepo, contentExtractorFactory, docParserFactory, knowledgeUseCase, sentenceUseCase, imageUseCase, pipelineFactory, log)
		log.Info("initialized", "usecase", "SourceUseCase")

		// Create artifact usecase
		artifactUseCase := artifactusecase.NewArtifactUseCase(artifactRepo)
		log.Info("initialized", "usecase", "ArtifactUseCase")

		// Create mindmap usecase
		mindmapUseCase := mindmapusecase.NewMindmapUseCase(sourceRepo, artifactRepo, chatModel, log)
		log.Info("initialized", "usecase", "MindmapUseCase")

		// Interface Layer (HTTP Handlers)
		notebookHandler := handlers.NewNotebookHandler(notebookUseCase, log)
		sourceHandler := handlers.NewSourceHandler(sourceUseCase, log)
		knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeUseCase, sourceUseCase, log)
		var responseHandler *handlers.ResponseHandler
		if responseUseCase != nil {
			responseHandler = handlers.NewResponseHandler(responseUseCase, log)
			log.Info("initialized", "handler", "ResponseHandler")
		}

		conversationHandler := handlers.NewConversationHandler(conversationUseCase, log)
		log.Info("initialized", "handler", "ConversationHandler")

		// Create artifact handler
		artifactHandler := handlers.NewArtifactHandler(artifactUseCase, mindmapUseCase, log)
		log.Info("initialized", "handler", "ArtifactHandler")

		// Setup routes
		router := mux.NewRouter()
		httproutes.Setup(router, notebookHandler, knowledgeHandler, sourceHandler, responseHandler, conversationHandler, artifactHandler)
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
