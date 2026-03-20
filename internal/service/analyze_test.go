package service

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/idx"
)

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

func TestFetchForAnalyze(t *testing.T) {
	tests := []struct {
		name       string
		client     *fakeFetcher
		ticker     string
		wantErr    string
		wantStderr string
	}{
		{
			name:    "list error",
			client:  &fakeFetcher{listErr: errors.New("connection refused")},
			ticker:  "BBCA",
			wantErr: "connection refused",
		},
		{
			name: "no reports",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{"BBCA": {}},
			},
			ticker:  "BBCA",
			wantErr: "no reports found for BBCA on IDX",
		},
		{
			name: "download failure logs warning",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "report.pdf", FileType: "pdf", ReportYear: "2024", ReportPeriod: "Q3"}},
				},
				dlErr: errors.New("download failed"),
			},
			ticker:     "BBCA",
			wantStderr: "Warning: failed to download report.pdf",
		},
		{
			name: "successful fetch",
			client: &fakeFetcher{
				reports: map[string][]idx.Attachment{
					"BBCA": {{FileName: "report.pdf", FileType: "pdf", ReportYear: "2024", ReportPeriod: "Q3"}},
				},
			},
			ticker: "BBCA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IDXLENS_HOME", t.TempDir())

			var errBuf bytes.Buffer

			err := FetchForAnalyze(context.Background(), &errBuf, tt.client, tt.ticker, 2024, "Q3")

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

			if tt.wantStderr != "" && !strings.Contains(errBuf.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want containing %q", errBuf.String(), tt.wantStderr)
			}
		})
	}
}
