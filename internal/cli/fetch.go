package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lugassawan/idxlens/internal/idx"
	"github.com/lugassawan/idxlens/internal/service"
	"github.com/spf13/cobra"
)

type fetchSummary struct {
	Downloaded []string `json:"downloaded"`
	Failed     []string `json:"failed"`
}

var fetchCmd = &cobra.Command{
	Use:   "fetch TICKER[,TICKER...]",
	Short: "Download documents from IDX to local cache",
	Long: `Download financial report attachments and presentations for the given tickers.
IDX documents (XLSX, XBRL, PDFs) are fetched via the IDX API.
Presentations are fetched from the registry (company IR pages, no Cloudflare).`,
	Args: cobra.ExactArgs(1),
	RunE: runFetch,
}

func init() {
	registerYearPeriodFlags(fetchCmd, true)
	fetchCmd.Flags().String(flagFileType, "", "Filter by file type (e.g. pdf, xlsx, zip)")
	fetchCmd.Flags().Int(flagWorkers, defaultWorkers, "Number of concurrent downloads")
	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	tickers := strings.Split(strings.ToUpper(args[0]), ",")
	year, period := parseYearPeriodFlags(cmd)
	fileType, _ := cmd.Flags().GetString(flagFileType)

	client, err := idx.NewAuthenticatedClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	summary := fetchSummary{}

	if err := fetchIDXDocuments(ctx, client, tickers, year, period, fileType, &summary); err != nil {
		return err
	}

	presClient := idx.New(idx.WithBaseURL(""), idx.WithHTTPClient(&http.Client{}))
	regProvider := &service.DefaultRegistryProvider{}
	fetchPresentations(ctx, regProvider, presClient, tickers, year, period, &summary)

	out, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(out))

	return nil
}

func fetchIDXDocuments(
	ctx context.Context, client service.IDXFetcher, tickers []string,
	year int, period, fileType string, summary *fetchSummary,
) error {
	dataDir, err := idx.DataDir()
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}

	downloadedSlices := make([][]string, len(tickers))
	failedSlices := make([][]string, len(tickers))
	errSlices := make([]error, len(tickers))

	var wg sync.WaitGroup

	for i, ticker := range tickers {
		wg.Add(1)

		go func(index int, t string) {
			defer wg.Done()

			dl, fl, err := fetchTickerDocuments(ctx, client, t, dataDir, year, period, fileType)
			downloadedSlices[index] = dl
			failedSlices[index] = fl
			errSlices[index] = err
		}(i, ticker)
	}

	wg.Wait()

	var errs []error

	for i := range errSlices {
		if errSlices[i] != nil {
			errs = append(errs, fmt.Errorf("list reports for %s: %w", tickers[i], errSlices[i]))
		}

		summary.Downloaded = append(summary.Downloaded, downloadedSlices[i]...)
		summary.Failed = append(summary.Failed, failedSlices[i]...)
	}

	return errors.Join(errs...)
}

func fetchTickerDocuments(
	ctx context.Context, client service.IDXFetcher, ticker, dataDir string,
	year int, period, fileType string,
) (downloaded, failed []string, err error) {
	atts, err := client.ListReports(ctx, ticker, year, period)
	if err != nil {
		return nil, nil, err
	}

	filtered := filterAttachments(atts, fileType)
	if len(filtered) == 0 {
		return nil, nil, nil
	}

	for _, att := range filtered {
		destDir := filepath.Join(dataDir, ticker, att.ReportYear, att.ReportPeriod)

		result, dlErr := client.Download(ctx, att, destDir)
		if dlErr != nil {
			failed = append(failed, att.FileName)
			continue
		}

		downloaded = append(downloaded, result.LocalPath)
	}

	return downloaded, failed, nil
}

func fetchPresentations(
	ctx context.Context, reg service.RegistryProvider, dl service.FileDownloader,
	tickers []string, year int, period string, summary *fetchSummary,
) {
	registry, err := reg.Registry(ctx)
	if err != nil || registry == nil {
		return
	}

	for _, ticker := range tickers {
		company, ok := registry[ticker]
		if !ok || len(company.Presentations) == 0 {
			continue
		}

		fetchCompanyPresentations(ctx, dl, ticker, company, year, period, summary)
	}
}

func fetchCompanyPresentations(
	ctx context.Context, dl service.FileDownloader, ticker string,
	company idx.CompanyRegistry, year int, period string, summary *fetchSummary,
) {
	for _, pres := range company.Presentations {
		if year != 0 && pres.Year != year {
			continue
		}

		if period != "" && !strings.EqualFold(pres.Period, period) {
			continue
		}

		dataDir, err := idx.DataDir()
		if err != nil {
			continue
		}

		destDir := filepath.Join(dataDir, ticker, strconv.Itoa(pres.Year), pres.Period)
		att := idx.Attachment{
			FileName: filepath.Base(pres.URL),
			FilePath: pres.URL,
		}

		result, err := dl.Download(ctx, att, destDir)
		if err != nil {
			summary.Failed = append(summary.Failed, att.FileName)
			continue
		}

		summary.Downloaded = append(summary.Downloaded, result.LocalPath)
	}
}

func filterAttachments(atts []idx.Attachment, fileType string) []idx.Attachment {
	if fileType == "" {
		return atts
	}

	var filtered []idx.Attachment

	for _, att := range atts {
		if strings.EqualFold(att.FileType, fileType) {
			filtered = append(filtered, att)
		}
	}

	return filtered
}
