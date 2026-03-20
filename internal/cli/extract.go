package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

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

	for _, input := range inputs {
		if err := extractFile(w, input, mode, pretty); err != nil {
			return err
		}
	}

	return nil
}

func extractFile(w io.Writer, input InputFile, mode string, pretty bool) error {
	e, err := getExtractor(input.Format)
	if err != nil {
		return err
	}

	return e.Extract(w, input.Path, mode, pretty)
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
