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
			name:    "unsupported type returns error",
			docType: DocTypeIncomeStatement,
			wantErr: true,
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
			name:    "unknown language returns nil",
			text:    "Total Assets",
			lang:    "fr",
			wantNil: true,
		},
		{
			name:           "alternate label variant",
			text:           "Jumlah Aset",
			lang:           "id",
			wantKey:        "total_assets",
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
