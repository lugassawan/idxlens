package domain

import (
	"slices"
	"testing"
	"time"

	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/lugassawan/idxlens/internal/table"
)

func makeTable(headers []string, rows []table.Row) table.Table {
	cols := make([]table.Column, len(headers))
	for i := range headers {
		cols[i] = table.Column{
			Index:     i,
			X1:        float64(i * 200),
			X2:        float64((i + 1) * 200),
			Alignment: "left",
		}

		if i > 0 {
			cols[i].Alignment = "right"
		}
	}

	return table.Table{
		Headers: headers,
		Rows:    rows,
		Columns: cols,
		Bounds:  pdf.Rect{X1: 0, Y1: 0, X2: float64(len(headers) * 200), Y2: 400},
		PageNum: 2,
	}
}

func makeRow(index int, texts ...string) table.Row {
	cells := make([]table.Cell, len(texts))
	for i, text := range texts {
		cells[i] = table.Cell{Text: text, Row: index, Col: i, Bounds: pdf.Rect{}}
	}

	return table.Row{Index: index, Cells: cells}
}

func TestMapperMapErrors(t *testing.T) {
	m := NewMapper()

	_, err := m.Map(DocTypeBalanceSheet, nil)
	if err == nil {
		t.Error("Map() error = nil, want error for no tables")
	}
}

func TestMapperMapBalanceSheet(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "31 Desember 2023", "31 Desember 2022"},
		[]table.Row{
			makeRow(0, "Kas dan Setara Kas", "1.234.567", "987.654"),
			makeRow(1, "Piutang Usaha", "500.000", "400.000"),
			makeRow(2, "Jumlah Aset Lancar", "1.734.567", "1.387.654"),
		},
	)

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if len(stmt.Periods) != 2 {
		t.Errorf("periods = %d, want 2", len(stmt.Periods))
	}

	wantPeriods := []string{"2023-12-31", "2022-12-31"}
	for i, want := range wantPeriods {
		if i < len(stmt.Periods) && stmt.Periods[i] != want {
			t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
		}
	}

	if stmt.Language != "id" {
		t.Errorf("language = %q, want %q", stmt.Language, "id")
	}

	if len(stmt.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(stmt.Items))
	}

	item := stmt.Items[0]
	if item.Key != "cash_and_equivalents" {
		t.Errorf("item[0].Key = %q, want %q", item.Key, "cash_and_equivalents")
	}

	if item.Confidence < 0.7 {
		t.Errorf("item[0].Confidence = %f, want >= 0.7", item.Confidence)
	}

	if item.Section != "assets" {
		t.Errorf("item[0].Section = %q, want %q", item.Section, "assets")
	}

	period := stmt.Periods[0]
	if val, ok := item.Values[period]; !ok || val != 1234567 {
		t.Errorf("item[0].Values[%q] = %v, want 1234567", period, val)
	}

	if !stmt.Items[2].IsSubtotal {
		t.Error("item[2].IsSubtotal = false, want true")
	}
}

func TestMapperMapUnmatchedRows(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "31 Desember 2023"},
		[]table.Row{makeRow(0, "Unknown Custom Line Item XYZ", "100.000")},
	)

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if len(stmt.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(stmt.Items))
	}

	if stmt.Items[0].Key != "" {
		t.Errorf("item[0].Key = %q, want empty", stmt.Items[0].Key)
	}

	if stmt.Items[0].Confidence != 0 {
		t.Errorf("item[0].Confidence = %f, want 0", stmt.Items[0].Confidence)
	}
}

func TestMapperMapPeriodDetection(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "December 31, 2023", "December 31, 2022"},
		[]table.Row{makeRow(0, "Cash and Cash Equivalents", "1.234.567", "987.654")},
	)

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if stmt.Language != "en" {
		t.Errorf("language = %q, want %q", stmt.Language, "en")
	}

	if len(stmt.Periods) != 2 {
		t.Errorf("periods = %d, want 2", len(stmt.Periods))
	}

	if stmt.Items[0].Key != "cash_and_equivalents" {
		t.Errorf("item[0].Key = %q, want %q", stmt.Items[0].Key, "cash_and_equivalents")
	}
}

func TestMapperMapCurrencyUnit(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name         string
		header       string
		periodHeader string
		label        string
		wantCurrency string
		wantUnit     string
	}{
		{
			name:         "indonesian",
			header:       "Dalam Jutaan Rupiah",
			periodHeader: "31 Desember 2023",
			label:        "Kas dan Setara Kas",
			wantCurrency: "IDR",
			wantUnit:     "millions",
		},
		{
			name:         "english",
			header:       "In millions of Rupiah",
			periodHeader: "December 31, 2023",
			label:        "Cash and Cash Equivalents",
			wantCurrency: "IDR",
			wantUnit:     "millions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				[]string{tt.header, tt.periodHeader},
				[]table.Row{makeRow(0, tt.label, "100.000")},
			)

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if stmt.Currency != tt.wantCurrency {
				t.Errorf("currency = %q, want %q", stmt.Currency, tt.wantCurrency)
			}

			if stmt.Unit != tt.wantUnit {
				t.Errorf("unit = %q, want %q", stmt.Unit, tt.wantUnit)
			}
		})
	}
}

func TestMapperMapCompany(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"PT Bank Central Asia Tbk", "31 Desember 2023"},
		[]table.Row{makeRow(0, "Kas dan Setara Kas", "100.000")},
	)

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if stmt.Company != "PT Bank Central Asia Tbk" {
		t.Errorf("company = %q, want %q", stmt.Company, "PT Bank Central Asia Tbk")
	}
}

func TestMapperMapSubtotals(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "31 Desember 2023"},
		[]table.Row{
			makeRow(0, "Jumlah Aset", "1.000.000"),
			makeRow(1, "Total Liabilitas", "500.000"),
			makeRow(2, "Sub-total Ekuitas", "500.000"),
		},
	)

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	for i, item := range stmt.Items {
		if !item.IsSubtotal {
			t.Errorf("item[%d] %q: IsSubtotal = false, want true", i, item.Label)
		}
	}
}

func TestMapperMapIndentLevel(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "31 Desember 2023"},
		[]table.Row{
			makeRow(0, "Custom Item No Indent", "100"),
			makeRow(1, "    Indented Item", "50"),
		},
	)

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if stmt.Items[0].Level != 0 {
		t.Errorf("item[0].Level = %d, want 0", stmt.Items[0].Level)
	}

	if stmt.Items[1].Level != 2 {
		t.Errorf("item[1].Level = %d, want 2", stmt.Items[1].Level)
	}
}

func TestIsSubtotal(t *testing.T) {
	tests := []struct {
		label string
		want  bool
	}{
		{"Jumlah Aset", true},
		{"Total Assets", true},
		{"Subjumlah Aset Lancar", true},
		{"Sub-total Current Assets", true},
		{"Kas dan Setara Kas", false},
		{"Cash and Cash Equivalents", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := isSubtotal(tt.label); got != tt.want {
				t.Errorf("isSubtotal(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

func TestNormalizeUnit(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"jutaan", "millions"},
		{"millions", "millions"},
		{"miliar", "billions"},
		{"billions", "billions"},
		{"ribuan", "thousands"},
		{"thousands", "thousands"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeUnit(tt.input); got != tt.want {
				t.Errorf("normalizeUnit(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCurrency(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Rupiah", "IDR"},
		{"rupiah", "IDR"},
		{"Dolar", "USD"},
		{"dollars", "USD"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeCurrency(tt.input); got != tt.want {
				t.Errorf("normalizeCurrency(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDateISO(t *testing.T) {
	tests := []struct {
		name     string
		day      string
		month    string
		year     string
		months   map[string]int
		wantDate string
	}{
		{
			name:     "indonesian december",
			day:      "31",
			month:    "Desember",
			year:     "2025",
			months:   map[string]int{"desember": 12},
			wantDate: "2025-12-31",
		},
		{
			name:     "english march",
			day:      "1",
			month:    "March",
			year:     "2024",
			months:   map[string]int{"march": 3},
			wantDate: "2024-03-01",
		},
		{
			name:     "unknown month returns empty",
			day:      "1",
			month:    "Foo",
			year:     "2024",
			months:   map[string]int{"march": 3},
			wantDate: "",
		},
		{
			name:     "invalid day returns empty",
			day:      "abc",
			month:    "March",
			year:     "2024",
			months:   map[string]int{"march": 3},
			wantDate: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDateISO(tt.day, tt.month, tt.year, tt.months)
			if got != tt.wantDate {
				t.Errorf("formatDateISO() = %q, want %q", got, tt.wantDate)
			}
		})
	}
}

func TestMapperPeriodDateParsing(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name        string
		headers     []string
		wantPeriods []string
		wantLang    string
	}{
		{
			name:        "indonesian dates",
			headers:     []string{"", "31 Desember 2025", "31 Desember 2024"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "id",
		},
		{
			name:        "english dates month first",
			headers:     []string{"", "December 31, 2025", "December 31, 2024"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "en",
		},
		{
			name:        "english dates no comma",
			headers:     []string{"", "December 31 2025"},
			wantPeriods: []string{"2025-12-31"},
			wantLang:    "en",
		},
		{
			name:        "indonesian june",
			headers:     []string{"", "30 Juni 2025"},
			wantPeriods: []string{"2025-06-30"},
			wantLang:    "id",
		},
		{
			name:        "english dates day first",
			headers:     []string{"", "31 December 2025", "31 December 2024"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "en",
		},
		{
			name:        "english day first merged cell",
			headers:     []string{"31 December 2025 31 December 2024"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				tt.headers,
				[]table.Row{makeRow(0, "Kas dan Setara Kas", "100.000")},
			)

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if len(stmt.Periods) != len(tt.wantPeriods) {
				t.Fatalf("periods count = %d, want %d", len(stmt.Periods), len(tt.wantPeriods))
			}

			for i, want := range tt.wantPeriods {
				if stmt.Periods[i] != want {
					t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
				}
			}

			if stmt.Language != tt.wantLang {
				t.Errorf("language = %q, want %q", stmt.Language, tt.wantLang)
			}
		})
	}
}

func TestMapperCurrencyUnitExpanded(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name         string
		header       string
		wantCurrency string
		wantUnit     string
	}{
		{
			name:         "expressed in millions of rupiah",
			header:       "Expressed in millions of Rupiah",
			wantCurrency: "IDR",
			wantUnit:     "millions",
		},
		{
			name:         "dalam miliaran rupiah",
			header:       "Dalam Miliaran Rupiah",
			wantCurrency: "IDR",
			wantUnit:     "billions",
		},
		{
			name:         "in billions of dollars",
			header:       "In Billions of Dollars",
			wantCurrency: "USD",
			wantUnit:     "billions",
		},
		{
			name:         "slash format jutaan/in million",
			header:       "Jutaan / In Million Rupiah",
			wantCurrency: "IDR",
			wantUnit:     "millions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				[]string{tt.header, "31 Desember 2023"},
				[]table.Row{makeRow(0, "Kas dan Setara Kas", "100.000")},
			)

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if stmt.Currency != tt.wantCurrency {
				t.Errorf("currency = %q, want %q", stmt.Currency, tt.wantCurrency)
			}

			if stmt.Unit != tt.wantUnit {
				t.Errorf("unit = %q, want %q", stmt.Unit, tt.wantUnit)
			}
		})
	}
}

func TestFilterFinancialTables(t *testing.T) {
	tests := []struct {
		name      string
		tables    []table.Table
		wantCount int
		wantPages []int
	}{
		{
			name: "skips page 1 when multiple tables",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{PageNum: 2, Rows: []table.Row{makeRow(0, "Kas")}},
				{PageNum: 3, Rows: []table.Row{makeRow(0, "Utang")}},
			},
			wantCount: 2,
			wantPages: []int{2, 3},
		},
		{
			name: "keeps single table even on page 1",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Kas")}},
			},
			wantCount: 1,
			wantPages: []int{1},
		},
		{
			name: "skips subsidiary tables",
			tables: []table.Table{
				{
					PageNum: 2,
					Headers: []string{"Daftar Entitas Anak / Subsidiaries"},
					Rows:    []table.Row{makeRow(0, "PT Sub Corp")},
				},
				{PageNum: 3, Rows: []table.Row{makeRow(0, "Kas")}},
			},
			wantCount: 1,
			wantPages: []int{3},
		},
		{
			name: "returns all if filtering removes everything",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
			},
			wantCount: 1,
			wantPages: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFinancialTables(tt.tables, DocTypeBalanceSheet)

			if len(result) != tt.wantCount {
				t.Fatalf("count = %d, want %d", len(result), tt.wantCount)
			}

			for i, wantPage := range tt.wantPages {
				if result[i].PageNum != wantPage {
					t.Errorf("result[%d].PageNum = %d, want %d", i, result[i].PageNum, wantPage)
				}
			}
		})
	}
}

func TestIsSubsidiaryTable(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    bool
	}{
		{"subsidiary header", []string{"List of Subsidiaries"}, true},
		{"entitas anak header", []string{"Daftar Entitas Anak"}, true},
		{"anak perusahaan", []string{"Anak Perusahaan"}, true},
		{"financial header", []string{"31 Desember 2023"}, false},
		{"empty headers", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := table.Table{Headers: tt.headers}
			if got := isSubsidiaryTable(tbl); got != tt.want {
				t.Errorf("isSubsidiaryTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterFinancialTablesXBRL(t *testing.T) {
	tests := []struct {
		name      string
		tables    []table.Table
		docType   DocType
		wantCount int
		wantPages []int
	}{
		{
			name: "xbrl filters to balance sheet pages",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{
					PageNum:  4,
					PageText: []string{"[4220000] Statement of financial position"},
					Rows:     []table.Row{makeRow(0, "Kas", "100")},
				},
				{PageNum: 5, Rows: []table.Row{makeRow(0, "Piutang", "200")}},
				{
					PageNum:  13,
					PageText: []string{"[4322000] Statement of profit or loss"},
					Rows:     []table.Row{makeRow(0, "Revenue", "300")},
				},
			},
			docType:   DocTypeBalanceSheet,
			wantCount: 2,
			wantPages: []int{4, 5},
		},
		{
			name: "falls back to heuristic when no xbrl markers",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{PageNum: 2, Rows: []table.Row{makeRow(0, "Kas")}},
			},
			docType:   DocTypeBalanceSheet,
			wantCount: 1,
			wantPages: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFinancialTables(tt.tables, tt.docType)

			if len(result) != tt.wantCount {
				t.Fatalf("count = %d, want %d", len(result), tt.wantCount)
			}

			for i, wantPage := range tt.wantPages {
				if result[i].PageNum != wantPage {
					t.Errorf("result[%d].PageNum = %d, want %d", i, result[i].PageNum, wantPage)
				}
			}
		})
	}
}

func TestIsMetadataRow(t *testing.T) {
	tests := []struct {
		name string
		row  table.Row
		want bool
	}{
		{
			name: "date row english day first",
			row:  makeRow(0, "31 December 2025", "31 December 2024"),
			want: true,
		},
		{
			name: "date row indonesian",
			row:  makeRow(0, "31 Desember 2025"),
			want: true,
		},
		{
			name: "xbrl marker row",
			row:  makeRow(0, "[4220000] Statement of financial position"),
			want: true,
		},
		{
			name: "data row",
			row:  makeRow(0, "Kas dan Setara Kas", "1.234.567"),
			want: false,
		},
		{
			name: "empty row",
			row:  makeRow(0, ""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMetadataRow(tt.row); got != tt.want {
				t.Errorf("isMetadataRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractLeadingNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:  "number followed by text",
			input: "36,408,142 Current accounts with Bank",
			want:  36408142,
		},
		{
			name:  "negative number followed by text",
			input: "( 638 ) Allowance for impairment",
			want:  -638,
		},
		{
			name:    "only text",
			input:   "Current accounts with Bank",
			wantErr: true,
		},
		{
			name:    "only number returns error",
			input:   "36408142",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractLeadingNumber(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractLeadingNumber(%q) = %v, want error", tt.input, got)
				}

				return
			}

			if err != nil {
				t.Errorf("extractLeadingNumber(%q) error = %v", tt.input, err)

				return
			}

			if got != tt.want {
				t.Errorf("extractLeadingNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCollapseKernedDigits(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "kerned year",
			input: "202 5",
			want:  "2025",
		},
		{
			name:  "multiple kerned years",
			input: "31 DECEMBER  202 5 AND  202 4",
			want:  "31 DECEMBER  2025 AND  2024",
		},
		{
			name:  "no kerning",
			input: "31 December 2025",
			want:  "31 December 2025",
		},
		{
			name:  "kerned day and year",
			input: "3 1 December 202 5",
			want:  "31 December 2025",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collapseKernedDigits(tt.input)
			if got != tt.want {
				t.Errorf("collapseKernedDigits(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapperKernedPeriodDetection(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name        string
		headers     []string
		pageText    []string
		wantPeriods []string
		wantLang    string
	}{
		{
			name:        "kerned years in headers",
			headers:     []string{"", "31 December  202 5", "31 December  202 4"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "en",
		},
		{
			name:    "kerned years in page text only",
			headers: []string{"", "Notes"},
			pageText: []string{
				"31 DECEMBER  202 5 AND  202 4",
				"(Expressed in millions of Rupiah)",
			},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "en",
		},
		{
			name:    "period from page text with and",
			headers: []string{},
			pageText: []string{
				"31 December 2025 and 2024",
			},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
			wantLang:    "en",
		},
		{
			name:        "indonesian kerned year",
			headers:     []string{"", "31 Desember  202 5"},
			wantPeriods: []string{"2025-12-31"},
			wantLang:    "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				tt.headers,
				[]table.Row{makeRow(0, "Kas dan Setara Kas", "100.000")},
			)
			tbl.PageText = tt.pageText

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if len(stmt.Periods) != len(tt.wantPeriods) {
				t.Fatalf("periods count = %d, want %d", len(stmt.Periods), len(tt.wantPeriods))
			}

			for i, want := range tt.wantPeriods {
				if stmt.Periods[i] != want {
					t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
				}
			}

			if stmt.Language != tt.wantLang {
				t.Errorf("language = %q, want %q", stmt.Language, tt.wantLang)
			}
		})
	}
}

func TestDeduplicateItems(t *testing.T) {
	tests := []struct {
		name      string
		items     []LineItem
		wantCount int
		wantKeys  []string
		wantVals  []float64
	}{
		{
			name: "keeps non-zero over zero duplicate",
			items: []LineItem{
				{Key: "revenue", Label: "Revenue", Values: map[string]float64{"2025-12-31": 0, "2024-12-31": 0}},
				{
					Key:    "revenue",
					Label:  "Pendapatan",
					Values: map[string]float64{"2025-12-31": 1000, "2024-12-31": 800},
				},
			},
			wantCount: 1,
			wantKeys:  []string{"revenue"},
			wantVals:  []float64{1000},
		},
		{
			name: "keeps item with larger absolute values",
			items: []LineItem{
				{Key: "expenses", Label: "Expenses", Values: map[string]float64{"2025-12-31": -500}},
				{Key: "expenses", Label: "Beban", Values: map[string]float64{"2025-12-31": -1000}},
			},
			wantCount: 1,
			wantKeys:  []string{"expenses"},
			wantVals:  []float64{-1000},
		},
		{
			name: "preserves unkeyed items",
			items: []LineItem{
				{Key: "", Label: "Unknown Row", Values: map[string]float64{"2025-12-31": 100}},
				{Key: "cash", Label: "Cash", Values: map[string]float64{"2025-12-31": 500}},
				{Key: "", Label: "Another Unknown", Values: map[string]float64{"2025-12-31": 200}},
			},
			wantCount: 3,
			wantKeys:  []string{"", "cash", ""},
		},
		{
			name: "no duplicates unchanged",
			items: []LineItem{
				{Key: "cash", Label: "Cash", Values: map[string]float64{"2025-12-31": 100}},
				{Key: "debt", Label: "Debt", Values: map[string]float64{"2025-12-31": 200}},
			},
			wantCount: 2,
			wantKeys:  []string{"cash", "debt"},
		},
		{
			name: "drops unkeyed zero-value items",
			items: []LineItem{
				{Key: "", Label: "Penurunan (kenaikan)", Values: map[string]float64{"2025-12-31": 0, "2024-12-31": 0}},
				{Key: "cash", Label: "Cash", Values: map[string]float64{"2025-12-31": 500}},
				{Key: "", Label: "Non-zero unkeyed", Values: map[string]float64{"2025-12-31": 100}},
			},
			wantCount: 2,
			wantKeys:  []string{"cash", ""},
			wantVals:  []float64{500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateItems(tt.items)

			if len(got) != tt.wantCount {
				t.Fatalf("count = %d, want %d", len(got), tt.wantCount)
			}

			for i, wantKey := range tt.wantKeys {
				if got[i].Key != wantKey {
					t.Errorf("items[%d].Key = %q, want %q", i, got[i].Key, wantKey)
				}
			}

			for i, wantVal := range tt.wantVals {
				period := "2025-12-31"
				if v, ok := got[i].Values[period]; !ok || v != wantVal {
					t.Errorf("items[%d].Values[%q] = %v, want %v", i, period, v, wantVal)
				}
			}
		})
	}
}

func TestMapperPageTextMetadata(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "31 December 2025"},
		[]table.Row{makeRow(0, "Kas", "100,000")},
	)
	tbl.PageText = []string{
		"PT Bank Central Asia Tbk AND SUBSIDIARIES",
		"(Expressed in millions of Rupiah, unless otherwise stated)",
	}

	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if stmt.Company != "PT Bank Central Asia Tbk" {
		t.Errorf("company = %q, want %q", stmt.Company, "PT Bank Central Asia Tbk")
	}

	if stmt.Currency != "IDR" {
		t.Errorf("currency = %q, want %q", stmt.Currency, "IDR")
	}

	if stmt.Unit != "millions" {
		t.Errorf("unit = %q, want %q", stmt.Unit, "millions")
	}
}

func TestIsFiscalPeriodEnd(t *testing.T) {
	tests := []struct {
		name string
		day  int
		mon  int
		want bool
	}{
		{"dec 31 annual", 31, 12, true},
		{"mar 31 Q1", 31, 3, true},
		{"jun 30 Q2", 30, 6, true},
		{"sep 30 Q3", 30, 9, true},
		{"jan 26 audit date", 26, 1, false},
		{"jan 1 period start", 1, 1, false},
		{"oct 31 incorporation", 31, 10, false},
		{"mar 1 not quarter end", 1, 3, false},
		{"jun 15 mid month", 15, 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFiscalPeriodEnd(tt.day, tt.mon); got != tt.want {
				t.Errorf("isFiscalPeriodEnd(%d, %d) = %v, want %v",
					tt.day, tt.mon, got, tt.want)
			}
		})
	}
}

func TestPeriodDetectionFiltersNonFiscalDates(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name        string
		headers     []string
		pageText    []string
		rowTexts    []string
		wantPeriods []string
	}{
		{
			name:    "filters audit date and period start from body text",
			headers: []string{"", "31 December 2025", "31 December 2024"},
			pageText: []string{
				"January 26, 2026",
				"January 01, 2025",
			},
			rowTexts:    []string{"Cash", "100"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
		},
		{
			name:    "filters regulation and incorporation dates",
			headers: []string{"", "31 Desember 2025", "31 Desember 2024"},
			pageText: []string{
				"October 31, 2000",
				"March 15, 1998",
			},
			rowTexts:    []string{"Kas", "100"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
		},
		{
			name:        "keeps quarter-end dates",
			headers:     []string{"", "30 Juni 2025", "30 Juni 2024"},
			rowTexts:    []string{"Kas", "100"},
			wantPeriods: []string{"2025-06-30", "2024-06-30"},
		},
		{
			name:    "limits to max 3 periods",
			headers: []string{"", "31 December 2025", "31 December 2024"},
			pageText: []string{
				"31 December 2023",
				"31 December 2022",
			},
			rowTexts:    []string{"Cash", "100"},
			wantPeriods: []string{"2025-12-31", "2024-12-31", "2023-12-31"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				tt.headers,
				[]table.Row{makeRow(0, tt.rowTexts...)},
			)
			tbl.PageText = tt.pageText

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if len(stmt.Periods) != len(tt.wantPeriods) {
				t.Fatalf("periods count = %d (%v), want %d (%v)",
					len(stmt.Periods), stmt.Periods,
					len(tt.wantPeriods), tt.wantPeriods)
			}

			for i, want := range tt.wantPeriods {
				if stmt.Periods[i] != want {
					t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
				}
			}
		})
	}
}

func TestIsNoiseLabel(t *testing.T) {
	tests := []struct {
		label string
		want  bool
	}{
		{"202 5", true},
		{"2025", true},
		{"2024", true},
		{"42", true},
		{"AB", true},
		{"", true},
		{"Kas dan Setara Kas", false},
		{"Interest Income", false},
		{"Revenue", false},
		{"Laba", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := isNoiseLabel(tt.label); got != tt.want {
				t.Errorf("isNoiseLabel(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

func TestFilterColKeys(t *testing.T) {
	tests := []struct {
		name     string
		periods  []string
		values   map[string]float64
		wantKeys []string
	}{
		{
			name:    "removes col_N keys when periods exist",
			periods: []string{"2025-12-31", "2024-12-31"},
			values: map[string]float64{
				"2025-12-31": 100,
				"2024-12-31": 200,
				"col_3":      300,
				"col_4":      400,
			},
			wantKeys: []string{"2025-12-31", "2024-12-31"},
		},
		{
			name:    "keeps col_N keys when no periods",
			periods: nil,
			values: map[string]float64{
				"col_1": 100,
				"col_2": 200,
			},
			wantKeys: []string{"col_1", "col_2"},
		},
		{
			name:    "no values unchanged",
			periods: []string{"2025-12-31"},
			values:  map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &FinancialStatement{
				Periods: tt.periods,
				Items: []LineItem{
					{Key: "test", Values: tt.values},
				},
			}

			filterColKeys(stmt)

			for key := range stmt.Items[0].Values {
				if !slices.Contains(tt.wantKeys, key) {
					t.Errorf("unexpected key %q in values after filtering", key)
				}
			}

			if len(stmt.Items[0].Values) != len(tt.wantKeys) {
				t.Errorf("values count = %d, want %d",
					len(stmt.Items[0].Values), len(tt.wantKeys))
			}
		})
	}
}

func TestIsCompositeDocType(t *testing.T) {
	tests := []struct {
		docType DocType
		want    bool
	}{
		{DocTypeBalanceSheet, false},
		{DocTypeIncomeStatement, false},
		{DocTypeCashFlow, false},
		{DocTypeEquityChanges, false},
		{DocTypeAuditorReport, true},
		{DocTypeAnnualReport, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			if got := isCompositeDocType(tt.docType); got != tt.want {
				t.Errorf("isCompositeDocType(%q) = %v, want %v",
					tt.docType, got, tt.want)
			}
		})
	}
}

func TestMapperCompositeDocType(t *testing.T) {
	m := NewMapper()

	tbl := makeTable(
		[]string{"", "31 December 2025", "31 December 2024"},
		[]table.Row{
			makeRow(0, "Cash and Cash Equivalents", "1.000.000", "800.000"),
			makeRow(1, "Net Income", "500.000", "400.000"),
		},
	)

	stmt, err := m.Map(DocTypeAuditorReport, []table.Table{tbl})
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if len(stmt.Items) == 0 {
		t.Fatal("no items returned")
	}

	// Should match items from different dictionary types.
	matchedKeys := make(map[string]bool)
	for _, item := range stmt.Items {
		if item.Key != "" {
			matchedKeys[item.Key] = true
		}
	}

	if len(matchedKeys) == 0 {
		t.Error("no items matched from any dictionary")
	}
}

func TestHasNumericColumns(t *testing.T) {
	tests := []struct {
		name string
		tbl  table.Table
		want bool
	}{
		{
			name: "table with numeric data",
			tbl: table.Table{
				Rows: []table.Row{makeRow(0, "Cash", "1.000.000")},
			},
			want: true,
		},
		{
			name: "table with no numeric data",
			tbl: table.Table{
				Rows: []table.Row{makeRow(0, "Description", "Some text")},
			},
			want: false,
		},
		{
			name: "empty table",
			tbl: table.Table{
				Rows: []table.Row{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNumericColumns(tt.tbl); got != tt.want {
				t.Errorf("hasNumericColumns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterCompositeFinancialTables(t *testing.T) {
	tests := []struct {
		name      string
		tables    []table.Table
		wantCount int
	}{
		{
			name: "xbrl markers select financial pages",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{
					PageNum:  4,
					PageText: []string{"[4220000] Statement of financial position"},
					Rows:     []table.Row{makeRow(0, "Cash", "100")},
				},
				{PageNum: 5, Rows: []table.Row{makeRow(0, "Receivables", "200")}},
				{
					PageNum:  13,
					PageText: []string{"[4322000] Statement of profit or loss"},
					Rows:     []table.Row{makeRow(0, "Revenue", "300")},
				},
				{PageNum: 14, Rows: []table.Row{makeRow(0, "Expenses", "400")}},
			},
			wantCount: 4,
		},
		{
			name: "falls back to numeric content without xbrl",
			tables: []table.Table{
				{PageNum: 1, Rows: []table.Row{makeRow(0, "Cover")}},
				{PageNum: 2, Rows: []table.Row{makeRow(0, "Cash", "1.000")}},
				{PageNum: 3, Rows: []table.Row{makeRow(0, "Narrative text", "no numbers")}},
				{PageNum: 4, Rows: []table.Row{makeRow(0, "Revenue", "500")}},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterCompositeFinancialTables(tt.tables)

			if len(result) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestCurrencyUnitShortFormat(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name         string
		header       string
		wantCurrency string
		wantUnit     string
	}{
		{
			name:         "Rp tn trillions",
			header:       "Consolidated (Rp tn)",
			wantCurrency: "IDR",
			wantUnit:     "trillions",
		},
		{
			name:         "Rp bn billions",
			header:       "(Rp bn)",
			wantCurrency: "IDR",
			wantUnit:     "billions",
		},
		{
			name:         "Rp mn millions",
			header:       "Rp mn",
			wantCurrency: "IDR",
			wantUnit:     "millions",
		},
		{
			name:         "Rp triliun",
			header:       "(Rp triliun)",
			wantCurrency: "IDR",
			wantUnit:     "trillions",
		},
		{
			name:         "Rp miliar",
			header:       "Rp miliar",
			wantCurrency: "IDR",
			wantUnit:     "billions",
		},
		{
			name:         "USD bn",
			header:       "(USD bn)",
			wantCurrency: "USD",
			wantUnit:     "billions",
		},
		{
			name:         "IDR tn",
			header:       "IDR tn",
			wantCurrency: "IDR",
			wantUnit:     "trillions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				[]string{tt.header, "Dec-23"},
				[]table.Row{makeRow(0, "Cash and Equivalents", "100")},
			)

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if stmt.Currency != tt.wantCurrency {
				t.Errorf("currency = %q, want %q", stmt.Currency, tt.wantCurrency)
			}

			if stmt.Unit != tt.wantUnit {
				t.Errorf("unit = %q, want %q", stmt.Unit, tt.wantUnit)
			}
		})
	}
}

func TestAbbreviatedPeriodDetection(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name        string
		headers     []string
		wantPeriods []string
	}{
		{
			name:        "Dec-25 and Dec-24",
			headers:     []string{"", "Dec-25", "Dec-24"},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
		},
		{
			name:        "Sep-25",
			headers:     []string{"", "Sep-25"},
			wantPeriods: []string{"2025-09-30"},
		},
		{
			name:        "Mar-24",
			headers:     []string{"", "Mar-24"},
			wantPeriods: []string{"2024-03-31"},
		},
		{
			name:        "Jun-25",
			headers:     []string{"", "Jun-25"},
			wantPeriods: []string{"2025-06-30"},
		},
		{
			name:        "FY-25 fiscal year",
			headers:     []string{"", "FY-25"},
			wantPeriods: []string{"2025-12-31"},
		},
		{
			name:        "3Q-25 quarter",
			headers:     []string{"", "3Q-25"},
			wantPeriods: []string{"2025-09-30"},
		},
		{
			name:        "4Q-24 quarter",
			headers:     []string{"", "4Q-24"},
			wantPeriods: []string{"2024-12-31"},
		},
		{
			name:        "1Q-25 quarter",
			headers:     []string{"", "1Q-25"},
			wantPeriods: []string{"2025-03-31"},
		},
		{
			name:        "2Q-25 quarter",
			headers:     []string{"", "2Q-25"},
			wantPeriods: []string{"2025-06-30"},
		},
		{
			name:        "multiple abbreviated periods",
			headers:     []string{"Consolidated (Rp tn)", "Dec-23", "Dec-24", "Dec-25"},
			wantPeriods: []string{"2023-12-31", "2024-12-31", "2025-12-31"},
		},
		{
			name:        "non quarter end month",
			headers:     []string{"", "Feb-25"},
			wantPeriods: []string{"2025-02-28"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				tt.headers,
				[]table.Row{makeRow(0, "Total Assets", "100")},
			)

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if len(stmt.Periods) != len(tt.wantPeriods) {
				t.Fatalf("periods count = %d (%v), want %d (%v)",
					len(stmt.Periods), stmt.Periods,
					len(tt.wantPeriods), tt.wantPeriods)
			}

			for i, want := range tt.wantPeriods {
				if stmt.Periods[i] != want {
					t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
				}
			}
		})
	}
}

func TestGrowthRateColumnFiltering(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name       string
		headers    []string
		rowTexts   []string
		wantValues int
	}{
		{
			name:       "filters YoY and QoQ columns",
			headers:    []string{"", "Dec-24", "Dec-25", "YoY", "QoQ"},
			rowTexts:   []string{"Total Assets", "1,449", "1,587", "9.5%", "3.1%"},
			wantValues: 2,
		},
		{
			name:       "keeps all columns when no growth headers",
			headers:    []string{"", "Dec-24", "Dec-25"},
			rowTexts:   []string{"Total Assets", "1,449", "1,587"},
			wantValues: 2,
		},
		{
			name:       "filters percentage values even without growth headers",
			headers:    []string{"", "Dec-24", "Dec-25", "Change"},
			rowTexts:   []string{"Total Assets", "1,449", "1,587", "9.5%"},
			wantValues: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				tt.headers,
				[]table.Row{makeRow(0, tt.rowTexts...)},
			)

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if len(stmt.Items) == 0 {
				t.Fatal("expected at least 1 item")
			}

			gotValues := len(stmt.Items[0].Values)
			if gotValues != tt.wantValues {
				t.Errorf("values count = %d, want %d (values: %v)",
					gotValues, tt.wantValues, stmt.Items[0].Values)
			}
		})
	}
}

func TestAbbrevToISO(t *testing.T) {
	tests := []struct {
		name  string
		label string
		year  int
		want  string
	}{
		{"december", "Dec", 2025, "2025-12-31"},
		{"september", "Sep", 2025, "2025-09-30"},
		{"march", "Mar", 2024, "2024-03-31"},
		{"june", "Jun", 2025, "2025-06-30"},
		{"fiscal year", "FY", 2025, "2025-12-31"},
		{"Q1", "1Q", 2025, "2025-03-31"},
		{"Q2", "2Q", 2025, "2025-06-30"},
		{"Q3", "3Q", 2025, "2025-09-30"},
		{"Q4", "4Q", 2025, "2025-12-31"},
		{"february non quarter end", "Feb", 2024, "2024-02-29"},
		{"february non leap year", "Feb", 2025, "2025-02-28"},
		{"unknown", "XY", 2025, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := abbrevToISO(tt.label, tt.year)
			if got != tt.want {
				t.Errorf("abbrevToISO(%q, %d) = %q, want %q", tt.label, tt.year, got, tt.want)
			}
		})
	}
}

func TestGrowthRateColumns(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    map[int]bool
	}{
		{
			name:    "detects YoY and QoQ",
			headers: []string{"", "Dec-24", "Dec-25", "YoY", "QoQ"},
			want:    map[int]bool{3: true, 4: true},
		},
		{
			name:    "no growth headers",
			headers: []string{"", "Dec-24", "Dec-25"},
			want:    map[int]bool{},
		},
		{
			name:    "percent sign header",
			headers: []string{"", "Dec-24", "%"},
			want:    map[int]bool{2: true},
		},
		{
			name:    "growth header",
			headers: []string{"", "Dec-24", "Growth"},
			want:    map[int]bool{2: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := growthRateColumns(tt.headers)
			if len(got) != len(tt.want) {
				t.Fatalf("growthRateColumns() = %v, want %v", got, tt.want)
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("growthRateColumns()[%d] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestIsPercentageValue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"9.5%", true},
		{"-3.1%", true},
		{"100", false},
		{"1,234", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isPercentageValue(tt.input); got != tt.want {
				t.Errorf("isPercentageValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPresentationStyleFullExtraction(t *testing.T) {
	m := NewMapper()

	headers := []string{"Consolidated (Rp tn)", "Dec-23", "Dec-24", "Dec-25", "YoY", "Sep-25", "QoQ"}
	rows := []table.Row{
		makeRow(0, "Total Assets", "1,408", "1,449", "1,587", "9.5%", "1,539", "3.1%"),
		makeRow(1, "Loans", "810", "922", "993", "7.7%", "944", "5.2%"),
	}

	tbl := makeTable(headers, rows)
	stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})

	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}

	if stmt.Currency != "IDR" {
		t.Errorf("currency = %q, want %q", stmt.Currency, "IDR")
	}

	if stmt.Unit != "trillions" {
		t.Errorf("unit = %q, want %q", stmt.Unit, "trillions")
	}

	wantPeriods := []string{"2023-12-31", "2024-12-31", "2025-12-31"}
	if len(stmt.Periods) != len(wantPeriods) {
		t.Fatalf("periods = %v, want %v", stmt.Periods, wantPeriods)
	}

	for i, want := range wantPeriods {
		if stmt.Periods[i] != want {
			t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
		}
	}

	if len(stmt.Items) < 2 {
		t.Fatalf("items count = %d, want >= 2", len(stmt.Items))
	}

	// Verify YoY and QoQ values are filtered out — only period values remain.
	for _, item := range stmt.Items {
		for key := range item.Values {
			if key == "YoY" || key == "QoQ" {
				t.Errorf("item %q has growth rate key %q, should be filtered",
					item.Label, key)
			}
		}
	}
}

func TestPageTextAbbrevPeriodsIgnored(t *testing.T) {
	m := NewMapper()

	tests := []struct {
		name        string
		headers     []string
		pageText    []string
		wantPeriods []string
	}{
		{
			name:    "FY25 in footer ignored",
			headers: []string{"Consolidated (Rp tn)", "Dec-23", "Dec-24", "Dec-25"},
			pageText: []string{
				"Analyst Meeting FY25",
				"2021	2022	2023	2024",
			},
			wantPeriods: []string{"2023-12-31", "2024-12-31", "2025-12-31"},
		},
		{
			name:    "bare years in chart axis ignored",
			headers: []string{"", "Dec-25"},
			pageText: []string{
				"2021 2022 2023 2024 Oct-25",
			},
			wantPeriods: []string{"2025-12-31"},
		},
		{
			name:    "full form period in page text still detected",
			headers: []string{"", "Dec-25"},
			pageText: []string{
				"31 Desember 2024",
			},
			wantPeriods: []string{"2025-12-31", "2024-12-31"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := makeTable(
				tt.headers,
				[]table.Row{makeRow(0, "Total Assets", "100")},
			)
			tbl.PageText = tt.pageText

			stmt, err := m.Map(DocTypeBalanceSheet, []table.Table{tbl})
			if err != nil {
				t.Fatalf("Map() unexpected error: %v", err)
			}

			if len(stmt.Periods) != len(tt.wantPeriods) {
				t.Fatalf("periods count = %d (%v), want %d (%v)",
					len(stmt.Periods), stmt.Periods,
					len(tt.wantPeriods), tt.wantPeriods)
			}

			for i, want := range tt.wantPeriods {
				if stmt.Periods[i] != want {
					t.Errorf("periods[%d] = %q, want %q", i, stmt.Periods[i], want)
				}
			}
		})
	}
}

func TestFYRequiresHyphen(t *testing.T) {
	m := &mapper{}
	stmt := &FinancialStatement{}

	// "FY25" without hyphen should NOT be detected
	m.detectPeriod("Analyst Meeting FY25", stmt)

	if len(stmt.Periods) != 0 {
		t.Errorf("FY25 without hyphen should not match, got periods: %v", stmt.Periods)
	}

	// "FY-25" with hyphen SHOULD be detected
	m.detectPeriod("FY-25", stmt)

	if len(stmt.Periods) != 1 || stmt.Periods[0] != "2025-12-31" {
		t.Errorf("FY-25 with hyphen should match, got periods: %v", stmt.Periods)
	}
}

func TestCurrencyIDRPriority(t *testing.T) {
	m := &mapper{}

	tests := []struct {
		name         string
		texts        []string
		wantCurrency string
		wantUnit     string
	}{
		{
			name:         "IDR from Rp tn",
			texts:        []string{"Consolidated (Rp tn)"},
			wantCurrency: "IDR",
			wantUnit:     "trillions",
		},
		{
			name:         "USD found first then Rp overrides",
			texts:        []string{"USD Bn", "Rp tn"},
			wantCurrency: "IDR",
			wantUnit:     "trillions",
		},
		{
			name:         "Rp found first USD ignored",
			texts:        []string{"Rp tn", "USD Bn"},
			wantCurrency: "IDR",
			wantUnit:     "trillions",
		},
		{
			name:         "only USD when no IDR present",
			texts:        []string{"USD Bn"},
			wantCurrency: "USD",
			wantUnit:     "billions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &FinancialStatement{}
			for _, text := range tt.texts {
				m.detectCurrencyUnit(text, stmt)
			}

			if stmt.Currency != tt.wantCurrency {
				t.Errorf("currency = %q, want %q", stmt.Currency, tt.wantCurrency)
			}

			if stmt.Unit != tt.wantUnit {
				t.Errorf("unit = %q, want %q", stmt.Unit, tt.wantUnit)
			}
		})
	}
}

func TestParseCurrencyUnit(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		wantCurrency string
		wantUnit     string
	}{
		{"Rp tn", "Consolidated (Rp tn)", "IDR", "trillions"},
		{"USD Bn", "Revenue (USD Bn)", "USD", "billions"},
		{"Rp miliar", "(Rp miliar)", "IDR", "billions"},
		{"dalam jutaan rupiah", "dalam jutaan rupiah", "IDR", "millions"},
		{"in millions of rupiah", "in millions of rupiah", "IDR", "millions"},
		{"no match", "plain text", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currency, unit := parseCurrencyUnit(tt.text)
			if currency != tt.wantCurrency {
				t.Errorf("parseCurrencyUnit(%q) currency = %q, want %q",
					tt.text, currency, tt.wantCurrency)
			}

			if unit != tt.wantUnit {
				t.Errorf("parseCurrencyUnit(%q) unit = %q, want %q",
					tt.text, unit, tt.wantUnit)
			}
		})
	}
}

func TestIsFutureDate(t *testing.T) {
	// Fix time to 2026-03-19 for deterministic tests.
	original := timeNow
	timeNow = func() time.Time {
		return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { timeNow = original })

	tests := []struct {
		name  string
		year  int
		month int
		want  bool
	}{
		{"current year same month", 2026, 3, false},
		{"current year dec", 2026, 12, false},
		{"one year ahead mar", 2027, 3, false},
		{"one year ahead jun", 2027, 6, true},
		{"one year ahead dec", 2027, 12, true},
		{"two years ahead", 2028, 1, true},
		{"past year", 2025, 12, false},
		{"far past", 2020, 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFutureDate(tt.year, tt.month); got != tt.want {
				t.Errorf("isFutureDate(%d, %d) = %v, want %v",
					tt.year, tt.month, got, tt.want)
			}
		})
	}
}

func TestAddPeriodRejectsFutureDates(t *testing.T) {
	original := timeNow
	timeNow = func() time.Time {
		return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { timeNow = original })

	stmt := &FinancialStatement{}

	// 2027-06-30 is more than 1 year in the future — should be rejected.
	addPeriod(stmt, "30", "Juni", "2027", monthsID, "id")
	if len(stmt.Periods) != 0 {
		t.Errorf("expected future date 2027-06-30 to be rejected, got periods: %v",
			stmt.Periods)
	}

	// 2025-12-31 is in the past — should be accepted.
	addPeriod(stmt, "31", "Desember", "2025", monthsID, "id")
	if len(stmt.Periods) != 1 || stmt.Periods[0] != "2025-12-31" {
		t.Errorf("expected 2025-12-31 to be accepted, got periods: %v",
			stmt.Periods)
	}
}

func TestAddAbbrevPeriodRejectsFutureDates(t *testing.T) {
	original := timeNow
	timeNow = func() time.Time {
		return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { timeNow = original })

	stmt := &FinancialStatement{}

	// Jun-27 = 2027-06-30, more than 1 year ahead — should be rejected.
	addAbbrevPeriod(stmt, "Jun", "27")
	if len(stmt.Periods) != 0 {
		t.Errorf("expected future date Jun-27 to be rejected, got periods: %v",
			stmt.Periods)
	}

	// Dec-25 = 2025-12-31, in the past — should be accepted.
	addAbbrevPeriod(stmt, "Dec", "25")
	if len(stmt.Periods) != 1 || stmt.Periods[0] != "2025-12-31" {
		t.Errorf("expected Dec-25 to be accepted, got periods: %v",
			stmt.Periods)
	}
}

func TestAbbrevMonth(t *testing.T) {
	tests := []struct {
		label string
		want  int
	}{
		{"FY", 12},
		{"1Q", 3},
		{"2Q", 6},
		{"3Q", 9},
		{"4Q", 12},
		{"Dec", 12},
		{"Jun", 6},
		{"Mar", 3},
		{"Sep", 9},
		{"unknown", 12},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := abbrevMonth(tt.label); got != tt.want {
				t.Errorf("abbrevMonth(%q) = %d, want %d", tt.label, got, tt.want)
			}
		})
	}
}
