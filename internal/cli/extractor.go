package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/lugassawan/idxlens/internal/service"
	"github.com/lugassawan/idxlens/internal/xbrl"
	"github.com/lugassawan/idxlens/internal/xlsx"
)

// Extractor extracts financial data from a file and writes it to w.
type Extractor interface {
	Extract(w io.Writer, path string, mode string, pretty bool) error
}

// extractorRegistry maps file formats to their extractors.
var extractorRegistry = map[string]Extractor{}

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

func (xlsxExtractor) Extract(w io.Writer, path, mode string, pretty bool) error {
	stmt, err := xlsx.Parse(path)
	if err != nil {
		return fmt.Errorf("parse xlsx: %w", err)
	}

	return writeJSON(w, stmt, pretty)
}

// xbrlExtractor extracts financial data from XBRL (ZIP) files.
type xbrlExtractor struct{}

func (xbrlExtractor) Extract(w io.Writer, path, mode string, pretty bool) error {
	stmt, err := xbrl.ParseZip(path)
	if err != nil {
		return fmt.Errorf("parse xbrl: %w", err)
	}

	return writeJSON(w, stmt, pretty)
}

// pdfExtractor extracts data from PDF files.
type pdfExtractor struct{}

func (pdfExtractor) Extract(w io.Writer, path, mode string, pretty bool) error {
	if mode == modePresentation {
		pairs, err := service.ExtractPresentation(path)
		if err != nil {
			return fmt.Errorf("extract presentation: %w", err)
		}

		return writeJSON(w, pairs, pretty)
	}

	return errors.New("PDF financial extraction not supported in v2 (use XLSX or XBRL)")
}
