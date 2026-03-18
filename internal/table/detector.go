package table

import "github.com/lugassawan/idxlens/internal/layout"

// NewDetector creates a Detector that combines line-based table region
// detection with spatial grid assembly.
func NewDetector() Detector {
	return &detector{
		lineDetector: newLineDetector(),
		gridBuilder:  newGridBuilder(),
	}
}

type detector struct {
	lineDetector *lineDetector
	gridBuilder  *gridBuilder
}

// Detect finds all tables in a layout page by first identifying table regions
// and column boundaries via line detection, then assembling cells into a
// structured grid.
func (d *detector) Detect(page layout.LayoutPage) ([]Table, error) {
	if len(page.Lines) == 0 {
		return nil, nil
	}

	groups := d.lineDetector.findLineGroups(page.Lines)

	var tables []Table
	for _, group := range groups {
		columns := d.lineDetector.detectColumns(group)
		if len(columns) < d.lineDetector.minColumns {
			continue
		}

		tbl := d.gridBuilder.Build(group, columns, page.Number)
		if len(tbl.Rows) < d.lineDetector.minRows {
			continue
		}

		tables = append(tables, tbl)
	}

	return tables, nil
}
