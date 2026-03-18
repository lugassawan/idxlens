package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/lugassawan/idxlens/internal/domain"
	"github.com/lugassawan/idxlens/internal/layout"
	"github.com/lugassawan/idxlens/internal/output"
	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/lugassawan/idxlens/internal/table"
)

const maxClassifyPages = 3

//export ExtractJSON
func ExtractJSON(pdfPath *C.char, docType *C.char) *C.char {
	result, err := extractJSON(C.GoString(pdfPath), C.GoString(docType))
	if err != nil {
		return errorJSON(err)
	}

	return C.CString(result)
}

//export Classify
func Classify(pdfPath *C.char) *C.char {
	result, err := classify(C.GoString(pdfPath))
	if err != nil {
		return errorJSON(err)
	}

	return C.CString(result)
}

//export FreeString
func FreeString(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func main() {}

func extractJSON(pdfPath, docTypeStr string) (string, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := pdf.NewReader()
	if err := reader.Open(f); err != nil {
		return "", fmt.Errorf("parse pdf: %w", err)
	}
	defer reader.Close()

	pages, err := analyzeAllPages(reader)
	if err != nil {
		return "", err
	}

	resolvedType, err := resolveDocType(domain.DocType(docTypeStr), pages)
	if err != nil {
		return "", err
	}

	tables, err := detectTables(pages)
	if err != nil {
		return "", err
	}

	mapper := domain.NewMapper()

	stmt, err := mapper.Map(resolvedType, tables)
	if err != nil {
		return formatJSON(emptyStatement(resolvedType))
	}

	return formatJSON(stmt)
}

func classify(pdfPath string) (string, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := pdf.NewReader()
	if err := reader.Open(f); err != nil {
		return "", fmt.Errorf("parse pdf: %w", err)
	}
	defer reader.Close()

	pages, err := analyzePages(reader, maxClassifyPages)
	if err != nil {
		return "", err
	}

	classifier := domain.NewHeuristicClassifier()

	classification, err := classifier.Classify(pages)
	if err != nil {
		return "", fmt.Errorf("classify document: %w", err)
	}

	data, err := json.Marshal(classification)
	if err != nil {
		return "", fmt.Errorf("marshal classification: %w", err)
	}

	return string(data), nil
}

func analyzeAllPages(reader pdf.Reader) ([]layout.LayoutPage, error) {
	return analyzePages(reader, reader.PageCount())
}

func analyzePages(reader pdf.Reader, maxPages int) ([]layout.LayoutPage, error) {
	pageCount := min(reader.PageCount(), maxPages)
	analyzer := layout.NewAnalyzer()
	pages := make([]layout.LayoutPage, 0, pageCount)

	for i := 1; i <= pageCount; i++ {
		page, err := reader.Page(i)
		if err != nil {
			return nil, fmt.Errorf("read page %d: %w", i, err)
		}

		layoutPage, err := analyzer.Analyze(page)
		if err != nil {
			return nil, fmt.Errorf("analyze page %d: %w", i, err)
		}

		pages = append(pages, layoutPage)
	}

	return pages, nil
}

func resolveDocType(docType domain.DocType, pages []layout.LayoutPage) (domain.DocType, error) {
	if docType != "" {
		return docType, nil
	}

	classifier := domain.NewHeuristicClassifier()

	classification, err := classifier.Classify(pages)
	if err != nil {
		return "", fmt.Errorf("classify document: %w", err)
	}

	return classification.Type, nil
}

func detectTables(pages []layout.LayoutPage) ([]table.Table, error) {
	detector := table.NewDetector()

	var tables []table.Table

	for _, page := range pages {
		pageTables, err := detector.Detect(page)
		if err != nil {
			return nil, fmt.Errorf("detect tables on page %d: %w", page.Number, err)
		}

		tables = append(tables, pageTables...)
	}

	return tables, nil
}

func emptyStatement(docType domain.DocType) *domain.FinancialStatement {
	return &domain.FinancialStatement{
		Type:  docType,
		Items: []domain.LineItem{},
	}
}

func formatJSON(stmt *domain.FinancialStatement) (string, error) {
	formatter, err := output.NewFormatter(output.FormatJSON, output.WithPretty(true))
	if err != nil {
		return "", fmt.Errorf("create formatter: %w", err)
	}

	var buf strings.Builder

	if err := formatter.Format(&buf, stmt); err != nil {
		return "", fmt.Errorf("format output: %w", err)
	}

	return buf.String(), nil
}

func errorJSON(err error) *C.char {
	result := map[string]string{"error": err.Error()}

	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return C.CString(`{"error":"internal error"}`)
	}

	return C.CString(string(data))
}
