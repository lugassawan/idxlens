package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatchNoFilesMatched(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"batch", "/nonexistent/path/*.pdf"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for no matching files, got nil")
	}

	if !strings.Contains(err.Error(), "no files matched pattern") {
		t.Errorf("error should contain %q, got %q", "no files matched pattern", err.Error())
	}
}

func TestBatchInvalidGlobPattern(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"batch", "[invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid glob pattern, got nil")
	}

	if !strings.Contains(err.Error(), "invalid glob pattern") {
		t.Errorf("error should contain %q, got %q", "invalid glob pattern", err.Error())
	}
}

func TestBatchInvalidWorkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dummy.pdf")

	if err := os.WriteFile(path, []byte("not a PDF"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"batch", "--workers", "0", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for zero workers, got nil")
	}

	if !strings.Contains(err.Error(), "workers must be at least 1") {
		t.Errorf("error should contain %q, got %q", "workers must be at least 1", err.Error())
	}
}

func TestBatchProcessesFiles(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"a.pdf", "b.pdf"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("not a PDF"), 0o644); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
	}

	pattern := filepath.Join(dir, "*.pdf")

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"batch", "--workers", "2", pattern})

	err := cmd.Execute()
	// Files are not valid PDFs, so extraction will fail per file,
	// but the batch command itself should succeed and report results.
	if err != nil {
		t.Fatalf("batch command should succeed even with failed files, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "total_files") {
		t.Error("output should contain total_files field")
	}

	if !strings.Contains(output, `"failed": 2`) {
		t.Errorf("output should show 2 failed files, got: %s", output)
	}
}

func TestExpandGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid pattern",
			pattern: "[invalid",
			wantErr: true,
			errMsg:  "invalid glob pattern",
		},
		{
			name:    "no matches returns empty slice",
			pattern: "/nonexistent/path/*.pdf",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := expandGlob(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error should contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(matches) != 0 {
				t.Errorf("expected 0 matches, got %d", len(matches))
			}
		})
	}
}
