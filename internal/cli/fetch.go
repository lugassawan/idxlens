package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/lugassawan/idxlens/internal/idx"
	"github.com/spf13/cobra"
)

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
	fetchCmd.Flags().Int(flagYear, 0, "Filter by report year")
	fetchCmd.Flags().String(flagPeriod, "", "Filter by report period (e.g. Q1, Q2, Q3, Audit)")
	fetchCmd.Flags().String(flagFileType, "", "Filter by file type (e.g. pdf, xlsx, zip)")
	fetchCmd.Flags().Int(flagWorkers, defaultWorkers, "Number of concurrent downloads")
	rootCmd.AddCommand(fetchCmd)
}

type fetchSummary struct {
	Downloaded []string `json:"downloaded"`
	Failed     []string `json:"failed"`
}

func runFetch(cmd *cobra.Command, args []string) error {
	tickers := strings.Split(strings.ToUpper(args[0]), ",")
	year, _ := cmd.Flags().GetInt(flagYear)
	period, _ := cmd.Flags().GetString(flagPeriod)
	fileType, _ := cmd.Flags().GetString(flagFileType)
	workers, _ := cmd.Flags().GetInt(flagWorkers)

	ctx := cmd.Context()
	summary := fetchSummary{}

	if err := fetchIDXDocuments(ctx, tickers, year, period, fileType, workers, &summary); err != nil {
		return err
	}

	fetchPresentations(ctx, tickers, year, period, workers, &summary, cmd)

	out, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(out))

	return nil
}

func fetchIDXDocuments(
	ctx context.Context, tickers []string,
	year int, period, fileType string, workers int, summary *fetchSummary,
) error {
	cookiePath, err := idx.CookiePath()
	if err != nil {
		return fmt.Errorf("resolve cookie path: %w", err)
	}

	cookies, err := idx.LoadCookies(cookiePath)
	if err != nil {
		return fmt.Errorf("load cookies: %w", err)
	}

	client := idx.New(idx.WithCookies(cookies))

	for _, ticker := range tickers {
		atts, err := client.ListReports(ctx, ticker, year, period)
		if err != nil {
			return fmt.Errorf("list reports for %s: %w", ticker, err)
		}

		filtered := filterAttachments(atts, fileType)
		if len(filtered) == 0 {
			continue
		}

		dataDir, err := idx.DataDir()
		if err != nil {
			return fmt.Errorf("resolve data directory: %w", err)
		}

		for _, att := range filtered {
			destDir := filepath.Join(dataDir, ticker, att.ReportYear, att.ReportPeriod)

			result, err := client.Download(ctx, att, destDir)
			if err != nil {
				summary.Failed = append(summary.Failed, att.FileName)
				continue
			}

			summary.Downloaded = append(summary.Downloaded, result.LocalPath)
		}
	}

	return nil
}

func fetchPresentations(
	ctx context.Context, tickers []string,
	year int, period string, workers int, summary *fetchSummary, cmd *cobra.Command,
) {
	regPath, err := idx.RegistryPath()
	if err != nil {
		return
	}

	registry, err := idx.LoadCachedRegistry(regPath)
	if err != nil {
		registry, err = idx.FetchRegistry(ctx)
		if err != nil {
			return
		}

		_ = idx.SaveCachedRegistry(regPath, registry)
	}

	client := &http.Client{}

	for _, ticker := range tickers {
		company, ok := registry[ticker]
		if !ok || len(company.Presentations) == 0 {
			continue
		}

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

			destDir := filepath.Join(dataDir, ticker, fmt.Sprintf("%d", pres.Year), pres.Period)
			att := idx.Attachment{
				FileName: filepath.Base(pres.URL),
				FilePath: pres.URL,
			}

			presClient := idx.New(idx.WithBaseURL(""), idx.WithHTTPClient(client))
			result, err := presClient.Download(ctx, att, destDir)
			if err != nil {
				summary.Failed = append(summary.Failed, att.FileName)
				continue
			}

			summary.Downloaded = append(summary.Downloaded, result.LocalPath)
		}
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
