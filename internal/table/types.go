package table

import "github.com/lugassawan/idxlens/internal/pdf"

// Cell represents a single cell in a detected table.
type Cell struct {
	Text    string
	Row     int
	Col     int
	Bounds  pdf.Rect
	Merged  bool
	RowSpan int
	ColSpan int
}

// Row represents a table row.
type Row struct {
	Index int
	Cells []Cell
}

// Column represents metadata about a table column.
type Column struct {
	Index     int
	X1        float64 // left edge
	X2        float64 // right edge
	Alignment string  // "left", "right", "center"
}

// Table represents a detected table structure.
type Table struct {
	Rows     []Row
	Columns  []Column
	Bounds   pdf.Rect
	PageNum  int
	Headers  []string // column header texts
	PageText []string // non-table text lines from the same page
}
