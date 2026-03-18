package table

import (
	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

const mergedCellOverlapRatio = 0.5

type gridBuilder struct{}

func newGridBuilder() *gridBuilder {
	return &gridBuilder{}
}

// Build assembles a Table from text lines and column boundaries.
// It assigns text elements to columns, detects merged cells, and
// extracts the header row.
func (g *gridBuilder) Build(
	lines []layout.TextLine,
	columns []Column,
	pageNum int,
) Table {
	rows := g.buildRows(lines, columns)
	headers := g.extractHeaders(rows)
	bounds := g.computeBounds(lines)

	return Table{
		Rows:    rows,
		Columns: columns,
		Bounds:  bounds,
		PageNum: pageNum,
		Headers: headers,
	}
}

// buildRows assigns text elements from each line to column-based cells.
func (g *gridBuilder) buildRows(
	lines []layout.TextLine,
	columns []Column,
) []Row {
	rows := make([]Row, 0, len(lines))

	for i, line := range lines {
		cells := g.assignCells(line, columns, i)
		if len(cells) == 0 {
			continue
		}

		rows = append(rows, Row{
			Index: i,
			Cells: cells,
		})
	}

	return rows
}

// assignCells maps text elements from a line into cells based on column
// boundaries, detecting elements that span multiple columns.
func (g *gridBuilder) assignCells(
	line layout.TextLine,
	columns []Column,
	rowIndex int,
) []Cell {
	cells := make([]Cell, 0, len(columns))
	assigned := make(map[int]bool)

	// First pass: detect spanning elements and assign them
	for _, elem := range line.Elements {
		spanning := g.spannedColumns(elem, columns)
		if len(spanning) <= 1 {
			continue
		}

		firstCol := spanning[0]
		lastCol := spanning[len(spanning)-1]

		cells = append(cells, Cell{
			Text:    elem.Text,
			Row:     rowIndex,
			Col:     firstCol,
			Merged:  true,
			RowSpan: 1,
			ColSpan: len(spanning),
			Bounds: pdf.Rect{
				X1: columns[firstCol].X1,
				Y1: line.Bounds.Y1,
				X2: columns[lastCol].X2,
				Y2: line.Bounds.Y2,
			},
		})

		for _, ci := range spanning {
			assigned[ci] = true
		}
	}

	// Second pass: assign non-spanning elements to their columns
	for colIdx, col := range columns {
		if assigned[colIdx] {
			continue
		}

		text := g.collectColumnText(line, col)
		if text == "" {
			continue
		}

		cells = append(cells, Cell{
			Text:    text,
			Row:     rowIndex,
			Col:     colIdx,
			RowSpan: 1,
			ColSpan: 1,
			Bounds: pdf.Rect{
				X1: col.X1,
				Y1: line.Bounds.Y1,
				X2: col.X2,
				Y2: line.Bounds.Y2,
			},
		})
	}

	return cells
}

// collectColumnText concatenates text from all elements in a line that fall
// within the given column boundaries.
func (g *gridBuilder) collectColumnText(line layout.TextLine, col Column) string {
	var result string

	for _, elem := range line.Elements {
		if !g.elementInColumn(elem, col) {
			continue
		}

		if result != "" {
			result += " "
		}
		result += elem.Text
	}

	return result
}

// elementInColumn checks whether a text element falls within a column.
func (g *gridBuilder) elementInColumn(elem pdf.TextElement, col Column) bool {
	elemCenter := (elem.Bounds.X1 + elem.Bounds.X2) / 2
	return elemCenter >= col.X1 && elemCenter <= col.X2
}

// spannedColumns returns the indices of columns that a text element
// overlaps with, based on the element's horizontal extent.
func (g *gridBuilder) spannedColumns(
	elem pdf.TextElement,
	columns []Column,
) []int {
	var indices []int
	for i, col := range columns {
		if g.rangesOverlap(elem.Bounds.X1, elem.Bounds.X2, col.X1, col.X2) {
			indices = append(indices, i)
		}
	}

	return indices
}

// rangesOverlap checks whether two horizontal ranges have meaningful overlap.
func (g *gridBuilder) rangesOverlap(ax1, ax2, bx1, bx2 float64) bool {
	overlapStart := ax1
	if bx1 > overlapStart {
		overlapStart = bx1
	}

	overlapEnd := ax2
	if bx2 < overlapEnd {
		overlapEnd = bx2
	}

	if overlapEnd <= overlapStart {
		return false
	}

	colWidth := bx2 - bx1
	if colWidth <= 0 {
		return false
	}

	return (overlapEnd-overlapStart)/colWidth >= mergedCellOverlapRatio
}

// extractHeaders returns text from the first row as column headers.
func (g *gridBuilder) extractHeaders(rows []Row) []string {
	if len(rows) == 0 {
		return nil
	}

	headers := make([]string, len(rows[0].Cells))
	for i, cell := range rows[0].Cells {
		headers[i] = cell.Text
	}

	return headers
}

// computeBounds calculates the bounding rectangle for a group of text lines.
func (g *gridBuilder) computeBounds(lines []layout.TextLine) pdf.Rect {
	if len(lines) == 0 {
		return pdf.Rect{}
	}

	bounds := lines[0].Bounds
	for _, line := range lines[1:] {
		if line.Bounds.X1 < bounds.X1 {
			bounds.X1 = line.Bounds.X1
		}
		if line.Bounds.Y1 < bounds.Y1 {
			bounds.Y1 = line.Bounds.Y1
		}
		if line.Bounds.X2 > bounds.X2 {
			bounds.X2 = line.Bounds.X2
		}
		if line.Bounds.Y2 > bounds.Y2 {
			bounds.Y2 = line.Bounds.Y2
		}
	}

	return bounds
}
