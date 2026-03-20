package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewLogger(t *testing.T) {
	t.Run("verbose enabled writes to stderr", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool(flagVerbose, false, "")
		_ = cmd.Flags().Set(flagVerbose, "true")

		var errBuf bytes.Buffer
		cmd.SetErr(&errBuf)

		logger := newLogger(cmd)
		logger.Info("test message")

		if !strings.Contains(errBuf.String(), "test message") {
			t.Errorf("expected log output, got: %q", errBuf.String())
		}
	})

	t.Run("verbose disabled discards logs", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool(flagVerbose, false, "")

		var errBuf bytes.Buffer
		cmd.SetErr(&errBuf)

		logger := newLogger(cmd)
		logger.Info("test message")

		if errBuf.Len() != 0 {
			t.Errorf("expected no output, got: %q", errBuf.String())
		}
	})
}
