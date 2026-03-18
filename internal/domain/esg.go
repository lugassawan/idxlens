package domain

import (
	"regexp"
	"slices"
	"strings"

	"github.com/lugassawan/idxlens/internal/table"
)

// GRI disclosure statuses.
const (
	StatusReported          = "reported"
	StatusPartiallyReported = "partially reported"
	StatusNotReported       = "not reported"
)

// griPattern matches GRI disclosure numbers in formats like "GRI 201-1",
// "201-1", or "GRI201-1".
var griPattern = regexp.MustCompile(`(?i)(?:GRI\s*)?(\d{3}-\d+)`)

// GRIDisclosure represents a single GRI Standards disclosure entry from an
// ESG/sustainability content index table.
type GRIDisclosure struct {
	Number      string `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PageRef     string `json:"page_ref"`
	Status      string `json:"status"`
}

// ESGReport holds all GRI disclosures extracted from ESG content index tables.
type ESGReport struct {
	Disclosures []GRIDisclosure `json:"disclosures"`
	Framework   string          `json:"framework"`
}

// ESGExtractor extracts GRI content index data from parsed tables.
type ESGExtractor struct{}

// NewESGExtractor creates a new ESGExtractor.
func NewESGExtractor() *ESGExtractor {
	return &ESGExtractor{}
}

// Extract scans the provided tables for GRI content index data and returns an
// ESGReport. Tables without GRI-related content are skipped. Returns nil if no
// GRI disclosures are found.
func (e *ESGExtractor) Extract(tables []table.Table) *ESGReport {
	var disclosures []GRIDisclosure

	for i := range tables {
		if !isGRITable(tables[i]) {
			continue
		}

		disclosures = append(disclosures, extractDisclosures(tables[i])...)
	}

	if len(disclosures) == 0 {
		return nil
	}

	return &ESGReport{
		Disclosures: disclosures,
		Framework:   "GRI",
	}
}

func isGRITable(t table.Table) bool {
	if slices.ContainsFunc(t.Headers, containsGRI) {
		return true
	}

	if len(t.Rows) > 0 {
		for _, cell := range t.Rows[0].Cells {
			if containsGRI(cell.Text) {
				return true
			}
		}
	}

	return false
}

func containsGRI(text string) bool {
	return strings.Contains(strings.ToUpper(text), "GRI")
}

func extractDisclosures(t table.Table) []GRIDisclosure {
	var disclosures []GRIDisclosure

	for _, row := range t.Rows {
		disclosure, ok := parseDisclosureRow(row)
		if ok {
			disclosures = append(disclosures, disclosure)
		}
	}

	return disclosures
}

func parseDisclosureRow(row table.Row) (GRIDisclosure, bool) {
	if len(row.Cells) == 0 {
		return GRIDisclosure{}, false
	}

	number := extractDisclosureNumber(row)
	if number == "" {
		return GRIDisclosure{}, false
	}

	disclosure := GRIDisclosure{
		Number: number,
	}

	assignDisclosureFields(&disclosure, row)

	return disclosure, true
}

func extractDisclosureNumber(row table.Row) string {
	for _, cell := range row.Cells {
		match := griPattern.FindStringSubmatch(cell.Text)
		if match != nil {
			return match[1]
		}
	}

	return ""
}

func assignDisclosureFields(d *GRIDisclosure, row table.Row) {
	cells := row.Cells
	numIdx := findDisclosureNumberIndex(cells)

	// Assign description from cells adjacent to the disclosure number.
	for i, cell := range cells {
		if i == numIdx {
			continue
		}

		text := strings.TrimSpace(cell.Text)
		if text == "" {
			continue
		}

		if isPageReference(text) {
			d.PageRef = text

			continue
		}

		if status := detectStatus(text); status != "" {
			d.Status = status

			continue
		}

		// First non-number text cell becomes the title, second becomes
		// the description.
		if d.Title == "" {
			d.Title = text
		} else if d.Description == "" {
			d.Description = text
		}
	}
}

func findDisclosureNumberIndex(cells []table.Cell) int {
	for i, cell := range cells {
		if griPattern.MatchString(cell.Text) {
			return i
		}
	}

	return -1
}

func isPageReference(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	for _, r := range trimmed {
		if (r < '0' || r > '9') && r != ',' && r != '-' && r != ' ' {
			return false
		}
	}

	return true
}

func detectStatus(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))

	switch {
	case strings.Contains(lower, "partially reported"):
		return StatusPartiallyReported
	case strings.Contains(lower, "not reported"):
		return StatusNotReported
	case strings.Contains(lower, "reported"):
		return StatusReported
	}

	return ""
}
