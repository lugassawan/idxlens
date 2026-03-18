package testutil

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCorpus(t *testing.T) {
	m, err := LoadCorpus()
	if err != nil {
		t.Fatalf("LoadCorpus() error: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}

	if len(m.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(m.Entries))
	}
}

func TestCorpusManifestParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVersion int
		wantEntries int
		wantErr     bool
	}{
		{
			name:        "empty entries",
			input:       `{"version": 1, "entries": []}`,
			wantVersion: 1,
			wantEntries: 0,
		},
		{
			name: "single entry",
			input: `{
				"version": 1,
				"entries": [{
					"file": "test.pdf",
					"classification": {"type": "balance_sheet", "language": "id"},
					"page_count": 3,
					"description": "test document"
				}]
			}`,
			wantVersion: 1,
			wantEntries: 1,
		},
		{
			name: "entry without classification",
			input: `{
				"version": 1,
				"entries": [{
					"file": "test.pdf",
					"page_count": 2,
					"description": "minimal entry"
				}]
			}`,
			wantVersion: 1,
			wantEntries: 1,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m CorpusManifest

			err := json.Unmarshal([]byte(tt.input), &m)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if m.Version != tt.wantVersion {
				t.Errorf("Version = %d, want %d", m.Version, tt.wantVersion)
			}

			if len(m.Entries) != tt.wantEntries {
				t.Errorf("Entries length = %d, want %d", len(m.Entries), tt.wantEntries)
			}
		})
	}
}

func TestCorpusEntryPDFPath(t *testing.T) {
	entry := CorpusEntry{File: "sample.pdf"}
	path := entry.PDFPath()

	if path == "" {
		t.Fatal("PDFPath() returned empty string")
	}

	wantSuffix := filepath.Join("testdata", "corpus", "sample.pdf")
	if !strings.HasSuffix(path, wantSuffix) {
		t.Errorf("PDFPath() = %q, want suffix %q", path, wantSuffix)
	}
}
