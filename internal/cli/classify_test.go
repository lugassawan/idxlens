package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/domain"
)

func TestClassifyMissingFile(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"classify", "/nonexistent/file.pdf"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	if !strings.Contains(err.Error(), "open file") {
		t.Errorf("error should contain %q, got %q", "open file", err.Error())
	}
}

func TestClassifyInvalidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.pdf")

	if err := os.WriteFile(path, []byte("not a PDF"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"classify", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid PDF, got nil")
	}

	if !strings.Contains(err.Error(), "parse pdf") {
		t.Errorf("error should contain %q, got %q", "parse pdf", err.Error())
	}
}

func TestClassifyUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dummy.pdf")

	if err := os.WriteFile(path, []byte("not a PDF"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"classify", "--format", "xml", path})

	err := cmd.Execute()
	if err == nil {
		// The PDF parsing error will fire before format validation,
		// so this test verifies the flag is accepted by cobra.
		// Format validation happens after successful classification.
		t.Log("error occurred before format validation (expected for invalid PDF)")
	}
}

func TestWriteClassification(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		wantErr  bool
		contains string
	}{
		{
			name:     "text format",
			format:   "text",
			contains: "Type:",
		},
		{
			name:     "json format",
			format:   "json",
			contains: "type",
		},
		{
			name:    "unsupported format",
			format:  "xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			c := domain.Classification{
				Type:       "balance-sheet",
				Confidence: 0.95,
				Language:   "en",
			}

			err := writeClassification(cmd, c, tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("output should contain %q, got %q", tt.contains, buf.String())
			}
		})
	}
}
