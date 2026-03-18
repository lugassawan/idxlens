package table

import (
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

func TestLineDetectorDetect(t *testing.T) {
	tests := []struct {
		name        string
		page        layout.LayoutPage
		wantTables  int
		wantColumns int
		wantRows    int
		wantHeaders []string
	}{
		{
			name: "empty page returns no tables",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines:  nil,
			},
			wantTables: 0,
		},
		{
			name: "single column is not a table",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeLine("Header", 50, 700),
					makeLine("Row 1", 50, 686),
					makeLine("Row 2", 50, 672),
				},
			},
			wantTables: 0,
		},
		{
			name: "evenly spaced columns detected as table",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Description", 10, 100),
						textAt("Amount", 200, 260),
					),
					makeMultiElementLine(686,
						textAt("Revenue", 10, 70),
						textAt("1,000", 220, 260),
					),
					makeMultiElementLine(672,
						textAt("Expenses", 10, 75),
						textAt("500", 230, 260),
					),
				},
			},
			wantTables:  1,
			wantColumns: 2,
			wantRows:    3,
			wantHeaders: []string{"Description", "Amount"},
		},
		{
			name: "right-aligned numeric column detected",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Item", 10, 50),
						textAt("Value", 198, 260),
					),
					makeMultiElementLine(686,
						textAt("Alpha", 10, 50),
						textAt("1,234", 220, 260),
					),
					makeMultiElementLine(672,
						textAt("Beta", 10, 45),
						textAt("56,789", 210, 260),
					),
				},
			},
			wantTables:  1,
			wantColumns: 2,
			wantRows:    3,
		},
		{
			name: "single row not enough for table",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Col A", 10, 50),
						textAt("Col B", 200, 260),
					),
				},
			},
			wantTables: 0,
		},
		{
			name: "three columns with header row",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Name", 10, 60),
						textAt("Q1", 150, 180),
						textAt("Q2", 280, 310),
					),
					makeMultiElementLine(686,
						textAt("Revenue", 10, 65),
						textAt("100", 160, 180),
						textAt("200", 290, 310),
					),
					makeMultiElementLine(672,
						textAt("Cost", 10, 40),
						textAt("50", 165, 180),
						textAt("80", 295, 310),
					),
				},
			},
			wantTables:  1,
			wantColumns: 3,
			wantRows:    3,
			wantHeaders: []string{"Name", "Q1", "Q2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newLineDetector()
			tables, err := d.Detect(tt.page)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			assertTableResult(t, tables, tt.page.Number, tt.wantTables, tt.wantColumns, tt.wantRows, tt.wantHeaders)
		})
	}
}

func TestLineDetectorDetectAlignment(t *testing.T) {
	d := newLineDetector()

	page := layout.LayoutPage{
		Number: 1,
		Size:   pdf.PageSize{Width: 612, Height: 792},
		Lines: []layout.TextLine{
			makeMultiElementLine(700,
				textAt("Item", 10, 50),
				textAt("Value", 198, 260),
			),
			makeMultiElementLine(686,
				textAt("Alpha", 10, 50),
				textAt("100", 230, 260),
			),
			makeMultiElementLine(672,
				textAt("Beta", 10, 45),
				textAt("2,000", 215, 260),
			),
		},
	}

	tables, err := d.Detect(page)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}

	// First column should be left-aligned (all start at X1=10)
	if tables[0].Columns[0].Alignment != "left" {
		t.Errorf("column 0 alignment = %q, want %q", tables[0].Columns[0].Alignment, "left")
	}

	// Second column should be right-aligned (all end at X2=260)
	if tables[0].Columns[1].Alignment != "right" {
		t.Errorf("column 1 alignment = %q, want %q", tables[0].Columns[1].Alignment, "right")
	}
}

func TestLineDetectorDetectBounds(t *testing.T) {
	d := newLineDetector()

	page := layout.LayoutPage{
		Number: 1,
		Size:   pdf.PageSize{Width: 612, Height: 792},
		Lines: []layout.TextLine{
			makeMultiElementLine(700,
				textAt("A", 10, 30),
				textAt("B", 200, 220),
			),
			makeMultiElementLine(680,
				textAt("C", 15, 35),
				textAt("D", 195, 225),
			),
		},
	}

	tables, err := d.Detect(page)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}

	b := tables[0].Bounds
	if b.X1 != 10 {
		t.Errorf("Bounds.X1 = %v, want 10", b.X1)
	}
	if b.X2 != 225 {
		t.Errorf("Bounds.X2 = %v, want 225", b.X2)
	}
}

func TestLineDetectorImplementsDetector(t *testing.T) {
	var _ Detector = newLineDetector()
}

// makeLine creates a TextLine with a single element starting at the given
// x-coordinate range and y position.
func makeLine(text string, x2, y float64) layout.TextLine {
	const x1 = 10.0

	elem := pdf.TextElement{
		Text:     text,
		FontName: "Arial",
		FontSize: 12,
		Bounds:   pdf.Rect{X1: x1, Y1: y, X2: x2, Y2: y + 12},
	}

	return layout.TextLine{
		Text:     text,
		Elements: []pdf.TextElement{elem},
		Bounds:   pdf.Rect{X1: x1, Y1: y, X2: x2, Y2: y + 12},
		FontName: "Arial",
		FontSize: 12,
	}
}

func textAt(text string, x1, x2 float64) pdf.TextElement {
	return pdf.TextElement{
		Text:     text,
		FontName: "Arial",
		FontSize: 12,
		Bounds:   pdf.Rect{X1: x1, Y1: 0, X2: x2, Y2: 12},
	}
}

func makeMultiElementLine(y float64, elems ...pdf.TextElement) layout.TextLine {
	for i := range elems {
		elems[i].Bounds.Y1 = y
		elems[i].Bounds.Y2 = y + 12
	}

	parts := make([]string, 0, len(elems))
	minX := elems[0].Bounds.X1
	maxX := elems[0].Bounds.X2

	for _, e := range elems {
		parts = append(parts, e.Text)
		if e.Bounds.X1 < minX {
			minX = e.Bounds.X1
		}
		if e.Bounds.X2 > maxX {
			maxX = e.Bounds.X2
		}
	}

	text := strings.Join(parts, " ")

	return layout.TextLine{
		Text:     text,
		Elements: elems,
		Bounds:   pdf.Rect{X1: minX, Y1: y, X2: maxX, Y2: y + 12},
		FontName: "Arial",
		FontSize: 12,
	}
}
