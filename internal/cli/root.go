package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "idxlens",
	Short: "Extract structured financial data from IDX reports",
	Long: `IDXLens extracts structured financial data from Indonesia Stock Exchange
(IDX) reports. Supports XLSX, XBRL, and PDF presentation formats.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}
