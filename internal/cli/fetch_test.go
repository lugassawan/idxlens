package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/idx"
)

func TestFetchCommandRegistered(t *testing.T) {
	assertCommandRegistered(t, "fetch TICKER[,TICKER...]")
}

func TestFetchCommandFlags(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{"year flag", "year"},
		{"period flag", "period"},
		{"file-type flag", "file-type"},
		{"workers flag", "workers"},
		{"dry-run flag", "dry-run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fetchCmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Errorf("fetch command missing --%s flag", tt.flag)
			}
		})
	}
}

func TestRunFetchNoCookies(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	rootCmd.SetArgs([]string{"fetch", "BBCA", "--year", "2025"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when cookies file is missing")
	}
}

func TestRunFetchWithCookiesServerDown(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IDXLENS_HOME", dir)

	// Write valid but empty cookies file
	cookiePath := filepath.Join(dir, "cookies.json")
	if err := os.WriteFile(cookiePath, []byte("[]"), 0o600); err != nil {
		t.Fatalf("write cookies: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fetch", "BBCA", "--year", "2025"})

	// Will try to contact real IDX API and fail - exercises runFetch wiring
	err := rootCmd.Execute()
	// We don't care if it succeeds or fails - we're testing the wiring path
	_ = err
}

func TestFetchRequiresYear(t *testing.T) {
	// Reset flag to ensure no leftover state from other tests
	_ = fetchCmd.Flags().Set(flagYear, "0")
	fetchCmd.Flags().Lookup(flagYear).Changed = false

	rootCmd.SetArgs([]string{"fetch", "BBCA"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --year is missing")
	}
}

func TestFilterAttachments(t *testing.T) {
	atts := []idx.Attachment{
		{FileName: "report.pdf", FileType: "pdf"},
		{FileName: "data.xlsx", FileType: "xlsx"},
		{FileName: "xbrl.zip", FileType: "zip"},
		{FileName: "other.pdf", FileType: "pdf"},
	}

	tests := []struct {
		name     string
		fileType string
		want     int
	}{
		{"empty filter returns all", "", 4},
		{"filter pdf", "pdf", 2},
		{"filter xlsx", "xlsx", 1},
		{"filter zip", "zip", 1},
		{"filter nonexistent", "csv", 0},
		{"case insensitive", "PDF", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAttachments(atts, tt.fileType)
			if len(got) != tt.want {
				t.Errorf("filterAttachments(%q) returned %d, want %d", tt.fileType, len(got), tt.want)
			}
		})
	}
}

type fakeRegistry struct {
	registry map[string]idx.CompanyRegistry
	err      error
}

func (f *fakeRegistry) Registry(_ context.Context) (map[string]idx.CompanyRegistry, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.registry, nil
}

type fakeFetcher struct {
	reports map[string][]idx.Attachment
	listErr error
	dlErr   error
}

func (f *fakeFetcher) ListReports(_ context.Context, ticker string, _ int, _ string) ([]idx.Attachment, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	return f.reports[ticker], nil
}

func (f *fakeFetcher) Download(_ context.Context, att idx.Attachment, destDir string) (*idx.DownloadResult, error) {
	if f.dlErr != nil {
		return nil, f.dlErr
	}

	return &idx.DownloadResult{
		Attachment: att,
		LocalPath:  filepath.Join(destDir, att.FileName),
	}, nil
}

func TestFetchIDXDocuments(t *testing.T) {
	tests := []struct {
		name           string
		client         *fakeFetcher
		tickers        []string
		fileType       string
		wantErr        string
		wantDownloaded int
		wantFailed     int
	}{
		{
			name: "successful fetch single ticker",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{FileName: "report.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"},
					},
				},
			},
			tickers:        []string{"BBCA"},
			wantDownloaded: 1,
		},
		{
			name: "successful fetch multiple tickers",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "bbca.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"}},
					"BBRI": {{FileName: "bbri.xlsx", FileType: "xlsx", ReportYear: "2025", ReportPeriod: "Q1"}},
				},
			},
			tickers:        []string{"BBCA", "BBRI"},
			wantDownloaded: 2,
		},
		{
			name: "file type filter",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{FileName: "report.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"},
						{FileName: "data.xlsx", FileType: "xlsx", ReportYear: "2025", ReportPeriod: "Q1"},
					},
				},
			},
			tickers:        []string{"BBCA"},
			fileType:       "pdf",
			wantDownloaded: 1,
		},
		{
			name: "download error records failure",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "report.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"}},
				},
				dlErr: errors.New("download failed"),
			},
			tickers:    []string{"BBCA"},
			wantFailed: 1,
		},
		{
			name:    "list error returns error",
			client:  &fakeFetcher{listErr: errors.New("server error")},
			tickers: []string{"BBCA"},
			wantErr: "list reports for BBCA: server error",
		},
		{
			name: "empty results skipped",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{"BBCA": {}},
			},
			tickers: []string{"BBCA"},
		},
		{
			name: "filter removes all attachments",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "data.xlsx", FileType: "xlsx", ReportYear: "2025", ReportPeriod: "Q1"}},
				},
			},
			tickers:  []string{"BBCA"},
			fileType: "pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := fetchSummary{}

			err := fetchIDXDocuments(context.Background(), tt.client, tt.tickers, 0, "", tt.fileType, &summary)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}

				if err.Error() != tt.wantErr {
					t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(summary.Downloaded) != tt.wantDownloaded {
				t.Errorf("downloaded = %d, want %d", len(summary.Downloaded), tt.wantDownloaded)
			}

			if len(summary.Failed) != tt.wantFailed {
				t.Errorf("failed = %d, want %d", len(summary.Failed), tt.wantFailed)
			}
		})
	}
}

func TestFetchTickerDocuments(t *testing.T) {
	tests := []struct {
		name           string
		client         *fakeFetcher
		ticker         string
		fileType       string
		wantErr        bool
		wantDownloaded int
		wantFailed     int
	}{
		{
			name: "successful fetch",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{FileName: "report.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"},
					},
				},
			},
			ticker:         "BBCA",
			wantDownloaded: 1,
		},
		{
			name:    "list error",
			client:  &fakeFetcher{listErr: errors.New("server error")},
			ticker:  "BBCA",
			wantErr: true,
		},
		{
			name: "empty after filter",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "data.xlsx", FileType: "xlsx", ReportYear: "2025", ReportPeriod: "Q1"}},
				},
			},
			ticker:   "BBCA",
			fileType: "pdf",
		},
		{
			name: "download error records failure",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "report.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"}},
				},
				dlErr: errors.New("download failed"),
			},
			ticker:     "BBCA",
			wantFailed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloaded, failed, err := fetchTickerDocuments(
				context.Background(), tt.client, tt.ticker, t.TempDir(), 0, "", tt.fileType,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(downloaded) != tt.wantDownloaded {
				t.Errorf("downloaded = %d, want %d", len(downloaded), tt.wantDownloaded)
			}

			if len(failed) != tt.wantFailed {
				t.Errorf("failed = %d, want %d", len(failed), tt.wantFailed)
			}
		})
	}
}

func TestFetchIDXDocumentsConcurrency(t *testing.T) {
	client := &fakeFetcher{
		reports: map[string][]idx.Attachment{
			"BBCA": {{FileName: "bbca.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"}},
			"BBRI": {{FileName: "bbri.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"}},
			"BMRI": {{FileName: "bmri.pdf", FileType: "pdf", ReportYear: "2025", ReportPeriod: "Q1"}},
		},
	}

	summary := fetchSummary{}

	err := fetchIDXDocuments(context.Background(), client, []string{"BBCA", "BBRI", "BMRI"}, 0, "", "", &summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary.Downloaded) != 3 {
		t.Errorf("downloaded = %d, want 3", len(summary.Downloaded))
	}
}

func TestFetchPresentations(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		reg := &fakeRegistry{
			registry: map[string]idx.CompanyRegistry{
				"IPCM": {
					Name: "Jasa Armada",
					Presentations: []idx.PresentationEntry{
						{URL: "https://example.com/q1.pdf", Year: 2025, Period: "annual"},
					},
				},
			},
		}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 0, "", &summary)

		if len(summary.Downloaded) != 1 {
			t.Errorf("downloaded = %d, want 1", len(summary.Downloaded))
		}
	})

	t.Run("registry error returns early", func(t *testing.T) {
		reg := &fakeRegistry{err: errors.New("fetch failed")}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 0, "", &summary)

		if len(summary.Downloaded) != 0 {
			t.Errorf("downloaded = %d, want 0", len(summary.Downloaded))
		}
	})

	t.Run("ticker not in registry", func(t *testing.T) {
		reg := &fakeRegistry{registry: map[string]idx.CompanyRegistry{}}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"UNKNOWN"}, 0, "", &summary)

		if len(summary.Downloaded) != 0 {
			t.Errorf("downloaded = %d, want 0", len(summary.Downloaded))
		}
	})

	t.Run("year filter", func(t *testing.T) {
		reg := &fakeRegistry{
			registry: map[string]idx.CompanyRegistry{
				"IPCM": {
					Presentations: []idx.PresentationEntry{
						{URL: "https://example.com/2024.pdf", Year: 2024, Period: "annual"},
						{URL: "https://example.com/2025.pdf", Year: 2025, Period: "annual"},
					},
				},
			},
		}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 2025, "", &summary)

		if len(summary.Downloaded) != 1 {
			t.Errorf("downloaded = %d, want 1", len(summary.Downloaded))
		}
	})

	t.Run("period filter", func(t *testing.T) {
		reg := &fakeRegistry{
			registry: map[string]idx.CompanyRegistry{
				"IPCM": {
					Presentations: []idx.PresentationEntry{
						{URL: "https://example.com/annual.pdf", Year: 2025, Period: "annual"},
						{URL: "https://example.com/q1.pdf", Year: 2025, Period: "Q1"},
					},
				},
			},
		}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 0, "Q1", &summary)

		if len(summary.Downloaded) != 1 {
			t.Errorf("downloaded = %d, want 1", len(summary.Downloaded))
		}
	})

	t.Run("nil registry returns early", func(t *testing.T) {
		reg := &fakeRegistry{registry: nil}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 0, "", &summary)

		if len(summary.Downloaded) != 0 {
			t.Errorf("downloaded = %d, want 0", len(summary.Downloaded))
		}
	})

	t.Run("company with no presentations", func(t *testing.T) {
		reg := &fakeRegistry{
			registry: map[string]idx.CompanyRegistry{
				"IPCM": {Name: "Jasa Armada", Presentations: nil},
			},
		}
		dl := &fakeFetcher{}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 0, "", &summary)

		if len(summary.Downloaded) != 0 {
			t.Errorf("downloaded = %d, want 0", len(summary.Downloaded))
		}
	})

	t.Run("download error records failure", func(t *testing.T) {
		reg := &fakeRegistry{
			registry: map[string]idx.CompanyRegistry{
				"IPCM": {
					Presentations: []idx.PresentationEntry{
						{URL: "https://example.com/q1.pdf", Year: 2025, Period: "annual"},
					},
				},
			},
		}
		dl := &fakeFetcher{dlErr: errors.New("download failed")}
		summary := fetchSummary{}

		fetchPresentations(context.Background(), reg, dl, []string{"IPCM"}, 0, "", &summary)

		if len(summary.Failed) != 1 {
			t.Errorf("failed = %d, want 1", len(summary.Failed))
		}
	})
}

func TestDryRunFetch(t *testing.T) {
	tests := []struct {
		name     string
		client   *fakeFetcher
		tickers  []string
		fileType string
		wantErr  string
		wantOut  []string
	}{
		{
			name: "lists files without downloading",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{
							EmitenCode: "BBCA", FileName: "report.pdf", FileType: "pdf",
							FileSize: 1024, ReportYear: "2025", ReportPeriod: "Q1",
						},
						{
							EmitenCode: "BBCA", FileName: "data.xlsx", FileType: "xlsx",
							FileSize: 2048, ReportYear: "2025", ReportPeriod: "Q1",
						},
					},
				},
			},
			tickers: []string{"BBCA"},
			wantOut: []string{"TICKER", "report.pdf", "data.xlsx"},
		},
		{
			name: "filters by file type",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{
							EmitenCode: "BBCA", FileName: "report.pdf", FileType: "pdf",
							FileSize: 1024, ReportYear: "2025", ReportPeriod: "Q1",
						},
						{
							EmitenCode: "BBCA", FileName: "data.xlsx", FileType: "xlsx",
							FileSize: 2048, ReportYear: "2025", ReportPeriod: "Q1",
						},
					},
				},
			},
			tickers:  []string{"BBCA"},
			fileType: "pdf",
			wantOut:  []string{"report.pdf"},
		},
		{
			name:    "list error",
			client:  &fakeFetcher{listErr: errors.New("connection refused")},
			tickers: []string{"BBCA"},
			wantErr: "list reports for BBCA",
		},
		{
			name: "multiple tickers",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{
							EmitenCode:   "BBCA",
							FileName:     "bbca.pdf",
							FileType:     "pdf",
							ReportYear:   "2025",
							ReportPeriod: "Q1",
						},
					},
					"BBRI": {
						{
							EmitenCode:   "BBRI",
							FileName:     "bbri.xlsx",
							FileType:     "xlsx",
							ReportYear:   "2025",
							ReportPeriod: "Q1",
						},
					},
				},
			},
			tickers: []string{"BBCA", "BBRI"},
			wantOut: []string{"bbca.pdf", "bbri.xlsx"},
		},
		{
			name: "empty results",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{"BBCA": {}},
			},
			tickers: []string{"BBCA"},
			wantOut: []string{"TICKER"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := dryRunFetch(context.Background(), &buf, tt.client, tt.tickers, 0, "", tt.fileType)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want containing %q", err.Error(), tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOut {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\ngot: %s", want, output)
				}
			}
		})
	}
}
