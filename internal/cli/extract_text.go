package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

var extractTextCmd = &cobra.Command{
	Use:   "text [pdf-path]",
	Short: "Extract text lines from a PDF",
	Long: `Extract text lines from a PDF by running the L0 (PDF parser) and
L1 (layout analyzer) pipeline. Outputs one text line per line, grouped by page.`,
	Args: cobra.ExactArgs(1),
	RunE: runExtractText,
}

func init() {
	extractCmd.AddCommand(extractTextCmd)
	extractTextCmd.Flags().String("pages", "", "page range (e.g. \"1-3,5,7-9\")")
}

func runExtractText(cmd *cobra.Command, args []string) error {
	pdfPath := args[0]

	pagesFlag, err := cmd.Flags().GetString("pages")
	if err != nil {
		return fmt.Errorf("get pages flag: %w", err)
	}

	f, err := os.Open(pdfPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := pdf.NewReader()

	if err := reader.Open(f); err != nil {
		return fmt.Errorf("parse pdf: %w", err)
	}
	defer reader.Close()

	pages, err := resolvePages(pagesFlag, reader.PageCount())
	if err != nil {
		return fmt.Errorf("resolve pages: %w", err)
	}

	analyzer := layout.NewAnalyzer()
	out := cmd.OutOrStdout()

	for i, pageNum := range pages {
		page, err := reader.Page(pageNum)
		if err != nil {
			return fmt.Errorf("read page %d: %w", pageNum, err)
		}

		layoutPage, err := analyzer.Analyze(page)
		if err != nil {
			return fmt.Errorf("analyze page %d: %w", pageNum, err)
		}

		fmt.Fprintf(out, "--- Page %d ---\n", pageNum)

		for _, line := range layoutPage.Lines {
			fmt.Fprintln(out, line.Text)
		}

		if i < len(pages)-1 {
			fmt.Fprintln(out)
		}
	}

	return nil
}

func resolvePages(pagesFlag string, totalPages int) ([]int, error) {
	if pagesFlag == "" {
		return allPages(totalPages), nil
	}

	return parsePageRange(pagesFlag, totalPages)
}

func allPages(totalPages int) []int {
	pages := make([]int, totalPages)
	for i := range pages {
		pages[i] = i + 1
	}

	return pages
}

func parsePageRange(spec string, totalPages int) ([]int, error) {
	seen := make(map[int]bool)

	for part := range strings.SplitSeq(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if err := parsePart(part, totalPages, seen); err != nil {
			return nil, err
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("empty page range: %q", spec)
	}

	return sortedKeys(seen), nil
}

func parsePart(part string, totalPages int, seen map[int]bool) error {
	if strings.Contains(part, "-") {
		return parseRangePart(part, totalPages, seen)
	}

	return parseSinglePage(part, totalPages, seen)
}

func parseSinglePage(s string, totalPages int, seen map[int]bool) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid page number: %q", s)
	}

	if n < 1 || n > totalPages {
		return fmt.Errorf("page %d out of range [1, %d]", n, totalPages)
	}

	seen[n] = true

	return nil
}

func parseRangePart(part string, totalPages int, seen map[int]bool) error {
	bounds := strings.SplitN(part, "-", 2)
	if len(bounds) != 2 || bounds[0] == "" || bounds[1] == "" {
		return fmt.Errorf("invalid range: %q", part)
	}

	start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
	if err != nil {
		return fmt.Errorf("invalid range start: %q", bounds[0])
	}

	end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
	if err != nil {
		return fmt.Errorf("invalid range end: %q", bounds[1])
	}

	if start > end {
		return fmt.Errorf("invalid range: start %d > end %d", start, end)
	}

	if start < 1 || end > totalPages {
		return fmt.Errorf("range %d-%d out of range [1, %d]", start, end, totalPages)
	}

	for i := start; i <= end; i++ {
		seen[i] = true
	}

	return nil
}

func sortedKeys(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	return keys
}
