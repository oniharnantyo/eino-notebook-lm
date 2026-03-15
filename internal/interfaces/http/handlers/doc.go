// Package handlers contains HTTP request handlers (controllers).
//
// Handlers are the interface adapters that translate HTTP requests
// into use case calls and HTTP responses from DTOs.
//
// Example:
//
//	type NotebookHandler struct {
//	    useCase usecases.NotebookUseCase
//	    logger  *logger.Logger
//	}
//
//	func (h *NotebookHandler) Create(w http.ResponseWriter, r *http.Request) {
//	    var req dtos.CreateNotebookRequest
//	    json.NewDecoder(r.Body).Decode(&req)
//	    notebook, _ := h.useCase.Create(r.Context(), &req)
//	    json.NewEncoder(w).Encode(notebook)
//	}
//
// Guidelines:
//   - Thin layer - delegate to use cases
//   - Handle HTTP concerns (status codes, headers)
//   - Don't include business logic
//   - Use middleware for cross-cutting concerns
//   - Return appropriate HTTP status codes
package handlers
