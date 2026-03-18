package cli

import (
	"testing"
)

func TestExecute(t *testing.T) {
	// Reset args to prevent interference from test flags.
	rootCmd.SetArgs([]string{"version"})

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
}
