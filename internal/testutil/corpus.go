package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// CorpusManifest holds the test corpus definition.
type CorpusManifest struct {
	Version int           `json:"version"`
	Entries []CorpusEntry `json:"entries"`
}

// CorpusEntry describes a single test PDF and its expected properties.
type CorpusEntry struct {
	File           string               `json:"file"`
	Classification CorpusClassification `json:"classification,omitzero"`
	PageCount      int                  `json:"page_count"`
	Description    string               `json:"description"`
}

// CorpusClassification holds expected classification for a corpus entry.
type CorpusClassification struct {
	Type     string `json:"type"`
	Language string `json:"language"`
}

const corpusDir = "corpus"

// LoadCorpus reads the corpus manifest from testdata/corpus/.
func LoadCorpus() (*CorpusManifest, error) {
	path := filepath.Join(projectRoot(), "testdata", corpusDir, "manifest.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read corpus manifest: %w", err)
	}

	var m CorpusManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse corpus manifest: %w", err)
	}

	return &m, nil
}

// PDFPath returns the full path to a corpus PDF file.
func (e *CorpusEntry) PDFPath() string {
	return filepath.Join(projectRoot(), "testdata", corpusDir, e.File)
}

// projectRoot finds the project root by walking up from the current file.
func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// internal/testutil/corpus.go -> go up 2 levels
	return filepath.Join(filepath.Dir(filename), "..", "..")
}
