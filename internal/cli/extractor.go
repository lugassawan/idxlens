package cli

import (
	"errors"
	"fmt"

	"github.com/lugassawan/idxlens/internal/service"
	"github.com/lugassawan/idxlens/internal/xbrl"
	"github.com/lugassawan/idxlens/internal/xlsx"
)

// ExtractResult wraps extractor output for uniform handling.
type ExtractResult struct {
	value any
}

// Extractor extracts financial data from a file and returns structured data.
type Extractor interface {
	Extract(path string, mode string) (ExtractResult, error)
}

// metaSetter allows setting fallback metadata on extraction results.
type metaSetter interface {
	SetMeta(ticker string, year int, period string)
}

// extractorRegistry maps file formats to their extractors.
var extractorRegistry = map[string]Extractor{}

// NewExtractResult creates a new ExtractResult.
func NewExtractResult(v any) ExtractResult { return ExtractResult{value: v} }

// Value returns the underlying extraction result.
func (r ExtractResult) Value() any { return r.value }

func init() {
	registerExtractor(formatXLSX, xlsxExtractor{})
	registerExtractor(formatXBRL, xbrlExtractor{})
	registerExtractor(formatPDF, pdfExtractor{})
}

// registerExtractor registers an extractor for the given format.
func registerExtractor(format string, e Extractor) {
	extractorRegistry[format] = e
}

// getExtractor returns the extractor for the given format, or an error.
func getExtractor(format string) (Extractor, error) {
	e, ok := extractorRegistry[format]
	if !ok {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return e, nil
}

// xlsxExtractor extracts financial data from XLSX files.
type xlsxExtractor struct{}

func (xlsxExtractor) Extract(path, mode string) (ExtractResult, error) {
	stmt, err := xlsx.Parse(path)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("parse xlsx: %w", err)
	}

	return NewExtractResult(stmt), nil
}

// xbrlExtractor extracts financial data from XBRL (ZIP) files.
type xbrlExtractor struct{}

func (xbrlExtractor) Extract(path, mode string) (ExtractResult, error) {
	stmt, err := xbrl.ParseZip(path)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("parse xbrl: %w", err)
	}

	return NewExtractResult(stmt), nil
}

// pdfExtractor extracts data from PDF files.
type pdfExtractor struct{}

func (pdfExtractor) Extract(path, mode string) (ExtractResult, error) {
	if mode == modePresentation {
		pairs, err := service.ExtractPresentation(path)
		if err != nil {
			return ExtractResult{}, fmt.Errorf("extract presentation: %w", err)
		}

		return NewExtractResult(pairs), nil
	}

	return ExtractResult{}, errors.New("PDF financial extraction not supported in v2 (use XLSX or XBRL)")
}
