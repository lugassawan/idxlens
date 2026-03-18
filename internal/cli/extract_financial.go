package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/lugassawan/idxlens/internal/domain"
	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/output"
	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/lugassawan/idxlens/internal/table"
)

var errUnknownDocType = errors.New("classify document: unable to determine report type, use --type flag")

var extractFinancialCmd = &cobra.Command{
	Use:   "financial [pdf-path]",
	Short: "Extract structured financial data from an IDX PDF report",
	Long: `Extract structured financial data by running the full L0-L4 pipeline:
PDF parsing, layout analysis, document classification, table detection,
financial statement mapping, and output formatting.`,
	Args: cobra.ExactArgs(1),
	RunE: runExtractFinancial,
}

func init() {
	extractCmd.AddCommand(extractFinancialCmd)
	extractFinancialCmd.Flags().StringP("type", "t", "", "report type (e.g. balance-sheet, income-statement)")
	extractFinancialCmd.Flags().StringP(flagFormat, "f", "json", "output format (json, csv)")
	extractFinancialCmd.Flags().StringP("output", "o", "", "output file path (default: stdout)")
	extractFinancialCmd.Flags().Bool("pretty", false, "pretty-print output (JSON only)")
}

func runExtractFinancial(cmd *cobra.Command, args []string) error {
	pdfPath := args[0]

	flags, err := parseFinancialFlags(cmd)
	if err != nil {
		return err
	}

	stmt, err := extractStatement(pdfPath, flags.docType)
	if err != nil {
		return err
	}

	return writeStatement(cmd, stmt, flags)
}

type financialFlags struct {
	docType    domain.DocType
	format     output.Format
	outputPath string
	pretty     bool
}

func parseFinancialFlags(cmd *cobra.Command) (financialFlags, error) {
	typeStr, err := cmd.Flags().GetString("type")
	if err != nil {
		return financialFlags{}, fmt.Errorf("read type flag: %w", err)
	}

	formatStr, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return financialFlags{}, fmt.Errorf("read format flag: %w", err)
	}

	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		return financialFlags{}, fmt.Errorf("read output flag: %w", err)
	}

	pretty, err := cmd.Flags().GetBool("pretty")
	if err != nil {
		return financialFlags{}, fmt.Errorf("read pretty flag: %w", err)
	}

	return financialFlags{
		docType:    domain.DocType(typeStr),
		format:     output.Format(formatStr),
		outputPath: outputPath,
		pretty:     pretty,
	}, nil
}

func extractStatement(pdfPath string, docType domain.DocType) (*domain.FinancialStatement, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := pdf.NewReader()
	if err := reader.Open(f); err != nil {
		return nil, fmt.Errorf("parse pdf: %w", err)
	}
	defer reader.Close()

	pages, err := analyzeAllPages(reader)
	if err != nil {
		return nil, err
	}

	resolvedType, err := resolveDocType(docType, pages)
	if err != nil {
		return nil, err
	}

	tables, err := detectTables(pages)
	if err != nil {
		return nil, err
	}

	mapper := domain.NewMapper()
	stmt, err := mapper.Map(resolvedType, tables)
	if err != nil {
		return nil, fmt.Errorf("map financial data: %w", err)
	}

	return stmt, nil
}

func analyzeAllPages(reader pdf.Reader) ([]layout.LayoutPage, error) {
	analyzer := layout.NewAnalyzer()
	pages := make([]layout.LayoutPage, 0, reader.PageCount())

	for i := 1; i <= reader.PageCount(); i++ {
		page, err := reader.Page(i)
		if err != nil {
			return nil, fmt.Errorf("read page %d: %w", i, err)
		}

		layoutPage, err := analyzer.Analyze(page)
		if err != nil {
			return nil, fmt.Errorf("analyze page %d: %w", i, err)
		}

		pages = append(pages, layoutPage)
	}

	return pages, nil
}

func resolveDocType(docType domain.DocType, pages []layout.LayoutPage) (domain.DocType, error) {
	if docType != "" {
		return docType, nil
	}

	classifier := domain.NewHeuristicClassifier()
	classification, err := classifier.Classify(pages)
	if err != nil {
		return "", fmt.Errorf("classify document: %w", err)
	}

	if classification.Type == domain.DocTypeUnknown {
		return "", errUnknownDocType
	}

	return classification.Type, nil
}

func detectTables(pages []layout.LayoutPage) ([]table.Table, error) {
	detector := table.NewDetector()

	var tables []table.Table
	for _, page := range pages {
		pageTables, err := detector.Detect(page)
		if err != nil {
			return nil, fmt.Errorf("detect tables on page %d: %w", page.Number, err)
		}

		tables = append(tables, pageTables...)
	}

	return tables, nil
}

func writeStatement(cmd *cobra.Command, stmt *domain.FinancialStatement, flags financialFlags) error {
	formatter, err := output.NewFormatter(flags.format, output.WithPretty(flags.pretty))
	if err != nil {
		return fmt.Errorf("create formatter: %w", err)
	}

	w, err := resolveWriter(cmd, flags.outputPath)
	if err != nil {
		return err
	}

	if closer, ok := w.(io.Closer); ok {
		defer closer.Close()
	}

	if err := formatter.Format(w, stmt); err != nil {
		return fmt.Errorf("format output: %w", err)
	}

	return nil
}

func resolveWriter(cmd *cobra.Command, outputPath string) (io.Writer, error) {
	if outputPath == "" {
		return cmd.OutOrStdout(), nil
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("create output file: %w", err)
	}

	return f, nil
}
