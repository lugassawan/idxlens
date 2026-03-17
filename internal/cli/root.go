package cli

import "github.com/spf13/cobra"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "idxlens",
	Short: "Extract structured financial data from IDX PDF reports",
	Long: `IDXLens is a CLI tool for extracting structured financial data
from Indonesia Stock Exchange (IDX) PDF reports. It converts
unstructured PDF tables into clean, machine-readable formats.`,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(classifyCmd)
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
