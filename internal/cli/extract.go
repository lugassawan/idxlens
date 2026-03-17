package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract [pdf-path]",
	Short: "Extract structured data from an IDX PDF report",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("extract: not yet implemented")
		return nil
	},
}

func init() {
	extractCmd.Flags().StringP("type", "t", "", "report type (e.g. balance-sheet, income-statement)")
	extractCmd.Flags().StringP("output", "o", "", "output file path")
}
