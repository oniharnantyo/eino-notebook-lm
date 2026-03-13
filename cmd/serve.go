package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/persistence"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/handlers"
	httproutes "github.com/oniharnantyo/eino-notebook/internal/interfaces/http/routes"
	"github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	servePort int
	serveHost string
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

		// Initialize dependencies (Hexagonal Architecture - Dependency Injection)
		// Infrastructure Layer

		// Create database connection pool
		dbPool, err := pgxpool.New(ctx, cfg.Database.GetDSN())
		if err != nil {
			return fmt.Errorf("failed to create database pool: %w", err)
		}
		defer dbPool.Close()
		log.Info("initialized", "db_pool", "pgxpool")

		// Create pgvector indexer
		vectorIndexer, err := pgvector.NewIndexer(ctx, &pgvector.Config{
			Pool:              dbPool,
			Dimension:         cfg.Gemini.Dimension,
			AutoCreateTable:   true,
			CreateIndexIfNotExists: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create pgvector indexer: %w", err)
		}
		log.Info("initialized", "indexer", "pgvector", "dimension", cfg.Gemini.Dimension)

		notebookRepo := persistence.NewInMemoryNotebookRepository()
		log.Info("initialized", "repository", "InMemoryNotebookRepository")

		// Application Layer (Use Cases)
		notebookUseCase := usecases.NewNotebookUseCase(notebookRepo)
		log.Info("initialized", "usecase", "NotebookUseCase")

		// TODO: Initialize document repository when implemented
		// documentRepo := ...

		documentUseCase := usecases.NewDocumentUseCase(nil, vectorIndexer)
		log.Info("initialized", "usecase", "DocumentUseCase")

		// Interface Layer (HTTP Handlers)
		notebookHandler := handlers.NewNotebookHandler(notebookUseCase, log)
		documentHandler := handlers.NewDocumentHandler(documentUseCase, log)

		// Setup routes
		router := mux.NewRouter()
		httproutes.Setup(router, notebookHandler, documentHandler)
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
