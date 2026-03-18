package layout

import "github.com/lugassawan/idxlens/internal/pdf"

// Analyzer processes raw PDF pages into structured layouts.
type Analyzer interface {
	// Analyze takes a PDF page and returns the assembled layout.
	Analyze(page pdf.Page) (LayoutPage, error)
}
