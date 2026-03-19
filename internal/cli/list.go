package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/lugassawan/idxlens/internal/idx"
	"github.com/lugassawan/idxlens/internal/service"
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
	listCmd.Flags().IntP(flagYear, "y", 0, descYearRequired)
	listCmd.Flags().StringP(flagPeriod, "p", "", descPeriod)
	_ = listCmd.MarkFlagRequired(flagYear)
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	tickers := strings.Split(strings.ToUpper(args[0]), ",")
	year, _ := cmd.Flags().GetInt(flagYear)
	period, _ := cmd.Flags().GetString(flagPeriod)

	client, err := idx.NewAuthenticatedClient()
	if err != nil {
		return err
	}

	return listReports(cmd.Context(), cmd.OutOrStdout(), client, tickers, year, period)
}

// listReports is the testable core: accepts interfaces, no infrastructure construction.
func listReports(
	ctx context.Context,
	w io.Writer,
	lister service.ReportLister,
	tickers []string,
	year int,
	period string,
) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TICKER\tFILENAME\tTYPE\tSIZE\tPERIOD\tYEAR")

	attResults := make([][]idx.Attachment, len(tickers))
	errResults := make([]error, len(tickers))

	var wg sync.WaitGroup

	for i, ticker := range tickers {
		wg.Add(1)

		go func(index int, t string) {
			defer wg.Done()

			atts, err := lister.ListReports(ctx, t, year, period)
			attResults[index] = atts
			errResults[index] = err
		}(i, ticker)
	}

	wg.Wait()

	var errs []error

	for i := range errResults {
		if errResults[i] != nil {
			errs = append(errs, fmt.Errorf("list reports for %s: %w", tickers[i], errResults[i]))

			continue
		}

		for _, att := range attResults[i] {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
				att.EmitenCode, att.FileName, att.FileType, att.FileSize,
				att.ReportPeriod, att.ReportYear)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	return tw.Flush()
}
