package domain

// DocType represents the type of an IDX financial report.
type DocType string

const (
	DocTypeBalanceSheet          DocType = "balance-sheet"
	DocTypeIncomeStatement       DocType = "income-statement"
	DocTypeCashFlow              DocType = "cash-flow"
	DocTypeEquityChanges         DocType = "equity-changes"
	DocTypeNotes                 DocType = "notes"
	DocTypeAuditorReport         DocType = "auditor-report"
	DocTypeSustainabilityReport  DocType = "sustainability-report"
	DocTypeAnnualReport          DocType = "annual-report"
	DocTypeCorporatePresentation DocType = "corporate-presentation"
	DocTypeUnknown               DocType = "unknown"
)

// Classification holds the result of document classification.
type Classification struct {
	Type       DocType `json:"type"`
	Confidence float64 `json:"confidence"`      // 0.0 to 1.0
	Language   string  `json:"language"`        // "id" or "en"
	Title      string  `json:"title,omitempty"` // detected report title
}

// AllDocTypes returns all known document types (excluding unknown).
func AllDocTypes() []DocType {
	return []DocType{
		DocTypeBalanceSheet,
		DocTypeIncomeStatement,
		DocTypeCashFlow,
		DocTypeEquityChanges,
		DocTypeNotes,
		DocTypeAuditorReport,
		DocTypeSustainabilityReport,
		DocTypeAnnualReport,
		DocTypeCorporatePresentation,
	}
}
