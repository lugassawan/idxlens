package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/lugassawan/idxlens/internal/domain"
	"github.com/lugassawan/idxlens/internal/output"
)

var batchCmd = &cobra.Command{
	Use:   "batch [glob-pattern]",
	Short: "Process multiple PDFs in batch",
	Long: `Process multiple PDF files matching a glob pattern using bounded
concurrency. Each file runs through the full extraction pipeline and
results are written to the specified output directory.`,
	Args: cobra.ExactArgs(1),
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.Flags().IntP(flagWorkers, "w", defaultWorkers, "number of concurrent workers")
	batchCmd.Flags().StringP(flagOutputDir, "d", "", "output directory for results")
	batchCmd.Flags().StringP(flagFormat, "f", defaultFormat, "output format (json, csv)")
	batchCmd.Flags().StringP(flagType, "t", "", "report type (e.g. balance-sheet, income-statement)")
}

func runBatch(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	flags, err := parseBatchFlags(cmd)
	if err != nil {
		return err
	}

	paths, err := expandGlob(pattern)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		return fmt.Errorf("no files matched pattern: %s", pattern)
	}

	processor := &domain.BatchProcessor{
		Workers:   flags.workers,
		OutputDir: flags.outputDir,
		DocType:   flags.docType,
		Extractor: &pipelineExtractor{},
	}

	summary := processor.Process(paths)

	return writeBatchSummary(cmd, summary)
}

type batchFlags struct {
	workers   int
	outputDir string
	format    output.Format
	docType   domain.DocType
}

func parseBatchFlags(cmd *cobra.Command) (batchFlags, error) {
	workers, err := cmd.Flags().GetInt(flagWorkers)
	if err != nil {
		return batchFlags{}, fmt.Errorf("read workers flag: %w", err)
	}

	if workers < 1 {
		return batchFlags{}, fmt.Errorf("workers must be at least 1, got %d", workers)
	}

	maxWorkers := runtime.NumCPU()
	if workers > maxWorkers {
		workers = maxWorkers
	}

	outputDir, err := cmd.Flags().GetString(flagOutputDir)
	if err != nil {
		return batchFlags{}, fmt.Errorf("read output-dir flag: %w", err)
	}

	formatStr, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return batchFlags{}, fmt.Errorf("read format flag: %w", err)
	}

	typeStr, err := cmd.Flags().GetString(flagType)
	if err != nil {
		return batchFlags{}, fmt.Errorf("read type flag: %w", err)
	}

	return batchFlags{
		workers:   workers,
		outputDir: outputDir,
		format:    output.Format(formatStr),
		docType:   domain.DocType(typeStr),
	}, nil
}

func expandGlob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	return matches, nil
}

// pipelineExtractor implements domain.FileExtractor using the existing
// extraction pipeline from extract_financial.go.
type pipelineExtractor struct{}

func (pe *pipelineExtractor) Extract(pdfPath string, docType domain.DocType) (*domain.FinancialStatement, error) {
	return extractStatement(pdfPath, docType)
}

func writeBatchSummary(cmd *cobra.Command, summary domain.BatchSummary) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")

	if err := enc.Encode(summary); err != nil {
		return fmt.Errorf("encode batch summary: %w", err)
	}

	return nil
}
