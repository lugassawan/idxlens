package table

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

func TestDetectorDetect(t *testing.T) {
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
			name: "simple two column table",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Name", 10, 80),
						textAt("Value", 200, 260),
					),
					makeMultiElementLine(686,
						textAt("Alpha", 10, 60),
						textAt("100", 230, 260),
					),
					makeMultiElementLine(672,
						textAt("Beta", 10, 50),
						textAt("200", 230, 260),
					),
				},
			},
			wantTables:  1,
			wantColumns: 2,
			wantRows:    3,
			wantHeaders: []string{"Name", "Value"},
		},
		{
			name: "financial table with right-aligned numbers",
			page: layout.LayoutPage{
				Number: 2,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Description", 10, 100),
						textAt("2024", 200, 260),
						textAt("2023", 350, 410),
					),
					makeMultiElementLine(686,
						textAt("Revenue", 10, 70),
						textAt("1,500,000", 195, 260),
						textAt("1,200,000", 345, 410),
					),
					makeMultiElementLine(672,
						textAt("Expenses", 10, 75),
						textAt("800,000", 205, 260),
						textAt("750,000", 355, 410),
					),
				},
			},
			wantTables:  1,
			wantColumns: 3,
			wantRows:    3,
			wantHeaders: []string{"Description", "2024", "2023"},
		},
		{
			name: "table with header row",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeMultiElementLine(700,
						textAt("Item", 10, 60),
						textAt("Amount", 200, 260),
					),
					makeMultiElementLine(686,
						textAt("Cash", 10, 45),
						textAt("50,000", 215, 260),
					),
					makeMultiElementLine(672,
						textAt("Receivables", 10, 85),
						textAt("30,000", 215, 260),
					),
				},
			},
			wantTables:  1,
			wantColumns: 2,
			wantRows:    3,
			wantHeaders: []string{"Item", "Amount"},
		},
		{
			name: "page with no tables returns empty result",
			page: layout.LayoutPage{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Lines: []layout.TextLine{
					makeLine("This is just a paragraph.", 300, 700),
					makeLine("Another paragraph line.", 280, 686),
				},
			},
			wantTables: 0,
		},
		{
			name: "single row insufficient for table",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDetector()
			tables, err := d.Detect(tt.page)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			assertTableResult(t, tables, tt.page.Number, tt.wantTables, tt.wantColumns, tt.wantRows, tt.wantHeaders)
		})
	}
}

func TestDetectorImplementsInterface(t *testing.T) {
	d := NewDetector()
	if _, err := d.Detect(layout.LayoutPage{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectorMultipleTables(t *testing.T) {
	// Two table groups separated by a large vertical gap
	page := layout.LayoutPage{
		Number: 1,
		Size:   pdf.PageSize{Width: 612, Height: 792},
		Lines: []layout.TextLine{
			// First table group
			makeMultiElementLine(700,
				textAt("A", 10, 50),
				textAt("B", 200, 260),
			),
			makeMultiElementLine(686,
				textAt("a1", 10, 40),
				textAt("b1", 220, 260),
			),
			// Large gap (> 2.5x median gap of 14)
			// Second table group
			makeMultiElementLine(500,
				textAt("X", 10, 50),
				textAt("Y", 200, 260),
			),
			makeMultiElementLine(486,
				textAt("x1", 10, 40),
				textAt("y1", 220, 260),
			),
		},
	}

	d := NewDetector()
	tables, err := d.Detect(page)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(tables))
	}

	if tables[0].Headers[0] != "A" {
		t.Errorf("first table header[0] = %q, want %q", tables[0].Headers[0], "A")
	}

	if tables[1].Headers[0] != "X" {
		t.Errorf("second table header[0] = %q, want %q", tables[1].Headers[0], "X")
	}
}

func TestGridBuilderMergedCells(t *testing.T) {
	columns := []Column{
		{Index: 0, X1: 10, X2: 100, Alignment: "left"},
		{Index: 1, X1: 150, X2: 250, Alignment: "right"},
		{Index: 2, X1: 300, X2: 400, Alignment: "right"},
	}

	// "Spanning Value" element at 140-410 overlaps columns 1 and 2
	lines := []layout.TextLine{
		makeMultiElementLine(700,
			textAt("Label", 10, 90),
			textAt("Spanning Value", 140, 410),
		),
		makeMultiElementLine(686,
			textAt("Row 2", 10, 60),
			textAt("100", 200, 250),
			textAt("200", 350, 400),
		),
	}

	gb := newGridBuilder()
	tbl := gb.Build(lines, columns, 1)

	// First row should have a merged cell spanning columns 1 and 2
	var foundMerged bool
	for _, cell := range tbl.Rows[0].Cells {
		if cell.Merged {
			foundMerged = true
			if cell.ColSpan != 2 {
				t.Errorf("merged cell ColSpan = %d, want 2", cell.ColSpan)
			}
			if cell.Col != 1 {
				t.Errorf("merged cell Col = %d, want 1", cell.Col)
			}
		}
	}

	if !foundMerged {
		t.Error("expected merged cell in first row, found none")
	}
}

func TestGridBuilderEmptyLines(t *testing.T) {
	gb := newGridBuilder()
	columns := []Column{
		{Index: 0, X1: 10, X2: 100, Alignment: "left"},
		{Index: 1, X1: 200, X2: 300, Alignment: "right"},
	}

	tbl := gb.Build(nil, columns, 1)

	if len(tbl.Rows) != 0 {
		t.Errorf("got %d rows, want 0", len(tbl.Rows))
	}

	if tbl.Headers != nil {
		t.Errorf("got headers %v, want nil", tbl.Headers)
	}
}
