package layout

import (
	"fmt"
	"testing"

	"github.com/lugassawan/idxlens/internal/pdf"
)

func BenchmarkAnalyze(b *testing.B) {
	tests := []struct {
		name         string
		elementCount int
	}{
		{name: "10 elements", elementCount: 10},
		{name: "50 elements", elementCount: 50},
		{name: "100 elements", elementCount: 100},
		{name: "500 elements", elementCount: 500},
	}

	for _, tc := range tests {
		page := buildSyntheticPage(tc.elementCount)

		b.Run(tc.name, func(b *testing.B) {
			a := NewAnalyzer()

			for range b.N {
				if _, err := a.Analyze(page); err != nil {
					b.Fatalf("Analyze: %v", err)
				}
			}
		})
	}
}

func BenchmarkFindDominantFontSize(b *testing.B) {
	tests := []struct {
		name         string
		elementCount int
	}{
		{name: "10 elements", elementCount: 10},
		{name: "100 elements", elementCount: 100},
		{name: "500 elements", elementCount: 500},
	}

	for _, tc := range tests {
		elements := buildSyntheticElements(tc.elementCount)

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				findDominantFontSize(elements)
			}
		})
	}
}

func BenchmarkClusterByY(b *testing.B) {
	tests := []struct {
		name         string
		elementCount int
	}{
		{name: "10 elements", elementCount: 10},
		{name: "100 elements", elementCount: 100},
		{name: "500 elements", elementCount: 500},
	}

	for _, tc := range tests {
		elements := buildSyntheticElements(tc.elementCount)
		a := &analyzer{lineThreshold: defaultLineThreshold}

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				a.clusterByY(elements, 12.0)
			}
		})
	}
}

func BenchmarkDetectRegions(b *testing.B) {
	tests := []struct {
		name      string
		lineCount int
	}{
		{name: "10 lines", lineCount: 10},
		{name: "50 lines", lineCount: 50},
		{name: "100 lines", lineCount: 100},
	}

	for _, tc := range tests {
		lines := buildSyntheticLines(tc.lineCount)

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				detectRegions(lines)
			}
		})
	}
}

// buildSyntheticPage creates a pdf.Page with the specified number of text
// elements arranged in a grid pattern typical of financial reports.
func buildSyntheticPage(elementCount int) pdf.Page {
	return pdf.Page{
		Number:   1,
		Size:     pdf.PageSize{Width: 595, Height: 842},
		Elements: buildSyntheticElements(elementCount),
	}
}

// buildSyntheticElements generates text elements in a two-column layout
// resembling a financial report page.
func buildSyntheticElements(count int) []pdf.TextElement {
	elements := make([]pdf.TextElement, count)

	for i := range count {
		row := i / 2
		col := i % 2

		x := 72.0 + float64(col)*250.0
		y := 700.0 - float64(row)*14.0
		fontSize := 12.0

		// Mix in some headers with larger font.
		if i%20 == 0 {
			fontSize = 14.0
		}

		elements[i] = pdf.TextElement{
			Text:     fmt.Sprintf("Element %d", i+1),
			FontName: "Helvetica",
			FontSize: fontSize,
			Bounds: pdf.Rect{
				X1: x,
				Y1: y,
				X2: x + 100.0,
				Y2: y + fontSize,
			},
		}
	}

	return elements
}

// buildSyntheticLines generates TextLine values simulating a structured
// document layout.
func buildSyntheticLines(count int) []TextLine {
	lines := make([]TextLine, count)

	for i := range count {
		y := 700.0 - float64(i)*14.0
		fontName := "Helvetica"
		fontSize := 12.0

		// Alternate font every 10 lines to create region boundaries.
		if (i/10)%2 == 1 {
			fontName = "Helvetica-Bold"
		}

		lines[i] = TextLine{
			Text:     fmt.Sprintf("Line %d content", i+1),
			FontName: fontName,
			FontSize: fontSize,
			Bounds: pdf.Rect{
				X1: 72.0,
				Y1: y,
				X2: 523.0,
				Y2: y + fontSize,
			},
		}
	}

	return lines
}
