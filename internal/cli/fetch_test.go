package cli

import "testing"

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
