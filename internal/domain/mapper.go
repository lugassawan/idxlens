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
	currencyIDR      = "IDR"
	currencyUSD      = "USD"
	maxPeriods       = 3
	metadataScanRows = 5
	unitMillions     = "millions"
	unitBillions     = "billions"
	unitTrillions    = "trillions"
	unitThousands    = "thousands"
	fmtAnnualEnd     = "%04d-12-31"
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
		`(?i)(January|February|March|April|May|June|July|` +
			`August|September|October|November|December)` +
			`\s+(\d{1,2}),?\s+(\d{4})`,
	)
	periodPatternENDayFirst = regexp.MustCompile(
		`(?i)(\d{1,2})\s+` +
			`(January|February|March|April|May|June|July|` +
			`August|September|October|November|December)` +
			`\s+(\d{4})`,
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
	kernedDigitPattern = regexp.MustCompile(`\d(?:\s+\d)+`)
	noiseYearPattern   = regexp.MustCompile(`^\d{4}$`)
	periodWithAndEN    = regexp.MustCompile(
		`(?i)(\d{1,2})\s+` +
			`(January|February|March|April|May|June|July|` +
			`August|September|October|November|December)` +
			`\s+(\d{4})\s+(?:AND|DAN)\s+(\d{4})`,
	)
	xbrlCodePattern    = regexp.MustCompile(`\[(\d{7})\]`)
	leadingNumberRegex = regexp.MustCompile(
		`^(\(?\s*-?\s*[\d][,.\d]*\s*\)?)`,
	)
	// currencyUnitShort matches compact presentation-style formats like
	// "Rp tn", "(Rp bn)", "Rp miliar", "USD mn", "Rp triliun".
	currencyUnitShort = regexp.MustCompile(
		`(?i)\(?\s*(Rp|USD|IDR)\s+` +
			`(tn|triliun|trillions?|bn|miliar(?:an)?|billions?|` +
			`mn|juta(?:an)?|millions?|ribu(?:an)?|thousands?)\s*\)?`,
	)
	// periodAbbrev matches abbreviated period headers like "Dec-25",
	// "Sep-24", "FY-25", "3Q-25", "4Q24".
	periodAbbrev = regexp.MustCompile(
		`(?i)\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|FY|[1-4]Q)-?(\d{2})\b`,
	)
	// growthRateHeader matches column headers that represent growth rates
	// rather than financial amounts: "YoY", "QoQ", "%", "Growth".
	growthRateHeader = regexp.MustCompile(
		`(?i)^(YoY|QoQ|[Gg]rowth|%|vs)$`,
	)
	monthAbbrevEN = map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4,
		"may": 5, "jun": 6, "jul": 7, "aug": 8,
		"sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}
)

// xbrlPageRange represents a contiguous page range for one XBRL section.
type xbrlPageRange struct {
	start int
	end   int
}

// pageMarker associates a page number with a detected document type from an
// XBRL taxonomy code.
type pageMarker struct {
	page    int
	docType DocType
}

// bestEntry tracks the item index with the highest absolute value total for
// deduplication.
type bestEntry struct {
	index    int
	absTotal float64
}

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

	dict, err := LoadAllDictionaries()
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

	financialTables := filterFinancialTables(tables, docType)

	for _, tbl := range financialTables {
		items := m.mapTableRows(tbl, dict, stmt)
		stmt.Items = append(stmt.Items, items...)
	}

	stmt.Items = deduplicateItems(stmt.Items)
	stmt.Items = filterPageReferences(stmt.Items, stmt.Unit)
	filterColKeys(stmt)

	return stmt, nil
}

// isCompositeDocType returns true for document types that contain multiple
// financial statement sections (e.g., audited reports and annual reports).
func isCompositeDocType(docType DocType) bool {
	return docType == DocTypeAuditorReport || docType == DocTypeAnnualReport
}

// deduplicateItems removes duplicate keyed items, keeping the one with the
// largest total absolute value. Unkeyed items with all-zero values are
// dropped as noise.
func deduplicateItems(items []LineItem) []LineItem {
	best := bestEntryByKey(items)
	result := make([]LineItem, 0, len(items))
	kept := make(map[string]bool, len(best))

	for _, item := range items {
		if item.Key == "" {
			if absValueTotal(item.Values) > 0 {
				result = append(result, item)
			}

			continue
		}

		if kept[item.Key] {
			continue
		}

		kept[item.Key] = true
		result = append(result, items[best[item.Key].index])
	}

	return result
}

// filterColKeys removes col_N keys from item values when period labels
// have been detected. These keys are placeholders for columns that
// could not be mapped to a known period and become noise once real
// periods are available.
func filterColKeys(stmt *FinancialStatement) {
	if len(stmt.Periods) == 0 {
		return
	}

	for i := range stmt.Items {
		for key := range stmt.Items[i].Values {
			if strings.HasPrefix(key, "col_") {
				delete(stmt.Items[i].Values, key)
			}
		}
	}
}

func bestEntryByKey(items []LineItem) map[string]bestEntry {
	best := make(map[string]bestEntry)

	for i, item := range items {
		if item.Key == "" {
			continue
		}

		total := absValueTotal(item.Values)

		prev, exists := best[item.Key]
		if !exists || total > prev.absTotal {
			best[item.Key] = bestEntry{index: i, absTotal: total}
		}
	}

	return best
}

func absValueTotal(values map[string]float64) float64 {
	var total float64

	for _, v := range values {
		if v < 0 {
			total -= v
		} else {
			total += v
		}
	}

	return total
}

func (m *mapper) extractMetadata(tbl table.Table, stmt *FinancialStatement) {
	for _, header := range tbl.Headers {
		m.detectPeriod(header, stmt)
		m.detectCurrencyUnit(header, stmt)
		m.detectCompany(header, stmt)
	}

	scanRows := min(metadataScanRows, len(tbl.Rows))

	for i := range scanRows {
		for _, cell := range tbl.Rows[i].Cells {
			m.detectPeriod(cell.Text, stmt)
			m.detectCurrencyUnit(cell.Text, stmt)
			m.detectCompany(cell.Text, stmt)
		}
	}

	for _, text := range tbl.PageText {
		m.detectPeriod(text, stmt)
		m.detectCurrencyUnit(text, stmt)
		m.detectCompany(text, stmt)
	}
}

func (m *mapper) mapTableRows(
	tbl table.Table, dict *Dictionary, stmt *FinancialStatement,
) []LineItem {
	var items []LineItem

	skipCols := growthRateColumns(tbl.Headers)
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

		if isMetadataRow(row) {
			continue
		}

		if isNoiseLabel(label) {
			continue
		}

		if isGarbledText(label) || isSpacedOutText(label) {
			continue
		}

		item := m.mapRow(row, label, dict, stmt, skipCols)
		items = append(items, item)
	}

	return items
}

func (m *mapper) mapRow(
	row table.Row, label string, dict *Dictionary,
	stmt *FinancialStatement, skipCols map[int]bool,
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

	m.parseRowValues(row, stmt.Periods, item.Values, skipCols)

	return item
}

func (m *mapper) parseRowValues(
	row table.Row, periods []string, values map[string]float64,
	skipCols map[int]bool,
) {
	for i := 1; i < len(row.Cells); i++ {
		if skipCols[i] {
			continue
		}

		text := strings.TrimSpace(row.Cells[i].Text)
		if text == "" || isPercentageValue(text) {
			continue
		}

		val, err := ParseNumber(text)
		if err != nil {
			val, err = extractLeadingNumber(text)
			if err != nil {
				continue
			}
		}

		periodIdx := i - 1
		if periodIdx < len(periods) {
			values[periods[periodIdx]] = val
		} else {
			values[fmt.Sprintf("col_%d", i)] = val
		}
	}
}

func extractLeadingNumber(text string) (float64, error) {
	match := leadingNumberRegex.FindString(text)
	if match == "" {
		return 0, ErrNotANumber
	}

	match = strings.TrimSpace(match)
	if match == text {
		return 0, ErrNotANumber
	}

	return ParseNumber(match)
}

func (m *mapper) detectPeriod(text string, stmt *FinancialStatement) {
	m.detectPeriodFromText(text, stmt)

	// Retry with kern-safe normalization if no periods found yet.
	normalized := collapseKernedDigits(text)
	if normalized != text {
		m.detectPeriodFromText(normalized, stmt)
	}
}

func (m *mapper) detectPeriodFromText(text string, stmt *FinancialStatement) {
	// Indonesian: "31 Desember 2025" (DD Month YYYY)
	for _, groups := range periodPatternID.FindAllStringSubmatch(text, -1) {
		addPeriod(stmt, groups[1], groups[2], groups[3], monthsID, "id")
	}

	// English month-first: "December 31, 2025"
	for _, groups := range periodPatternEN.FindAllStringSubmatch(text, -1) {
		addPeriod(stmt, groups[2], groups[1], groups[3], monthsEN, "en")
	}

	// English day-first: "31 December 2025"
	for _, groups := range periodPatternENDayFirst.FindAllStringSubmatch(text, -1) {
		addPeriod(stmt, groups[1], groups[2], groups[3], monthsEN, "en")
	}

	// "31 December 2025 AND 2024" — implied same day/month for second year.
	for _, groups := range periodWithAndEN.FindAllStringSubmatch(text, -1) {
		addPeriod(stmt, groups[1], groups[2], groups[3], monthsEN, "en")
		addPeriod(stmt, groups[1], groups[2], groups[4], monthsEN, "en")
	}

	// Abbreviated: "Dec-25", "Sep-24", "FY-25", "3Q-25"
	for _, groups := range periodAbbrev.FindAllStringSubmatch(text, -1) {
		addAbbrevPeriod(stmt, groups[1], groups[2])
	}
}

// collapseKernedDigits removes spaces within digit sequences caused by PDF
// kerning. For example, "202 5" becomes "2025" and "31 DECEMBER  202 5"
// becomes "31 DECEMBER 2025".
func collapseKernedDigits(text string) string {
	return kernedDigitPattern.ReplaceAllStringFunc(text, func(match string) string {
		return strings.ReplaceAll(match, " ", "")
	})
}

func addPeriod(stmt *FinancialStatement, day, month, year string, months map[string]int, lang string) {
	if len(stmt.Periods) >= maxPeriods {
		return
	}

	iso := formatDateISO(day, month, year, months)
	if iso == "" || slices.Contains(stmt.Periods, iso) {
		return
	}

	dayInt, _ := strconv.Atoi(day)
	monthInt := months[strings.ToLower(month)]

	if !isFiscalPeriodEnd(dayInt, monthInt) {
		return
	}

	stmt.Periods = append(stmt.Periods, iso)

	if stmt.Language == "" {
		stmt.Language = lang
	}
}

// addAbbrevPeriod converts abbreviated period labels like "Dec-25" or
// "FY-25" into ISO dates and adds them to the statement.
func addAbbrevPeriod(stmt *FinancialStatement, label, yearShort string) {
	if len(stmt.Periods) >= maxPeriods {
		return
	}

	year, err := strconv.Atoi(yearShort)
	if err != nil {
		return
	}

	year += 2000

	iso := abbrevToISO(label, year)
	if iso == "" || slices.Contains(stmt.Periods, iso) {
		return
	}

	stmt.Periods = append(stmt.Periods, iso)

	if stmt.Language == "" {
		stmt.Language = "en"
	}
}

func abbrevToISO(label string, year int) string {
	lower := strings.ToLower(label)

	// Fiscal year: "FY" -> December 31
	if lower == "fy" {
		return fmt.Sprintf(fmtAnnualEnd, year)
	}

	// Quarterly: "1Q" -> Mar 31, "2Q" -> Jun 30, "3Q" -> Sep 30, "4Q" -> Dec 31
	if len(lower) == 2 && lower[1] == 'q' {
		return quarterEndISO(lower[0]-'0', year)
	}

	// Month abbreviation: "Dec" -> last day of month
	if month, ok := monthAbbrevEN[lower]; ok {
		return fiscalPeriodISO(month, year)
	}

	return ""
}

func quarterEndISO(quarter byte, year int) string {
	switch quarter {
	case 1:
		return fmt.Sprintf("%04d-03-31", year)
	case 2:
		return fmt.Sprintf("%04d-06-30", year)
	case 3:
		return fmt.Sprintf("%04d-09-30", year)
	case 4:
		return fmt.Sprintf(fmtAnnualEnd, year)
	default:
		return ""
	}
}

func fiscalPeriodISO(month, year int) string {
	switch month {
	case 3:
		return fmt.Sprintf("%04d-03-31", year)
	case 6:
		return fmt.Sprintf("%04d-06-30", year)
	case 9:
		return fmt.Sprintf("%04d-09-30", year)
	case 12:
		return fmt.Sprintf(fmtAnnualEnd, year)
	default:
		// Non-quarter-end months: use last day of month.
		lastDay := lastDayOfMonth(month, year)
		return fmt.Sprintf("%04d-%02d-%02d", year, month, lastDay)
	}
}

func lastDayOfMonth(month, year int) int {
	switch month {
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	default:
		return 31
	}
}

// isFiscalPeriodEnd returns true if the date is a fiscal quarter-end or
// year-end: Dec 31, Mar 31, Jun 30, or Sep 30.
func isFiscalPeriodEnd(day, month int) bool {
	switch {
	case month == 12 && day == 31: // Annual / Q4
		return true
	case month == 3 && day == 31: // Q1
		return true
	case month == 6 && day == 30: // Q2
		return true
	case month == 9 && day == 30: // Q3
		return true
	default:
		return false
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

		return
	}

	if matches := currencyUnitShort.FindStringSubmatch(text); len(matches) == 3 {
		stmt.Currency = normalizeShortCurrency(matches[1])
		stmt.Unit = normalizeUnit(matches[2])
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

func filterFinancialTables(tables []table.Table, docType DocType) []table.Table {
	if len(tables) <= 1 {
		return tables
	}

	if docType == DocTypeAnnualReport {
		return filterAnnualReportTables(tables)
	}

	if isCompositeDocType(docType) {
		return filterCompositeFinancialTables(tables)
	}

	pageRange := detectXBRLPageRange(tables, docType)
	if pageRange != nil {
		return filterByPageRange(tables, pageRange)
	}

	return filterByHeuristic(tables)
}

// filterAnnualReportTables filters tables specifically for annual reports.
// It first tries XBRL markers, then falls back to financial section header
// detection which is stricter than generic numeric content filtering.
func filterAnnualReportTables(tables []table.Table) []table.Table {
	markers := scanXBRLMarkers(tables)
	if len(markers) > 0 {
		return filterByXBRLMarkedPages(tables, markers)
	}

	return filterByFinancialContent(tables)
}

// filterCompositeFinancialTables filters tables for composite document types
// (audited reports, annual reports). It collects all pages marked by any
// XBRL financial marker and includes them. If no XBRL markers are found,
// it falls back to filtering tables that contain numeric data columns.
func filterCompositeFinancialTables(tables []table.Table) []table.Table {
	markers := scanXBRLMarkers(tables)
	if len(markers) > 0 {
		return filterByXBRLMarkedPages(tables, markers)
	}

	return filterByNumericContent(tables)
}

// filterByXBRLMarkedPages keeps tables whose page numbers fall within any
// XBRL-marked financial section range.
func filterByXBRLMarkedPages(tables []table.Table, markers []pageMarker) []table.Table {
	pages := xbrlFinancialPages(markers, maxTablePage(tables))

	var result []table.Table

	for _, tbl := range tables {
		if !pages[tbl.PageNum] {
			continue
		}

		if isSubsidiaryTable(tbl) {
			continue
		}

		result = append(result, tbl)
	}

	if len(result) == 0 {
		return filterByNumericContent(tables)
	}

	return result
}

// xbrlFinancialPages builds a set of page numbers that belong to any
// XBRL-marked financial section.
func xbrlFinancialPages(markers []pageMarker, lastPage int) map[int]bool {
	pages := make(map[int]bool)

	for i, m := range markers {
		endPage := lastPage
		for j := i + 1; j < len(markers); j++ {
			if markers[j].page > m.page {
				endPage = markers[j].page - 1

				break
			}
		}

		for p := m.page; p <= endPage; p++ {
			pages[p] = true
		}
	}

	return pages
}

// filterByNumericContent keeps tables that have rows with numeric values,
// filtering out narrative/prose pages in annual reports.
func filterByNumericContent(tables []table.Table) []table.Table {
	var result []table.Table

	for _, tbl := range tables {
		if tbl.PageNum == 1 {
			continue
		}

		if isSubsidiaryTable(tbl) {
			continue
		}

		if !hasNumericColumns(tbl) {
			continue
		}

		result = append(result, tbl)
	}

	if len(result) == 0 {
		return filterByHeuristic(tables)
	}

	return result
}

// hasNumericColumns checks whether a table has at least one row with
// parseable numeric values beyond the label column, indicating it is a
// financial data table rather than a narrative page.
func hasNumericColumns(tbl table.Table) bool {
	checkRows := min(metadataScanRows, len(tbl.Rows))

	for i := range checkRows {
		row := tbl.Rows[i]
		for j := 1; j < len(row.Cells); j++ {
			text := strings.TrimSpace(row.Cells[j].Text)
			if text == "" {
				continue
			}

			if _, err := ParseNumber(text); err == nil {
				return true
			}
		}
	}

	return false
}

func filterByPageRange(tables []table.Table, pr *xbrlPageRange) []table.Table {
	var result []table.Table

	for _, tbl := range tables {
		if tbl.PageNum < pr.start || tbl.PageNum > pr.end {
			continue
		}

		if isSubsidiaryTable(tbl) {
			continue
		}

		result = append(result, tbl)
	}

	if len(result) == 0 {
		return filterByHeuristic(tables)
	}

	return result
}

func filterByHeuristic(tables []table.Table) []table.Table {
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

func detectXBRLPageRange(tables []table.Table, docType DocType) *xbrlPageRange {
	markers := scanXBRLMarkers(tables)
	if len(markers) == 0 {
		return nil
	}

	targetPage := findTargetPage(markers, docType)
	if targetPage == 0 {
		return nil
	}

	endPage := findEndPage(markers, tables, docType, targetPage)

	return &xbrlPageRange{start: targetPage, end: endPage}
}

func scanXBRLMarkers(tables []table.Table) []pageMarker {
	var markers []pageMarker

	for _, tbl := range tables {
		for _, text := range collectTableText(tbl) {
			for code, dt := range xbrlMarkers {
				if strings.Contains(text, code) {
					markers = append(markers, pageMarker{page: tbl.PageNum, docType: dt})
				}
			}
		}
	}

	return markers
}

func findTargetPage(markers []pageMarker, docType DocType) int {
	for _, m := range markers {
		if m.docType == docType {
			return m.page
		}
	}

	return 0
}

func findEndPage(markers []pageMarker, tables []table.Table, docType DocType, targetPage int) int {
	endPage := maxTablePage(tables)

	for _, m := range markers {
		if m.page > targetPage && m.page <= endPage && m.docType != docType {
			endPage = m.page - 1
		}
	}

	return endPage
}

func collectTableText(tbl table.Table) []string {
	texts := make([]string, 0, len(tbl.Headers)+len(tbl.PageText))
	texts = append(texts, tbl.Headers...)
	texts = append(texts, tbl.PageText...)

	scanRows := min(metadataScanRows, len(tbl.Rows))

	for i := range scanRows {
		for _, cell := range tbl.Rows[i].Cells {
			texts = append(texts, cell.Text)
		}
	}

	return texts
}

func maxTablePage(tables []table.Table) int {
	maxPage := 0

	for _, tbl := range tables {
		if tbl.PageNum > maxPage {
			maxPage = tbl.PageNum
		}
	}

	return maxPage
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

func isMetadataRow(row table.Row) bool {
	for _, cell := range row.Cells {
		text := strings.TrimSpace(cell.Text)
		if text == "" {
			continue
		}

		if containsPeriodDate(text) {
			return true
		}

		if xbrlCodePattern.MatchString(text) {
			return true
		}
	}

	return false
}

// isNoiseLabel returns true if the label is noise rather than a financial
// line item: year numbers (possibly kerned), purely numeric strings, or
// labels shorter than 3 characters.
func isNoiseLabel(label string) bool {
	collapsed := collapseKernedDigits(label)

	if len([]rune(collapsed)) < 3 {
		return true
	}

	if noiseYearPattern.MatchString(collapsed) {
		return true
	}

	trimmed := strings.TrimSpace(collapsed)
	if _, err := strconv.Atoi(trimmed); err == nil {
		return true
	}

	return false
}

func containsPeriodDate(text string) bool {
	if periodPatternID.MatchString(text) ||
		periodPatternEN.MatchString(text) ||
		periodPatternENDayFirst.MatchString(text) ||
		periodAbbrev.MatchString(text) {
		return true
	}

	normalized := collapseKernedDigits(text)

	return normalized != text &&
		(periodPatternID.MatchString(normalized) ||
			periodPatternEN.MatchString(normalized) ||
			periodPatternENDayFirst.MatchString(normalized))
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

// growthRateColumns returns the set of column indices whose headers
// indicate growth rates (YoY, QoQ, %) rather than financial amounts.
func growthRateColumns(headers []string) map[int]bool {
	skip := make(map[int]bool)

	for i, h := range headers {
		trimmed := strings.TrimSpace(h)
		if growthRateHeader.MatchString(trimmed) {
			skip[i] = true
		}
	}

	return skip
}

func isPercentageValue(text string) bool {
	return strings.HasSuffix(text, "%")
}

func normalizeUnit(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "juta"),
		strings.HasPrefix(lower, "million"),
		lower == "mn":
		return unitMillions
	case strings.HasPrefix(lower, "miliar"),
		strings.HasPrefix(lower, "billion"),
		lower == "bn":
		return unitBillions
	case strings.HasPrefix(lower, "triliun"),
		strings.HasPrefix(lower, "trillion"),
		lower == "tn":
		return unitTrillions
	case strings.HasPrefix(lower, "ribu"),
		strings.HasPrefix(lower, "thousand"):
		return unitThousands
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

func normalizeShortCurrency(raw string) string {
	upper := strings.ToUpper(raw)
	switch upper {
	case "RP", currencyIDR:
		return currencyIDR
	case currencyUSD:
		return currencyUSD
	default:
		return upper
	}
}
