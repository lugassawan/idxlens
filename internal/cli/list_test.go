package cli

import "testing"

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
