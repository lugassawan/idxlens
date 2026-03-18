package domain

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed dictionaries/*.json
var dictionaryFS embed.FS

// DictionaryItem represents a single financial line item definition.
type DictionaryItem struct {
	Key     string              `json:"key"`
	Labels  map[string][]string `json:"labels"`
	Section string              `json:"section"`
	Level   int                 `json:"level"`
}

// Dictionary holds all line items for a specific report type.
type Dictionary struct {
	Type    string           `json:"type"`
	Version int              `json:"version"`
	Items   []DictionaryItem `json:"items"`
}

var multiSpacePattern = regexp.MustCompile(`\s+`)

// financialDocTypes lists the document types that have dictionaries.
var financialDocTypes = []DocType{
	DocTypeBalanceSheet,
	DocTypeIncomeStatement,
	DocTypeCashFlow,
	DocTypeEquityChanges,
}

// LoadDictionary loads a dictionary for the given report type.
func LoadDictionary(docType DocType) (*Dictionary, error) {
	filename, err := docTypeFilename(docType)
	if err != nil {
		return nil, err
	}

	data, err := dictionaryFS.ReadFile("dictionaries/" + filename)
	if err != nil {
		return nil, fmt.Errorf("load dictionary %s: %w", docType, err)
	}

	var dict Dictionary
	if err := json.Unmarshal(data, &dict); err != nil {
		return nil, fmt.Errorf("parse dictionary %s: %w", docType, err)
	}

	return &dict, nil
}

// LoadAllDictionaries loads dictionaries for all financial statement types
// and merges their items into a single dictionary. This enables matching
// labels from composite reports (audited, annual) that contain multiple
// statement types.
func LoadAllDictionaries() (*Dictionary, error) {
	merged := &Dictionary{Type: "all", Version: 1}

	for _, dt := range financialDocTypes {
		d, err := LoadDictionary(dt)
		if err != nil {
			return nil, fmt.Errorf("load all dictionaries: %w", err)
		}

		merged.Items = append(merged.Items, d.Items...)
	}

	return merged, nil
}

// MatchLabel finds the best matching dictionary item for a given text label.
// It checks the preferred language first, then falls back to all other
// languages. IDX reports are bilingual — date headers may be English while
// line-item labels are Indonesian.
// Returns nil and 0 if no match is found.
func (d *Dictionary) MatchLabel(text string, lang string) (*DictionaryItem, float64) {
	normalized := normalizeWhitespace(text)
	if normalized == "" {
		return nil, 0
	}

	lowered := strings.ToLower(normalized)

	var bestItem *DictionaryItem
	var bestConfidence float64

	for i := range d.Items {
		item := &d.Items[i]
		confidence := matchItemAllLanguages(normalized, lowered, item, lang)

		if confidence > bestConfidence {
			bestItem = item
			bestConfidence = confidence
		}
	}

	return bestItem, bestConfidence
}

func matchItemAllLanguages(
	normalized string, lowered string, item *DictionaryItem, preferredLang string,
) float64 {
	// Try preferred language first.
	if labels, ok := item.Labels[preferredLang]; ok {
		if c := matchLabels(normalized, lowered, labels); c > 0 {
			return c
		}
	}

	// Fall back to all other languages.
	var best float64

	for lang, labels := range item.Labels {
		if lang == preferredLang {
			continue
		}

		if c := matchLabels(normalized, lowered, labels); c > best {
			best = c
		}
	}

	return best
}

func matchLabels(normalized string, lowered string, labels []string) float64 {
	var best float64

	for _, label := range labels {
		if normalized == label {
			return 1.0
		}

		labelLower := strings.ToLower(label)

		if best < 0.9 && lowered == labelLower {
			best = 0.9
		}

		if best < 0.7 && strings.Contains(lowered, labelLower) {
			best = 0.7
		}

		if best < 0.6 && len(lowered) > 3 && strings.Contains(labelLower, lowered) {
			best = 0.6
		}
	}

	return best
}

func docTypeFilename(docType DocType) (string, error) {
	filenames := map[DocType]string{
		DocTypeBalanceSheet:    "balance_sheet.json",
		DocTypeIncomeStatement: "income_statement.json",
		DocTypeCashFlow:        "cash_flow.json",
		DocTypeEquityChanges:   "equity_changes.json",
	}

	filename, ok := filenames[docType]
	if !ok {
		return "", fmt.Errorf("no dictionary for document type: %s", docType)
	}

	return filename, nil
}

// normalizeWhitespace replaces non-breaking spaces and other Unicode
// whitespace with ASCII spaces, collapses runs, and trims the result.
func normalizeWhitespace(text string) string {
	// Replace common non-ASCII whitespace characters with ASCII space.
	s := strings.Map(func(r rune) rune {
		switch r {
		case '\u00A0', '\u2002', '\u2003', '\u2009', '\u200A':
			return ' '
		default:
			return r
		}
	}, text)

	return strings.TrimSpace(multiSpacePattern.ReplaceAllString(s, " "))
}
