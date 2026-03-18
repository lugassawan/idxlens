package table

import (
	"math"
	"sort"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

const (
	defaultMinColumns   = 2
	defaultMinRows      = 2
	defaultColumnGapMin = 10.0
	alignmentTolerance  = 3.0
	alignLeft           = "left"
	alignRight          = "right"
)

type lineDetector struct {
	minColumns   int
	minRows      int
	columnGapMin float64
}

func newLineDetector() *lineDetector {
	return &lineDetector{
		minColumns:   defaultMinColumns,
		minRows:      defaultMinRows,
		columnGapMin: defaultColumnGapMin,
	}
}

// Detect finds all tables in a layout page by analyzing text alignment patterns.
func (d *lineDetector) Detect(page layout.LayoutPage) ([]Table, error) {
	if len(page.Lines) == 0 {
		return nil, nil
	}

	groups := d.findLineGroups(page.Lines)

	var tables []Table
	for _, group := range groups {
		columns := d.detectColumns(group)
		if len(columns) < d.minColumns {
			continue
		}

		rows := d.buildRows(group, columns)
		if len(rows) < d.minRows {
			continue
		}

		bounds := d.computeBounds(group)
		tables = append(tables, Table{
			Rows:    rows,
			Columns: columns,
			Bounds:  bounds,
			PageNum: page.Number,
			Headers: d.extractHeaders(rows),
		})
	}

	return tables, nil
}

// findLineGroups splits lines into groups separated by large vertical gaps.
func (d *lineDetector) findLineGroups(lines []layout.TextLine) [][]layout.TextLine {
	if len(lines) <= 1 {
		return [][]layout.TextLine{lines}
	}

	gaps := make([]float64, len(lines)-1)
	for i := range len(lines) - 1 {
		gaps[i] = math.Abs(lines[i].Bounds.Y1 - lines[i+1].Bounds.Y1)
	}

	medianGap := median(gaps)
	threshold := medianGap * 2.5

	var groups [][]layout.TextLine
	start := 0

	for i, gap := range gaps {
		if gap > threshold && threshold > 0 {
			groups = append(groups, lines[start:i+1])
			start = i + 1
		}
	}
	groups = append(groups, lines[start:])

	return groups
}

// detectColumns identifies column boundaries by clustering text element
// X-coordinates across multiple lines.
func (d *lineDetector) detectColumns(lines []layout.TextLine) []Column {
	var xPositions []xEdge
	for _, line := range lines {
		for _, elem := range line.Elements {
			xPositions = append(xPositions, xEdge{
				left:  elem.Bounds.X1,
				right: elem.Bounds.X2,
			})
		}
	}

	if len(xPositions) == 0 {
		return nil
	}

	clusters := d.clusterXPositions(xPositions)
	if len(clusters) < d.minColumns {
		return nil
	}

	columns := make([]Column, len(clusters))
	for i, c := range clusters {
		columns[i] = Column{
			Index:     i,
			X1:        c.left,
			X2:        c.right,
			Alignment: d.detectAlignment(lines, c),
		}
	}

	return columns
}

// clusterXPositions groups X-edge positions into column clusters based on gaps.
func (d *lineDetector) clusterXPositions(edges []xEdge) []xCluster {
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].left < edges[j].left
	})

	var clusters []xCluster
	current := xCluster{left: edges[0].left, right: edges[0].right}

	for i := 1; i < len(edges); i++ {
		gap := edges[i].left - current.right
		if gap >= d.columnGapMin {
			clusters = append(clusters, current)
			current = xCluster{left: edges[i].left, right: edges[i].right}
		} else if edges[i].right > current.right {
			current.right = edges[i].right
		}
	}
	clusters = append(clusters, current)

	return clusters
}

// detectAlignment determines whether elements in a column are left-aligned,
// right-aligned, or centered.
func (d *lineDetector) detectAlignment(lines []layout.TextLine, cluster xCluster) string {
	var leftAligned, rightAligned int

	for _, line := range lines {
		for _, elem := range line.Elements {
			if !d.elementInCluster(elem, cluster) {
				continue
			}

			if math.Abs(elem.Bounds.X1-cluster.left) <= alignmentTolerance {
				leftAligned++
			}

			if math.Abs(elem.Bounds.X2-cluster.right) <= alignmentTolerance {
				rightAligned++
			}
		}
	}

	if rightAligned > leftAligned {
		return alignRight
	}

	return alignLeft
}

// elementInCluster checks whether a text element falls within a column cluster.
func (d *lineDetector) elementInCluster(elem pdf.TextElement, cluster xCluster) bool {
	elemCenter := (elem.Bounds.X1 + elem.Bounds.X2) / 2
	return elemCenter >= cluster.left && elemCenter <= cluster.right
}

// buildRows assigns text elements from each line into column-based cells.
func (d *lineDetector) buildRows(lines []layout.TextLine, columns []Column) []Row {
	rows := make([]Row, 0, len(lines))

	for i, line := range lines {
		cells := d.assignCells(line, columns, i)
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

// assignCells maps text elements from a line into cells based on column boundaries.
func (d *lineDetector) assignCells(
	line layout.TextLine,
	columns []Column,
	rowIndex int,
) []Cell {
	cells := make([]Cell, 0, len(columns))

	for colIdx, col := range columns {
		text := d.collectColumnText(line, col)
		if text == "" {
			continue
		}

		cells = append(cells, Cell{
			Text: text,
			Row:  rowIndex,
			Col:  colIdx,
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
func (d *lineDetector) collectColumnText(line layout.TextLine, col Column) string {
	var result string

	for _, elem := range line.Elements {
		cluster := xCluster{left: col.X1, right: col.X2}
		if !d.elementInCluster(elem, cluster) {
			continue
		}

		if result != "" {
			result += " "
		}
		result += elem.Text
	}

	return result
}

// computeBounds calculates the bounding rectangle for a group of text lines.
func (d *lineDetector) computeBounds(lines []layout.TextLine) pdf.Rect {
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

// extractHeaders returns text from the first row as column headers.
func (d *lineDetector) extractHeaders(rows []Row) []string {
	if len(rows) == 0 {
		return nil
	}

	headers := make([]string, len(rows[0].Cells))
	for i, cell := range rows[0].Cells {
		headers[i] = cell.Text
	}

	return headers
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}

	return sorted[mid]
}
