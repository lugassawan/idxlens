package domain

import (
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
)

func makeTestPages(texts []string) []layout.LayoutPage {
	pages := make([]layout.LayoutPage, 0, len(texts))
	for i, text := range texts {
		lines := strings.Split(text, "\n")
		textLines := make([]layout.TextLine, 0, len(lines))
		for _, line := range lines {
			textLines = append(textLines, layout.TextLine{Text: line})
		}
		pages = append(pages, layout.LayoutPage{
			Number: i + 1,
			Lines:  textLines,
		})
	}
	return pages
}

func TestHeuristicClassifierClassify(t *testing.T) {
	tests := []struct {
		name       string
		pageTexts  []string
		wantType   DocType
		wantLang   string
		wantMinCon float64
	}{
		{
			name:       "indonesian balance sheet",
			pageTexts:  []string{"PT EXAMPLE Tbk\nLAPORAN POSISI KEUANGAN\n31 Desember 2023"},
			wantType:   DocTypeBalanceSheet,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "english balance sheet",
			pageTexts:  []string{"PT EXAMPLE Tbk\nSTATEMENT OF FINANCIAL POSITION\nAs of December 31, 2023"},
			wantType:   DocTypeBalanceSheet,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:       "indonesian income statement",
			pageTexts:  []string{"PT EXAMPLE Tbk\nLAPORAN LABA RUGI\nUntuk tahun yang berakhir"},
			wantType:   DocTypeIncomeStatement,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "english income statement",
			pageTexts:  []string{"PT EXAMPLE Tbk\nSTATEMENT OF PROFIT OR LOSS\nFor the year ended"},
			wantType:   DocTypeIncomeStatement,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:       "indonesian income statement comprehensive",
			pageTexts:  []string{"LAPORAN LABA RUGI DAN PENGHASILAN KOMPREHENSIF LAIN"},
			wantType:   DocTypeIncomeStatement,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "indonesian cash flow",
			pageTexts:  []string{"PT EXAMPLE Tbk\nLAPORAN ARUS KAS\nUntuk tahun yang berakhir"},
			wantType:   DocTypeCashFlow,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "english cash flow",
			pageTexts:  []string{"PT EXAMPLE Tbk\nSTATEMENT OF CASH FLOWS\nFor the year ended"},
			wantType:   DocTypeCashFlow,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:       "english cash flow alternative",
			pageTexts:  []string{"PT EXAMPLE Tbk\nCASH FLOW STATEMENT\nFor the year ended"},
			wantType:   DocTypeCashFlow,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:       "indonesian equity changes",
			pageTexts:  []string{"PT EXAMPLE Tbk\nLAPORAN PERUBAHAN EKUITAS\nUntuk tahun yang berakhir"},
			wantType:   DocTypeEquityChanges,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "english equity changes",
			pageTexts:  []string{"PT EXAMPLE Tbk\nSTATEMENT OF CHANGES IN EQUITY\nFor the year ended"},
			wantType:   DocTypeEquityChanges,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:       "indonesian notes",
			pageTexts:  []string{"PT EXAMPLE Tbk\nCATATAN ATAS LAPORAN KEUANGAN\n31 Desember 2023"},
			wantType:   DocTypeNotes,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "english notes",
			pageTexts:  []string{"PT EXAMPLE Tbk\nNOTES TO THE FINANCIAL STATEMENTS\nDecember 31, 2023"},
			wantType:   DocTypeNotes,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:       "indonesian auditor report",
			pageTexts:  []string{"LAPORAN AUDITOR INDEPENDEN\nKepada Pemegang Saham"},
			wantType:   DocTypeAuditorReport,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "english auditor report",
			pageTexts:  []string{"INDEPENDENT AUDITOR\nReport on the Financial Statements"},
			wantType:   DocTypeAuditorReport,
			wantLang:   "en",
			wantMinCon: 0.1,
		},
		{
			name:      "unknown document",
			pageTexts: []string{"Some random text that does not match"},
			wantType:  DocTypeUnknown,
		},
		{
			name:     "empty pages",
			wantType: DocTypeUnknown,
		},
		{
			name:       "multiple pages with keywords on second page",
			pageTexts:  []string{"PT EXAMPLE Tbk", "LAPORAN ARUS KAS\nPeriode 2023"},
			wantType:   DocTypeCashFlow,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "mixed case keywords",
			pageTexts:  []string{"laporan Posisi Keuangan"},
			wantType:   DocTypeBalanceSheet,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
		{
			name:       "keywords with extra whitespace",
			pageTexts:  []string{"LAPORAN  POSISI   KEUANGAN"},
			wantType:   DocTypeBalanceSheet,
			wantLang:   "id",
			wantMinCon: 0.1,
		},
	}

	classifier := NewHeuristicClassifier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := makeTestPages(tt.pageTexts)
			got, err := classifier.Classify(pages)
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if tt.wantLang != "" && got.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", got.Language, tt.wantLang)
			}
			if tt.wantMinCon > 0 && got.Confidence < tt.wantMinCon {
				t.Errorf("Confidence = %f, want >= %f", got.Confidence, tt.wantMinCon)
			}
			if tt.wantType == DocTypeUnknown && got.Confidence != 0 {
				t.Errorf("Confidence = %f for unknown, want 0", got.Confidence)
			}
		})
	}
}

func TestHeuristicClassifierConfidence(t *testing.T) {
	classifier := NewHeuristicClassifier()

	t.Run("more keyword matches yield higher confidence", func(t *testing.T) {
		singleMatch := makeTestPages([]string{"NERACA"})
		multiMatch := makeTestPages([]string{"NERACA\nLAPORAN POSISI KEUANGAN"})

		gotSingle, err := classifier.Classify(singleMatch)
		if err != nil {
			t.Fatalf("Classify() error = %v", err)
		}
		gotMulti, err := classifier.Classify(multiMatch)
		if err != nil {
			t.Fatalf("Classify() error = %v", err)
		}

		if gotMulti.Confidence <= gotSingle.Confidence {
			t.Errorf("multi-match confidence (%f) should be > single-match (%f)",
				gotMulti.Confidence, gotSingle.Confidence)
		}
	})
}
