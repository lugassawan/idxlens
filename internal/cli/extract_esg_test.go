package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/domain"
)

func TestExtractESGMissingFile(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"extract", "esg", "/nonexistent/file.pdf"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	if !strings.Contains(err.Error(), "open file") {
		t.Errorf("error should contain %q, got %q", "open file", err.Error())
	}
}

func TestExtractESGInvalidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.pdf")

	if err := os.WriteFile(path, []byte("not a PDF"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"extract", "esg", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid PDF, got nil")
	}

	if !strings.Contains(err.Error(), "parse pdf") {
		t.Errorf("error should contain %q, got %q", "parse pdf", err.Error())
	}
}

func TestEmptyESGReport(t *testing.T) {
	report := emptyESGReport()

	if report == nil {
		t.Fatal("emptyESGReport returned nil")
	}

	if report.Framework != "" {
		t.Errorf("Framework = %q, want empty", report.Framework)
	}

	if report.Disclosures == nil {
		t.Error("Disclosures is nil, want non-nil empty slice")
	}

	if len(report.Disclosures) != 0 {
		t.Errorf("Disclosures length = %d, want 0", len(report.Disclosures))
	}
}

func TestWriteESGReport(t *testing.T) {
	tests := []struct {
		name   string
		report *domain.ESGReport
		want   string
	}{
		{
			name: "empty report",
			report: &domain.ESGReport{
				Framework:   "",
				Disclosures: []domain.GRIDisclosure{},
			},
			want: `"disclosures": []`,
		},
		{
			name: "report with disclosures",
			report: &domain.ESGReport{
				Framework: "GRI",
				Disclosures: []domain.GRIDisclosure{
					{
						Number: "201-1",
						Title:  "Direct economic value",
					},
				},
			},
			want: `"number": "201-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			err := writeESGReport(cmd, tt.report)
			if err != nil {
				t.Fatalf("writeESGReport: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("output should contain %q, got %q", tt.want, output)
			}

			// Verify valid JSON.
			var parsed map[string]any
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				t.Errorf("output is not valid JSON: %v", err)
			}
		})
	}
}
