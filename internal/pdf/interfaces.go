package pdf

import "io"

// Reader provides access to PDF document content.
// Implementations must be safe to call Pages() multiple times.
type Reader interface {
	// Open initializes the reader from a byte stream.
	Open(r io.ReadSeeker) error

	// Metadata returns document-level metadata.
	Metadata() (Metadata, error)

	// PageCount returns the total number of pages.
	PageCount() int

	// Page extracts content from a single page (1-indexed).
	Page(number int) (Page, error)

	// Close releases any resources held by the reader.
	Close() error
}
