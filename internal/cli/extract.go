package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/lugassawan/idxlens/internal/service"
	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract [TICKER|FILE]",
	Short: "Extract financial data from local cache or file",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)
	extractCmd.Flags().String(flagMode, modeFinancial, "extraction mode (financial, presentation)")
	registerYearPeriodFlags(extractCmd, false)
	registerOutputFlags(extractCmd)
}

func runExtract(cmd *cobra.Command, args []string) error {
	year, period := parseYearPeriodFlags(cmd)

	inputs, err := ResolveInputs(args[0], year, period)
	if err != nil {
		return fmt.Errorf("resolve inputs: %w", err)
	}

	outputPath, pretty := parseOutputFlags(cmd)

	w, cleanup, err := openWriter(cmd, outputPath)
	if err != nil {
		return err
	}
	defer cleanup()

	mode, _ := cmd.Flags().GetString(flagMode)

	var results []any

	for _, input := range inputs {
		result, err := service.ExtractFile(
			input.Path, input.Format, mode, input.Ticker, input.Year, input.Period,
		)
		if err != nil {
			return err
		}

		results = append(results, result)
	}

	return writeResults(w, results, pretty)
}

// writeResults writes a single result or a JSON array of multiple results.
func writeResults(w io.Writer, results []any, pretty bool) error {
	if len(results) == 0 {
		return nil
	}

	if len(results) == 1 {
		return writeJSON(w, results[0], pretty)
	}

	return writeJSON(w, results, pretty)
}

func writeJSON(w io.Writer, v any, pretty bool) error {
	data, err := marshalJSON(v, pretty)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	data = append(data, '\n')

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func marshalJSON(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}

	return json.Marshal(v)
}

func openWriter(cmd *cobra.Command, path string) (io.Writer, func(), error) {
	if path == "" {
		return cmd.OutOrStdout(), func() {}, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("create output file: %w", err)
	}

	bw := bufio.NewWriter(f)

	cleanup := func() {
		_ = bw.Flush()
		_ = f.Close()
	}

	return bw, cleanup, nil
}
