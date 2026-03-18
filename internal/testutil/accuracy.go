package testutil

import (
	"fmt"

	"github.com/lugassawan/idxlens/internal/domain"
)

// AccuracyError describes a single mismatch between extracted and ground truth data.
type AccuracyError struct {
	Type     string // "missing", "extra", "wrong_value", "wrong_key"
	Expected string
	Got      string
	LineItem string
}

// AccuracyResult holds accuracy metrics for a single file comparison.
type AccuracyResult struct {
	File          string
	Precision     float64
	Recall        float64
	F1Score       float64
	ValueAccuracy float64
	Errors        []AccuracyError
}

// AggregateReport holds accuracy metrics aggregated across multiple files.
type AggregateReport struct {
	Results           []AccuracyResult
	MeanPrecision     float64
	MeanRecall        float64
	MeanF1Score       float64
	MeanValueAccuracy float64
	TotalFiles        int
	PassedFiles       int
}

// MeasureAccuracy compares extracted output against ground truth, computing
// precision, recall, F1 score, and value accuracy for line items matched by key.
func MeasureAccuracy(extracted, groundTruth *domain.FinancialStatement) AccuracyResult {
	truthMap := indexLineItems(groundTruth.Items)
	extractedMap := indexLineItems(extracted.Items)

	var truePositives int
	var correctValues int
	var totalValues int
	var errors []AccuracyError

	// Check each ground truth item against extracted items.
	for key, truthItem := range truthMap {
		extItem, found := extractedMap[key]
		if !found {
			errors = append(errors, AccuracyError{
				Type:     "missing",
				Expected: truthItem.Label,
				LineItem: key,
			})

			continue
		}

		truePositives++

		valueErrors, matched, total := compareValues(key, extItem.Values, truthItem.Values)
		correctValues += matched
		totalValues += total
		errors = append(errors, valueErrors...)
	}

	// Check for extra items not in ground truth.
	for key, extItem := range extractedMap {
		if _, found := truthMap[key]; !found {
			errors = append(errors, AccuracyError{
				Type:     "extra",
				Got:      extItem.Label,
				LineItem: key,
			})
		}
	}

	precision := safeDivide(float64(truePositives), float64(len(extractedMap)))
	recall := safeDivide(float64(truePositives), float64(len(truthMap)))
	f1 := computeF1(precision, recall)
	valueAccuracy := safeDivide(float64(correctValues), float64(totalValues))

	return AccuracyResult{
		Precision:     precision,
		Recall:        recall,
		F1Score:       f1,
		ValueAccuracy: valueAccuracy,
		Errors:        errors,
	}
}

// Aggregate computes mean metrics across multiple accuracy results.
// A file is considered passed when its F1 score is 1.0.
func Aggregate(results []AccuracyResult) AggregateReport {
	if len(results) == 0 {
		return AggregateReport{}
	}

	var sumPrecision, sumRecall, sumF1, sumValueAccuracy float64
	var passed int

	for _, r := range results {
		sumPrecision += r.Precision
		sumRecall += r.Recall
		sumF1 += r.F1Score
		sumValueAccuracy += r.ValueAccuracy

		if r.F1Score == 1.0 {
			passed++
		}
	}

	n := float64(len(results))

	return AggregateReport{
		Results:           results,
		MeanPrecision:     sumPrecision / n,
		MeanRecall:        sumRecall / n,
		MeanF1Score:       sumF1 / n,
		MeanValueAccuracy: sumValueAccuracy / n,
		TotalFiles:        len(results),
		PassedFiles:       passed,
	}
}

func indexLineItems(items []domain.LineItem) map[string]domain.LineItem {
	m := make(map[string]domain.LineItem, len(items))
	for _, item := range items {
		m[item.Key] = item
	}

	return m
}

func compareValues(
	lineItem string,
	extracted, truth map[string]float64,
) ([]AccuracyError, int, int) {
	var errors []AccuracyError
	var matched int

	total := len(truth)

	for period, truthVal := range truth {
		extVal, found := extracted[period]
		if !found {
			errors = append(errors, AccuracyError{
				Type:     "wrong_value",
				Expected: formatFloat(truthVal),
				Got:      "",
				LineItem: lineItem,
			})

			continue
		}

		if extVal == truthVal {
			matched++
		} else {
			errors = append(errors, AccuracyError{
				Type:     "wrong_value",
				Expected: formatFloat(truthVal),
				Got:      formatFloat(extVal),
				LineItem: lineItem,
			})
		}
	}

	return errors, matched, total
}

func safeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func computeF1(precision, recall float64) float64 {
	sum := precision + recall
	if sum == 0 {
		return 0
	}

	return 2 * precision * recall / sum
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%g", v)
}
