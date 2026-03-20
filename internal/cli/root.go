package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:           "idxlens",
	Short:         "Extract structured financial data from IDX reports",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, _ []string) {
		printBanner(cmd)
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().Bool(flagVerbose, false, "enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}
