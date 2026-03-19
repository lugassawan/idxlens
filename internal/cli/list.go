package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/lugassawan/idxlens/internal/idx"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list TICKER[,TICKER...]",
	Short: "List available documents on IDX",
	Long:  `List financial report attachments available on IDX for the given tickers.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runList,
}

func init() {
	listCmd.Flags().Int(flagYear, 0, "Filter by report year")
	listCmd.Flags().String(flagPeriod, "", "Filter by report period (e.g. Q1, Q2, Q3, Audit)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	tickers := strings.Split(strings.ToUpper(args[0]), ",")
	year, _ := cmd.Flags().GetInt(flagYear)
	period, _ := cmd.Flags().GetString(flagPeriod)

	cookiePath, err := idx.CookiePath()
	if err != nil {
		return fmt.Errorf("resolve cookie path: %w", err)
	}

	cookies, err := idx.LoadCookies(cookiePath)
	if err != nil {
		return fmt.Errorf("load cookies: %w", err)
	}

	client := idx.New(idx.WithCookies(cookies))

	return listReports(cmd.Context(), cmd.OutOrStdout(), client, tickers, year, period)
}

// listReports is the testable core: accepts interfaces, no infrastructure construction.
func listReports(
	ctx context.Context,
	w io.Writer,
	lister ReportLister,
	tickers []string,
	year int,
	period string,
) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TICKER\tFILENAME\tTYPE\tSIZE\tPERIOD\tYEAR")

	for _, ticker := range tickers {
		attachments, err := lister.ListReports(ctx, ticker, year, period)
		if err != nil {
			return fmt.Errorf("list reports for %s: %w", ticker, err)
		}

		for _, att := range attachments {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
				att.EmitenCode, att.FileName, att.FileType, att.FileSize,
				att.ReportPeriod, att.ReportYear)
		}
	}

	return tw.Flush()
}
