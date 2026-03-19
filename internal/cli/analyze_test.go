package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestAnalyzeCommandRegistered(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "analyze TICKER[,TICKER...]" {
			found = true

			if cmd.Short == "" {
				t.Error("analyze command has empty Short description")
			}

			break
		}
	}

	if !found {
		t.Error("analyze command not registered on rootCmd")
	}
}

func TestAnalyzeCommandFlags(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{"year flag", "year"},
		{"period flag", "period"},
		{"format flag", "format"},
		{"output flag", "output"},
		{"pretty flag", "pretty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := analyzeCmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Errorf("analyze command missing --%s flag", tt.flag)
			}
		})
	}
}

func TestRunAnalyzeNoCookies(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	rootCmd.SetArgs([]string{"analyze", "BBCA", "--year", "2024", "--period", "Q3"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when cookies file is missing")
	}
}

func TestRunAnalyzeWithExistingFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	// Create a fake PDF in the expected location
	tickerDir := filepath.Join(dir, "data", "BBCA", "2024", "Q3")
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	pdfPath := filepath.Join(tickerDir, "presentation.pdf")
	if err := os.WriteFile(pdfPath, []byte("fake pdf"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"analyze", "BBCA", "--year", "2024", "--period", "Q3"})

	// Will fail trying to parse the fake PDF, but it exercises analyzeTicker
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error parsing fake PDF")
	}
}

func TestRunAnalyzeWithXLSX(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	tickerDir := filepath.Join(dir, "data", "BBCA", "2024", "Q3")
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	// Create a real XLSX file
	xlsxPath := filepath.Join(tickerDir, "FinancialStatement-2024-Q3-BBCA.xlsx")
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

	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	f.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"analyze", "BBCA", "--year", "2024", "--period", "Q3"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected JSON output, got empty")
	}
}

func TestBestFormat(t *testing.T) {
	tests := []struct {
		name   string
		files  []InputFile
		want   string
		wantNl bool
	}{
		{
			name:   "empty returns nil",
			files:  nil,
			wantNl: true,
		},
		{
			name: "prefers xlsx over pdf",
			files: []InputFile{
				{Path: "a.pdf", Format: "pdf"},
				{Path: "b.xlsx", Format: "xlsx"},
			},
			want: "xlsx",
		},
		{
			name: "prefers xlsx over xbrl",
			files: []InputFile{
				{Path: "a.zip", Format: "xbrl"},
				{Path: "b.xlsx", Format: "xlsx"},
			},
			want: "xlsx",
		},
		{
			name: "prefers xbrl over pdf",
			files: []InputFile{
				{Path: "a.pdf", Format: "pdf"},
				{Path: "b.zip", Format: "xbrl"},
			},
			want: "xbrl",
		},
		{
			name: "single pdf",
			files: []InputFile{
				{Path: "a.pdf", Format: "pdf"},
			},
			want: "pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestFormat(tt.files)

			if tt.wantNl {
				if got != nil {
					t.Errorf("bestFormat() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("bestFormat() = nil, want non-nil")
			}

			if got.Format != tt.want {
				t.Errorf("bestFormat().Format = %q, want %q", got.Format, tt.want)
			}
		})
	}
}
