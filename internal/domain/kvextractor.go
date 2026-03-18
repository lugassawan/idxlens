package domain

import (
	"strings"

	"github.com/lugassawan/idxlens/internal/layout"
)

// Confidence levels for key-value pair extraction patterns.
const (
	confidenceColonSeparated  = 0.9
	confidenceLabelAboveValue = 0.7
)

// minTabularColumns is the minimum number of whitespace-separated segments
// that indicates a line is table content rather than a key-value pair.
const minTabularColumns = 3

// horizontalAlignmentThreshold is the maximum horizontal distance (in PDF
// units) between the start of two consecutive lines for them to be considered
// vertically aligned (label-above-value pattern).
const horizontalAlignmentThreshold = 50.0

// KeyValuePair represents a single key-value pair extracted from PDF content.
type KeyValuePair struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	PageNum    int     `json:"page_num"`
}

// KVExtractor extracts key-value pairs from non-tabular PDF content.
type KVExtractor struct{}

// NewKVExtractor creates a new KVExtractor.
func NewKVExtractor() *KVExtractor {
	return &KVExtractor{}
}

// Extract scans layout pages for key-value pairs using multiple detection
// patterns: colon-separated pairs and label-above-value patterns. Lines that
// appear to be table content are skipped.
func (e *KVExtractor) Extract(pages []layout.LayoutPage) []KeyValuePair {
	pairs := make([]KeyValuePair, 0, len(pages))

	for _, page := range pages {
		pairs = append(pairs, extractFromPage(page)...)
	}

	return pairs
}

func extractFromPage(page layout.LayoutPage) []KeyValuePair {
	var pairs []KeyValuePair

	consumed := make([]bool, len(page.Lines))

	// First pass: colon-separated pairs (higher confidence).
	for i, line := range page.Lines {
		if isTabularContent(line.Text) {
			continue
		}

		pair, ok := extractColonPair(line.Text, page.Number)
		if ok {
			pairs = append(pairs, pair)
			consumed[i] = true
		}
	}

	// Second pass: label-above-value pairs from unconsumed lines.
	for i := range len(page.Lines) - 1 {
		if consumed[i] || consumed[i+1] {
			continue
		}

		pair, ok := extractLabelAboveValue(page.Lines[i], page.Lines[i+1], page.Number)
		if ok {
			pairs = append(pairs, pair)
			consumed[i] = true
			consumed[i+1] = true
		}
	}

	return pairs
}

func extractColonPair(text string, pageNum int) (KeyValuePair, bool) {
	before, after, found := strings.Cut(text, ":")
	if !found {
		return KeyValuePair{}, false
	}

	key := strings.TrimSpace(before)
	value := strings.TrimSpace(after)

	if key == "" || value == "" {
		return KeyValuePair{}, false
	}

	return KeyValuePair{
		Key:        key,
		Value:      value,
		Confidence: confidenceColonSeparated,
		PageNum:    pageNum,
	}, true
}

func extractLabelAboveValue(label, value layout.TextLine, pageNum int) (KeyValuePair, bool) {
	if label.FontSize <= value.FontSize {
		return KeyValuePair{}, false
	}

	if isTabularContent(label.Text) || isTabularContent(value.Text) {
		return KeyValuePair{}, false
	}

	horizontalDist := label.Bounds.X1 - value.Bounds.X1
	if horizontalDist < 0 {
		horizontalDist = -horizontalDist
	}

	if horizontalDist > horizontalAlignmentThreshold {
		return KeyValuePair{}, false
	}

	labelText := strings.TrimSpace(label.Text)
	valueText := strings.TrimSpace(value.Text)

	if labelText == "" || valueText == "" {
		return KeyValuePair{}, false
	}

	return KeyValuePair{
		Key:        labelText,
		Value:      valueText,
		Confidence: confidenceLabelAboveValue,
		PageNum:    pageNum,
	}, true
}

func isTabularContent(text string) bool {
	segments := strings.Fields(text)
	if len(segments) < minTabularColumns {
		return false
	}

	// Count segments that look like numbers or are separated by wide gaps.
	// Table content typically has multiple numeric columns.
	numericCount := 0

	for _, seg := range segments {
		if looksNumeric(seg) {
			numericCount++
		}
	}

	return numericCount >= minTabularColumns-1
}

func looksNumeric(s string) bool {
	cleaned := strings.NewReplacer(",", "", ".", "", "(", "", ")", "", "-", "").Replace(s)
	if cleaned == "" {
		return false
	}

	for _, r := range cleaned {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
