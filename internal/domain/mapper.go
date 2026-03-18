package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/lugassawan/idxlens/internal/table"
)

// Mapper maps raw table data into a standardized FinancialStatement.
type Mapper interface {
	Map(docType DocType, tables []table.Table) (*FinancialStatement, error)
}

var (
	periodPatternID = regexp.MustCompile(
		`\d{1,2}\s+` +
			`(?:Januari|Februari|Maret|April|Mei|Juni|Juli|` +
			`Agustus|September|Oktober|November|Desember)` +
			`\s+\d{4}`,
	)
	periodPatternEN = regexp.MustCompile(
		`(?:January|February|March|April|May|June|July|` +
			`August|September|October|November|December)` +
			`\s+\d{1,2},?\s+\d{4}`,
	)
	companyPattern = regexp.MustCompile(`PT\s+.+\s+Tbk`)
	currencyUnitID = regexp.MustCompile(
		`(?i)dalam\s+(jutaan|miliar|ribuan)\s+(rupiah|dolar)`,
	)
	currencyUnitEN = regexp.MustCompile(
		`(?i)in\s+(millions?|billions?|thousands?)\s+of\s+(rupiah|dollars?)`,
	)
)

// NewMapper creates a new Mapper that uses embedded dictionaries for label
// matching and Indonesian number format parsing.
func NewMapper() Mapper {
	return &mapper{}
}

type mapper struct{}

func (m *mapper) Map(docType DocType, tables []table.Table) (*FinancialStatement, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("map %s: no tables provided", docType)
	}

	dict, err := LoadDictionary(docType)
	if err != nil {
		return nil, fmt.Errorf("map %s: %w", docType, err)
	}

	stmt := &FinancialStatement{
		Type: docType,
	}

	for _, tbl := range tables {
		m.extractMetadata(tbl, stmt)
	}

	if stmt.Language == "" {
		stmt.Language = "id"
	}

	for _, tbl := range tables {
		items := m.mapTableRows(tbl, dict, stmt)
		stmt.Items = append(stmt.Items, items...)
	}

	return stmt, nil
}

func (m *mapper) extractMetadata(tbl table.Table, stmt *FinancialStatement) {
	for _, header := range tbl.Headers {
		m.detectPeriod(header, stmt)
		m.detectCurrencyUnit(header, stmt)
		m.detectCompany(header, stmt)
	}

	if len(tbl.Rows) > 0 {
		for _, cell := range tbl.Rows[0].Cells {
			m.detectPeriod(cell.Text, stmt)
			m.detectCurrencyUnit(cell.Text, stmt)
			m.detectCompany(cell.Text, stmt)
		}
	}
}

func (m *mapper) mapTableRows(
	tbl table.Table, dict *Dictionary, stmt *FinancialStatement,
) []LineItem {
	var items []LineItem

	startRow := headerRowOffset(tbl)

	for i := startRow; i < len(tbl.Rows); i++ {
		row := tbl.Rows[i]
		if len(row.Cells) == 0 {
			continue
		}

		label := strings.TrimSpace(row.Cells[0].Text)
		if label == "" {
			continue
		}

		item := m.mapRow(row, label, dict, stmt)
		items = append(items, item)
	}

	return items
}

func (m *mapper) mapRow(
	row table.Row, label string, dict *Dictionary, stmt *FinancialStatement,
) LineItem {
	item := LineItem{
		Label:      label,
		Values:     make(map[string]float64),
		IsSubtotal: isSubtotal(label),
	}

	matched, confidence := dict.MatchLabel(label, stmt.Language)
	if matched != nil {
		item.Key = matched.Key
		item.Section = matched.Section
		item.Level = matched.Level
		item.Confidence = confidence
	} else {
		item.Key = ""
		item.Level = detectIndentLevel(row.Cells[0])
	}

	m.parseRowValues(row, stmt.Periods, item.Values)

	return item
}

func (m *mapper) parseRowValues(
	row table.Row, periods []string, values map[string]float64,
) {
	for i := 1; i < len(row.Cells); i++ {
		text := strings.TrimSpace(row.Cells[i].Text)
		if text == "" {
			continue
		}

		val, err := ParseNumber(text)
		if err != nil {
			continue
		}

		periodIdx := i - 1
		if periodIdx < len(periods) {
			values[periods[periodIdx]] = val
		} else {
			values[fmt.Sprintf("col_%d", i)] = val
		}
	}
}

func (m *mapper) detectPeriod(text string, stmt *FinancialStatement) {
	for _, match := range periodPatternID.FindAllString(text, -1) {
		if !slices.Contains(stmt.Periods, match) {
			stmt.Periods = append(stmt.Periods, match)

			if stmt.Language == "" {
				stmt.Language = "id"
			}
		}
	}

	for _, match := range periodPatternEN.FindAllString(text, -1) {
		if !slices.Contains(stmt.Periods, match) {
			stmt.Periods = append(stmt.Periods, match)

			if stmt.Language == "" {
				stmt.Language = "en"
			}
		}
	}
}

func (m *mapper) detectCurrencyUnit(text string, stmt *FinancialStatement) {
	if stmt.Currency != "" && stmt.Unit != "" {
		return
	}

	if matches := currencyUnitID.FindStringSubmatch(text); len(matches) == 3 {
		stmt.Unit = normalizeUnit(matches[1])
		stmt.Currency = normalizeCurrency(matches[2])

		return
	}

	if matches := currencyUnitEN.FindStringSubmatch(text); len(matches) == 3 {
		stmt.Unit = normalizeUnit(matches[1])
		stmt.Currency = normalizeCurrency(matches[2])
	}
}

func (m *mapper) detectCompany(text string, stmt *FinancialStatement) {
	if stmt.Company != "" {
		return
	}

	if match := companyPattern.FindString(text); match != "" {
		stmt.Company = match
	}
}

func headerRowOffset(tbl table.Table) int {
	if len(tbl.Headers) > 0 {
		return 0
	}

	if len(tbl.Rows) > 1 {
		firstRow := tbl.Rows[0]
		hasNumeric := false

		for i := 1; i < len(firstRow.Cells); i++ {
			text := strings.TrimSpace(firstRow.Cells[i].Text)
			if _, err := ParseNumber(text); err == nil {
				hasNumeric = true

				break
			}
		}

		if !hasNumeric {
			return 1
		}
	}

	return 0
}

func isSubtotal(label string) bool {
	lower := strings.ToLower(label)

	return strings.HasPrefix(lower, "jumlah") ||
		strings.HasPrefix(lower, "total") ||
		strings.HasPrefix(lower, "subjumlah") ||
		strings.HasPrefix(lower, "sub-total")
}

func detectIndentLevel(cell table.Cell) int {
	text := cell.Text
	trimmed := strings.TrimLeft(text, " \t")
	spaces := len(text) - len(trimmed)

	switch {
	case spaces >= 8:
		return 3
	case spaces >= 4:
		return 2
	case spaces >= 2:
		return 1
	default:
		return 0
	}
}

func normalizeUnit(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "juta"),
		strings.HasPrefix(lower, "million"):
		return "millions"
	case strings.HasPrefix(lower, "miliar"),
		strings.HasPrefix(lower, "billion"):
		return "billions"
	case strings.HasPrefix(lower, "ribu"),
		strings.HasPrefix(lower, "thousand"):
		return "thousands"
	default:
		return lower
	}
}

func normalizeCurrency(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "rupiah"):
		return "IDR"
	case strings.HasPrefix(lower, "dolar"),
		strings.HasPrefix(lower, "dollar"):
		return "USD"
	default:
		return strings.ToUpper(raw)
	}
}
