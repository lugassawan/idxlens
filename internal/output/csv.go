package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/lugassawan/idxlens/internal/domain"
)

type csvFormatter struct{}

func newCSVFormatter(_ *formatterConfig) *csvFormatter {
	return &csvFormatter{}
}

func (f *csvFormatter) Format(w io.Writer, stmt *domain.FinancialStatement) error {
	cw := csv.NewWriter(w)
	sorted := sortPeriods(stmt.Periods)

	header := buildHeader(sorted)
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for i := range stmt.Items {
		row := buildRow(&stmt.Items[i], sorted)
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write csv row %d: %w", i, err)
		}
	}

	cw.Flush()

	if err := cw.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}

func buildHeader(sortedPeriods []string) []string {
	header := make([]string, 0, 6+len(sortedPeriods))
	header = append(header, "Key", "Label", "Section", "Level", "IsSubtotal", "Confidence")
	header = append(header, sortedPeriods...)

	return header
}

func buildRow(item *domain.LineItem, sortedPeriods []string) []string {
	row := []string{
		item.Key,
		item.Label,
		item.Section,
		strconv.Itoa(item.Level),
		strconv.FormatBool(item.IsSubtotal),
		strconv.FormatFloat(item.Confidence, 'f', -1, 64),
	}

	for _, p := range sortedPeriods {
		v, ok := item.Values[p]
		if ok {
			row = append(row, strconv.FormatFloat(v, 'f', -1, 64))
		} else {
			row = append(row, "")
		}
	}

	return row
}

func sortPeriods(periods []string) []string {
	sorted := make([]string, len(periods))
	copy(sorted, periods)
	sort.Strings(sorted)

	return sorted
}
