package layout

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/pdf"
)

func TestAnalyzerAnalyze(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name       string
		page       pdf.Page
		wantLines  int
		wantText   []string
		wantRegion int
		wantErr    bool
	}{
		{
			name: "empty page returns zero lines",
			page: pdf.Page{
				Number:   1,
				Size:     pdf.PageSize{Width: 612, Height: 792},
				Elements: nil,
			},
			wantLines:  0,
			wantRegion: 0,
		},
		{
			name: "single element page",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{
						Text:     "Hello",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 10, Y1: 700, X2: 50, Y2: 712},
					},
				},
			},
			wantLines:  1,
			wantText:   []string{"Hello"},
			wantRegion: 1,
		},
		{
			name: "multiple elements same line sorted by X with column gap",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{
						Text:     "World",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 100, Y1: 700, X2: 140, Y2: 712},
					},
					{
						Text:     "Hello",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 10, Y1: 700, X2: 50, Y2: 712},
					},
				},
			},
			wantLines:  1,
			wantText:   []string{"Hello\tWorld"},
			wantRegion: 1,
		},
		{
			name: "two lines at different Y positions ordered top to bottom",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{
						Text:     "Bottom",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 10, Y1: 600, X2: 60, Y2: 612},
					},
					{Text: "Top", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 10, Y1: 700, X2: 40, Y2: 712}},
				},
			},
			wantLines:  2,
			wantText:   []string{"Top", "Bottom"},
			wantRegion: 1,
		},
		{
			name: "elements with gaps get spaces inserted",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{
						Text:     "Hello",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 10, Y1: 700, X2: 50, Y2: 712},
					},
					{
						Text:     "World",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 60, Y1: 700, X2: 100, Y2: 712},
					},
				},
			},
			wantLines: 1,
			wantText:  []string{"Hello World"},
		},
		{
			name: "adjacent elements without gap are joined without space",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{Text: "Hel", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 10, Y1: 700, X2: 34, Y2: 712}},
					{Text: "lo", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 34, Y1: 700, X2: 50, Y2: 712}},
				},
			},
			wantLines: 1,
			wantText:  []string{"Hello"},
		},
		{
			name: "elements with slight Y variance cluster into same line",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{
						Text:     "Hello",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 10, Y1: 700, X2: 50, Y2: 712},
					},
					{
						Text:     "World",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 60, Y1: 701, X2: 100, Y2: 713},
					},
				},
			},
			wantLines: 1,
			wantText:  []string{"Hello World"},
		},
		{
			name: "different font sizes create separate regions",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{
						Text:     "Title",
						FontName: "Arial-Bold",
						FontSize: 18,
						Bounds:   pdf.Rect{X1: 10, Y1: 750, X2: 80, Y2: 768},
					},
					{
						Text:     "Body text",
						FontName: "Arial",
						FontSize: 12,
						Bounds:   pdf.Rect{X1: 10, Y1: 700, X2: 90, Y2: 712},
					},
				},
			},
			wantLines:  2,
			wantText:   []string{"Title", "Body text"},
			wantRegion: 2,
		},
		{
			name: "bilingual multi-column layout uses tabs at column boundaries",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{Text: "Kas", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 50, Y1: 500, X2: 70, Y2: 508}},
					{Text: "25,305,031", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 200, Y1: 500, X2: 250, Y2: 508}},
					{Text: "29,315,878", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 300, Y1: 500, X2: 350, Y2: 508}},
					{Text: "Cash", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 450, Y1: 500, X2: 470, Y2: 508}},
					{Text: "Dana yang dibatasi", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 50, Y1: 480, X2: 140, Y2: 488}},
					{Text: "0", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 240, Y1: 480, X2: 245, Y2: 488}},
					{Text: "0", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 340, Y1: 480, X2: 345, Y2: 488}},
					{Text: "Restricted funds", FontName: "Arial", FontSize: 8, Bounds: pdf.Rect{X1: 450, Y1: 480, X2: 520, Y2: 488}},
				},
			},
			wantLines: 2,
			wantText: []string{
				"Kas\t25,305,031\t29,315,878\tCash",
				"Dana yang dibatasi\t0\t0\tRestricted funds",
			},
		},
		{
			name: "small gaps within words use spaces not tabs",
			page: pdf.Page{
				Number: 1,
				Size:   pdf.PageSize{Width: 612, Height: 792},
				Elements: []pdf.TextElement{
					{Text: "Hello", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 10, Y1: 700, X2: 50, Y2: 712}},
					{Text: "World", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 58, Y1: 700, X2: 98, Y2: 712}},
				},
			},
			wantLines: 1,
			wantText:  []string{"Hello World"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := a.Analyze(tt.page)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Analyze() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(result.Lines) != tt.wantLines {
				t.Fatalf("got %d lines, want %d", len(result.Lines), tt.wantLines)
			}

			for i, wantText := range tt.wantText {
				if result.Lines[i].Text != wantText {
					t.Errorf("line[%d].Text = %q, want %q", i, result.Lines[i].Text, wantText)
				}
			}

			if tt.wantRegion > 0 && len(result.Regions) != tt.wantRegion {
				t.Errorf("got %d regions, want %d", len(result.Regions), tt.wantRegion)
			}

			if result.Number != tt.page.Number {
				t.Errorf("Number = %d, want %d", result.Number, tt.page.Number)
			}

			if result.Size != tt.page.Size {
				t.Errorf("Size = %v, want %v", result.Size, tt.page.Size)
			}
		})
	}
}

func TestAnalyzerAnalyzeBoundsUnion(t *testing.T) {
	a := NewAnalyzer()

	page := pdf.Page{
		Number: 1,
		Size:   pdf.PageSize{Width: 612, Height: 792},
		Elements: []pdf.TextElement{
			{Text: "Hello", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 10, Y1: 700, X2: 50, Y2: 712}},
			{Text: "World", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 55, Y1: 698, X2: 95, Y2: 714}},
		},
	}

	result, err := a.Analyze(page)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if len(result.Lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(result.Lines))
	}

	b := result.Lines[0].Bounds
	if b.X1 != 10 || b.X2 != 95 {
		t.Errorf("bounds X = [%v, %v], want [10, 95]", b.X1, b.X2)
	}

	if b.Y1 != 698 || b.Y2 != 714 {
		t.Errorf("bounds Y = [%v, %v], want [698, 714]", b.Y1, b.Y2)
	}
}

func TestAnalyzerAnalyzeLineElements(t *testing.T) {
	a := NewAnalyzer()

	page := pdf.Page{
		Number: 1,
		Size:   pdf.PageSize{Width: 612, Height: 792},
		Elements: []pdf.TextElement{
			{Text: "World", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 50, Y1: 700, X2: 90, Y2: 712}},
			{Text: "Hello", FontName: "Arial", FontSize: 12, Bounds: pdf.Rect{X1: 10, Y1: 700, X2: 45, Y2: 712}},
		},
	}

	result, err := a.Analyze(page)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if len(result.Lines[0].Elements) != 2 {
		t.Fatalf("got %d elements, want 2", len(result.Lines[0].Elements))
	}

	if result.Lines[0].Elements[0].Text != "Hello" {
		t.Errorf("first element = %q, want %q", result.Lines[0].Elements[0].Text, "Hello")
	}

	if result.Lines[0].Elements[1].Text != "World" {
		t.Errorf("second element = %q, want %q", result.Lines[0].Elements[1].Text, "World")
	}
}
