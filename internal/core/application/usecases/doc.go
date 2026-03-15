// Package usecases contains application business logic.
//
// Use cases orchestrate domain objects to perform specific user goals.
// They act as the application layer in Clean Architecture, coordinating
// between domain entities and repositories.
//
// # Structure
//
// The usecases package is organized into subpackages by domain:
//
//	- notebook: Notebook-related business logic
//	- knowledge: Knowledge management and content extraction
//	- extractor: Content extraction from various sources (files, URLs, text)
//	- document: Document parsing (Kreuzberg integration)
//
// # Example
//
//	type NotebookUseCase interface {
//	    Create(ctx context.Context, req *dtos.CreateNotebookRequest) (*dtos.NotebookResponse, error)
//	    GetByID(ctx context.Context, id string) (*dtos.NotebookResponse, error)
//	}
//
// # Guidelines
//
//   - Define interface first, then implement
//   - Use dependency injection (accept repositories via constructor)
//   - Handle transaction boundaries
//   - Return DTOs, not domain entities
//   - Don't expose domain details to interfaces
//   - One use case = one user goal
package usecases
