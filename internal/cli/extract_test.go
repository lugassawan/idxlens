package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractCommandRegistration(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "extract [TICKER|FILE]" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("extract command not registered on root")
	}
}

func TestExtractCommandFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"mode", ""},
		{flagYear, "y"},
		{flagPeriod, "p"},
		{flagFormat, "f"},
		{"output", "o"},
		{"pretty", ""},
	}

	for _, tt := range flags {
		t.Run(tt.name, func(t *testing.T) {
			f := extractCmd.Flags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("flag %q not found", tt.name)
			}

			if tt.shorthand != "" && f.Shorthand != tt.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", tt.name, f.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestExtractXLSXFile(t *testing.T) {
	path := createTestXLSX(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"extract", path})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("expected JSON output, got empty")
	}

	if !bytes.Contains([]byte(output), []byte(`"ticker"`)) {
		t.Errorf("output missing ticker field: %s", output)
	}
}

func TestExtractXBRLNotImplemented(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.zip")

	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	rootCmd.SetArgs([]string{"extract", path})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for XBRL extraction")
	}
}

func TestExtractPDFNotImplemented(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")

	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	rootCmd.SetArgs([]string{"extract", path})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for PDF extraction")
	}
}

func TestExtractNonExistentFile(t *testing.T) {
	rootCmd.SetArgs([]string{"extract", "/nonexistent/file.xlsx"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent file")
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

	if err := f.DeleteSheet("Sheet1"); err != nil {
		t.Fatalf("delete default sheet: %v", err)
	}

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
