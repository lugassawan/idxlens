package xlsx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParse(t *testing.T) {
	fixturePath := createFixture(t)

	stmt, err := Parse(fixturePath)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if stmt.Ticker != "BBCA" {
		t.Errorf("Ticker = %q, want %q", stmt.Ticker, "BBCA")
	}

	if stmt.Year != 2024 {
		t.Errorf("Year = %d, want %d", stmt.Year, 2024)
	}

	if stmt.Period != "Q3" {
		t.Errorf("Period = %q, want %q", stmt.Period, "Q3")
	}

	if len(stmt.Sheets) != 1 {
		t.Fatalf("Sheets count = %d, want 1", len(stmt.Sheets))
	}

	sheet := stmt.Sheets[0]
	if sheet.Name != "Balance Sheet" {
		t.Errorf("Sheet.Name = %q, want %q", sheet.Name, "Balance Sheet")
	}

	if len(sheet.Items) != 3 {
		t.Fatalf("Items count = %d, want 3", len(sheet.Items))
	}

	tests := []struct {
		name  string
		label string
		key   string
		want  float64
	}{
		{"first item 2024", "Total Assets", "2024", 1000000},
		{"first item 2023", "Total Assets", "2023", 900000},
		{"second item 2024", "Total Liabilities", "2024", 600000},
		{"third item 2024", "Total Equity", "2024", 400000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var item *LineItem

			for i := range sheet.Items {
				if sheet.Items[i].Label == tt.label {
					item = &sheet.Items[i]
					break
				}
			}

			if item == nil {
				t.Fatalf("item %q not found", tt.label)
			}

			got, ok := item.Values[tt.key]
			if !ok {
				t.Fatalf("key %q not found in Values", tt.key)
			}

			if got != tt.want {
				t.Errorf("Values[%q] = %f, want %f", tt.key, got, tt.want)
			}
		})
	}
}

func TestParseMeta(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		ticker   string
		year     int
		period   string
	}{
		{
			name:     "standard IDX filename",
			filename: "FinancialStatement-2024-Q3-BBCA.xlsx",
			ticker:   "BBCA",
			year:     2024,
			period:   "Q3",
		},
		{
			name:     "audit period",
			filename: "FinancialStatement-2023-Audit-TLKM.xlsx",
			ticker:   "TLKM",
			year:     2023,
			period:   "Audit",
		},
		{
			name:     "non-matching filename",
			filename: "random-file.xlsx",
			ticker:   "",
			year:     0,
			period:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &Statement{}
			parseMeta(stmt, tt.filename)

			if stmt.Ticker != tt.ticker {
				t.Errorf("Ticker = %q, want %q", stmt.Ticker, tt.ticker)
			}

			if stmt.Year != tt.year {
				t.Errorf("Year = %d, want %d", stmt.Year, tt.year)
			}

			if stmt.Period != tt.period {
				t.Errorf("Period = %q, want %q", stmt.Period, tt.period)
			}
		})
	}
}

func TestParseEmptySheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	f := excelize.NewFile()
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("create fixture: %v", err)
	}

	f.Close()

	stmt, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(stmt.Sheets) != 0 {
		t.Errorf("Sheets count = %d, want 0", len(stmt.Sheets))
	}
}

func TestParseNonExistent(t *testing.T) {
	_, err := Parse("/nonexistent/file.xlsx")
	if err == nil {
		t.Fatal("Parse() expected error for non-existent file")
	}
}

func createFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "FinancialStatement-2024-Q3-BBCA.xlsx")

	f := excelize.NewFile()

	sheetName := "Balance Sheet"
	idx, err := f.NewSheet(sheetName)
	if err != nil {
		t.Fatalf("create sheet: %v", err)
	}

	f.SetActiveSheet(idx)

	if err := f.DeleteSheet("Sheet1"); err != nil {
		t.Fatalf("delete default sheet: %v", err)
	}

	headers := []string{"Account", "2024", "2023"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
	}

	rows := []struct {
		label string
		v2024 float64
		v2023 float64
	}{
		{"Total Assets", 1000000, 900000},
		{"Total Liabilities", 600000, 550000},
		{"Total Equity", 400000, 350000},
	}

	for i, row := range rows {
		r := i + 2
		cellA, _ := excelize.CoordinatesToCellName(1, r)
		cellB, _ := excelize.CoordinatesToCellName(2, r)
		cellC, _ := excelize.CoordinatesToCellName(3, r)
		_ = f.SetCellValue(sheetName, cellA, row.label)
		_ = f.SetCellValue(sheetName, cellB, row.v2024)
		_ = f.SetCellValue(sheetName, cellC, row.v2023)
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	f.Close()

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture not created: %v", err)
	}

	return path
}
