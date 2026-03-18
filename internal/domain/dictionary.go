package domain

import (
	"embed"
	"encoding/json"
	"fmt"
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

// MatchLabel finds the best matching dictionary item for a given text label.
// Returns nil and 0 if no match is found.
func (d *Dictionary) MatchLabel(text string, lang string) (*DictionaryItem, float64) {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return nil, 0
	}

	lowered := strings.ToLower(normalized)

	var bestItem *DictionaryItem
	var bestConfidence float64

	for i := range d.Items {
		item := &d.Items[i]
		labels, ok := item.Labels[lang]
		if !ok {
			continue
		}

		confidence := matchLabels(normalized, lowered, labels)
		if confidence > bestConfidence {
			bestItem = item
			bestConfidence = confidence
		}
	}

	return bestItem, bestConfidence
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
