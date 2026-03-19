package cli

import "testing"

func TestSilentError(t *testing.T) {
	err := &SilentError{ExitCode: 2}

	if err.Error() != "" {
		t.Errorf("Error() = %q, want empty string", err.Error())
	}

	if err.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", err.ExitCode)
	}
}
