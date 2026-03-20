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
	Short: "Fetch and extract financial data",
	Long: `Full pipeline: fetch all available formats from IDX, then extract from
the best format (XBRL > XLSX > PDF for presentations). Falls back to locally
cached files if fetch fails.`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	registerYearPeriodFlags(analyzeCmd, true)
	registerOutputFlags(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	tickers := strings.Split(strings.ToUpper(args[0]), ",")
	year, period := parseYearPeriodFlags(cmd)
	outputPath, pretty := parseOutputFlags(cmd)

	w, cleanup, err := openWriter(cmd, outputPath)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := cmd.Context()

	var errs []error

	errW := cmd.ErrOrStderr()

	for _, ticker := range tickers {
		if err := analyzeTicker(ctx, w, errW, ticker, year, period, pretty); err != nil {
			errs = append(errs, fmt.Errorf("analyze %s: %w", ticker, err))
		}
	}

	return errors.Join(errs...)
}

func analyzeTicker(
	ctx context.Context, w io.Writer, errW io.Writer,
	ticker string, year int, period string, pretty bool,
) error {
	fmt.Fprintf(errW, "Analyzing %s...\n", ticker)

	// Always fetch to ensure all formats are available for best selection.
	// Auth or fetch errors are non-fatal if local files already exist.
	var fetchErr error

	client, err := idx.NewAuthenticatedClient()
	if err != nil {
		fetchErr = fmt.Errorf("create client: %w", err)
	} else {
		fetchErr = fetchForTicker(ctx, errW, client, ticker, year, period)
	}

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

func fetchForTicker(
	ctx context.Context, errW io.Writer, client service.IDXFetcher,
	ticker string, year int, period string,
) error {
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
			fmt.Fprintf(errW, "Warning: failed to download %s: %v\n", att.FileName, dlErr)
			continue
		}
	}

	return nil
}
