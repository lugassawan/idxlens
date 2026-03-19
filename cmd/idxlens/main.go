package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/lugassawan/idxlens/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		var silent *cli.SilentError
		if errors.As(err, &silent) {
			os.Exit(silent.ExitCode)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
