package domain

// FinancialStatement represents a fully parsed financial statement with
// standardized line items mapped from raw table data.
type FinancialStatement struct {
	Type     DocType    `json:"type"`
	Company  string     `json:"company"`
	Periods  []string   `json:"periods"`
	Currency string     `json:"currency"`
	Unit     string     `json:"unit"`
	Items    []LineItem `json:"items"`
	Language string     `json:"language"`
}

// LineItem represents a single row in a financial statement, mapped to a
// standardized key from the dictionary.
type LineItem struct {
	Key        string             `json:"key"`
	Label      string             `json:"label"`
	Section    string             `json:"section"`
	Level      int                `json:"level"`
	Values     map[string]float64 `json:"values"`
	IsSubtotal bool               `json:"is_subtotal"`
	Confidence float64            `json:"confidence"`
}
