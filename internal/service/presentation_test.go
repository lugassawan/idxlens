package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPresentation(t *testing.T) {
	t.Run("real PDF extraction", func(t *testing.T) {
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

		for i, p := range pairs {
			if p.Key == "" {
				t.Errorf("pair[%d] has empty key", i)
			}

			if p.PageNum < 1 {
				t.Errorf("pair[%d] has invalid page number %d", i, p.PageNum)
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "non-existent file",
				setup: func(t *testing.T) string {
					t.Helper()
					return "/nonexistent/file.pdf"
				},
			},
			{
				name: "invalid PDF",
				setup: func(t *testing.T) string {
					t.Helper()
					dir := t.TempDir()
					path := filepath.Join(dir, "fake.pdf")
					if err := os.WriteFile(path, []byte("not a pdf"), 0o644); err != nil {
						t.Fatalf("write file: %v", err)
					}
					return path
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path := tt.setup(t)

				_, err := ExtractPresentation(path)
				if err == nil {
					t.Fatal("expected error")
				}
			})
		}
	})
}
