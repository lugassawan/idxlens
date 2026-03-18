package domain

import "github.com/lugassawan/idxlens/internal/layout"

// Classifier determines the type of an IDX financial report
// from its layout-analyzed pages.
type Classifier interface {
	// Classify analyzes the first few pages and returns a classification.
	Classify(pages []layout.LayoutPage) (Classification, error)
}
