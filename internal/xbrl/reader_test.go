package xbrl

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

const inlineXBRLHTML = `<html
  xmlns:ix="http://www.xbrl.org/2013/inlineXBRL"
  xmlns:ifrs="http://xbrl.ifrs.org/taxonomy/2023">
<body>
<ix:nonFraction name="ifrs-full:Revenue" unitRef="IDR"
  contextRef="FY2024" decimals="-6">1500000</ix:nonFraction>
<ix:nonFraction name="ifrs-full:CostOfSales" unitRef="IDR"
  contextRef="FY2024" decimals="-6">800000</ix:nonFraction>
<ix:nonNumeric name="ifrs-full:NameOfEntity"
  contextRef="FY2024">PT Example Tbk</ix:nonNumeric>
</body>
</html>`

const standardXBRL = `<?xml version="1.0" encoding="UTF-8"?>
<xbrl xmlns="http://www.xbrl.org/2003/instance"
  xmlns:ifrs="http://xbrl.ifrs.org/taxonomy/2023">
<ifrs:Revenue contextRef="FY2024" unitRef="IDR" decimals="-6">2500000</ifrs:Revenue>
<ifrs:Assets contextRef="FY2024" unitRef="IDR" decimals="-6">9000000</ifrs:Assets>
</xbrl>`

func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}

	w := zip.NewWriter(f)

	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}

		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	return path
}

func TestParseZip(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantFacts int
		wantErr   bool
	}{
		{
			name:      "inline XBRL HTML",
			files:     map[string]string{"report.htm": inlineXBRLHTML},
			wantFacts: 3,
		},
		{
			name:      "standard XBRL",
			files:     map[string]string{"report.xbrl": standardXBRL},
			wantFacts: 2,
		},
		{
			name:    "empty zip",
			files:   map[string]string{},
			wantErr: true,
		},
		{
			name:    "no XBRL files",
			files:   map[string]string{"readme.txt": "hello"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestZip(t, tt.files)
			stmt, err := ParseZip(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(stmt.Facts) != tt.wantFacts {
				t.Errorf("got %d facts, want %d", len(stmt.Facts), tt.wantFacts)
			}
		})
	}
}

func TestParseZipNonExistentFile(t *testing.T) {
	_, err := ParseZip("/nonexistent/path/test.zip")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestParseZipInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.zip")

	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ParseZip(path)
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestParseZipMetadata(t *testing.T) {
	files := map[string]string{"report.htm": inlineXBRLHTML}

	dir := t.TempDir()
	path := filepath.Join(dir, "FinancialStatement-2024-Q3-BBCA.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	w := zip.NewWriter(f)

	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry: %v", err)
		}

		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("write entry: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	stmt, err := ParseZip(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stmt.Ticker != "BBCA" {
		t.Errorf("ticker = %q, want %q", stmt.Ticker, "BBCA")
	}

	if stmt.Year != 2024 {
		t.Errorf("year = %d, want %d", stmt.Year, 2024)
	}

	if stmt.Period != "Q3" {
		t.Errorf("period = %q, want %q", stmt.Period, "Q3")
	}
}

func TestParseZipFactValues(t *testing.T) {
	path := createTestZip(t, map[string]string{"report.htm": inlineXBRLHTML})

	stmt, err := ParseZip(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false

	for _, fact := range stmt.Facts {
		if fact.Concept == "ifrs-full:Revenue" {
			found = true

			if fact.Value != 1500000 {
				t.Errorf("revenue value = %f, want %f", fact.Value, 1500000.0)
			}

			if fact.Unit != "IDR" {
				t.Errorf("unit = %q, want %q", fact.Unit, "IDR")
			}

			if fact.Decimals != "-6" {
				t.Errorf("decimals = %q, want %q", fact.Decimals, "-6")
			}
		}
	}

	if !found {
		t.Error("ifrs-full:Revenue fact not found")
	}
}
