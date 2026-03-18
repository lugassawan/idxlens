package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lugassawan/idxlens/internal/domain"
	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/spf13/cobra"
)

const maxClassifyPages = 3

var classifyCmd = &cobra.Command{
	Use:   "classify [pdf-path]",
	Short: "Classify an IDX PDF report by type",
	Args:  cobra.ExactArgs(1),
	RunE:  runClassify,
}

func init() {
	rootCmd.AddCommand(classifyCmd)
	classifyCmd.Flags().StringP("format", "f", "text", "output format (text, json)")
}

func runClassify(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("read format flag: %w", err)
	}

	classification, err := classifyFile(filePath)
	if err != nil {
		return err
	}

	return writeClassification(cmd, classification, format)
}

func classifyFile(filePath string) (domain.Classification, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return domain.Classification{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := pdf.NewReader()
	if err := reader.Open(f); err != nil {
		return domain.Classification{}, fmt.Errorf("parse pdf: %w", err)
	}
	defer reader.Close()

	pages, err := analyzePages(reader)
	if err != nil {
		return domain.Classification{}, err
	}

	classifier := domain.NewHeuristicClassifier()
	classification, err := classifier.Classify(pages)
	if err != nil {
		return domain.Classification{}, fmt.Errorf("classify document: %w", err)
	}

	return classification, nil
}

func analyzePages(reader pdf.Reader) ([]layout.LayoutPage, error) {
	pageCount := min(reader.PageCount(), maxClassifyPages)

	analyzer := layout.NewAnalyzer()
	pages := make([]layout.LayoutPage, 0, pageCount)

	for i := 1; i <= pageCount; i++ {
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

func writeClassification(cmd *cobra.Command, c domain.Classification, format string) error {
	w := cmd.OutOrStdout()

	switch format {
	case "json":
		return writeClassificationJSON(cmd, c)
	case "text":
		fmt.Fprintf(w, "Type:       %s\n", c.Type)
		fmt.Fprintf(w, "Confidence: %.0f%%\n", c.Confidence*100)
		fmt.Fprintf(w, "Language:   %s\n", c.Language)
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func writeClassificationJSON(cmd *cobra.Command, c domain.Classification) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
