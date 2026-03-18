package domain

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/lugassawan/idxlens/internal/layout"
)

// OpinionType represents the auditor's opinion classification.
type OpinionType string

const (
	OpinionUnqualified         OpinionType = "unqualified"
	OpinionQualified           OpinionType = "qualified"
	OpinionAdverse             OpinionType = "adverse"
	OpinionDisclaimer          OpinionType = "disclaimer"
	OpinionUnqualifiedEmphasis OpinionType = "unqualified-emphasis"
	OpinionUnknown             OpinionType = "unknown"
)

// AuditorReport holds structured data from an auditor's report.
type AuditorReport struct {
	Firm            string      `json:"firm"`
	Opinion         OpinionType `json:"opinion"`
	ReportDate      string      `json:"report_date"`
	KeyAuditMatters []string    `json:"key_audit_matters"`
	Language        string      `json:"language"`
}

// AuditorParser extracts structured data from auditor report pages.
type AuditorParser struct {
	firmPatterns     []*regexp.Regexp
	datePattern      *regexp.Regexp
	kamHeaderPattern *regexp.Regexp
}

var (
	disclaimerPhrases = []string{
		"tidak memberikan pendapat",
		"disclaimer of opinion",
		"disclaim",
	}

	adversePhrases = []string{
		"tidak wajar",
		"adverse opinion",
	}

	qualifiedPhrases = []string{
		"wajar dengan pengecualian",
		"qualified opinion",
		"except for",
	}

	unqualifiedPhrases = []string{
		"wajar tanpa pengecualian",
		"unqualified opinion",
		"wajar, dalam semua hal yang material",
	}

	emphasisPhrases = []string{
		"paragraf penjelasan",
		"emphasis of matter",
		"penekanan suatu hal",
	}
)

// NewAuditorParser creates a new AuditorParser.
func NewAuditorParser() *AuditorParser {
	return &AuditorParser{
		firmPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)kantor\s+akuntan\s+publik\s+([^\n]+)`),
			regexp.MustCompile(`(?i)(?:registered\s+)?public\s+account(?:ing|ant)\s+firm\s+([^\n]+)`),
		},
		datePattern: regexp.MustCompile(
			`(\d{1,2})\s+` +
				`(januari|februari|maret|april|mei|juni|juli|agustus|september|` +
				`oktober|november|desember|` +
				`january|february|march|may|june|july|august|` +
				`october|december)\s+(\d{4})`,
		),
		kamHeaderPattern: regexp.MustCompile(`(?i)(key\s+audit\s+matters?|hal\s+audit\s+utama)`),
	}
}

// Parse extracts an AuditorReport from layout-analyzed pages.
func (p *AuditorParser) Parse(pages []layout.LayoutPage) (*AuditorReport, error) {
	if len(pages) == 0 {
		return &AuditorReport{Opinion: OpinionUnknown}, nil
	}

	text := extractText(pages)
	normalized := normalizeText(text)

	report := &AuditorReport{
		Firm:            p.detectFirm(text),
		Opinion:         detectOpinion(normalized),
		ReportDate:      p.detectDate(normalized),
		KeyAuditMatters: p.detectKeyAuditMatters(pages),
		Language:        detectLanguage(normalized),
	}

	if report.Opinion == OpinionUnqualified && hasEmphasis(normalized) {
		report.Opinion = OpinionUnqualifiedEmphasis
	}

	return report, nil
}

func (p *AuditorParser) detectFirm(text string) string {
	for _, pat := range p.firmPatterns {
		matches := pat.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

func detectOpinion(normalized string) OpinionType {
	// Check disclaimer first — "tidak memberikan pendapat" must not match
	// as adverse ("tidak wajar").
	if containsAny(normalized, disclaimerPhrases) {
		return OpinionDisclaimer
	}

	if containsAny(normalized, adversePhrases) {
		return OpinionAdverse
	}

	if containsAny(normalized, qualifiedPhrases) {
		return OpinionQualified
	}

	if containsAny(normalized, unqualifiedPhrases) {
		return OpinionUnqualified
	}

	return OpinionUnknown
}

func hasEmphasis(normalized string) bool {
	return containsAny(normalized, emphasisPhrases)
}

func detectLanguage(normalized string) string {
	idCount := countMatches(normalized, []string{
		"wajar", "pendapat", "laporan", "keuangan", "auditor independen",
	})
	enCount := countMatches(normalized, []string{
		"opinion", "financial statements", "independent auditor", "audit",
	})

	if idCount > enCount {
		return "id"
	}

	if enCount > 0 {
		return "en"
	}

	return ""
}

func (p *AuditorParser) detectDate(normalized string) string {
	matches := p.datePattern.FindStringSubmatch(normalized)
	if len(matches) < 4 {
		return ""
	}

	return matches[1] + " " + capitalizeFirst(matches[2]) + " " + matches[3]
}

func (p *AuditorParser) detectKeyAuditMatters(pages []layout.LayoutPage) []string {
	allLines := collectLines(pages)

	startIdx := -1

	for i, line := range allLines {
		normalized := normalizeText(line)
		if p.kamHeaderPattern.MatchString(normalized) {
			startIdx = i + 1

			break
		}
	}

	if startIdx < 0 {
		return nil
	}

	var matters []string

	for i := startIdx; i < len(allLines); i++ {
		trimmed := strings.TrimSpace(allLines[i])
		if trimmed == "" {
			continue
		}

		if isEndOfSection(trimmed) {
			break
		}

		if isMatterHeading(trimmed) {
			matters = append(matters, trimmed)
		}
	}

	return matters
}

func collectLines(pages []layout.LayoutPage) []string {
	var lines []string

	for _, page := range pages {
		for _, line := range page.Lines {
			lines = append(lines, line.Text)
		}
	}

	return lines
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

func containsAny(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}

	return false
}

func countMatches(text string, phrases []string) int {
	count := 0

	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			count++
		}
	}

	return count
}

func isEndOfSection(line string) bool {
	endMarkers := []string{
		"laporan auditor",
		"independent auditor",
		"tanggung jawab",
		"responsibility",
		"basis for",
		"dasar",
	}

	lower := strings.ToLower(line)

	for _, marker := range endMarkers {
		if strings.HasPrefix(lower, marker) {
			return true
		}
	}

	return false
}

func isMatterHeading(line string) bool {
	// KAM headings are typically short lines (under 100 chars) that don't
	// start with lowercase and are not purely numeric.
	if len(line) > 100 {
		return false
	}

	if len(line) < 3 {
		return false
	}

	first := rune(line[0])

	return first >= 'A' && first <= 'Z'
}
