package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var classifyCmd = &cobra.Command{
	Use:   "classify [pdf-path]",
	Short: "Classify an IDX PDF report by type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("classify: not yet implemented")
		return nil
	},
}
