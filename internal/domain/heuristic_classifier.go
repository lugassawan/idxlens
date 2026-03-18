package domain

import (
	"regexp"
	"strings"

	"github.com/lugassawan/idxlens/internal/layout"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

// xbrlMarkerWeight is the score weight applied to XBRL section markers,
// which are highly reliable classification signals found in IDX PDFs.
const xbrlMarkerWeight = 10.0

// xbrlMarkers maps XBRL taxonomy codes to document types.
// These codes appear in IDX PDF XBRL metadata sections as "[code] Title".
// Multiple codes map to the same type because IDX uses different taxonomy
// versions (OJK 2020 uses 3xxx/5xxx/6xxx, OJK 2023+ uses 4xxx).
var xbrlMarkers = map[string]DocType{
	"[4220000]": DocTypeBalanceSheet,
	"[3210000]": DocTypeIncomeStatement,
	"[4322000]": DocTypeIncomeStatement,
	"[5310000]": DocTypeCashFlow,
	"[4510000]": DocTypeCashFlow,
	"[6110000]": DocTypeEquityChanges,
	"[4410000]": DocTypeEquityChanges,
}

// coverPagePhrases are terms found on bureaucratic cover pages that should
// reduce confidence in keyword matches from those pages. Their presence
// indicates the page is administrative, not financial content.
var coverPagePhrases = []string{
	"penyampaian",
	"nomor surat",
	"kode emiten",
}

// NewHeuristicClassifier creates a classifier that uses keyword matching
// to identify IDX financial report types.
func NewHeuristicClassifier() Classifier {
	return &heuristicClassifier{
		rules: buildClassificationRules(),
	}
}

type classificationRule struct {
	docType  DocType
	keywords []keyword
}

type keyword struct {
	phrase string
	lang   string
}

type heuristicClassifier struct {
	rules []classificationRule
}

type docTypeScore struct {
	score      float64
	totalPoss  float64
	bestWeight float64
	lang       string
}

func buildClassificationRules() []classificationRule {
	return []classificationRule{
		{
			docType: DocTypeBalanceSheet,
			keywords: []keyword{
				{phrase: "laporan posisi keuangan", lang: "id"},
				{phrase: "neraca", lang: "id"},
				{phrase: "statement of financial position", lang: "en"},
				{phrase: "balance sheet", lang: "en"},
			},
		},
		{
			docType: DocTypeIncomeStatement,
			keywords: []keyword{
				{phrase: "laporan laba rugi dan penghasilan komprehensif lain", lang: "id"},
				{phrase: "laporan laba rugi", lang: "id"},
				{phrase: "income statement", lang: "en"},
				{phrase: "profit or loss", lang: "en"},
			},
		},
		{
			docType: DocTypeCashFlow,
			keywords: []keyword{
				{phrase: "laporan arus kas", lang: "id"},
				{phrase: "statement of cash flows", lang: "en"},
				{phrase: "cash flow statement", lang: "en"},
			},
		},
		{
			docType: DocTypeEquityChanges,
			keywords: []keyword{
				{phrase: "laporan perubahan ekuitas", lang: "id"},
				{phrase: "statement of changes in equity", lang: "en"},
			},
		},
		{
			docType: DocTypeNotes,
			keywords: []keyword{
				{phrase: "catatan atas laporan keuangan", lang: "id"},
				{phrase: "notes to the financial statements", lang: "en"},
			},
		},
		{
			docType: DocTypeAuditorReport,
			keywords: []keyword{
				{phrase: "laporan auditor independen", lang: "id"},
				{phrase: "independent auditor", lang: "en"},
				{phrase: "laporan auditor", lang: "id"},
				{phrase: "opini", lang: "id"},
			},
		},
	}
}

func (c *heuristicClassifier) Classify(pages []layout.LayoutPage) (Classification, error) {
	if len(pages) == 0 {
		return Classification{Type: DocTypeUnknown}, nil
	}

	text := extractText(pages)
	normalized := normalizeText(text)

	scores := c.scoreAllTypes(normalized)
	applyXBRLMarkers(normalized, scores)
	penalizeCoverPage(normalized, scores)

	bestType, best := findBestMatch(scores)
	if bestType == DocTypeUnknown {
		return Classification{Type: DocTypeUnknown}, nil
	}

	confidence := best.score / best.totalPoss

	return Classification{
		Type:       bestType,
		Confidence: confidence,
		Language:   best.lang,
	}, nil
}

func extractText(pages []layout.LayoutPage) string {
	var sb strings.Builder
	for _, page := range pages {
		for _, line := range page.Lines {
			sb.WriteString(line.Text)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func normalizeText(text string) string {
	lowered := strings.ToLower(text)
	return whitespaceRe.ReplaceAllString(lowered, " ")
}

func (c *heuristicClassifier) scoreAllTypes(normalized string) map[DocType]*docTypeScore {
	scores := make(map[DocType]*docTypeScore)

	for _, rule := range c.rules {
		s := &docTypeScore{}
		scores[rule.docType] = s

		for _, kw := range rule.keywords {
			weight := phraseWeight(kw.phrase)
			s.totalPoss += weight

			if strings.Contains(normalized, kw.phrase) {
				s.score += weight

				if weight > s.bestWeight {
					s.bestWeight = weight
					s.lang = kw.lang
				}
			}
		}
	}

	return scores
}

func phraseWeight(phrase string) float64 {
	words := strings.Fields(phrase)
	return float64(len(words))
}

func findBestMatch(scores map[DocType]*docTypeScore) (DocType, docTypeScore) {
	bestType := DocTypeUnknown
	var best docTypeScore

	for dt, s := range scores {
		if s.score > best.score {
			bestType = dt
			best = *s
		}
	}

	if best.score == 0 {
		return DocTypeUnknown, docTypeScore{}
	}

	return bestType, best
}

// applyXBRLMarkers scans for XBRL taxonomy codes (e.g., "[4220000]") and
// adds high-weight scores to the matching document type. These markers
// appear in IDX PDF XBRL metadata sections and are very reliable signals.
func applyXBRLMarkers(normalized string, scores map[DocType]*docTypeScore) {
	for code, docType := range xbrlMarkers {
		if !strings.Contains(normalized, code) {
			continue
		}

		s, ok := scores[docType]
		if !ok {
			s = &docTypeScore{}
			scores[docType] = s
		}

		s.score += xbrlMarkerWeight
		s.totalPoss += xbrlMarkerWeight

		if xbrlMarkerWeight > s.bestWeight {
			s.bestWeight = xbrlMarkerWeight
			s.lang = detectXBRLLang(normalized, code)
		}
	}
}

// detectXBRLLang determines the language from text near an XBRL marker.
// XBRL markers are followed by English section titles, so default to "en".
func detectXBRLLang(normalized string, code string) string {
	idx := strings.Index(normalized, code)
	if idx < 0 {
		return "en"
	}

	// Look at text after the code for language hints.
	end := min(idx+len(code)+200, len(normalized))
	after := normalized[idx:end]

	if strings.Contains(after, "posisi keuangan") ||
		strings.Contains(after, "laba rugi") ||
		strings.Contains(after, "arus kas") ||
		strings.Contains(after, "perubahan ekuitas") {
		return "id"
	}

	return "en"
}

// penalizeCoverPage detects bureaucratic cover pages and reduces the
// auditor-report score when "diaudit" / "audited" appears in metadata
// context rather than as a section header.
func penalizeCoverPage(normalized string, scores map[DocType]*docTypeScore) {
	if !isCoverPage(normalized) {
		return
	}

	// On cover pages, "diaudit" / "audited" appear as metadata labels
	// (e.g., "Diaudit / Audited"), not as auditor-report section headers.
	// Reduce the auditor-report score to prevent false positives.
	auditorScore, ok := scores[DocTypeAuditorReport]
	if !ok {
		return
	}

	if hasMetadataAuditLabel(normalized) && !hasAuditorReportHeader(normalized) {
		auditorScore.score *= 0.25
	}
}

// isCoverPage returns true if the text contains phrases typical of
// IDX bureaucratic cover pages.
func isCoverPage(normalized string) bool {
	for _, phrase := range coverPagePhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

// hasMetadataAuditLabel returns true if "diaudit" or "audited" appears
// in a metadata context (e.g., "diaudit / audited" as a label).
func hasMetadataAuditLabel(normalized string) bool {
	return strings.Contains(normalized, "diaudit") ||
		strings.Contains(normalized, "/ audited")
}

// hasAuditorReportHeader returns true if the text contains an explicit
// auditor report section header.
func hasAuditorReportHeader(normalized string) bool {
	return strings.Contains(normalized, "laporan auditor independen") ||
		strings.Contains(normalized, "independent auditor")
}
