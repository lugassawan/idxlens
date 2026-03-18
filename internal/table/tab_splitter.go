package table

import (
	"strings"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

// hasTabSeparators checks whether a group of lines uses tab characters as
// column separators. Returns true if at least half the lines contain tabs.
func hasTabSeparators(lines []layout.TextLine) bool {
	if len(lines) == 0 {
		return false
	}

	tabCount := 0
	for _, line := range lines {
		if strings.Contains(line.Text, "\t") {
			tabCount++
		}
	}

	return tabCount*2 >= len(lines)
}

// splitTabColumns determines column boundaries from tab-separated lines by
// counting the maximum number of tab-delimited fields.
func splitTabColumns(lines []layout.TextLine) []Column {
	maxFields := 0

	for _, line := range lines {
		if !strings.Contains(line.Text, "\t") {
			continue
		}

		fields := strings.Split(line.Text, "\t")
		if len(fields) > maxFields {
			maxFields = len(fields)
		}
	}

	if maxFields < 2 {
		return nil
	}

	columns := make([]Column, maxFields)
	for i := range columns {
		columns[i] = Column{
			Index:     i,
			X1:        float64(i * 100),
			X2:        float64((i + 1) * 100),
			Alignment: columnAlignment(i, maxFields),
		}
	}

	return columns
}

// columnAlignment returns the alignment for a column based on its position.
// First column is left-aligned (labels), middle columns are right-aligned
// (numeric values), last column is left-aligned when there are 4+ columns
// (English label in bilingual IDX format: label, val1, val2, en_label).
func columnAlignment(index, total int) string {
	if index == 0 {
		return alignLeft
	}

	if index == total-1 && total > 3 {
		return alignLeft
	}

	return alignRight
}

// buildTabRows splits each tab-separated line into cells, one per column.
func buildTabRows(lines []layout.TextLine, columns []Column) []Row {
	rows := make([]Row, 0, len(lines))

	for i, line := range lines {
		cells := splitTabLine(line, columns, i)
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

// splitTabLine splits a single tab-separated line into cells aligned with
// the given columns.
func splitTabLine(line layout.TextLine, columns []Column, rowIndex int) []Cell {
	parts := strings.Split(line.Text, "\t")

	cells := make([]Cell, 0, len(parts))

	for i, part := range parts {
		text := strings.TrimSpace(part)
		if text == "" || i >= len(columns) {
			continue
		}

		cells = append(cells, Cell{
			Text:    text,
			Row:     rowIndex,
			Col:     i,
			RowSpan: 1,
			ColSpan: 1,
			Bounds: pdf.Rect{
				X1: columns[i].X1,
				Y1: line.Bounds.Y1,
				X2: columns[i].X2,
				Y2: line.Bounds.Y2,
			},
		})
	}

	return cells
}
