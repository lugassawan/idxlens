package cli

import "github.com/spf13/cobra"

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract structured data from an IDX PDF report",
}

func init() {
	rootCmd.AddCommand(extractCmd)
}
