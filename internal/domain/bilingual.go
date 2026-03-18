package domain

import (
	"strings"

	"github.com/lugassawan/idxlens/internal/layout"
)

// Language represents a supported language.
type Language string

const (
	LangIndonesian Language = "id"
	LangEnglish    Language = "en"
	LangUnknown    Language = "unknown"
)

// indonesianMarkers contains words that are strong indicators of Indonesian text.
var indonesianMarkers = []string{
	"dan", "atau", "yang", "dari", "untuk", "atas", "pada", "dengan",
	"jumlah", "catatan", "pendapatan", "beban", "laba", "rugi", "aset",
	"liabilitas", "ekuitas", "kas", "piutang", "utang",
}

// englishMarkers contains words that are strong indicators of English text.
var englishMarkers = []string{
	"and", "the", "for", "from", "with", "total", "notes",
	"revenue", "expense", "profit", "loss", "assets",
	"liabilities", "equity", "cash", "receivables", "payables",
}

// markerThreshold is the minimum number of marker hits required to classify
// a language with confidence.
const markerThreshold = 2

// BilingualRouter detects languages and routes content for matching.
type BilingualRouter struct{}

// NewBilingualRouter creates a new BilingualRouter.
func NewBilingualRouter() *BilingualRouter {
	return &BilingualRouter{}
}

// DetectLanguage analyzes text content to determine the primary language.
// It counts occurrences of language-specific marker words and returns the
// language with higher marker density. Returns LangUnknown if neither
// language reaches the threshold or if both have equal counts.
func (r *BilingualRouter) DetectLanguage(text string) Language {
	if strings.TrimSpace(text) == "" {
		return LangUnknown
	}

	idCount := countMarkers(text, indonesianMarkers)
	enCount := countMarkers(text, englishMarkers)

	if idCount < markerThreshold && enCount < markerThreshold {
		return LangUnknown
	}

	if idCount > enCount {
		return LangIndonesian
	}

	if enCount > idCount {
		return LangEnglish
	}

	return LangUnknown
}

// IsBilingual checks if content appears to contain both languages.
// Returns true when both Indonesian and English marker counts meet the
// threshold.
func (r *BilingualRouter) IsBilingual(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	idCount := countMarkers(text, indonesianMarkers)
	enCount := countMarkers(text, englishMarkers)

	return idCount >= markerThreshold && enCount >= markerThreshold
}

// SplitBilingual separates bilingual content into language-specific parts
// based on spatial layout. IDX reports typically place Indonesian text on
// the left half and English text on the right half of the page. Lines are
// assigned based on whether their horizontal midpoint falls left or right
// of the page center.
func (r *BilingualRouter) SplitBilingual(lines []layout.TextLine) (indonesian, english []layout.TextLine) {
	if len(lines) == 0 {
		return nil, nil
	}

	midX := findPageMidpoint(lines)

	for _, line := range lines {
		lineMidX := (line.Bounds.X1 + line.Bounds.X2) / 2
		if lineMidX < midX {
			indonesian = append(indonesian, line)
		} else {
			english = append(english, line)
		}
	}

	return indonesian, english
}

// countMarkers counts how many marker words appear in the given text.
// Matching is case-insensitive and uses word boundary detection to avoid
// partial matches.
func countMarkers(text string, markers []string) int {
	lower := strings.ToLower(text)
	words := strings.Fields(lower)

	count := 0

	for _, marker := range markers {
		for _, word := range words {
			cleaned := strings.Trim(word, ".,;:!?()[]{}\"'")
			if cleaned == marker {
				count++
			}
		}
	}

	return count
}

// findPageMidpoint calculates the horizontal midpoint of the content area
// by finding the minimum and maximum X coordinates across all lines.
func findPageMidpoint(lines []layout.TextLine) float64 {
	minX := lines[0].Bounds.X1
	maxX := lines[0].Bounds.X2

	for _, line := range lines[1:] {
		if line.Bounds.X1 < minX {
			minX = line.Bounds.X1
		}

		if line.Bounds.X2 > maxX {
			maxX = line.Bounds.X2
		}
	}

	return (minX + maxX) / 2
}
