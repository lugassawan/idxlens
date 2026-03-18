package domain

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

func TestBilingualRouterDetectLanguage(t *testing.T) {
	router := NewBilingualRouter()

	tests := []struct {
		name string
		text string
		want Language
	}{
		{
			name: "pure Indonesian text",
			text: "Pendapatan dari penjualan dan jasa untuk periode yang berakhir pada tanggal",
			want: LangIndonesian,
		},
		{
			name: "pure English text",
			text: "Revenue from sales and services for the period with total assets",
			want: LangEnglish,
		},
		{
			name: "empty string",
			text: "",
			want: LangUnknown,
		},
		{
			name: "whitespace only",
			text: "   \t\n  ",
			want: LangUnknown,
		},
		{
			name: "short ambiguous text with no markers",
			text: "12.345.678",
			want: LangUnknown,
		},
		{
			name: "single marker below threshold",
			text: "dan",
			want: LangUnknown,
		},
		{
			name: "Indonesian with financial terms",
			text: "Jumlah aset dan liabilitas dari laporan keuangan",
			want: LangIndonesian,
		},
		{
			name: "English with financial terms",
			text: "Total assets and liabilities from the financial statements",
			want: LangEnglish,
		},
		{
			name: "equal marker counts",
			text: "dan atau and the",
			want: LangUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.DetectLanguage(tt.text)
			if got != tt.want {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestBilingualRouterIsBilingual(t *testing.T) {
	router := NewBilingualRouter()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "bilingual content",
			text: "Pendapatan dan jasa Revenue and services from the company untuk periode yang berakhir",
			want: true,
		},
		{
			name: "Indonesian only",
			text: "Pendapatan dari penjualan dan jasa untuk periode yang berakhir",
			want: false,
		},
		{
			name: "English only",
			text: "Revenue from sales and services for the period with total",
			want: false,
		},
		{
			name: "empty string",
			text: "",
			want: false,
		},
		{
			name: "numbers only",
			text: "12.345.678 99.999",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.IsBilingual(tt.text)
			if got != tt.want {
				t.Errorf("IsBilingual(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestBilingualRouterSplitBilingual(t *testing.T) {
	router := NewBilingualRouter()

	tests := []struct {
		name        string
		lines       []layout.TextLine
		wantIDCount int
		wantENCount int
		wantIDTexts []string
		wantENTexts []string
	}{
		{
			name:        "empty lines",
			lines:       nil,
			wantIDCount: 0,
			wantENCount: 0,
		},
		{
			name: "left right split",
			lines: []layout.TextLine{
				{Text: "Pendapatan", Bounds: pdf.Rect{X1: 50, Y1: 100, X2: 200, Y2: 120}},
				{Text: "Revenue", Bounds: pdf.Rect{X1: 400, Y1: 100, X2: 550, Y2: 120}},
				{Text: "Beban usaha", Bounds: pdf.Rect{X1: 50, Y1: 130, X2: 200, Y2: 150}},
				{Text: "Operating expenses", Bounds: pdf.Rect{X1: 400, Y1: 130, X2: 580, Y2: 150}},
			},
			wantIDCount: 2,
			wantENCount: 2,
			wantIDTexts: []string{"Pendapatan", "Beban usaha"},
			wantENTexts: []string{"Revenue", "Operating expenses"},
		},
		{
			name: "lines clustered on left with one outlier on right",
			lines: []layout.TextLine{
				{Text: "Line A", Bounds: pdf.Rect{X1: 50, Y1: 100, X2: 150, Y2: 120}},
				{Text: "Line B", Bounds: pdf.Rect{X1: 50, Y1: 130, X2: 150, Y2: 150}},
				{Text: "Line C", Bounds: pdf.Rect{X1: 400, Y1: 160, X2: 550, Y2: 180}},
			},
			wantIDCount: 2,
			wantENCount: 1,
			wantIDTexts: []string{"Line A", "Line B"},
			wantENTexts: []string{"Line C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indonesian, english := router.SplitBilingual(tt.lines)

			if len(indonesian) != tt.wantIDCount {
				t.Errorf("SplitBilingual() indonesian count = %d, want %d", len(indonesian), tt.wantIDCount)
			}

			if len(english) != tt.wantENCount {
				t.Errorf("SplitBilingual() english count = %d, want %d", len(english), tt.wantENCount)
			}

			if tt.wantIDTexts != nil {
				for i, want := range tt.wantIDTexts {
					if i < len(indonesian) && indonesian[i].Text != want {
						t.Errorf("indonesian[%d].Text = %q, want %q", i, indonesian[i].Text, want)
					}
				}
			}

			if tt.wantENTexts != nil {
				for i, want := range tt.wantENTexts {
					if i < len(english) && english[i].Text != want {
						t.Errorf("english[%d].Text = %q, want %q", i, english[i].Text, want)
					}
				}
			}
		})
	}
}

func TestCountMarkers(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		markers []string
		want    int
	}{
		{
			name:    "exact word matches",
			text:    "dan atau yang",
			markers: []string{"dan", "atau", "yang"},
			want:    3,
		},
		{
			name:    "case insensitive",
			text:    "DAN Atau YANG",
			markers: []string{"dan", "atau", "yang"},
			want:    3,
		},
		{
			name:    "no partial matches",
			text:    "pendanaan autan",
			markers: []string{"dan", "aut"},
			want:    0,
		},
		{
			name:    "markers with punctuation",
			text:    "dan, atau. yang;",
			markers: []string{"dan", "atau", "yang"},
			want:    3,
		},
		{
			name:    "empty text",
			text:    "",
			markers: []string{"dan"},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countMarkers(tt.text, tt.markers)
			if got != tt.want {
				t.Errorf("countMarkers(%q, ...) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}
