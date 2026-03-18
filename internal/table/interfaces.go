package table

import "github.com/lugassawan/idxlens/internal/layout"

// Detector identifies and extracts tables from layout-analyzed pages.
type Detector interface {
	// Detect finds all tables in a layout page.
	Detect(page layout.LayoutPage) ([]Table, error)
}
