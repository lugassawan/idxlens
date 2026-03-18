package domain

import (
	"errors"
	"testing"
)

type stubExtractor struct {
	results map[string]*FinancialStatement
	errs    map[string]error
}

func (s *stubExtractor) Extract(pdfPath string, docType DocType) (*FinancialStatement, error) {
	if err, ok := s.errs[pdfPath]; ok {
		return nil, err
	}

	if stmt, ok := s.results[pdfPath]; ok {
		return stmt, nil
	}

	return &FinancialStatement{Type: docType}, nil
}

func TestBatchProcessorAllSuccess(t *testing.T) {
	extractor := &stubExtractor{
		results: map[string]*FinancialStatement{
			"a.pdf": {Type: DocTypeBalanceSheet},
			"b.pdf": {Type: DocTypeIncomeStatement},
		},
	}

	processor := &BatchProcessor{
		Workers:   2,
		Extractor: extractor,
	}

	summary := processor.Process([]string{"a.pdf", "b.pdf"})

	if summary.TotalFiles != 2 {
		t.Errorf("TotalFiles = %d, want 2", summary.TotalFiles)
	}

	if summary.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", summary.Succeeded)
	}

	if summary.Failed != 0 {
		t.Errorf("Failed = %d, want 0", summary.Failed)
	}

	if len(summary.Results) != 2 {
		t.Fatalf("Results length = %d, want 2", len(summary.Results))
	}

	for _, r := range summary.Results {
		if !r.Success {
			t.Errorf("result for %s should be successful", r.InputPath)
		}

		if r.Duration <= 0 {
			t.Errorf("result for %s should have positive duration", r.InputPath)
		}
	}
}

func TestBatchProcessorPartialFailure(t *testing.T) {
	extractor := &stubExtractor{
		results: map[string]*FinancialStatement{
			"a.pdf": {Type: DocTypeBalanceSheet},
		},
		errs: map[string]error{
			"b.pdf": errors.New("parse error"),
		},
	}

	processor := &BatchProcessor{
		Workers:   2,
		Extractor: extractor,
	}

	summary := processor.Process([]string{"a.pdf", "b.pdf"})

	if summary.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", summary.Succeeded)
	}

	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
}

func TestBatchProcessorEmptyPaths(t *testing.T) {
	processor := &BatchProcessor{
		Workers:   2,
		Extractor: &stubExtractor{},
	}

	summary := processor.Process([]string{})

	if summary.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", summary.TotalFiles)
	}

	if summary.Succeeded != 0 {
		t.Errorf("Succeeded = %d, want 0", summary.Succeeded)
	}

	if summary.Duration <= 0 {
		t.Errorf("Duration should be positive, got %v", summary.Duration)
	}
}

func TestBatchProcessorSingleWorker(t *testing.T) {
	extractor := &stubExtractor{
		results: map[string]*FinancialStatement{
			"a.pdf": {Type: DocTypeBalanceSheet},
			"b.pdf": {Type: DocTypeIncomeStatement},
			"c.pdf": {Type: DocTypeCashFlow},
		},
	}

	processor := &BatchProcessor{
		Workers:   1,
		Extractor: extractor,
	}

	summary := processor.Process([]string{"a.pdf", "b.pdf", "c.pdf"})

	if summary.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", summary.TotalFiles)
	}

	if summary.Succeeded != 3 {
		t.Errorf("Succeeded = %d, want 3", summary.Succeeded)
	}
}

func TestBatchResultErrorMsg(t *testing.T) {
	extractor := &stubExtractor{
		errs: map[string]error{
			"bad.pdf": errors.New("corrupted file"),
		},
	}

	processor := &BatchProcessor{
		Workers:   1,
		Extractor: extractor,
	}

	summary := processor.Process([]string{"bad.pdf"})

	if len(summary.Results) != 1 {
		t.Fatalf("Results length = %d, want 1", len(summary.Results))
	}

	r := summary.Results[0]
	if r.ErrorMsg != "corrupted file" {
		t.Errorf("ErrorMsg = %q, want %q", r.ErrorMsg, "corrupted file")
	}

	if r.Success {
		t.Error("result should not be successful")
	}
}
