package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/xuri/excelize/v2"
)

// errWriter is an io.Writer that always returns an error.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

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

func TestExtractPDFFinancialModeError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")

	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	rootCmd.SetArgs([]string{"extract", path})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for PDF financial extraction")
	}
}

func TestExtractPresentationWithRealPDF(t *testing.T) {
	// Use a real PDF from testdata to exercise the full extractPresentation path
	pdfPath := filepath.Join("..", "..", "tmp", "testdata", "IPCM", "IPCM_2025-Annual-Public-Expose.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not available")
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"extract", pdfPath, "--mode", "presentation"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected JSON output, got empty")
	}
}

func TestExtractPresentationWithRealPDFPretty(t *testing.T) {
	pdfPath := filepath.Join("..", "..", "tmp", "testdata", "IPCM", "IPCM_2025-Annual-Public-Expose.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not available")
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"extract", pdfPath, "--mode", "presentation", "--pretty"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected JSON output, got empty")
	}
}

func TestExtractPDFPresentationModeRouting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")

	// Write a fake PDF — the presentation pipeline will attempt to parse it
	// and fail, confirming the routing reached extractPresentation.
	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	rootCmd.SetArgs([]string{"extract", path, "--mode", "presentation"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error parsing fake PDF in presentation mode")
	}

	// The error should come from PDF parsing, not the "not supported" message.
	got := err.Error()
	if got == "PDF financial extraction not supported in v2 (use XLSX or XBRL)" {
		t.Error("presentation mode should not return financial mode error")
	}
}

func TestExtractXLSXToFile(t *testing.T) {
	xlsxPath := createTestXLSX(t)
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "output.json")

	rootCmd.SetArgs([]string{"extract", xlsxPath, "--output", outputPath, "--pretty"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("output file is empty")
	}
}

func TestExtractNonExistentFile(t *testing.T) {
	rootCmd.SetArgs([]string{"extract", "/nonexistent/file.xlsx"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestExtractFileUnsupportedFormat(t *testing.T) {
	_, err := extractFile(InputFile{Path: "test.dat", Format: "dat"}, "financial")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}

	want := "unsupported format: dat"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestExtractFileAppliesMetadata(t *testing.T) {
	path := createTestXLSX(t)

	// InputFile has metadata but the file is named with standard IDX pattern,
	// so parseMeta will fill it from filename. Test with a non-matching name.
	dir := t.TempDir()
	nonStandardPath := filepath.Join(dir, "report.xlsx")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	if err := os.WriteFile(nonStandardPath, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	input := InputFile{
		Path:   nonStandardPath,
		Format: formatXLSX,
		Ticker: "PGAS",
		Year:   2025,
		Period: "Audit",
	}

	result, err := extractFile(input, modeFinancial)
	if err != nil {
		t.Fatalf("extractFile() error: %v", err)
	}

	// The result should have metadata from InputFile since filename doesn't match
	var buf bytes.Buffer
	if err := writeJSON(&buf, result.Value(), false); err != nil {
		t.Fatalf("writeJSON() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte(`"ticker":"PGAS"`)) {
		t.Errorf("expected ticker PGAS in output: %s", output)
	}

	if !bytes.Contains([]byte(output), []byte(`"year":2025`)) {
		t.Errorf("expected year 2025 in output: %s", output)
	}

	if !bytes.Contains([]byte(output), []byte(`"period":"Audit"`)) {
		t.Errorf("expected period Audit in output: %s", output)
	}
}

func TestWriteResultsSingle(t *testing.T) {
	var buf bytes.Buffer

	results := []any{map[string]int{"a": 1}}
	if err := writeResults(&buf, results, false); err != nil {
		t.Fatalf("writeResults() error: %v", err)
	}

	want := "{\"a\":1}\n"
	if buf.String() != want {
		t.Errorf("writeResults() = %q, want %q", buf.String(), want)
	}
}

func TestWriteResultsMultiple(t *testing.T) {
	var buf bytes.Buffer

	results := []any{
		map[string]int{"a": 1},
		map[string]int{"b": 2},
	}

	if err := writeResults(&buf, results, false); err != nil {
		t.Fatalf("writeResults() error: %v", err)
	}

	want := "[{\"a\":1},{\"b\":2}]\n"
	if buf.String() != want {
		t.Errorf("writeResults() = %q, want %q", buf.String(), want)
	}
}

func TestWriteResultsMultiplePretty(t *testing.T) {
	var buf bytes.Buffer

	results := []any{
		map[string]int{"a": 1},
		map[string]int{"b": 2},
	}

	if err := writeResults(&buf, results, true); err != nil {
		t.Fatalf("writeResults() error: %v", err)
	}

	output := buf.String()
	if output[0] != '[' {
		t.Errorf("expected JSON array, got: %s", output)
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name   string
		v      any
		pretty bool
		want   string
	}{
		{
			name: "compact",
			v:    map[string]int{"a": 1},
			want: "{\"a\":1}\n",
		},
		{
			name:   "pretty",
			v:      map[string]int{"a": 1},
			pretty: true,
			want:   "{\n  \"a\": 1\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeJSON(&buf, tt.v, tt.pretty); err != nil {
				t.Fatalf("writeJSON() error: %v", err)
			}

			if buf.String() != tt.want {
				t.Errorf("writeJSON() = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestWriteJSONMarshalError(t *testing.T) {
	// Channels cannot be marshalled to JSON
	var buf bytes.Buffer
	err := writeJSON(&buf, make(chan int), false)
	if err == nil {
		t.Fatal("expected error for unmarshalable value")
	}
}

func TestWriteJSONWriteError(t *testing.T) {
	w := &errWriter{}
	err := writeJSON(w, map[string]int{"a": 1}, false)
	if err == nil {
		t.Fatal("expected error for write failure")
	}
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		v      any
		pretty bool
		want   string
	}{
		{"compact", []int{1, 2}, false, "[1,2]"},
		{"pretty", []int{1, 2}, true, "[\n  1,\n  2\n]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalJSON(tt.v, tt.pretty)
			if err != nil {
				t.Fatalf("marshalJSON() error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("marshalJSON() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestOpenWriter(t *testing.T) {
	t.Run("empty path returns stdout", func(t *testing.T) {
		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		w, cleanup, err := openWriter(cmd, "")
		if err != nil {
			t.Fatalf("openWriter() error: %v", err)
		}
		defer cleanup()

		if w == nil {
			t.Fatal("writer is nil")
		}

		// Write and verify it goes to the cmd output
		_, _ = w.Write([]byte("hello"))
		if buf.String() != "hello" {
			t.Errorf("output = %q, want %q", buf.String(), "hello")
		}
	})

	t.Run("file path creates file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "output.json")

		cmd := &cobra.Command{}

		w, cleanup, err := openWriter(cmd, path)
		if err != nil {
			t.Fatalf("openWriter() error: %v", err)
		}

		_, _ = w.Write([]byte("test data"))
		cleanup()

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read output file: %v", err)
		}

		if string(got) != "test data" {
			t.Errorf("file content = %q, want %q", string(got), "test data")
		}
	})

	t.Run("invalid path returns error", func(t *testing.T) {
		cmd := &cobra.Command{}

		_, _, err := openWriter(cmd, "/nonexistent/dir/output.json")
		if err == nil {
			t.Fatal("expected error for invalid path")
		}
	})
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
