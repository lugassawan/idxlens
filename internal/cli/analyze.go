package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lugassawan/idxlens/internal/idx"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze TICKER[,TICKER...]",
	Short: "Fetch (if needed) and extract financial data",
	Long: `Full pipeline: check local cache for files, fetch from IDX if missing,
then extract from the best available format (XLSX > XBRL > PDF for presentations).`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().IntP(flagYear, "y", 0, "report year")
	analyzeCmd.Flags().StringP(flagPeriod, "p", "", "report period")
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

	for _, ticker := range tickers {
		if err := analyzeTicker(ctx, w, ticker, year, period, pretty); err != nil {
			return fmt.Errorf("analyze %s: %w", ticker, err)
		}
	}

	return nil
}

func analyzeTicker(
	ctx context.Context, w io.Writer,
	ticker string, year int, period string, pretty bool,
) error {
	files, err := ResolveInputs(ticker, year, period)
	if err != nil {
		if fetchErr := fetchForTicker(ctx, ticker, year, period); fetchErr != nil {
			return fmt.Errorf("fetch: %w", fetchErr)
		}

		files, err = ResolveInputs(ticker, year, period)
		if err != nil {
			return fmt.Errorf("resolve after fetch: %w", err)
		}
	}

	best := bestFormat(files)
	if best == nil {
		return fmt.Errorf("no extractable files for %s", ticker)
	}

	mode := "financial"
	if best.Format == formatPDF {
		mode = "presentation"
	}

	return extractFile(w, *best, mode, pretty)
}

func bestFormat(files []InputFile) *InputFile {
	priority := map[string]int{formatXLSX: 3, formatXBRL: 2, formatPDF: 1}

	var best *InputFile

	for i := range files {
		if best == nil || priority[files[i].Format] > priority[best.Format] {
			best = &files[i]
		}
	}

	return best
}

func fetchForTicker(ctx context.Context, ticker string, year int, period string) error {
	cookiePath, err := idx.CookiePath()
	if err != nil {
		return fmt.Errorf("resolve cookie path: %w", err)
	}

	cookies, err := idx.LoadCookies(cookiePath)
	if err != nil {
		return fmt.Errorf("load cookies: %w", err)
	}

	client := idx.New(idx.WithCookies(cookies))

	atts, err := client.ListReports(ctx, ticker, year, period)
	if err != nil {
		return fmt.Errorf("list reports: %w", err)
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
