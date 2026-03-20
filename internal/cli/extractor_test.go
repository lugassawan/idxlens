package cli

import "testing"

func TestGetExtractor(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"xlsx registered", formatXLSX, false},
		{"xbrl registered", formatXBRL, false},
		{"pdf registered", formatPDF, false},
		{"unknown format", "csv", true},
		{"empty format", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := getExtractor(tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if e == nil {
				t.Fatal("expected non-nil extractor")
			}
		})
	}
}

func TestExtractorRegistryCompleteness(t *testing.T) {
	formats := []string{formatXLSX, formatXBRL, formatPDF}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			if _, ok := extractorRegistry[f]; !ok {
				t.Errorf("format %q not registered", f)
			}
		})
	}
}
