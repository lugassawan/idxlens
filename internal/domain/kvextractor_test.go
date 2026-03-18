package domain

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/pdf"
)

func TestKVExtractorExtractColonSeparated(t *testing.T) {
	extractor := NewKVExtractor()

	tests := []struct {
		name      string
		pages     []layout.LayoutPage
		wantPairs []KeyValuePair
	}{
		{
			name: "single colon-separated pair",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Company Name: PT Example Tbk", FontSize: 10},
					},
				},
			},
			wantPairs: []KeyValuePair{
				{Key: "Company Name", Value: "PT Example Tbk", Confidence: 0.9, PageNum: 1},
			},
		},
		{
			name: "multiple colon-separated pairs from multiple pages",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Company Name: PT Example Tbk", FontSize: 10},
						{Text: "Stock Code: EXAM", FontSize: 10},
					},
				},
				{
					Number: 2,
					Lines: []layout.TextLine{
						{Text: "Currency: IDR", FontSize: 10},
					},
				},
			},
			wantPairs: []KeyValuePair{
				{Key: "Company Name", Value: "PT Example Tbk", Confidence: 0.9, PageNum: 1},
				{Key: "Stock Code", Value: "EXAM", Confidence: 0.9, PageNum: 1},
				{Key: "Currency", Value: "IDR", Confidence: 0.9, PageNum: 2},
			},
		},
		{
			name: "colon in value splits at first colon only",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Time: 10:30", FontSize: 10},
					},
				},
			},
			wantPairs: []KeyValuePair{
				{Key: "Time", Value: "10:30", Confidence: 0.9, PageNum: 1},
			},
		},
		{
			name: "line with no colon is skipped",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "This line has no key value pair", FontSize: 10},
					},
				},
			},
			wantPairs: nil,
		},
		{
			name: "no pairs found returns empty slice",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines:  []layout.TextLine{},
				},
			},
			wantPairs: nil,
		},
		{
			name:      "empty pages returns empty slice",
			pages:     nil,
			wantPairs: nil,
		},
		{
			name: "colon at end of line with no value is skipped",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Section Title:", FontSize: 10},
					},
				},
			},
			wantPairs: nil,
		},
		{
			name: "table-like content with multiple spaces is skipped",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Revenue    1,000,000    2,000,000    3,000,000", FontSize: 10},
					},
				},
			},
			wantPairs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.Extract(tt.pages)
			assertKeyValuePairs(t, got, tt.wantPairs)
		})
	}
}

func TestKVExtractorExtractLabelAboveValue(t *testing.T) {
	extractor := NewKVExtractor()

	tests := []struct {
		name      string
		pages     []layout.LayoutPage
		wantPairs []KeyValuePair
	}{
		{
			name: "label above value with different font sizes",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Company Name", FontSize: 14, Bounds: pdf.Rect{X1: 50, Y1: 100, X2: 200, Y2: 114}},
						{Text: "PT Example Tbk", FontSize: 10, Bounds: pdf.Rect{X1: 50, Y1: 120, X2: 200, Y2: 130}},
					},
				},
			},
			wantPairs: []KeyValuePair{
				{Key: "Company Name", Value: "PT Example Tbk", Confidence: 0.7, PageNum: 1},
			},
		},
		{
			name: "same font size lines are not label-above-value",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Line One", FontSize: 10, Bounds: pdf.Rect{X1: 50, Y1: 100, X2: 200, Y2: 110}},
						{Text: "Line Two", FontSize: 10, Bounds: pdf.Rect{X1: 50, Y1: 120, X2: 200, Y2: 130}},
					},
				},
			},
			wantPairs: nil,
		},
		{
			name: "label above value not matched when lines are far apart horizontally",
			pages: []layout.LayoutPage{
				{
					Number: 1,
					Lines: []layout.TextLine{
						{Text: "Label", FontSize: 14, Bounds: pdf.Rect{X1: 50, Y1: 100, X2: 150, Y2: 114}},
						{Text: "Value", FontSize: 10, Bounds: pdf.Rect{X1: 400, Y1: 120, X2: 500, Y2: 130}},
					},
				},
			},
			wantPairs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.Extract(tt.pages)
			assertKeyValuePairs(t, got, tt.wantPairs)
		})
	}
}

func assertKeyValuePairs(t *testing.T, got, want []KeyValuePair) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("Extract() returned %d pairs, want %d", len(got), len(want))
	}

	for i, w := range want {
		if got[i].Key != w.Key {
			t.Errorf("pair[%d].Key = %q, want %q", i, got[i].Key, w.Key)
		}

		if got[i].Value != w.Value {
			t.Errorf("pair[%d].Value = %q, want %q", i, got[i].Value, w.Value)
		}

		if got[i].Confidence != w.Confidence {
			t.Errorf("pair[%d].Confidence = %v, want %v", i, got[i].Confidence, w.Confidence)
		}

		if got[i].PageNum != w.PageNum {
			t.Errorf("pair[%d].PageNum = %d, want %d", i, got[i].PageNum, w.PageNum)
		}
	}
}
