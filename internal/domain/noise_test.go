package domain

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/table"
)

func TestIsGarbledText(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  bool
	}{
		{
			name:  "encoded characters from PDF font issues",
			label: "K _ G > i > S & S ^ > * K l m K _",
			want:  true,
		},
		{
			name:  "heavy symbol content",
			label: "> ^ ~ [ ] > * > < |",
			want:  true,
		},
		{
			name:  "normal financial label indonesian",
			label: "Kas dan Setara Kas",
			want:  false,
		},
		{
			name:  "normal financial label english",
			label: "Cash and Cash Equivalents",
			want:  false,
		},
		{
			name:  "label with parentheses is fine",
			label: "Laba (Rugi) Bersih",
			want:  false,
		},
		{
			name:  "label with few symbols below threshold",
			label: "Revenue - Net",
			want:  false,
		},
		{
			name:  "empty string",
			label: "",
			want:  false,
		},
		{
			name:  "mostly symbols",
			label: ">>>^^^~~~",
			want:  true,
		},
		{
			name:  "mixed garbled with some text",
			label: "A > B ^ C > D ~ E",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGarbledText(tt.label); got != tt.want {
				t.Errorf("isGarbledText(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

func TestIsPageRefValue(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]float64
		unit   string
		want   bool
	}{
		{
			name:   "page number in millions context",
			values: map[string]float64{"2025-12-31": 294},
			unit:   "millions",
			want:   true,
		},
		{
			name:   "page numbers in billions context",
			values: map[string]float64{"2025-12-31": 42, "2024-12-31": 38},
			unit:   "billions",
			want:   true,
		},
		{
			name:   "real financial value in millions",
			values: map[string]float64{"2025-12-31": 1234567},
			unit:   "millions",
			want:   false,
		},
		{
			name:   "mixed values one large",
			values: map[string]float64{"2025-12-31": 100, "2024-12-31": 50000},
			unit:   "millions",
			want:   false,
		},
		{
			name:   "zero values are not page refs",
			values: map[string]float64{"2025-12-31": 0},
			unit:   "millions",
			want:   false,
		},
		{
			name:   "empty values",
			values: map[string]float64{},
			unit:   "millions",
			want:   false,
		},
		{
			name:   "no unit set returns false",
			values: map[string]float64{"2025-12-31": 42},
			unit:   "",
			want:   false,
		},
		{
			name:   "thousands unit does not filter",
			values: map[string]float64{"2025-12-31": 42},
			unit:   "thousands",
			want:   false,
		},
		{
			name:   "fractional value is not a page ref",
			values: map[string]float64{"2025-12-31": 42.5},
			unit:   "millions",
			want:   false,
		},
		{
			name:   "negative page-like number",
			values: map[string]float64{"2025-12-31": -294},
			unit:   "millions",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPageRefValue(tt.values, tt.unit); got != tt.want {
				t.Errorf("isPageRefValue(%v, %q) = %v, want %v",
					tt.values, tt.unit, got, tt.want)
			}
		})
	}
}

func TestFilterPageReferences(t *testing.T) {
	tests := []struct {
		name      string
		items     []LineItem
		unit      string
		wantCount int
	}{
		{
			name: "removes page references keeps financial items",
			items: []LineItem{
				{Key: "cash", Label: "Cash", Values: map[string]float64{"2025-12-31": 1234567}},
				{Key: "", Label: "Terealisasi", Values: map[string]float64{"2025-12-31": 294}},
				{Key: "revenue", Label: "Revenue", Values: map[string]float64{"2025-12-31": 500000}},
			},
			unit:      "millions",
			wantCount: 2,
		},
		{
			name: "no filtering when unit is empty",
			items: []LineItem{
				{Key: "", Label: "Some item", Values: map[string]float64{"2025-12-31": 42}},
			},
			unit:      "",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterPageReferences(tt.items, tt.unit)
			if len(got) != tt.wantCount {
				t.Errorf("filterPageReferences() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestIsNonFinancialTable(t *testing.T) {
	tests := []struct {
		name string
		tbl  table.Table
		want bool
	}{
		{
			name: "gcg table by header",
			tbl: table.Table{
				Headers: []string{"Good Corporate Governance"},
				Rows:    []table.Row{makeRow(0, "Komite Audit", "4")},
			},
			want: true,
		},
		{
			name: "anti-fraud table",
			tbl: table.Table{
				Headers: []string{"Anti-Fraud Strategy"},
				Rows:    []table.Row{makeRow(0, "Preventif", "100%")},
			},
			want: true,
		},
		{
			name: "conflict of interest in rows",
			tbl: table.Table{
				Headers: []string{"Governance"},
				Rows: []table.Row{
					makeRow(0, "Benturan Kepentingan", "Nihil"),
				},
			},
			want: true,
		},
		{
			name: "financial table",
			tbl: table.Table{
				Headers: []string{"31 Desember 2025"},
				Rows:    []table.Row{makeRow(0, "Kas dan Setara Kas", "1.000.000")},
			},
			want: false,
		},
		{
			name: "empty table",
			tbl: table.Table{
				Headers: []string{},
				Rows:    []table.Row{},
			},
			want: false,
		},
		{
			name: "whistleblowing in rows",
			tbl: table.Table{
				Rows: []table.Row{
					makeRow(0, "Whistleblowing System", "Active"),
				},
			},
			want: true,
		},
		{
			name: "risk management table",
			tbl: table.Table{
				Headers: []string{"Manajemen Risiko"},
				Rows:    []table.Row{makeRow(0, "Risiko Kredit", "Low")},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonFinancialTable(tt.tbl); got != tt.want {
				t.Errorf("isNonFinancialTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasFinancialSectionHeader(t *testing.T) {
	tests := []struct {
		name string
		tbl  table.Table
		want bool
	}{
		{
			name: "ikhtisar keuangan in page text",
			tbl: table.Table{
				PageText: []string{"Ikhtisar Keuangan"},
			},
			want: true,
		},
		{
			name: "financial highlights in header",
			tbl: table.Table{
				Headers: []string{"Financial Highlights"},
			},
			want: true,
		},
		{
			name: "statement of financial position",
			tbl: table.Table{
				PageText: []string{"Laporan Posisi Keuangan"},
			},
			want: true,
		},
		{
			name: "no financial header",
			tbl: table.Table{
				PageText: []string{"Good Corporate Governance"},
				Headers:  []string{"Board of Commissioners"},
			},
			want: false,
		},
		{
			name: "balance sheet header",
			tbl: table.Table{
				Headers: []string{"Neraca Konsolidasi"},
			},
			want: true,
		},
		{
			name: "income statement",
			tbl: table.Table{
				PageText: []string{"Statement of Profit or Loss"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasFinancialSectionHeader(tt.tbl); got != tt.want {
				t.Errorf("hasFinancialSectionHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterByFinancialContent(t *testing.T) {
	tests := []struct {
		name      string
		tables    []table.Table
		wantCount int
		wantPages []int
	}{
		{
			name: "keeps financial tables skips governance",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{
					PageNum:  10,
					PageText: []string{"Ikhtisar Keuangan"},
					Rows:     []table.Row{makeRow(0, "Kas", "1.000.000")},
				},
				{
					PageNum: 11,
					Rows:    []table.Row{makeRow(0, "Piutang", "500.000")},
				},
				{
					PageNum: 50,
					Headers: []string{"Good Corporate Governance"},
					Rows:    []table.Row{makeRow(0, "Komite Audit", "4")},
				},
				{
					PageNum: 60,
					Rows:    []table.Row{makeRow(0, "Attendance", "12")},
				},
			},
			wantCount: 2,
			wantPages: []int{10, 11},
		},
		{
			name: "falls back to numeric content when no financial headers",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{PageNum: 2, Rows: []table.Row{makeRow(0, "Cash", "1.000")}},
			},
			wantCount: 1,
			wantPages: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterByFinancialContent(tt.tables)

			if len(result) != tt.wantCount {
				t.Fatalf("count = %d, want %d", len(result), tt.wantCount)
			}

			for i, wantPage := range tt.wantPages {
				if result[i].PageNum != wantPage {
					t.Errorf("result[%d].PageNum = %d, want %d",
						i, result[i].PageNum, wantPage)
				}
			}
		})
	}
}

func TestFilterAnnualReportTables(t *testing.T) {
	tests := []struct {
		name      string
		tables    []table.Table
		wantCount int
	}{
		{
			name: "uses xbrl markers when available",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{
					PageNum:  4,
					PageText: []string{"[4220000] Statement of financial position"},
					Rows:     []table.Row{makeRow(0, "Cash", "100")},
				},
				{PageNum: 5, Rows: []table.Row{makeRow(0, "Receivables", "200")}},
			},
			wantCount: 2,
		},
		{
			name: "falls back to financial content filter without xbrl",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{
					PageNum:  10,
					PageText: []string{"Ikhtisar Keuangan"},
					Rows:     []table.Row{makeRow(0, "Kas", "1.000")},
				},
				{
					PageNum: 50,
					Headers: []string{"Good Corporate Governance"},
					Rows:    []table.Row{makeRow(0, "GCG", "4")},
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterAnnualReportTables(tt.tables)

			if len(result) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestCollapseSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal text", "hello world", "hello world"},
		{"multiple spaces", "hello   world", "hello world"},
		{"tabs and spaces", "hello\t\t  world", "hello world"},
		{"empty", "", ""},
		{"only spaces", "   ", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collapseSpaces(tt.input); got != tt.want {
				t.Errorf("collapseSpaces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
