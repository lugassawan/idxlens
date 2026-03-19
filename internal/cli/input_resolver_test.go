package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFilePath(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{"file with extension", "report.xlsx", true},
		{"absolute path", "/data/report.xlsx", true},
		{"relative path", "data/report.xlsx", true},
		{"ticker only", "BBCA", false},
		{"multi-ticker", "BBCA,TLKM", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFilePath(tt.arg)
			if got != tt.want {
				t.Errorf("isFilePath(%q) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"xlsx", "report.xlsx", "xlsx"},
		{"zip", "report.zip", "xbrl"},
		{"pdf", "report.pdf", "pdf"},
		{"uppercase", "REPORT.XLSX", "xlsx"},
		{"unknown", "report.txt", ""},
		{"no extension", "report", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormat(tt.path)
			if got != tt.want {
				t.Errorf("detectFormat(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveFile(t *testing.T) {
	t.Run("valid xlsx file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.xlsx")

		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("create file: %v", err)
		}

		files, err := resolveFile(path)
		if err != nil {
			t.Fatalf("resolveFile() error: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("files count = %d, want 1", len(files))
		}

		if files[0].Format != "xlsx" {
			t.Errorf("Format = %q, want %q", files[0].Format, "xlsx")
		}

		if files[0].Path != path {
			t.Errorf("Path = %q, want %q", files[0].Path, path)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := resolveFile("/nonexistent/file.xlsx")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		dir := t.TempDir()

		_, err := resolveFile(dir)
		if err == nil {
			t.Fatal("expected error for directory")
		}
	})

	t.Run("unsupported extension", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.txt")

		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("create file: %v", err)
		}

		_, err := resolveFile(path)
		if err == nil {
			t.Fatal("expected error for unsupported extension")
		}
	})
}

func TestBuildGlobPattern(t *testing.T) {
	tests := []struct {
		name    string
		dataDir string
		ticker  string
		year    int
		period  string
		want    string
	}{
		{
			name:    "all wildcards",
			dataDir: "/data",
			ticker:  "BBCA",
			year:    0,
			period:  "",
			want:    filepath.Join("/data", "BBCA", "*", "*", "*"),
		},
		{
			name:    "with year",
			dataDir: "/data",
			ticker:  "BBCA",
			year:    2024,
			period:  "",
			want:    filepath.Join("/data", "BBCA", "2024", "*", "*"),
		},
		{
			name:    "with year and period",
			dataDir: "/data",
			ticker:  "BBCA",
			year:    2024,
			period:  "Q3",
			want:    filepath.Join("/data", "BBCA", "2024", "Q3", "*"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildGlobPattern(tt.dataDir, tt.ticker, tt.year, tt.period)
			if got != tt.want {
				t.Errorf("buildGlobPattern() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveInputsFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")

	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	files, err := ResolveInputs(path, 0, "")
	if err != nil {
		t.Fatalf("ResolveInputs() error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("files count = %d, want 1", len(files))
	}

	if files[0].Format != "pdf" {
		t.Errorf("Format = %q, want %q", files[0].Format, "pdf")
	}
}

func TestResolveInputsTicker(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	tickerDir := filepath.Join(dir, "data", "BBCA", "2024", "Q3")
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	xlsxPath := filepath.Join(tickerDir, "FinancialStatement-2024-Q3-BBCA.xlsx")
	if err := os.WriteFile(xlsxPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	files, err := ResolveInputs("BBCA", 2024, "Q3")
	if err != nil {
		t.Fatalf("ResolveInputs() error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("files count = %d, want 1", len(files))
	}

	if files[0].Format != "xlsx" {
		t.Errorf("Format = %q, want %q", files[0].Format, "xlsx")
	}

	if files[0].Ticker != "BBCA" {
		t.Errorf("Ticker = %q, want %q", files[0].Ticker, "BBCA")
	}
}

func TestResolveInputsTickerNoFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	_, err := ResolveInputs("XXXX", 0, "")
	if err == nil {
		t.Fatal("expected error for ticker with no files")
	}
}
