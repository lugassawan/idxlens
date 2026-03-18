package table

import (
	"fmt"
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

func BenchmarkDetect(b *testing.B) {
	tests := []struct {
		name string
		rows int
		cols int
	}{
		{name: "5x3 table", rows: 5, cols: 3},
		{name: "20x4 table", rows: 20, cols: 4},
		{name: "50x5 table", rows: 50, cols: 5},
		{name: "100x6 table", rows: 100, cols: 6},
	}

	for _, tc := range tests {
		page := buildBenchLayoutPage(tc.rows, tc.cols)

		b.Run(tc.name, func(b *testing.B) {
			d := NewDetector()

			for range b.N {
				if _, err := d.Detect(page); err != nil {
					b.Fatalf("Detect: %v", err)
				}
			}
		})
	}
}

func BenchmarkFindLineGroups(b *testing.B) {
	tests := []struct {
		name      string
		lineCount int
	}{
		{name: "10 lines", lineCount: 10},
		{name: "50 lines", lineCount: 50},
		{name: "100 lines", lineCount: 100},
	}

	for _, tc := range tests {
		lines := buildBenchLines(tc.lineCount, 3)
		ld := newLineDetector()

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				ld.findLineGroups(lines)
			}
		})
	}
}

func BenchmarkDetectColumns(b *testing.B) {
	tests := []struct {
		name string
		rows int
		cols int
	}{
		{name: "10 rows 3 cols", rows: 10, cols: 3},
		{name: "50 rows 5 cols", rows: 50, cols: 5},
		{name: "100 rows 6 cols", rows: 100, cols: 6},
	}

	for _, tc := range tests {
		lines := buildBenchLines(tc.rows, tc.cols)
		ld := newLineDetector()

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				ld.detectColumns(lines)
			}
		})
	}
}

func BenchmarkBuildRows(b *testing.B) {
	tests := []struct {
		name string
		rows int
		cols int
	}{
		{name: "10 rows 3 cols", rows: 10, cols: 3},
		{name: "50 rows 5 cols", rows: 50, cols: 5},
	}

	for _, tc := range tests {
		lines := buildBenchLines(tc.rows, tc.cols)
		ld := newLineDetector()
		columns := ld.detectColumns(lines)

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				ld.buildRows(lines, columns)
			}
		})
	}
}

// buildBenchLayoutPage creates a LayoutPage with a synthetic table grid
// for benchmark testing.
func buildBenchLayoutPage(rows, cols int) layout.LayoutPage {
	return layout.LayoutPage{
		Number: 1,
		Size:   pdf.PageSize{Width: 595, Height: 842},
		Lines:  buildBenchLines(rows, cols),
	}
}

// buildBenchLines generates TextLine values arranged in a tabular grid with
// the specified number of rows and columns.
func buildBenchLines(rows, cols int) []layout.TextLine {
	colWidth := 400.0 / float64(cols)
	lines := make([]layout.TextLine, rows)

	for r := range rows {
		y := 700.0 - float64(r)*14.0
		elements := make([]pdf.TextElement, cols)

		for c := range cols {
			x := 72.0 + float64(c)*colWidth + float64(c)*15.0

			text := fmt.Sprintf("R%dC%d", r+1, c+1)
			if c == 0 {
				text = fmt.Sprintf("Label row %d", r+1)
			}

			elements[c] = pdf.TextElement{
				Text:     text,
				FontName: "Helvetica",
				FontSize: 10.0,
				Bounds: pdf.Rect{
					X1: x,
					Y1: y,
					X2: x + colWidth,
					Y2: y + 10.0,
				},
			}
		}

		lines[r] = layout.TextLine{
			Text:     fmt.Sprintf("Row %d", r+1),
			Elements: elements,
			FontName: "Helvetica",
			FontSize: 10.0,
			Bounds: pdf.Rect{
				X1: 72.0,
				Y1: y,
				X2: 72.0 + float64(cols)*colWidth + float64(cols-1)*15.0,
				Y2: y + 10.0,
			},
		}
	}

	return lines
}
