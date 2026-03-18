package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lugassawan/idxlens/internal/domain"
)

func sampleStatement() *domain.FinancialStatement {
	return &domain.FinancialStatement{
		Type:     domain.DocTypeBalanceSheet,
		Company:  "PT Test Tbk",
		Periods:  []string{"2024-12-31", "2023-12-31"},
		Currency: "IDR",
		Unit:     "millions",
		Language: "id",
		Items: []domain.LineItem{
			{
				Key:        "cash_and_equivalents",
				Label:      "Kas dan setara kas",
				Section:    "current_assets",
				Level:      1,
				Values:     map[string]float64{"2024-12-31": 1500000, "2023-12-31": 1200000},
				IsSubtotal: false,
				Confidence: 0.95,
			},
			{
				Key:        "total_current_assets",
				Label:      "Jumlah aset lancar",
				Section:    "current_assets",
				Level:      0,
				Values:     map[string]float64{"2024-12-31": 5000000, "2023-12-31": 4000000},
				IsSubtotal: true,
				Confidence: 0.9,
			},
		},
	}
}

func emptyStatement() *domain.FinancialStatement {
	return &domain.FinancialStatement{
		Type:     domain.DocTypeIncomeStatement,
		Company:  "PT Empty Tbk",
		Periods:  []string{},
		Currency: "IDR",
		Unit:     "units",
		Language: "en",
		Items:    []domain.LineItem{},
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name    string
		format  Format
		wantErr bool
	}{
		{name: "json format", format: FormatJSON, wantErr: false},
		{name: "csv format", format: FormatCSV, wantErr: false},
		{name: "unsupported format", format: "xml", wantErr: true},
		{name: "empty format", format: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFormatter(tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if f == nil {
				t.Fatal("expected formatter, got nil")
			}
		})
	}
}

func TestJSONFormatterCompact(t *testing.T) {
	f, err := NewFormatter(FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, sampleStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "  ") {
		t.Error("compact JSON should not contain indentation")
	}

	var parsed domain.FinancialStatement
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.Company != "PT Test Tbk" {
		t.Errorf("company = %q, want %q", parsed.Company, "PT Test Tbk")
	}

	if len(parsed.Items) != 2 {
		t.Errorf("items count = %d, want 2", len(parsed.Items))
	}
}

func TestJSONFormatterPretty(t *testing.T) {
	f, err := NewFormatter(FormatJSON, WithPretty(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, sampleStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "  ") {
		t.Error("pretty JSON should contain indentation")
	}

	if !strings.Contains(output, "\n") {
		t.Error("pretty JSON should contain newlines")
	}

	var parsed domain.FinancialStatement
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestJSONFormatterEmpty(t *testing.T) {
	f, err := NewFormatter(FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, emptyStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed domain.FinancialStatement
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(parsed.Items) != 0 {
		t.Errorf("items count = %d, want 0", len(parsed.Items))
	}
}

func TestCSVFormatterHeaders(t *testing.T) {
	f, err := NewFormatter(FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, sampleStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}

	header := lines[0]
	wantHeader := "Key,Label,Section,Level,IsSubtotal,Confidence,2023-12-31,2024-12-31"
	if header != wantHeader {
		t.Errorf("header = %q, want %q", header, wantHeader)
	}
}

func TestCSVFormatterRows(t *testing.T) {
	f, err := NewFormatter(FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, sampleStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatal("expected at least 2 lines")
	}

	row1 := lines[1]
	if !strings.Contains(row1, "cash_and_equivalents") {
		t.Errorf("first row should contain key, got %q", row1)
	}

	if !strings.Contains(row1, "Kas dan setara kas") {
		t.Errorf("first row should contain label, got %q", row1)
	}

	if !strings.Contains(row1, "1200000") {
		t.Errorf("first row should contain 2023 value, got %q", row1)
	}

	if !strings.Contains(row1, "1500000") {
		t.Errorf("first row should contain 2024 value, got %q", row1)
	}
}

func TestCSVFormatterEmpty(t *testing.T) {
	f, err := NewFormatter(FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, emptyStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (header only), got %d", len(lines))
	}

	wantHeader := "Key,Label,Section,Level,IsSubtotal,Confidence"
	if lines[0] != wantHeader {
		t.Errorf("header = %q, want %q", lines[0], wantHeader)
	}
}

func TestCSVFormatterSpecialCharacters(t *testing.T) {
	stmt := &domain.FinancialStatement{
		Type:     domain.DocTypeBalanceSheet,
		Company:  "PT Test Tbk",
		Periods:  []string{"2024-12-31"},
		Currency: "IDR",
		Unit:     "millions",
		Language: "id",
		Items: []domain.LineItem{
			{
				Key:        "special_item",
				Label:      "Label with, comma and \"quotes\"",
				Section:    "section",
				Level:      1,
				Values:     map[string]float64{"2024-12-31": 100},
				IsSubtotal: false,
				Confidence: 1,
			},
		},
	}

	f, err := NewFormatter(FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"Label with, comma and ""quotes"""`) {
		t.Errorf("special characters not properly escaped in CSV: %q", output)
	}
}

func TestJSONFormatterStructure(t *testing.T) {
	f, err := NewFormatter(FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, sampleStatement()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	wantKeys := []string{"type", "company", "periods", "currency", "unit", "items", "language"}
	for _, key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}

	items, ok := raw["items"].([]any)
	if !ok {
		t.Fatal("items should be an array")
	}

	if len(items) != 2 {
		t.Errorf("items count = %d, want 2", len(items))
	}

	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("item should be an object")
	}

	itemKeys := []string{"key", "label", "section", "level", "values", "is_subtotal", "confidence"}
	for _, key := range itemKeys {
		if _, ok := item[key]; !ok {
			t.Errorf("missing key %q in line item JSON", key)
		}
	}
}

func TestCSVFormatterMissingValue(t *testing.T) {
	stmt := &domain.FinancialStatement{
		Type:     domain.DocTypeBalanceSheet,
		Company:  "PT Test Tbk",
		Periods:  []string{"2024-12-31", "2023-12-31"},
		Currency: "IDR",
		Unit:     "millions",
		Language: "id",
		Items: []domain.LineItem{
			{
				Key:        "partial_item",
				Label:      "Partial data",
				Section:    "assets",
				Level:      1,
				Values:     map[string]float64{"2024-12-31": 500},
				IsSubtotal: false,
				Confidence: 0.8,
			},
		},
	}

	f, err := NewFormatter(FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Format(&buf, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Row should have empty value for 2023-12-31 (sorted first)
	row := lines[1]
	if !strings.Contains(row, ",,") || !strings.HasPrefix(row, "partial_item") {
		// The 2023-12-31 column (sorted first) should be empty, followed by 500
		parts := strings.Split(row, ",")
		// header: Key,Label,Section,Level,IsSubtotal,Confidence,2023-12-31,2024-12-31
		// index:  0    1     2       3     4          5          6          7
		if len(parts) < 8 {
			t.Fatalf("expected at least 8 columns, got %d", len(parts))
		}

		if parts[6] != "" {
			t.Errorf("2023-12-31 value = %q, want empty", parts[6])
		}

		if parts[7] != "500" {
			t.Errorf("2024-12-31 value = %q, want %q", parts[7], "500")
		}
	}
}
