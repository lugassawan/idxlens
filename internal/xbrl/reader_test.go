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

const inlineXBRLHTML2 = `<html
  xmlns:ix="http://www.xbrl.org/2013/inlineXBRL"
  xmlns:ifrs="http://xbrl.ifrs.org/taxonomy/2023">
<body>
<ix:nonFraction name="ifrs-full:Assets" unitRef="IDR"
  contextRef="FY2024" decimals="-6">9000000</ix:nonFraction>
<ix:nonFraction name="ifrs-full:Liabilities" unitRef="IDR"
  contextRef="FY2024" decimals="-6">3000000</ix:nonFraction>
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

	return createTestZipNamed(t, "test.zip", files)
}

func createTestZipNamed(t *testing.T, zipName string, files map[string]string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), zipName)

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
			name: "multiple inline XBRL HTML files",
			files: map[string]string{
				"1000000.html": inlineXBRLHTML,
				"1210000.html": inlineXBRLHTML2,
			},
			wantFacts: 5,
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
	path := createTestZipNamed(t, "FinancialStatement-2024-Q3-BBCA.zip",
		map[string]string{"report.htm": inlineXBRLHTML})

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

func TestParseZipRealData(t *testing.T) {
	paths := []struct {
		path   string
		ticker string
	}{
		{"../../tmp/testdata/ADRO/inlineXBRL.zip", "ADRO"},
		{"../../tmp/testdata/IPCM/inlineXBRL.zip", "IPCM"},
	}

	for _, p := range paths {
		t.Run(p.ticker, func(t *testing.T) {
			if _, err := os.Stat(p.path); os.IsNotExist(err) {
				t.Skip("test data not available")
			}

			stmt, err := ParseZip(p.path)
			if err != nil {
				t.Fatalf("ParseZip() error: %v", err)
			}

			if len(stmt.Facts) <= 41 {
				t.Errorf("expected more than 41 facts (metadata only), got %d", len(stmt.Facts))
			}

			t.Logf("%s: %d facts extracted", p.ticker, len(stmt.Facts))
		})
	}
}
