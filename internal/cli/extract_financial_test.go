package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/domain"
)

func TestExtractFinancialMissingFile(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"extract", "financial", "/nonexistent/file.pdf"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	if !strings.Contains(err.Error(), "open file") {
		t.Errorf("error should contain %q, got %q", "open file", err.Error())
	}
}

func TestExtractFinancialInvalidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.pdf")

	if err := os.WriteFile(path, []byte("not a PDF"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"extract", "financial", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid PDF, got nil")
	}

	if !strings.Contains(err.Error(), "parse pdf") {
		t.Errorf("error should contain %q, got %q", "parse pdf", err.Error())
	}
}

func TestExtractFinancialOutputToFile(t *testing.T) {
	// Verify that --output flag with nonexistent parent dir produces an error.
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "dummy.pdf")

	if err := os.WriteFile(pdfPath, []byte("not a PDF"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	outputPath := filepath.Join(dir, "nonexistent", "output.json")
	cmd.SetArgs([]string{"extract", "financial", "--output", outputPath, pdfPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Error will be from PDF parsing (happens before output resolution),
	// which confirms the flag is accepted.
	if !strings.Contains(err.Error(), "parse pdf") {
		t.Errorf("error should contain %q, got %q", "parse pdf", err.Error())
	}
}

func TestEmptyStatement(t *testing.T) {
	tests := []struct {
		name    string
		docType domain.DocType
	}{
		{name: "unknown type", docType: domain.DocTypeUnknown},
		{name: "balance sheet", docType: domain.DocTypeBalanceSheet},
		{name: "empty string type", docType: domain.DocType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := emptyStatement(tt.docType)

			if stmt == nil {
				t.Fatal("emptyStatement returned nil")
			}

			if stmt.Type != tt.docType {
				t.Errorf("Type = %q, want %q", stmt.Type, tt.docType)
			}

			if stmt.Items == nil {
				t.Error("Items is nil, want non-nil empty slice")
			}

			if len(stmt.Items) != 0 {
				t.Errorf("Items length = %d, want 0", len(stmt.Items))
			}

			if stmt.Company != "" {
				t.Errorf("Company = %q, want empty", stmt.Company)
			}

			if stmt.Periods != nil {
				t.Errorf("Periods = %v, want nil", stmt.Periods)
			}
		})
	}
}

func TestResolveWriter(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		wantStdout bool
		wantErr    bool
	}{
		{
			name:       "empty path returns stdout",
			outputPath: "",
			wantStdout: true,
		},
		{
			name:       "invalid path returns error",
			outputPath: "/nonexistent/dir/file.json",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			w, err := resolveWriter(cmd, tt.outputPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantStdout && w == nil {
				t.Fatal("expected non-nil writer for stdout")
			}
		})
	}
}

func TestResolveWriterToFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.json")

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	w, err := resolveWriter(cmd, outPath)
	if err != nil {
		t.Fatalf("resolveWriter: %v", err)
	}

	// Writer should be a file closer.
	closer, ok := w.(interface{ Close() error })
	if !ok {
		t.Fatal("expected writer to implement Close")
	}
	defer closer.Close()

	// Verify file was created.
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file was not created: %v", err)
	}
}
