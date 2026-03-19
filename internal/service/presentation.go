package service

import (
	"fmt"
	"os"

	"github.com/lugassawan/idxlens/internal/domain"
	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

// ExtractPresentation runs the PDF -> Layout -> KV extraction pipeline.
func ExtractPresentation(pdfPath string) ([]domain.KeyValuePair, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	reader := pdf.NewReader()

	if err := reader.Open(f); err != nil {
		return nil, fmt.Errorf("parse pdf: %w", err)
	}
	defer reader.Close()

	analyzer := layout.NewAnalyzer()
	pages := make([]layout.LayoutPage, 0, reader.PageCount())

	for i := 1; i <= reader.PageCount(); i++ {
		page, err := reader.Page(i)
		if err != nil {
			return nil, fmt.Errorf("read page %d: %w", i, err)
		}

		lp, err := analyzer.Analyze(page)
		if err != nil {
			return nil, fmt.Errorf("analyze page %d: %w", i, err)
		}

		pages = append(pages, lp)
	}

	extractor := domain.NewKVExtractor()
	pairs := extractor.Extract(pages)

	return pairs, nil
}
