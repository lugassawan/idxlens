package cli

import (
	"bytes"
	"testing"
)

func TestRunAuthFails(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"auth"})

	// Will fail because Chrome/chromedp is not available in test env
	err := rootCmd.Execute()
	if err == nil {
		t.Skip("auth succeeded (Chrome available)")
	}
}

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
