package domain

import "testing"

func TestLoadDictionary(t *testing.T) {
	tests := []struct {
		name    string
		docType DocType
		wantErr bool
	}{
		{
			name:    "balance sheet loads successfully",
			docType: DocTypeBalanceSheet,
		},
		{
			name:    "unknown type returns error",
			docType: DocTypeUnknown,
			wantErr: true,
		},
		{
			name:    "income statement loads successfully",
			docType: DocTypeIncomeStatement,
		},
		{
			name:    "cash flow loads successfully",
			docType: DocTypeCashFlow,
		},
		{
			name:    "equity changes loads successfully",
			docType: DocTypeEquityChanges,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dict, err := LoadDictionary(tt.docType)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if dict.Type != string(tt.docType) {
				t.Errorf("type = %q, want %q", dict.Type, tt.docType)
			}

			if dict.Version < 1 {
				t.Errorf("version = %d, want >= 1", dict.Version)
			}

			if len(dict.Items) == 0 {
				t.Error("expected items, got none")
			}
		})
	}
}

func TestDictionaryMatchLabel(t *testing.T) {
	dict, err := LoadDictionary(DocTypeBalanceSheet)
	if err != nil {
		t.Fatalf("failed to load dictionary: %v", err)
	}

	tests := []struct {
		name           string
		text           string
		lang           string
		wantKey        string
		wantConfidence float64
		wantNil        bool
	}{
		{
			name:           "exact match Indonesian",
			text:           "Kas dan Setara Kas",
			lang:           "id",
			wantKey:        "cash_and_equivalents",
			wantConfidence: 1.0,
		},
		{
			name:           "exact match English",
			text:           "Total Assets",
			lang:           "en",
			wantKey:        "total_assets",
			wantConfidence: 1.0,
		},
		{
			name:           "case insensitive match",
			text:           "total assets",
			lang:           "en",
			wantKey:        "total_assets",
			wantConfidence: 0.9,
		},
		{
			name:           "case insensitive Indonesian",
			text:           "kas dan setara kas",
			lang:           "id",
			wantKey:        "cash_and_equivalents",
			wantConfidence: 0.9,
		},
		{
			name:           "contains match",
			text:           "Total Assets (Consolidated)",
			lang:           "en",
			wantKey:        "total_assets",
			wantConfidence: 0.7,
		},
		{
			name:    "unknown label returns nil",
			text:    "Something Completely Unknown",
			lang:    "en",
			wantNil: true,
		},
		{
			name:    "empty text returns nil",
			text:    "",
			lang:    "en",
			wantNil: true,
		},
		{
			name:    "whitespace only returns nil",
			text:    "   ",
			lang:    "en",
			wantNil: true,
		},
		{
			name:           "unknown language falls back to other languages",
			text:           "Total Assets",
			lang:           "fr",
			wantKey:        "total_assets",
			wantConfidence: 1.0,
		},
		{
			name:           "alternate label variant",
			text:           "Jumlah Aset",
			lang:           "id",
			wantKey:        "total_assets",
			wantConfidence: 1.0,
		},
		{
			name:           "indonesian label matched when language is english",
			text:           "Kas dan Setara Kas",
			lang:           "en",
			wantKey:        "cash_and_equivalents",
			wantConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, confidence := dict.MatchLabel(tt.text, tt.lang)

			if tt.wantNil {
				if item != nil {
					t.Errorf("expected nil item, got %q", item.Key)
				}
				if confidence != 0 {
					t.Errorf("expected 0 confidence, got %f", confidence)
				}
				return
			}

			if item == nil {
				t.Fatal("expected item, got nil")
			}

			if item.Key != tt.wantKey {
				t.Errorf("key = %q, want %q", item.Key, tt.wantKey)
			}

			if confidence != tt.wantConfidence {
				t.Errorf("confidence = %f, want %f", confidence, tt.wantConfidence)
			}
		})
	}
}

func TestDictionaryMatchLabelIncomeStatement(t *testing.T) {
	dict, err := LoadDictionary(DocTypeIncomeStatement)
	if err != nil {
		t.Fatalf("failed to load dictionary: %v", err)
	}

	tests := []struct {
		name           string
		text           string
		lang           string
		wantKey        string
		wantConfidence float64
	}{
		{
			name:           "interest income exact case variant",
			text:           "Interest income",
			lang:           "en",
			wantKey:        "interest_income_en",
			wantConfidence: 1.0,
		},
		{
			name:           "interest expenses exact variant",
			text:           "Interest expenses",
			lang:           "en",
			wantKey:        "interest_expense_en",
			wantConfidence: 1.0,
		},
		{
			name:           "INTEREST INCOME all caps",
			text:           "INTEREST INCOME",
			lang:           "en",
			wantKey:        "interest_income_en",
			wantConfidence: 1.0,
		},
		{
			name:           "non-breaking space normalization",
			text:           "Interest\u00A0Income",
			lang:           "en",
			wantKey:        "interest_income_bank",
			wantConfidence: 1.0,
		},
		{
			name:           "profit or loss partial match",
			text:           "profit or loss",
			lang:           "en",
			wantKey:        "oci_not_reclassified",
			wantConfidence: 0.6,
		},
		{
			name:           "suffix stripped net match",
			text:           "Fee and Commission Income - net",
			lang:           "en",
			wantKey:        "fee_and_commission_income",
			wantConfidence: 0.85,
		},
		{
			name:           "case insensitive bersih label match",
			text:           "Pendapatan Provisi dan Komisi - bersih",
			lang:           "id",
			wantKey:        "fee_and_commission_income",
			wantConfidence: 0.9,
		},
		{
			name:           "suffix stripped bersih match",
			text:           "Beban Provisi dan Komisi - bersih",
			lang:           "id",
			wantKey:        "fee_and_commission_expense",
			wantConfidence: 0.85,
		},
		{
			name:           "suffix stripped net case insensitive",
			text:           "Other Income (Expense) - Net",
			lang:           "en",
			wantKey:        "other_income_expense_net",
			wantConfidence: 1.0,
		},
		{
			name:           "plural to singular match expenses to expense",
			text:           "Selling Expense",
			lang:           "en",
			wantKey:        "selling_expenses",
			wantConfidence: 0.85,
		},
		{
			name:           "singular to plural match expense to expenses",
			text:           "Rental Expense",
			lang:           "en",
			wantKey:        "rental_expenses",
			wantConfidence: 0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, confidence := dict.MatchLabel(tt.text, tt.lang)

			if item == nil {
				t.Fatalf("expected item, got nil for %q", tt.text)
			}

			if item.Key != tt.wantKey {
				t.Errorf("key = %q, want %q", item.Key, tt.wantKey)
			}

			if confidence != tt.wantConfidence {
				t.Errorf("confidence = %f, want %f", confidence, tt.wantConfidence)
			}
		})
	}
}

func TestStripSuffix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "strip net", input: "income - net", want: "income"},
		{name: "strip bersih", input: "pendapatan - bersih", want: "pendapatan"},
		{name: "strip net no space before dash", input: "income -net", want: "income"},
		{name: "no suffix unchanged", input: "interest income", want: "interest income"},
		{name: "net in middle unchanged", input: "net income", want: "net income"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripSuffix(tt.input)
			if got != tt.want {
				t.Errorf("stripSuffix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchPlural(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "expense vs expenses", a: "expense", b: "expenses", want: true},
		{name: "expenses vs expense", a: "expenses", b: "expense", want: true},
		{name: "selling expense vs selling expenses", a: "selling expense", b: "selling expenses", want: true},
		{name: "fee vs fees", a: "fee", b: "fees", want: true},
		{name: "same word no plural", a: "income", b: "income", want: false},
		{name: "different words", a: "income", b: "expense", want: false},
		{name: "different word count", a: "selling", b: "selling expenses", want: false},
		{name: "different prefix", a: "selling expense", b: "rental expenses", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPlural(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("matchPlural(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDictionaryValidation(t *testing.T) {
	docTypes := []DocType{
		DocTypeBalanceSheet,
		DocTypeIncomeStatement,
		DocTypeCashFlow,
		DocTypeEquityChanges,
	}

	totalItems := 0

	for _, docType := range docTypes {
		t.Run("validate_"+string(docType), func(t *testing.T) {
			dict, err := LoadDictionary(docType)
			if err != nil {
				t.Fatalf("failed to load %s: %v", docType, err)
			}

			if len(dict.Items) == 0 {
				t.Fatalf("dictionary %s has no items", docType)
			}

			totalItems += len(dict.Items)

			seen := make(map[string]bool)
			for _, item := range dict.Items {
				if seen[item.Key] {
					t.Errorf("duplicate key %q in %s", item.Key, docType)
				}
				seen[item.Key] = true

				idLabels := item.Labels["id"]
				if len(idLabels) == 0 {
					t.Errorf("item %q missing Indonesian labels", item.Key)
				}

				enLabels := item.Labels["en"]
				if len(enLabels) == 0 {
					t.Errorf("item %q missing English labels", item.Key)
				}

				if item.Section == "" {
					t.Errorf("item %q has empty section", item.Key)
				}
			}
		})
	}

	t.Run("total item count", func(t *testing.T) {
		if totalItems < 200 {
			t.Errorf("total items across all dictionaries = %d, want >= 200", totalItems)
		}
	})
}

func TestLoadAllDictionaries(t *testing.T) {
	dict, err := LoadAllDictionaries()
	if err != nil {
		t.Fatalf("LoadAllDictionaries() error: %v", err)
	}

	if dict.Type != "all" {
		t.Errorf("type = %q, want %q", dict.Type, "all")
	}

	// Should contain items from all four financial dictionaries.
	if len(dict.Items) < 200 {
		t.Errorf("items = %d, want >= 200", len(dict.Items))
	}

	// Verify items from different statement types are present by checking
	// for known keys from each dictionary.
	wantKeys := map[string]bool{
		"cash_and_equivalents": false, // balance sheet
		"net_income":           false, // income statement
	}

	for _, item := range dict.Items {
		if _, ok := wantKeys[item.Key]; ok {
			wantKeys[item.Key] = true
		}
	}

	for key, found := range wantKeys {
		if !found {
			t.Errorf("expected key %q not found in merged dictionary", key)
		}
	}
}
