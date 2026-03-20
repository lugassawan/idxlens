package cli

import (
	"io"
	"log/slog"

	"github.com/spf13/cobra"
)

// newLogger creates a slog.Logger based on the --verbose flag.
// When verbose is enabled, logs go to stderr. When disabled, logs are discarded.
func newLogger(cmd *cobra.Command) *slog.Logger {
	verbose, _ := cmd.Flags().GetBool(flagVerbose)

	var w io.Writer
	if verbose {
		w = cmd.ErrOrStderr()
	} else {
		w = io.Discard
	}

	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
