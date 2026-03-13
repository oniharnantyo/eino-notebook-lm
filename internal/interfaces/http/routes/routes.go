package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/handlers"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/middleware"
)

// Setup configures all application routes
func Setup(router *mux.Router, notebookHandler *handlers.NotebookHandler, documentHandler *handlers.DocumentHandler) {
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
	notebooks.HandleFunc("/{id}/archive", notebookHandler.Archive).Methods(http.MethodPost)

	// Document routes
	documents := api.PathPrefix("/documents").Subrouter()
	documents.HandleFunc("", documentHandler.Create).Methods(http.MethodPost)
	documents.HandleFunc("", documentHandler.List).Methods(http.MethodGet)
	documents.HandleFunc("/{id}", documentHandler.GetByID).Methods(http.MethodGet)
	documents.HandleFunc("/{id}", documentHandler.Update).Methods(http.MethodPut)
	documents.HandleFunc("/{id}", documentHandler.Delete).Methods(http.MethodDelete)
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
