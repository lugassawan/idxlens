package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/idx"
)

func TestListCommandRegistered(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "list TICKER[,TICKER...]" {
			found = true

			if cmd.Short == "" {
				t.Error("list command has empty Short description")
			}

			break
		}
	}

	if !found {
		t.Error("list command not registered on rootCmd")
	}
}

func TestListCommandFlags(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{"year flag", "year"},
		{"period flag", "period"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := listCmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Errorf("list command missing --%s flag", tt.flag)
			}
		})
	}
}

type fakeReportLister struct {
	reports map[string][]idx.Attachment
	err     error
}

func (f *fakeReportLister) ListReports(_ context.Context, ticker string, _ int, _ string) ([]idx.Attachment, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.reports[ticker], nil
}

func TestListReports(t *testing.T) {
	tests := []struct {
		name    string
		lister  *fakeReportLister
		tickers []string
		wantErr string
		wantOut []string
	}{
		{
			name: "single ticker with results",
			lister: &fakeReportLister{
				reports: map[string][]idx.Attachment{
					"BBCA": {
						{
							EmitenCode:   "BBCA",
							FileName:     "report.pdf",
							FileType:     "pdf",
							FileSize:     1024,
							ReportPeriod: "Q1",
							ReportYear:   "2025",
						},
					},
				},
			},
			tickers: []string{"BBCA"},
			wantOut: []string{"BBCA", "report.pdf", "pdf", "1024", "Q1", "2025"},
		},
		{
			name: "multiple tickers",
			lister: &fakeReportLister{
				reports: map[string][]idx.Attachment{
					"BBCA": {{EmitenCode: "BBCA", FileName: "bbca.pdf", FileType: "pdf"}},
					"BBRI": {{EmitenCode: "BBRI", FileName: "bbri.xlsx", FileType: "xlsx"}},
				},
			},
			tickers: []string{"BBCA", "BBRI"},
			wantOut: []string{"BBCA", "bbca.pdf", "BBRI", "bbri.xlsx"},
		},
		{
			name:    "empty results",
			lister:  &fakeReportLister{reports: map[string][]idx.Attachment{}},
			tickers: []string{"BBCA"},
			wantOut: []string{"TICKER"},
		},
		{
			name:    "lister error",
			lister:  &fakeReportLister{err: errors.New("connection refused")},
			tickers: []string{"BBCA"},
			wantErr: "list reports for BBCA: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := listReports(context.Background(), &buf, tt.lister, tt.tickers, 0, "")

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

			output := buf.String()
			for _, want := range tt.wantOut {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\ngot: %s", want, output)
				}
			}
		})
	}
}

func TestListReportsHeader(t *testing.T) {
	lister := &fakeReportLister{reports: map[string][]idx.Attachment{}}
	var buf bytes.Buffer

	err := listReports(context.Background(), &buf, lister, []string{"BBCA"}, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	header := "TICKER  FILENAME  TYPE  SIZE  PERIOD  YEAR"
	if !strings.Contains(buf.String(), header) {
		t.Errorf("output missing header\ngot: %s", buf.String())
	}
}
