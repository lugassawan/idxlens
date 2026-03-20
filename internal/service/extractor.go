package service

import (
	"errors"
	"fmt"

	"github.com/lugassawan/idxlens/internal/xbrl"
	"github.com/lugassawan/idxlens/internal/xlsx"
)

// extractor extracts financial data from a file and returns structured data.
type extractor interface {
	extract(path string, mode string) (any, error)
}

// metaSetter allows setting fallback metadata on extraction results.
type metaSetter interface {
	SetMeta(ticker string, year int, period string)
}

var extractorRegistry = map[string]extractor{}

// ExtractFile extracts financial data from the given file and applies
// fallback metadata when the filename doesn't encode ticker/year/period.
func ExtractFile(path, format, mode, ticker string, year int, period string) (any, error) {
	e, err := getExtractor(format)
	if err != nil {
		return nil, err
	}

	result, err := e.extract(path, mode)
	if err != nil {
		return nil, err
	}

	if ms, ok := result.(metaSetter); ok {
		ms.SetMeta(ticker, year, period)
	}

	return result, nil
}

func init() {
	extractorRegistry["xlsx"] = xlsxExtractor{}
	extractorRegistry["xbrl"] = xbrlExtractor{}
	extractorRegistry["pdf"] = pdfExtractor{}
}

func getExtractor(format string) (extractor, error) {
	e, ok := extractorRegistry[format]
	if !ok {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return e, nil
}

type xlsxExtractor struct{}

func (xlsxExtractor) extract(path, mode string) (any, error) {
	stmt, err := xlsx.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse xlsx: %w", err)
	}

	return stmt, nil
}

type xbrlExtractor struct{}

func (xbrlExtractor) extract(path, mode string) (any, error) {
	stmt, err := xbrl.ParseZip(path)
	if err != nil {
		return nil, fmt.Errorf("parse xbrl: %w", err)
	}

	return stmt, nil
}

type pdfExtractor struct{}

func (pdfExtractor) extract(path, mode string) (any, error) {
	if mode == "presentation" {
		pairs, err := ExtractPresentation(path)
		if err != nil {
			return nil, fmt.Errorf("extract presentation: %w", err)
		}

		return pairs, nil
	}

	return nil, errors.New("PDF financial extraction not supported in v2 (use XLSX or XBRL)")
}
