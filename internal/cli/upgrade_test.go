package cli

import "testing"

func TestUpgradeCommandRegistered(t *testing.T) {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "upgrade" {
			found = true

			if cmd.Short == "" {
				t.Error("upgrade command has empty Short description")
			}

			break
		}
	}

	if !found {
		t.Error("upgrade command not registered on rootCmd")
	}
}
