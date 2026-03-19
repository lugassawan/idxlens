package xlsx

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Statement represents the extracted data from an XLSX financial report.
type Statement struct {
	Ticker string  `json:"ticker"`
	Year   int     `json:"year"`
	Period string  `json:"period"`
	Sheets []Sheet `json:"sheets"`
}

// Sheet represents a single worksheet within the report.
type Sheet struct {
	Name  string     `json:"name"`
	Items []LineItem `json:"items"`
}

// LineItem represents a single row of financial data.
type LineItem struct {
	Label  string             `json:"label"`
	Values map[string]float64 `json:"values"`
}

// filenamePattern matches IDX financial statement filenames:
// FinancialStatement-{year}-{period}-{ticker}.xlsx
var filenamePattern = regexp.MustCompile(
	`(?i)FinancialStatement-(\d{4})-([^-]+)-([A-Z]{4})\.xlsx$`,
)

// Parse reads an XLSX file and extracts structured financial data.
func Parse(path string) (*Statement, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	stmt := &Statement{}
	parseMeta(stmt, filepath.Base(path))

	for _, name := range f.GetSheetList() {
		sheet, err := parseSheet(f, name)
		if err != nil {
			return nil, fmt.Errorf("parse sheet %q: %w", name, err)
		}

		if len(sheet.Items) > 0 {
			stmt.Sheets = append(stmt.Sheets, *sheet)
		}
	}

	return stmt, nil
}

func parseMeta(stmt *Statement, filename string) {
	m := filenamePattern.FindStringSubmatch(filename)
	if len(m) != 4 {
		return
	}

	stmt.Year, _ = strconv.Atoi(m[1])
	stmt.Period = m[2]
	stmt.Ticker = m[3]
}

func parseSheet(f *excelize.File, name string) (*Sheet, error) {
	rows, err := f.GetRows(name)
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	if len(rows) < 2 {
		return &Sheet{Name: name}, nil
	}

	headers := rows[0]
	sheet := &Sheet{Name: name}

	for _, row := range rows[1:] {
		item := parseRow(row, headers)
		if item != nil {
			sheet.Items = append(sheet.Items, *item)
		}
	}

	return sheet, nil
}

func parseRow(row, headers []string) *LineItem {
	if len(row) == 0 {
		return nil
	}

	label := strings.TrimSpace(row[0])
	if label == "" {
		return nil
	}

	item := &LineItem{
		Label:  label,
		Values: make(map[string]float64),
	}

	for i := 1; i < len(row) && i < len(headers); i++ {
		cell := strings.TrimSpace(row[i])
		if cell == "" {
			continue
		}

		v, err := strconv.ParseFloat(cell, 64)
		if err != nil {
			continue
		}

		key := strings.TrimSpace(headers[i])
		if key != "" {
			item.Values[key] = v
		}
	}

	return item
}
