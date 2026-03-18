package domain

import (
	"testing"

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

	tests := []struct {
		name    string
		docType DocType
		tables  []table.Table
	}{
		{name: "no tables", docType: DocTypeBalanceSheet, tables: nil},
		{name: "unsupported doc type", docType: DocTypeUnknown, tables: []table.Table{{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.Map(tt.docType, tt.tables)
			if err == nil {
				t.Error("Map() error = nil, want error")
			}
		})
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
			result := filterFinancialTables(tt.tables)

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
