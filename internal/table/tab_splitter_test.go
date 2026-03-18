package table

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

func TestHasTabSeparators(t *testing.T) {
	tests := []struct {
		name  string
		lines []layout.TextLine
		want  bool
	}{
		{
			name:  "empty lines",
			lines: nil,
			want:  false,
		},
		{
			name: "no tabs",
			lines: []layout.TextLine{
				{Text: "Hello World"},
				{Text: "Another line"},
			},
			want: false,
		},
		{
			name: "all lines have tabs",
			lines: []layout.TextLine{
				{Text: "Kas\t25,305,031\t29,315,878\tCash"},
				{Text: "Dana\t0\t0\tRestricted funds"},
			},
			want: true,
		},
		{
			name: "majority have tabs",
			lines: []layout.TextLine{
				{Text: "Header without tabs"},
				{Text: "Kas\t25,305,031\tCash"},
				{Text: "Dana\t0\tFunds"},
				{Text: "Giro\t47,768\tCurrent"},
			},
			want: true,
		},
		{
			name: "minority have tabs",
			lines: []layout.TextLine{
				{Text: "No tabs here"},
				{Text: "Still no tabs"},
				{Text: "One\ttab"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTabSeparators(tt.lines)
			if got != tt.want {
				t.Errorf("hasTabSeparators() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitTabColumns(t *testing.T) {
	tests := []struct {
		name       string
		lines      []layout.TextLine
		wantCount  int
		wantNil    bool
		wantAligns []string
	}{
		{
			name:    "no tab lines",
			lines:   []layout.TextLine{{Text: "no tabs"}},
			wantNil: true,
		},
		{
			name: "single tab gives two columns",
			lines: []layout.TextLine{
				{Text: "Label\tValue"},
			},
			wantCount:  2,
			wantAligns: []string{"left", "right"},
		},
		{
			name: "bilingual four columns",
			lines: []layout.TextLine{
				{Text: "Kas\t25,305,031\t29,315,878\tCash"},
				{Text: "Dana\t0\t0\tRestricted funds"},
			},
			wantCount:  4,
			wantAligns: []string{"left", "right", "right", "left"},
		},
		{
			name: "three columns",
			lines: []layout.TextLine{
				{Text: "Kas\t25,305,031\t29,315,878"},
			},
			wantCount:  3,
			wantAligns: []string{"left", "right", "right"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := splitTabColumns(tt.lines)

			if tt.wantNil {
				if columns != nil {
					t.Fatalf("got %d columns, want nil", len(columns))
				}
				return
			}

			if len(columns) != tt.wantCount {
				t.Fatalf("got %d columns, want %d", len(columns), tt.wantCount)
			}

			for i, want := range tt.wantAligns {
				if columns[i].Alignment != want {
					t.Errorf("column[%d].Alignment = %q, want %q", i, columns[i].Alignment, want)
				}
			}

			for i, col := range columns {
				if col.Index != i {
					t.Errorf("column[%d].Index = %d, want %d", i, col.Index, i)
				}
			}
		})
	}
}

func TestColumnAlignment(t *testing.T) {
	tests := []struct {
		name  string
		index int
		total int
		want  string
	}{
		{name: "first of two", index: 0, total: 2, want: "left"},
		{name: "last of two", index: 1, total: 2, want: "right"},
		{name: "first of four", index: 0, total: 4, want: "left"},
		{name: "second of four", index: 1, total: 4, want: "right"},
		{name: "third of four", index: 2, total: 4, want: "right"},
		{name: "last of four", index: 3, total: 4, want: "left"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := columnAlignment(tt.index, tt.total)
			if got != tt.want {
				t.Errorf("columnAlignment(%d, %d) = %q, want %q", tt.index, tt.total, got, tt.want)
			}
		})
	}
}

func TestBuildTabRows(t *testing.T) {
	columns := splitTabColumns([]layout.TextLine{
		{Text: "Kas\t25,305,031\t29,315,878\tCash"},
	})

	lines := []layout.TextLine{
		makeTabLine("Kas\t25,305,031\t29,315,878\tCash", 700),
		makeTabLine("Dana\t0\t0\tRestricted funds", 686),
	}

	rows := buildTabRows(lines, columns)

	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// First row
	if len(rows[0].Cells) != 4 {
		t.Fatalf("row 0: got %d cells, want 4", len(rows[0].Cells))
	}
	if rows[0].Cells[0].Text != "Kas" {
		t.Errorf("row 0 cell 0 = %q, want %q", rows[0].Cells[0].Text, "Kas")
	}
	if rows[0].Cells[1].Text != "25,305,031" {
		t.Errorf("row 0 cell 1 = %q, want %q", rows[0].Cells[1].Text, "25,305,031")
	}
	if rows[0].Cells[2].Text != "29,315,878" {
		t.Errorf("row 0 cell 2 = %q, want %q", rows[0].Cells[2].Text, "29,315,878")
	}
	if rows[0].Cells[3].Text != "Cash" {
		t.Errorf("row 0 cell 3 = %q, want %q", rows[0].Cells[3].Text, "Cash")
	}

	// Check column indices
	for i, cell := range rows[0].Cells {
		if cell.Col != i {
			t.Errorf("row 0 cell %d Col = %d, want %d", i, cell.Col, i)
		}
	}
}

func TestSplitTabLineSkipsEmpty(t *testing.T) {
	columns := splitTabColumns([]layout.TextLine{
		{Text: "A\tB\tC"},
	})

	line := makeTabLine("Label\t\tValue", 700)
	cells := splitTabLine(line, columns, 0)

	// Empty middle field should be skipped
	if len(cells) != 2 {
		t.Fatalf("got %d cells, want 2", len(cells))
	}

	if cells[0].Text != "Label" {
		t.Errorf("cell 0 = %q, want %q", cells[0].Text, "Label")
	}

	if cells[1].Text != "Value" {
		t.Errorf("cell 1 = %q, want %q", cells[1].Text, "Value")
	}
}

func TestSplitTabLineTrimsWhitespace(t *testing.T) {
	columns := splitTabColumns([]layout.TextLine{
		{Text: "A\tB"},
	})

	line := makeTabLine("  Label  \t  100  ", 700)
	cells := splitTabLine(line, columns, 0)

	if len(cells) != 2 {
		t.Fatalf("got %d cells, want 2", len(cells))
	}

	if cells[0].Text != "Label" {
		t.Errorf("cell 0 = %q, want %q", cells[0].Text, "Label")
	}

	if cells[1].Text != "100" {
		t.Errorf("cell 1 = %q, want %q", cells[1].Text, "100")
	}
}

// makeTabLine creates a TextLine with tab-separated text for testing.
func makeTabLine(text string, y float64) layout.TextLine {
	return layout.TextLine{
		Text:     text,
		Elements: nil,
		Bounds:   pdf.Rect{X1: 10, Y1: y, X2: 500, Y2: y + 12},
		FontName: "Arial",
		FontSize: 12,
	}
}
