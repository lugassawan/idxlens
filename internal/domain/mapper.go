package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/lugassawan/idxlens/internal/table"
)

const (
	currencyIDR = "IDR"
	currencyUSD = "USD"
)

// Mapper maps raw table data into a standardized FinancialStatement.
type Mapper interface {
	Map(docType DocType, tables []table.Table) (*FinancialStatement, error)
}

var (
	periodPatternID = regexp.MustCompile(
		`(\d{1,2})\s+` +
			`(Januari|Februari|Maret|April|Mei|Juni|Juli|` +
			`Agustus|September|Oktober|November|Desember)` +
			`\s+(\d{4})`,
	)
	periodPatternEN = regexp.MustCompile(
		`(January|February|March|April|May|June|July|` +
			`August|September|October|November|December)` +
			`\s+(\d{1,2}),?\s+(\d{4})`,
	)
	companyPattern = regexp.MustCompile(`PT\s+.+\s+Tbk`)
	currencyUnitID = regexp.MustCompile(
		`(?i)dalam\s+(jutaan|miliar(?:an)?|ribuan)\s+(rupiah|dolar)`,
	)
	currencyUnitEN = regexp.MustCompile(
		`(?i)(?:expressed\s+in\s+|in\s+)(millions?|billions?|thousands?)\s+(?:of\s+)?(rupiah|dollars?)`,
	)
	currencyUnitSlash = regexp.MustCompile(
		`(?i)(jutaan|miliar(?:an)?|ribuan|millions?|billions?|thousands?)` +
			`\s*/\s*` +
			`(?:in\s+)?(millions?|billions?|thousands?)`,
	)
	monthsID = map[string]int{
		"januari": 1, "februari": 2, "maret": 3, "april": 4,
		"mei": 5, "juni": 6, "juli": 7, "agustus": 8,
		"september": 9, "oktober": 10, "november": 11, "desember": 12,
	}
	monthsEN = map[string]int{
		"january": 1, "february": 2, "march": 3, "april": 4,
		"may": 5, "june": 6, "july": 7, "august": 8,
		"september": 9, "october": 10, "november": 11, "december": 12,
	}
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

	financialTables := filterFinancialTables(tables)

	for _, tbl := range financialTables {
		m.extractMetadata(tbl, stmt)
	}

	if stmt.Language == "" {
		stmt.Language = "id"
	}

	for _, tbl := range financialTables {
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
	for _, groups := range periodPatternID.FindAllStringSubmatch(text, -1) {
		iso := formatDateISO(groups[1], groups[2], groups[3], monthsID)
		if iso != "" && !slices.Contains(stmt.Periods, iso) {
			stmt.Periods = append(stmt.Periods, iso)

			if stmt.Language == "" {
				stmt.Language = "id"
			}
		}
	}

	for _, groups := range periodPatternEN.FindAllStringSubmatch(text, -1) {
		iso := formatDateISO(groups[2], groups[1], groups[3], monthsEN)
		if iso != "" && !slices.Contains(stmt.Periods, iso) {
			stmt.Periods = append(stmt.Periods, iso)

			if stmt.Language == "" {
				stmt.Language = "en"
			}
		}
	}
}

func formatDateISO(dayStr, monthStr, yearStr string, months map[string]int) string {
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		return ""
	}

	month, ok := months[strings.ToLower(monthStr)]
	if !ok {
		return ""
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
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

		return
	}

	if matches := currencyUnitSlash.FindStringSubmatch(text); len(matches) == 3 {
		stmt.Unit = normalizeUnit(matches[1])
		stmt.Currency = inferCurrencyFromContext(text)
	}
}

func inferCurrencyFromContext(text string) string {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "rupiah"):
		return currencyIDR
	case strings.Contains(lower, "dolar"), strings.Contains(lower, "dollar"):
		return currencyUSD
	default:
		return currencyIDR
	}
}

func filterFinancialTables(tables []table.Table) []table.Table {
	if len(tables) <= 1 {
		return tables
	}

	var result []table.Table

	for _, tbl := range tables {
		if tbl.PageNum == 1 {
			continue
		}

		if isSubsidiaryTable(tbl) {
			continue
		}

		result = append(result, tbl)
	}

	if len(result) == 0 {
		return tables
	}

	return result
}

func isSubsidiaryTable(tbl table.Table) bool {
	keywords := []string{
		"anak perusahaan", "entitas anak", "subsidiary",
		"subsidiaries", "daftar perusahaan",
	}

	for _, header := range tbl.Headers {
		lower := strings.ToLower(header)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}

	return false
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
		return currencyIDR
	case strings.HasPrefix(lower, "dolar"),
		strings.HasPrefix(lower, "dollar"):
		return currencyUSD
	default:
		return strings.ToUpper(raw)
	}
}
