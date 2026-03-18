package domain

import (
	"regexp"
	"strings"

	"github.com/lugassawan/idxlens/internal/layout"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

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
