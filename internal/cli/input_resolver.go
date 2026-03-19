package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lugassawan/idxlens/internal/idx"
)

// InputFile describes a resolved input file for extraction.
type InputFile struct {
	Path   string
	Format string // "xlsx", "xbrl", "pdf"
	Ticker string
	Year   int
	Period string
}

// ResolveInputs resolves the given argument into a list of input files.
// If arg looks like a file path (contains '.' or '/'), it is resolved as a
// direct file. Otherwise it is treated as a ticker symbol and the data
// directory is scanned for matching files.
func ResolveInputs(arg string, year int, period string) ([]InputFile, error) {
	if isFilePath(arg) {
		return resolveFile(arg)
	}

	return resolveTicker(strings.ToUpper(arg), year, period)
}

func isFilePath(arg string) bool {
	return strings.Contains(arg, ".") || strings.Contains(arg, "/")
}

func resolveFile(path string) ([]InputFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("expected file, got directory: %s", path)
	}

	format := detectFormat(path)
	if format == "" {
		return nil, fmt.Errorf("unsupported file extension: %s", filepath.Ext(path))
	}

	return []InputFile{{Path: path, Format: format}}, nil
}

func resolveTicker(ticker string, year int, period string) ([]InputFile, error) {
	dataDir, err := idx.DataDir()
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}

	pattern := buildGlobPattern(dataDir, ticker, year, period)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	var files []InputFile

	for _, m := range matches {
		format := detectFormat(m)
		if format == "" {
			continue
		}

		files = append(files, InputFile{
			Path:   m,
			Format: format,
			Ticker: ticker,
			Year:   year,
			Period: period,
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found for ticker %s", ticker)
	}

	return files, nil
}

func buildGlobPattern(dataDir, ticker string, year int, period string) string {
	yearPart := "*"
	if year != 0 {
		yearPart = fmt.Sprintf("%d", year)
	}

	periodPart := "*"
	if period != "" {
		periodPart = period
	}

	return filepath.Join(dataDir, ticker, yearPart, periodPart, "*")
}

func detectFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xlsx":
		return "xlsx"
	case ".zip":
		return "xbrl"
	case ".pdf":
		return "pdf"
	default:
		return ""
	}
}
