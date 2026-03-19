package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lugassawan/idxlens/internal/domain"
	"github.com/lugassawan/idxlens/internal/pdf"
)

var extractESGCmd = &cobra.Command{
	Use:   "esg [pdf-path]",
	Short: "Extract ESG/GRI data from a sustainability report",
	Long: `Extract ESG/GRI content index data by running the L0-L3 pipeline:
PDF parsing, layout analysis, table detection, and ESG extraction.
Outputs GRI disclosures as JSON.`,
	Args: cobra.ExactArgs(1),
	RunE: runExtractESG,
}

func init() {
	extractCmd.AddCommand(extractESGCmd)
}

func runExtractESG(cmd *cobra.Command, args []string) error {
	pdfPath := args[0]

	report, err := extractESGReport(pdfPath)
	if err != nil {
		return err
	}

	return writeESGReport(cmd, report)
}

func extractESGReport(pdfPath string) (*domain.ESGReport, error) {
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

	tables, err := detectTables(pages)
	if err != nil {
		return nil, err
	}

	extractor := domain.NewESGExtractor()
	report := extractor.Extract(tables)

	if report == nil {
		return emptyESGReport(), nil
	}

	return report, nil
}

func emptyESGReport() *domain.ESGReport {
	return &domain.ESGReport{
		Framework:   "",
		Disclosures: []domain.GRIDisclosure{},
	}
}

func writeESGReport(cmd *cobra.Command, report *domain.ESGReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	out := cmd.OutOrStdout()

	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}

	return nil
}
