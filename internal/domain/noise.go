package domain

import (
	"slices"
	"strings"
	"unicode"

	"github.com/lugassawan/idxlens/internal/table"
)

const (
	garbledSymbolThreshold = 0.30
	pageRefMaxValue        = 999
)

// garbledSymbols are characters commonly found in garbled PDF text caused by
// font encoding issues. Labels with a high ratio of these characters are
// noise rather than financial line items.
var garbledSymbols = [256]bool{
	'>': true, '<': true, '^': true, '~': true,
	'[': true, ']': true, '{': true, '}': true,
	'*': true, '|': true, '\\': true, '`': true,
	'@': true, '#': true, '$': true,
	'_': true, '&': true,
}

// nonFinancialKeywords are terms that appear in governance, compliance, and
// organizational tables in annual reports but not in financial statements.
var nonFinancialKeywords = []string{
	"anti-fraud", "anti fraud", "antifraud",
	"benturan kepentingan", "conflict of interest",
	"komite di bawah", "committee under",
	"good corporate governance", "gcg",
	"tata kelola",
	"rapat umum pemegang saham", "general meeting of shareholders",
	"dewan komisaris", "board of commissioners",
	"dewan direksi", "board of directors",
	"remunerasi", "remuneration",
	"whistle", "whistleblowing",
	"kode etik", "code of conduct",
	"corporate social responsibility", "csr",
	"manajemen risiko", "risk management",
	"sumber daya manusia", "human resources",
	"tanggung jawab sosial", "social responsibility",
}

// financialSectionHeaders are headings that identify financial statement
// sections within annual reports.
var financialSectionHeaders = []string{
	"ikhtisar keuangan", "financial highlights",
	"laporan posisi keuangan", "statement of financial position",
	"laporan laba rugi", "statement of profit or loss",
	"laporan arus kas", "statement of cash flows",
	"laporan perubahan ekuitas", "statement of changes in equity",
	"neraca", "balance sheet",
	"income statement", "profit and loss",
	"pendapatan dan beban", "revenue and expenses",
	"ringkasan keuangan", "financial summary",
}

// isGarbledText returns true if the label has a high ratio of symbol
// characters typically produced by PDF font encoding errors.
func isGarbledText(label string) bool {
	if len(label) == 0 {
		return false
	}

	symbolCount := 0
	nonSpaceCount := 0

	for _, r := range label {
		if unicode.IsSpace(r) {
			continue
		}

		nonSpaceCount++

		if r < 256 && garbledSymbols[r] {
			symbolCount++
		}
	}

	if nonSpaceCount == 0 {
		return false
	}

	return float64(symbolCount)/float64(nonSpaceCount) > garbledSymbolThreshold
}

// isPageRefValue returns true if all values in the item are small integers
// that likely represent page numbers rather than financial data. This
// catches table-of-contents entries that slip through as false positives.
func isPageRefValue(values map[string]float64, unit string) bool {
	if len(values) == 0 {
		return false
	}

	if unit != unitMillions && unit != unitBillions {
		return false
	}

	for _, v := range values {
		abs := v
		if abs < 0 {
			abs = -abs
		}

		if abs > pageRefMaxValue || abs == 0 {
			return false
		}

		if v != float64(int64(v)) {
			return false
		}
	}

	return true
}

// filterPageReferences removes items whose values all look like page numbers
// rather than financial data (small integers in a millions/billions context).
func filterPageReferences(items []LineItem, unit string) []LineItem {
	if unit == "" {
		return items
	}

	result := make([]LineItem, 0, len(items))

	for _, item := range items {
		if isPageRefValue(item.Values, unit) {
			continue
		}

		result = append(result, item)
	}

	return result
}

// isNonFinancialTable returns true if the table contains governance,
// compliance, or organizational content rather than financial data.
func isNonFinancialTable(tbl table.Table) bool {
	if slices.ContainsFunc(tbl.Headers, containsNonFinancialKeyword) {
		return true
	}

	scanRows := min(metadataScanRows, len(tbl.Rows))

	for i := range scanRows {
		for _, cell := range tbl.Rows[i].Cells {
			if containsNonFinancialKeyword(cell.Text) {
				return true
			}
		}
	}

	return false
}

// hasFinancialSectionHeader returns true if the table's page text or headers
// contain a recognized financial statement section heading.
func hasFinancialSectionHeader(tbl table.Table) bool {
	return slices.ContainsFunc(tbl.PageText, containsFinancialHeader) ||
		slices.ContainsFunc(tbl.Headers, containsFinancialHeader)
}

// filterByFinancialContent applies stricter filtering for annual reports.
// Tables must either have a financial section header on their page or appear
// in a cluster of tables with financial headers. Tables from non-financial
// sections (GCG, governance, CSR) are excluded.
func filterByFinancialContent(tables []table.Table) []table.Table {
	financialPages := detectFinancialPages(tables)

	var result []table.Table

	for _, tbl := range tables {
		if tbl.PageNum == 1 {
			continue
		}

		if isSubsidiaryTable(tbl) {
			continue
		}

		if isNonFinancialTable(tbl) {
			continue
		}

		if !hasNumericColumns(tbl) {
			continue
		}

		if len(financialPages) > 0 && !financialPages[tbl.PageNum] {
			continue
		}

		result = append(result, tbl)
	}

	if len(result) == 0 {
		return filterByNumericContent(tables)
	}

	return result
}

// detectFinancialPages scans all tables and returns a map of page numbers
// that belong to financial sections. If a table has a financial header, its
// page and nearby pages (within a reasonable range) are included.
func detectFinancialPages(tables []table.Table) map[int]bool {
	pages := make(map[int]bool)

	var anchorPages []int

	for _, tbl := range tables {
		if hasFinancialSectionHeader(tbl) {
			anchorPages = append(anchorPages, tbl.PageNum)
		}
	}

	if len(anchorPages) == 0 {
		return pages
	}

	for _, anchor := range anchorPages {
		end := anchor + financialPageSpan(tables, anchor)

		for p := anchor; p <= end; p++ {
			pages[p] = true
		}
	}

	return pages
}

// financialPageSpan returns how many pages after an anchor page should be
// considered part of the same financial section. It looks for the next
// anchor page or non-financial section break.
func financialPageSpan(tables []table.Table, anchor int) int {
	maxSpan := 15
	span := maxSpan

	for _, tbl := range tables {
		if tbl.PageNum <= anchor {
			continue
		}

		if isNonFinancialTable(tbl) && tbl.PageNum-anchor < span {
			span = tbl.PageNum - anchor - 1

			break
		}

		if hasFinancialSectionHeader(tbl) && tbl.PageNum > anchor {
			if tbl.PageNum-anchor < span {
				span = tbl.PageNum - anchor + maxSpan
			}

			break
		}
	}

	if span < 0 {
		span = 0
	}

	return span
}

func containsNonFinancialKeyword(text string) bool {
	lower := strings.ToLower(text)

	for _, kw := range nonFinancialKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

func containsFinancialHeader(text string) bool {
	lower := strings.ToLower(text)
	normalized := collapseSpaces(lower)

	for _, header := range financialSectionHeaders {
		if strings.Contains(normalized, header) {
			return true
		}
	}

	return false
}

// collapseSpaces reduces runs of whitespace to a single space.
func collapseSpaces(text string) string {
	var b strings.Builder

	b.Grow(len(text))
	prevSpace := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}

			continue
		}

		b.WriteRune(r)
		prevSpace = false
	}

	return b.String()
}
