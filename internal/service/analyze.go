package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/lugassawan/idxlens/internal/idx"
)

// FetchForAnalyze fetches all available report formats for a ticker.
// Download errors are logged as warnings to errW but are non-fatal.
func FetchForAnalyze(
	ctx context.Context,
	errW io.Writer,
	fetcher IDXFetcher,
	ticker string,
	year int,
	period string,
) error {
	atts, err := fetcher.ListReports(ctx, ticker, year, period)
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

		if _, dlErr := fetcher.Download(ctx, att, destDir); dlErr != nil {
			fmt.Fprintf(errW, "Warning: failed to download %s: %v\n", att.FileName, dlErr)
			continue
		}
	}

	return nil
}
