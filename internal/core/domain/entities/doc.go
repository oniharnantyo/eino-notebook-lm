// Package entities contains the core domain entities of the application.
//
// Domain entities represent business concepts with:
// - Unique identity (ID)
// - Business rules (validation, invariants)
// - Behavior (methods that operate on the entity's state)
//
// Example:
//
//	type Notebook struct {
//	    ID    uuid.UUID
//	    Title string
//	    Status NotebookStatus
//	}
//
//	func (n *Notebook) Archive() {
//	    n.Status = StatusArchived
//	}
//
// Guidelines:
//   - Entities should be self-contained
//   - Include business logic methods
//   - Use value objects for complex attributes
//   - No framework dependencies
//   - Focus on "what" the business does, not "how"
package entities
