package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	colorGreen = "\033[32m"
	colorBold  = "\033[1m"
	colorGray  = "\033[90m"
	colorReset = "\033[0m"
)

func printBanner(cmd *cobra.Command) {
	noColor := os.Getenv("NO_COLOR") != ""

	lines := []string{
		`  ___ ____  __  __ _`,
		` |_ _|  _ \ \ \/ /| |    ___ _ __  ___`,
		`  | || | | | \  / | |   / _ \ '_ \/ __|`,
		`  | || |_| | /  \ | |__|  __/ | | \__ \`,
		` |___|____/ /_/\_\|_____\___|_| |_|___/`,
	}

	versionStr := version
	if len(versionStr) > 0 && versionStr[0] != 'v' {
		versionStr = "v" + versionStr
	}

	for i, line := range lines {
		colored := paintLine(line, noColor)
		if i == 1 {
			fmt.Fprintf(cmd.OutOrStdout(), "%s    %s\n", colored, paintVersion(versionStr, noColor))
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), colored)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
}

func paintLine(s string, noColor bool) string {
	if noColor {
		return s
	}

	return colorGreen + colorBold + s + colorReset
}

func paintVersion(s string, noColor bool) string {
	if noColor {
		return s
	}

	return colorGray + s + colorReset
}
