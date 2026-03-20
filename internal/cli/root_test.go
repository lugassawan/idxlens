package cli

import (
	"testing"
)

func TestVerboseFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup(flagVerbose)
	if f == nil {
		t.Fatal("rootCmd missing --verbose persistent flag")
	}

	if f.DefValue != "false" {
		t.Errorf("verbose default = %q, want %q", f.DefValue, "false")
	}
}

func TestExecute(t *testing.T) {
	// Reset args to prevent interference from test flags.
	rootCmd.SetArgs([]string{"version"})

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
}
