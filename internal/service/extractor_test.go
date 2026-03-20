package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/lugassawan/idxlens/internal/xbrl"
	"github.com/lugassawan/idxlens/internal/xlsx"
	excelize "github.com/xuri/excelize/v2"
)

func TestGetExtractor(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"xlsx registered", "xlsx", false},
		{"xbrl registered", "xbrl", false},
		{"pdf registered", "pdf", false},
		{"unknown format", "csv", true},
		{"empty format", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := getExtractor(tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if e == nil {
				t.Fatal("expected non-nil extractor")
			}
		})
	}
}

func TestExtractorRegistryCompleteness(t *testing.T) {
	formats := []string{"xlsx", "xbrl", "pdf"}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			if _, ok := extractorRegistry[f]; !ok {
				t.Errorf("format %q not registered", f)
			}
		})
	}
}

func TestExtractFileUnsupportedFormat(t *testing.T) {
	_, err := ExtractFile("test.dat", "dat", "financial", "", 0, "")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}

	want := "unsupported format: dat"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestExtractFileXLSX(t *testing.T) {
	path := createTestXLSX(t)

	result, err := ExtractFile(path, "xlsx", "financial", "", 0, "")
	if err != nil {
		t.Fatalf("ExtractFile() error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestExtractFileXLSXAppliesMeta(t *testing.T) {
	path := createTestXLSX(t)

	// Use a non-matching filename so parseMeta leaves fields empty
	dir := t.TempDir()
	nonStandard := filepath.Join(dir, "report.xlsx")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	if err := os.WriteFile(nonStandard, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	result, err := ExtractFile(nonStandard, "xlsx", "financial", "PGAS", 2025, "Audit")
	if err != nil {
		t.Fatalf("ExtractFile() error: %v", err)
	}

	stmt, ok := result.(*xlsx.Statement)
	if !ok {
		t.Fatal("expected *xlsx.Statement")
	}

	if stmt.Ticker != "PGAS" {
		t.Errorf("Ticker = %q, want %q", stmt.Ticker, "PGAS")
	}

	if stmt.Year != 2025 {
		t.Errorf("Year = %d, want %d", stmt.Year, 2025)
	}

	if stmt.Period != "Audit" {
		t.Errorf("Period = %q, want %q", stmt.Period, "Audit")
	}
}

func TestExtractFileXBRLAppliesMeta(t *testing.T) {
	path := createTestXBRLZip(t)

	result, err := ExtractFile(path, "xbrl", "financial", "BBCA", 2025, "Q1")
	if err != nil {
		t.Fatalf("ExtractFile() error: %v", err)
	}

	stmt, ok := result.(*xbrl.Statement)
	if !ok {
		t.Fatal("expected *xbrl.Statement")
	}

	if stmt.Ticker != "BBCA" {
		t.Errorf("Ticker = %q, want %q", stmt.Ticker, "BBCA")
	}

	if stmt.Year != 2025 {
		t.Errorf("Year = %d, want %d", stmt.Year, 2025)
	}

	if stmt.Period != "Q1" {
		t.Errorf("Period = %q, want %q", stmt.Period, "Q1")
	}
}

func TestExtractFilePDFFinancialModeError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")

	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	_, err := ExtractFile(path, "pdf", "financial", "", 0, "")
	if err == nil {
		t.Fatal("expected error for PDF financial mode")
	}
}

func TestExtractFilePDFPresentationModeError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")

	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	// Fake PDF will fail parsing, confirming presentation mode routing
	_, err := ExtractFile(path, "pdf", "presentation", "", 0, "")
	if err == nil {
		t.Fatal("expected error parsing fake PDF in presentation mode")
	}

	// Error should come from PDF parsing, not the "not supported" message
	got := err.Error()
	if got == "PDF financial extraction not supported in v2 (use XLSX or XBRL)" {
		t.Error("presentation mode should not return financial mode error")
	}
}

func TestExtractFileXLSXInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.xlsx")

	if err := os.WriteFile(path, []byte("not xlsx"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	_, err := ExtractFile(path, "xlsx", "financial", "", 0, "")
	if err == nil {
		t.Fatal("expected error for invalid xlsx")
	}
}

func TestExtractFileXBRLInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.zip")

	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	_, err := ExtractFile(path, "xbrl", "financial", "", 0, "")
	if err == nil {
		t.Fatal("expected error for invalid xbrl zip")
	}
}

func createTestXLSX(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "FinancialStatement-2024-Q3-BBCA.xlsx")

	f := excelize.NewFile()

	sheetName := "Balance Sheet"
	sheetIdx, err := f.NewSheet(sheetName)
	if err != nil {
		t.Fatalf("create sheet: %v", err)
	}

	f.SetActiveSheet(sheetIdx)
	_ = f.DeleteSheet("Sheet1")
	_ = f.SetCellValue(sheetName, "A1", "Account")
	_ = f.SetCellValue(sheetName, "B1", "2024")
	_ = f.SetCellValue(sheetName, "A2", "Total Assets")
	_ = f.SetCellValue(sheetName, "B2", 1000000)

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	f.Close()

	return path
}

func createTestXBRLZip(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}

	w := zip.NewWriter(f)

	fw, err := w.Create("report.htm")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	content := `<html xmlns:ix="http://www.xbrl.org/2013/inlineXBRL">
<body>
<ix:nonFraction name="ifrs-full:Revenue" unitRef="IDR"
  contextRef="FY2024" decimals="-6">1500000</ix:nonFraction>
</body>
</html>`

	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	return path
}
