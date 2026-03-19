package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/lugassawan/idxlens/internal/xlsx"
)

var extractCmd = &cobra.Command{
	Use:   "extract [TICKER|FILE]",
	Short: "Extract financial data from local cache or file",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)
	extractCmd.Flags().String("mode", "financial", "extraction mode (financial, presentation)")
	extractCmd.Flags().IntP(flagYear, "y", 0, "report year")
	extractCmd.Flags().StringP(flagPeriod, "p", "", "report period")
	extractCmd.Flags().StringP(flagFormat, "f", defaultFormat, "output format (json, csv)")
	extractCmd.Flags().StringP("output", "o", "", "output file path")
	extractCmd.Flags().Bool("pretty", false, "pretty-print JSON output")
}

func runExtract(cmd *cobra.Command, args []string) error {
	year, _ := cmd.Flags().GetInt(flagYear)
	period, _ := cmd.Flags().GetString(flagPeriod)

	inputs, err := ResolveInputs(args[0], year, period)
	if err != nil {
		return fmt.Errorf("resolve inputs: %w", err)
	}

	pretty, _ := cmd.Flags().GetBool("pretty")
	outputPath, _ := cmd.Flags().GetString("output")

	w, cleanup, err := openWriter(cmd, outputPath)
	if err != nil {
		return err
	}
	defer cleanup()

	for _, input := range inputs {
		if err := extractFile(w, input, pretty); err != nil {
			return err
		}
	}

	return nil
}

func extractFile(w io.Writer, input InputFile, pretty bool) error {
	switch input.Format {
	case formatXLSX:
		return extractXLSX(w, input.Path, pretty)
	case formatXBRL:
		return errors.New("XBRL extraction not yet implemented")
	case formatPDF:
		return errors.New("PDF extraction not yet implemented")
	default:
		return fmt.Errorf("unsupported format: %s", input.Format)
	}
}

func extractXLSX(w io.Writer, path string, pretty bool) error {
	stmt, err := xlsx.Parse(path)
	if err != nil {
		return fmt.Errorf("parse xlsx: %w", err)
	}

	return writeJSON(w, stmt, pretty)
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

	cleanup := func() {
		_ = f.Close()
	}

	return f, cleanup, nil
}
