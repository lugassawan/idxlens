package cli

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/lugassawan/idxlens/internal/idx"
)

func TestFetchCommandRegistered(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "fetch TICKER[,TICKER...]" {
			found = true

			if cmd.Short == "" {
				t.Error("fetch command has empty Short description")
			}

			break
		}
	}

	if !found {
		t.Error("fetch command not registered on rootCmd")
	}
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
