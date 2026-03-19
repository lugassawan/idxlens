package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPresentation(t *testing.T) {
	pdfPath := filepath.Join("..", "..", "tmp", "testdata", "IPCM", "IPCM_2025-Annual-Public-Expose.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not available")
	}

	pairs, err := ExtractPresentation(pdfPath)
	if err != nil {
		t.Fatalf("ExtractPresentation() error: %v", err)
	}

	if len(pairs) == 0 {
		t.Fatal("expected non-empty key-value pairs")
	}

	// Verify pairs have required fields
	for i, p := range pairs {
		if p.Key == "" {
			t.Errorf("pair[%d] has empty key", i)
		}

		if p.PageNum < 1 {
			t.Errorf("pair[%d] has invalid page number %d", i, p.PageNum)
		}
	}
}

func TestExtractPresentationNonExistentFile(t *testing.T) {
	_, err := ExtractPresentation("/nonexistent/file.pdf")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestExtractPresentationInvalidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.pdf")

	if err := os.WriteFile(path, []byte("not a pdf"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ExtractPresentation(path)
	if err == nil {
		t.Fatal("expected error for invalid PDF")
	}
}
