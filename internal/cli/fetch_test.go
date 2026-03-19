package cli

import (
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
