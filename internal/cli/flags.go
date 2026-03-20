package cli

import "github.com/spf13/cobra"

// registerYearPeriodFlags adds --year/-y and --period/-p flags.
func registerYearPeriodFlags(cmd *cobra.Command, yearRequired bool) {
	cmd.Flags().IntP(flagYear, "y", 0, descYearRequired)
	cmd.Flags().StringP(flagPeriod, "p", "", descPeriod)

	if yearRequired {
		_ = cmd.MarkFlagRequired(flagYear)
	}
}

// registerOutputFlags adds --format/-f, --output/-o, and --pretty flags.
func registerOutputFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(flagFormat, "f", defaultFormat, "output format (json, csv)")
	cmd.Flags().StringP(flagOutput, "o", "", "output file path")
	cmd.Flags().Bool(flagPretty, false, "pretty-print JSON output")
}

// parseYearPeriodFlags reads year and period flag values from the command.
func parseYearPeriodFlags(cmd *cobra.Command) (int, string) {
	year, _ := cmd.Flags().GetInt(flagYear)
	period, _ := cmd.Flags().GetString(flagPeriod)

	return year, period
}

// parseOutputFlags reads output and pretty flag values from the command.
func parseOutputFlags(cmd *cobra.Command) (string, bool) {
	outputPath, _ := cmd.Flags().GetString(flagOutput)
	pretty, _ := cmd.Flags().GetBool(flagPretty)

	return outputPath, pretty
}
