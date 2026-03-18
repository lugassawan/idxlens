package domain

import "time"

// BatchResult holds the outcome of processing a single PDF file
// within a batch operation.
type BatchResult struct {
	InputPath  string        `json:"input_path"`
	OutputPath string        `json:"output_path,omitempty"`
	DocType    DocType       `json:"doc_type"`
	Success    bool          `json:"success"`
	Error      error         `json:"-"`
	ErrorMsg   string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
}

// BatchSummary aggregates results from a batch processing run.
type BatchSummary struct {
	TotalFiles int           `json:"total_files"`
	Succeeded  int           `json:"succeeded"`
	Failed     int           `json:"failed"`
	Duration   time.Duration `json:"duration"`
	Results    []BatchResult `json:"results"`
}

// BatchProcessor provides bounded-concurrency fan-out processing
// for multiple PDF files.
type BatchProcessor struct {
	Workers   int
	OutputDir string
	DocType   DocType
	Extractor FileExtractor
}

// FileExtractor defines the contract for extracting a financial statement
// from a single PDF file. Implementations live in the CLI layer.
type FileExtractor interface {
	Extract(pdfPath string, docType DocType) (*FinancialStatement, error)
}

// Process runs extraction on all provided file paths using bounded concurrency.
func (bp *BatchProcessor) Process(paths []string) BatchSummary {
	start := time.Now()

	results := make([]BatchResult, len(paths))

	sem := make(chan struct{}, bp.Workers)
	done := make(chan struct{})

	for i, path := range paths {
		go func(idx int, filePath string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = bp.processFile(filePath)
			done <- struct{}{}
		}(i, path)
	}

	for range paths {
		<-done
	}

	summary := BatchSummary{
		TotalFiles: len(paths),
		Duration:   time.Since(start),
		Results:    results,
	}

	for _, r := range results {
		if r.Success {
			summary.Succeeded++
		} else {
			summary.Failed++
		}
	}

	return summary
}

func (bp *BatchProcessor) processFile(pdfPath string) BatchResult {
	start := time.Now()

	stmt, err := bp.Extractor.Extract(pdfPath, bp.DocType)
	if err != nil {
		return BatchResult{
			InputPath: pdfPath,
			Success:   false,
			Error:     err,
			ErrorMsg:  err.Error(),
			Duration:  time.Since(start),
		}
	}

	return BatchResult{
		InputPath: pdfPath,
		DocType:   stmt.Type,
		Success:   true,
		Duration:  time.Since(start),
	}
}
