package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lugassawan/idxlens/internal/idx"
	"github.com/lugassawan/idxlens/internal/service"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze TICKER[,TICKER...]",
	Short: "Fetch (if needed) and extract financial data",
	Long: `Full pipeline: fetch all available formats from IDX, then extract from
the best format (XBRL > XLSX > PDF for presentations). Falls back to locally
cached files if fetch fails.`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().IntP(flagYear, "y", 0, descYearRequired)
	analyzeCmd.Flags().StringP(flagPeriod, "p", "", descPeriod)
	_ = analyzeCmd.MarkFlagRequired(flagYear)
	analyzeCmd.Flags().StringP(flagFormat, "f", defaultFormat, "output format (json, csv)")
	analyzeCmd.Flags().StringP(flagOutput, "o", "", "output file path")
	analyzeCmd.Flags().Bool(flagPretty, false, "pretty-print JSON output")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	tickers := strings.Split(strings.ToUpper(args[0]), ",")
	year, _ := cmd.Flags().GetInt(flagYear)
	period, _ := cmd.Flags().GetString(flagPeriod)
	pretty, _ := cmd.Flags().GetBool(flagPretty)
	outputPath, _ := cmd.Flags().GetString(flagOutput)

	w, cleanup, err := openWriter(cmd, outputPath)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := cmd.Context()

	var errs []error

	for _, ticker := range tickers {
		if err := analyzeTicker(ctx, w, ticker, year, period, pretty); err != nil {
			errs = append(errs, fmt.Errorf("analyze %s: %w", ticker, err))
		}
	}

	return errors.Join(errs...)
}

func analyzeTicker(
	ctx context.Context, w io.Writer,
	ticker string, year int, period string, pretty bool,
) error {
	client, err := idx.NewAuthenticatedClient()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Always fetch to ensure all formats are available for best selection.
	// Fetch errors are non-fatal if local files already exist.
	fetchErr := fetchForTicker(ctx, client, ticker, year, period)

	files, err := ResolveInputs(ticker, year, period)
	if err != nil {
		if fetchErr != nil {
			return fetchErr
		}

		return fmt.Errorf("no files available for %s: %w", ticker, err)
	}

	best := bestFormat(files)
	if best == nil {
		return fmt.Errorf("no extractable files for %s", ticker)
	}

	mode := modeFinancial
	if best.Format == formatPDF {
		mode = modePresentation
	}

	return extractFile(w, *best, mode, pretty)
}

func bestFormat(files []InputFile) *InputFile {
	priority := map[string]int{formatXBRL: 3, formatXLSX: 2, formatPDF: 1}

	var best *InputFile

	for i := range files {
		if best == nil || priority[files[i].Format] > priority[best.Format] {
			best = &files[i]
		}
	}

	return best
}

func fetchForTicker(ctx context.Context, client service.IDXFetcher, ticker string, year int, period string) error {
	atts, err := client.ListReports(ctx, ticker, year, period)
	if err != nil {
		return err
	}

	if len(atts) == 0 {
		return fmt.Errorf("no reports found for %s on IDX", ticker)
	}

	dataDir, err := idx.DataDir()
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}

	for _, att := range atts {
		destDir := filepath.Join(dataDir, ticker, att.ReportYear, att.ReportPeriod)

		if _, dlErr := client.Download(ctx, att, destDir); dlErr != nil {
			continue
		}
	}

	return nil
}
