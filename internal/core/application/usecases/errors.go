package usecases

import "errors"

// Common errors for usecases
var (
	// ErrNoAvailableParser is returned when no document parser is available
	ErrNoAvailableParser = errors.New("no available document parser")

	// ErrNoAvailablePDFExtractor is returned when no PDF extractor is available
	ErrNoAvailablePDFExtractor = errors.New("no available PDF extractor")
)
