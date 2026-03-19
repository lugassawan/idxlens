package cli

import "testing"

func TestAuthCommandRegistered(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "auth" {
			found = true

			if cmd.Short == "" {
				t.Error("auth command has empty Short description")
			}

			break
		}
	}

	if !found {
		t.Error("auth command not registered on rootCmd")
	}
}
