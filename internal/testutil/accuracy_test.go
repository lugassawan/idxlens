package testutil

import (
	"math"
	"testing"

	"github.com/lugassawan/idxlens/internal/domain"
)

func TestMeasureAccuracy(t *testing.T) {
	tests := []struct {
		name              string
		extracted         *domain.FinancialStatement
		groundTruth       *domain.FinancialStatement
		wantPrecision     float64
		wantRecall        float64
		wantF1            float64
		wantValueAccuracy float64
		wantErrorCount    int
	}{
		{
			name:      "perfect match",
			extracted: makeStatement("revenue", 100.0, "expenses", 50.0),
			groundTruth: makeStatement(
				"revenue", 100.0, "expenses", 50.0,
			),
			wantPrecision:     1.0,
			wantRecall:        1.0,
			wantF1:            1.0,
			wantValueAccuracy: 1.0,
			wantErrorCount:    0,
		},
		{
			name:              "no matches with disjoint keys",
			extracted:         makeStatement("assets", 200.0),
			groundTruth:       makeStatement("liabilities", 300.0),
			wantPrecision:     0.0,
			wantRecall:        0.0,
			wantF1:            0.0,
			wantValueAccuracy: 0.0,
			wantErrorCount:    2, // 1 missing + 1 extra
		},
		{
			name: "partial match with one hit one miss",
			extracted: makeStatement(
				"revenue", 100.0, "assets", 200.0,
			),
			groundTruth: makeStatement(
				"revenue", 100.0, "expenses", 50.0,
			),
			wantPrecision:     0.5,
			wantRecall:        0.5,
			wantF1:            0.5,
			wantValueAccuracy: 1.0,
			wantErrorCount:    2, // 1 missing + 1 extra
		},
		{
			name:      "value mismatch",
			extracted: makeStatement("revenue", 999.0),
			groundTruth: makeStatement(
				"revenue", 100.0,
			),
			wantPrecision:     1.0,
			wantRecall:        1.0,
			wantF1:            1.0,
			wantValueAccuracy: 0.0,
			wantErrorCount:    1, // 1 wrong_value
		},
		{
			name:              "empty statements",
			extracted:         makeStatement(),
			groundTruth:       makeStatement(),
			wantPrecision:     0.0,
			wantRecall:        0.0,
			wantF1:            0.0,
			wantValueAccuracy: 0.0,
			wantErrorCount:    0,
		},
		{
			name:      "extra items only",
			extracted: makeStatement("revenue", 100.0, "expenses", 50.0),
			groundTruth: makeStatement(
				"revenue", 100.0,
			),
			wantPrecision:     0.5,
			wantRecall:        1.0,
			wantF1:            2.0 / 3.0,
			wantValueAccuracy: 1.0,
			wantErrorCount:    1, // 1 extra
		},
		{
			name:      "missing items only",
			extracted: makeStatement("revenue", 100.0),
			groundTruth: makeStatement(
				"revenue", 100.0, "expenses", 50.0,
			),
			wantPrecision:     1.0,
			wantRecall:        0.5,
			wantF1:            2.0 / 3.0,
			wantValueAccuracy: 1.0,
			wantErrorCount:    1, // 1 missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MeasureAccuracy(tt.extracted, tt.groundTruth)

			assertFloat(t, "Precision", tt.wantPrecision, result.Precision)
			assertFloat(t, "Recall", tt.wantRecall, result.Recall)
			assertFloat(t, "F1Score", tt.wantF1, result.F1Score)
			assertFloat(t, "ValueAccuracy", tt.wantValueAccuracy, result.ValueAccuracy)

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf(
					"error count: got %d, want %d; errors: %v",
					len(result.Errors), tt.wantErrorCount, result.Errors,
				)
			}
		})
	}
}

func TestAggregate(t *testing.T) {
	tests := []struct {
		name              string
		results           []AccuracyResult
		wantMeanPrecision float64
		wantMeanF1        float64
		wantTotalFiles    int
		wantPassedFiles   int
	}{
		{
			name:              "empty results",
			results:           nil,
			wantMeanPrecision: 0.0,
			wantMeanF1:        0.0,
			wantTotalFiles:    0,
			wantPassedFiles:   0,
		},
		{
			name: "single perfect result",
			results: []AccuracyResult{
				{
					File:          "a.pdf",
					Precision:     1.0,
					Recall:        1.0,
					F1Score:       1.0,
					ValueAccuracy: 1.0,
				},
			},
			wantMeanPrecision: 1.0,
			wantMeanF1:        1.0,
			wantTotalFiles:    1,
			wantPassedFiles:   1,
		},
		{
			name: "mixed results",
			results: []AccuracyResult{
				{
					File:          "a.pdf",
					Precision:     1.0,
					Recall:        1.0,
					F1Score:       1.0,
					ValueAccuracy: 1.0,
				},
				{
					File:          "b.pdf",
					Precision:     0.5,
					Recall:        0.5,
					F1Score:       0.5,
					ValueAccuracy: 0.8,
				},
			},
			wantMeanPrecision: 0.75,
			wantMeanF1:        0.75,
			wantTotalFiles:    2,
			wantPassedFiles:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Aggregate(tt.results)

			assertFloat(t, "MeanPrecision", tt.wantMeanPrecision, report.MeanPrecision)
			assertFloat(t, "MeanF1Score", tt.wantMeanF1, report.MeanF1Score)

			if report.TotalFiles != tt.wantTotalFiles {
				t.Errorf("TotalFiles: got %d, want %d", report.TotalFiles, tt.wantTotalFiles)
			}

			if report.PassedFiles != tt.wantPassedFiles {
				t.Errorf("PassedFiles: got %d, want %d", report.PassedFiles, tt.wantPassedFiles)
			}
		})
	}
}

// makeStatement creates a FinancialStatement from key-value pairs.
// Arguments alternate between string keys and float64 values.
func makeStatement(keyValues ...any) *domain.FinancialStatement {
	var items []domain.LineItem
	for i := 0; i+1 < len(keyValues); i += 2 {
		key, _ := keyValues[i].(string)
		val, _ := keyValues[i+1].(float64)

		items = append(items, domain.LineItem{
			Key:   key,
			Label: key,
			Values: map[string]float64{
				"2024": val,
			},
		})
	}

	return &domain.FinancialStatement{Items: items}
}

func assertFloat(t *testing.T, name string, want, got float64) {
	t.Helper()

	if math.Abs(want-got) > 1e-9 {
		t.Errorf("%s: got %f, want %f", name, got, want)
	}
}
