package cli

import "testing"

func assertCommandRegistered(t *testing.T, use string) {
	t.Helper()

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == use {
			if cmd.Short == "" {
				t.Errorf("%s command has empty Short description", use)
			}

			return
		}
	}

	t.Fatalf("%s command not registered on rootCmd", use)
}
