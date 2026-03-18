package layout

import "github.com/lugassawan/idxlens/internal/pdf"

// TextLine represents a line of text composed of one or more text elements.
type TextLine struct {
	Text     string
	Elements []pdf.TextElement
	Bounds   pdf.Rect
	FontName string
	FontSize float64
}

// Region represents a group of text lines that form a logical region on a page.
type Region struct {
	Lines  []TextLine
	Bounds pdf.Rect
}

// LayoutPage represents a PDF page after layout analysis, with text organized
// into lines and regions.
type LayoutPage struct {
	Number  int
	Size    pdf.PageSize
	Lines   []TextLine
	Regions []Region
}
