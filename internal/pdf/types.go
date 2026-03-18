package pdf

// Rect represents a bounding rectangle in PDF coordinates.
type Rect struct {
	X1, Y1, X2, Y2 float64
}

// TextElement represents a single text element extracted from a PDF page.
type TextElement struct {
	Text     string
	FontName string
	FontSize float64
	Bounds   Rect
}

// PageSize represents the dimensions of a PDF page.
type PageSize struct {
	Width, Height float64
}

// Page represents a single parsed PDF page with its text elements.
type Page struct {
	Number   int
	Size     PageSize
	Elements []TextElement
}

// Metadata holds document-level metadata extracted from a PDF.
type Metadata struct {
	Title    string
	Author   string
	Creator  string
	Producer string
	Pages    int
}
