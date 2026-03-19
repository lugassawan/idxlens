package cli

import "testing"

func TestAnalyzeCommandRegistered(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "analyze TICKER[,TICKER...]" {
			found = true

			if cmd.Short == "" {
				t.Error("analyze command has empty Short description")
			}

			break
		}
	}

	if !found {
		t.Error("analyze command not registered on rootCmd")
	}
}

func TestAnalyzeCommandFlags(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{"year flag", "year"},
		{"period flag", "period"},
		{"format flag", "format"},
		{"output flag", "output"},
		{"pretty flag", "pretty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := analyzeCmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Errorf("analyze command missing --%s flag", tt.flag)
			}
		})
	}
}

func TestBestFormat(t *testing.T) {
	tests := []struct {
		name   string
		files  []InputFile
		want   string
		wantNl bool
	}{
		{
			name:   "empty returns nil",
			files:  nil,
			wantNl: true,
		},
		{
			name: "prefers xlsx over pdf",
			files: []InputFile{
				{Path: "a.pdf", Format: "pdf"},
				{Path: "b.xlsx", Format: "xlsx"},
			},
			want: "xlsx",
		},
		{
			name: "prefers xlsx over xbrl",
			files: []InputFile{
				{Path: "a.zip", Format: "xbrl"},
				{Path: "b.xlsx", Format: "xlsx"},
			},
			want: "xlsx",
		},
		{
			name: "prefers xbrl over pdf",
			files: []InputFile{
				{Path: "a.pdf", Format: "pdf"},
				{Path: "b.zip", Format: "xbrl"},
			},
			want: "xbrl",
		},
		{
			name: "single pdf",
			files: []InputFile{
				{Path: "a.pdf", Format: "pdf"},
			},
			want: "pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestFormat(tt.files)

			if tt.wantNl {
				if got != nil {
					t.Errorf("bestFormat() = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("bestFormat() = nil, want non-nil")
			}

			if got.Format != tt.want {
				t.Errorf("bestFormat().Format = %q, want %q", got.Format, tt.want)
			}
		})
	}
}
