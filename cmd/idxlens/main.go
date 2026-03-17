package main

import (
	"os"

	"github.com/lugassawan/idxlens/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
