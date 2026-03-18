package domain

import (
	"errors"
	"testing"
)

func TestParseNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr error
	}{
		// Basic integers
		{name: "simple integer", input: "1234", want: 1234},
		{name: "thousands separator", input: "1.234", want: 1234},
		{name: "millions", input: "1.234.567", want: 1234567},
		{name: "billions", input: "1.234.567.890", want: 1234567890},

		// Decimals
		{name: "decimal", input: "1.234,56", want: 1234.56},
		{name: "decimal no thousands", input: "123,45", want: 123.45},
		{name: "only decimal", input: "0,50", want: 0.50},

		// Negatives
		{name: "negative with minus", input: "-1.234", want: -1234},
		{name: "negative with parens", input: "(1.234)", want: -1234},
		{name: "negative decimal parens", input: "(1.234,56)", want: -1234.56},

		// Special values
		{name: "dash is zero", input: "-", want: 0},
		{name: "empty string", input: "", want: 0, wantErr: ErrEmptyValue},
		{name: "whitespace only", input: "   ", want: 0, wantErr: ErrEmptyValue},
		{name: "not a number", input: "N/A", want: 0, wantErr: ErrNotANumber},
		{name: "text string", input: "Total", want: 0, wantErr: ErrNotANumber},

		// English format (commas as thousands)
		{name: "english thousands", input: "1,234", want: 1234},
		{name: "english millions", input: "1,234,567", want: 1234567},
		{name: "english billions", input: "1,234,567,890", want: 1234567890},
		{name: "english with decimal", input: "1,234.56", want: 1234.56},
		{name: "english negative parens", input: "(1,234)", want: -1234},
		{name: "english negative millions", input: "( 16,780,115 )", want: -16780115},

		// Edge cases
		{name: "leading/trailing spaces", input: "  1.234  ", want: 1234},
		{name: "zero", input: "0", want: 0},
		{name: "negative zero", input: "-0", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNumber(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseNumber(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("ParseNumber(%q) unexpected error: %v", tt.input, err)

				return
			}

			if got != tt.want {
				t.Errorf("ParseNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
