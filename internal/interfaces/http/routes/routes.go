package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/handlers"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/middleware"
)

// Setup configures all application routes
func Setup(router *mux.Router, notebookHandler *handlers.NotebookHandler, knowledgeHandler *handlers.KnowledgeHandler, sourceHandler *handlers.SourceHandler, responseHandler *handlers.ResponseHandler, conversationHandler *handlers.ConversationHandler, artifactHandler *handlers.ArtifactHandler) {
	// Apply global middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recovery)
	router.Use(middleware.CORS)

	// Health check
	router.HandleFunc("/health", healthCheck).Methods(http.MethodGet)
	router.HandleFunc("/ready", readinessCheck).Methods(http.MethodGet)

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Notebook routes
	notebooks := api.PathPrefix("/notebooks").Subrouter()
	notebooks.HandleFunc("", notebookHandler.Create).Methods(http.MethodPost)
	notebooks.HandleFunc("", notebookHandler.List).Methods(http.MethodGet)
	notebooks.HandleFunc("/{id}", notebookHandler.GetByID).Methods(http.MethodGet)
	notebooks.HandleFunc("/{id}", notebookHandler.Update).Methods(http.MethodPut)
	notebooks.HandleFunc("/{id}", notebookHandler.Delete).Methods(http.MethodDelete)

	// Source routes (nested under notebooks)
	notebooks.HandleFunc("/{notebookId}/sources", sourceHandler.List).Methods(http.MethodGet)
	notebooks.HandleFunc("/{notebookId}/sources/{id}", sourceHandler.GetByID).Methods(http.MethodGet)
	notebooks.HandleFunc("/{notebookId}/sources/{id}", sourceHandler.Delete).Methods(http.MethodDelete)

	// Conversation routes (nested under notebooks)
	notebooks.HandleFunc("/{notebookId}/conversations", conversationHandler.ListByNotebook).Methods(http.MethodGet)

	// Knowledge routes (nested under notebooks)
	notebooks.HandleFunc("/{notebookId}/knowledges", knowledgeHandler.Create).Methods(http.MethodPost)
	notebooks.HandleFunc("/{notebookId}/knowledges/status/{sourceId}/stream", knowledgeHandler.StreamSourceStatus).Methods(http.MethodGet)

	// Artifact routes (nested under notebooks)
	notebooks.HandleFunc("/{notebookId}/mindmap", artifactHandler.GenerateMindmap).Methods(http.MethodPost)
	notebooks.HandleFunc("/{notebookId}/artifacts", artifactHandler.List).Methods(http.MethodGet)
	notebooks.HandleFunc("/{notebookId}/artifacts/{id}", artifactHandler.GetByID).Methods(http.MethodGet)

	// OpenAI Responses API (only if responseHandler is provided)
	if responseHandler != nil {
		router.HandleFunc("/v1/responses", responseHandler.CreateResponse).Methods(http.MethodPost)
	}
}

// healthCheck returns the health status
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// readinessCheck returns the readiness status
func readinessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
