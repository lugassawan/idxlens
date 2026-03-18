package pdf

import (
	"bytes"
	"strings"
	"testing"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// buildTestPDF creates a minimal valid PDF with the given number of pages.
// Each page uses an A4 media box (595x842). If withText is true,
// a simple text content stream is added to each page.
func buildTestPDF(t *testing.T, pageCount int, withText bool) *bytes.Reader {
	t.Helper()

	xRefTable, err := pdfcpu.CreateXRefTableWithRootDict()
	if err != nil {
		t.Fatalf("create xref table: %v", err)
	}

	mediaBox := types.NewRectangle(0, 0, 595, 842)

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("get catalog: %v", err)
	}

	pagesDict := types.NewDict()
	pagesDict.InsertName("Type", "Pages")
	pagesDict.InsertInt("Count", pageCount)

	pagesRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		t.Fatalf("create pages ref: %v", err)
	}

	rootDict.Insert("Pages", *pagesRef)

	var kids types.Array
	for i := range pageCount {
		pageDict := types.NewDict()
		pageDict.InsertName("Type", "Page")
		pageDict.Insert("Parent", *pagesRef)
		pageDict.Insert("MediaBox", mediaBox.Array())

		if withText {
			addTextContent(t, xRefTable, pageDict, i+1)
		}

		pageRef, err := xRefTable.IndRefForNewObject(pageDict)
		if err != nil {
			t.Fatalf("create page ref: %v", err)
		}

		kids = append(kids, *pageRef)
	}

	pagesDict.Insert("Kids", kids)

	xRefTable.PageCount = pageCount

	conf := model.NewDefaultConfiguration()
	ctx := pdfcpu.CreateContext(xRefTable, conf)

	var buf bytes.Buffer
	if err := pdfcpuapi.WriteContext(ctx, &buf); err != nil {
		t.Fatalf("write pdf: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}

// addTextContent adds a simple content stream with text to a page dict.
func addTextContent(t *testing.T, xRefTable *model.XRefTable, pageDict types.Dict, pageNum int) {
	t.Helper()

	// Build a basic font resource.
	fontDict := types.NewDict()
	fontDict.InsertName("Type", "Font")
	fontDict.InsertName("Subtype", "Type1")
	fontDict.InsertName("BaseFont", "Helvetica")

	fontRef, err := xRefTable.IndRefForNewObject(fontDict)
	if err != nil {
		t.Fatalf("create font ref: %v", err)
	}

	fontMap := types.NewDict()
	fontMap.Insert("F1", *fontRef)

	resDict := types.NewDict()
	resDict.Insert("Font", fontMap)
	pageDict.Insert("Resources", resDict)

	// Build content stream: "BT /F1 12 Tf 72 700 Td (Hello Page N) Tj ET"
	var content strings.Builder
	content.WriteString("BT\n")
	content.WriteString("/F1 12 Tf\n")
	content.WriteString("72 700 Td\n")

	content.WriteString("(Hello Page ")
	content.WriteString(strings.Repeat("I", pageNum)) // simple page indicator
	content.WriteString(") Tj\n")

	content.WriteString("ET\n")

	sd, err := xRefTable.NewStreamDictForBuf([]byte(content.String()))
	if err != nil {
		t.Fatalf("create stream dict: %v", err)
	}

	if err := sd.Encode(); err != nil {
		t.Fatalf("encode stream: %v", err)
	}

	streamRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		t.Fatalf("create stream ref: %v", err)
	}

	pageDict.Insert("Contents", *streamRef)
}

func TestNewReader(t *testing.T) {
	r := NewReader()
	if r == nil {
		t.Fatal("NewReader returned nil")
	}
}

func TestPageCount(t *testing.T) {
	tests := []struct {
		name  string
		pages int
	}{
		{name: "single page", pages: 1},
		{name: "three pages", pages: 3},
		{name: "five pages", pages: 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rs := buildTestPDF(t, tc.pages, false)
			r := NewReader()

			if err := r.Open(rs); err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer r.Close()

			got := r.PageCount()
			if got != tc.pages {
				t.Errorf("PageCount() = %d, want %d", got, tc.pages)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	rs := buildTestPDF(t, 2, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	meta, err := r.Metadata()
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}

	if meta.Pages != 2 {
		t.Errorf("Metadata.Pages = %d, want 2", meta.Pages)
	}

	// Producer is set by pdfcpu when writing.
	if meta.Producer == "" {
		t.Log("Metadata.Producer is empty (may vary by pdfcpu version)")
	}
}

func TestMetadataBeforeOpen(t *testing.T) {
	r := NewReader()

	_, err := r.Metadata()
	if err == nil {
		t.Fatal("expected error from Metadata before Open, got nil")
	}
}

func TestTextExtraction(t *testing.T) {
	rs := buildTestPDF(t, 1, true)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	page, err := r.Page(1)
	if err != nil {
		t.Fatalf("Page(1): %v", err)
	}

	if page.Number != 1 {
		t.Errorf("Page.Number = %d, want 1", page.Number)
	}

	// A4 dimensions: 595x842.
	if page.Size.Width != 595 || page.Size.Height != 842 {
		t.Errorf("Page.Size = {%v, %v}, want {595, 842}", page.Size.Width, page.Size.Height)
	}

	if len(page.Elements) == 0 {
		t.Fatal("expected at least one text element, got 0")
	}

	found := false
	for _, el := range page.Elements {
		if strings.Contains(el.Text, "Hello Page") {
			found = true

			if el.FontName != "F1" {
				t.Errorf("TextElement.FontName = %q, want %q", el.FontName, "F1")
			}

			if el.FontSize != 12 {
				t.Errorf("TextElement.FontSize = %v, want 12", el.FontSize)
			}

			if el.Bounds.X1 != 72 {
				t.Errorf("TextElement.Bounds.X1 = %v, want 72", el.Bounds.X1)
			}

			if el.Bounds.Y1 != 700 {
				t.Errorf("TextElement.Bounds.Y1 = %v, want 700", el.Bounds.Y1)
			}
		}
	}

	if !found {
		t.Errorf("expected text element containing %q, elements: %v", "Hello Page", page.Elements)
	}
}

func TestEmptyPage(t *testing.T) {
	rs := buildTestPDF(t, 1, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	page, err := r.Page(1)
	if err != nil {
		t.Fatalf("Page(1): %v", err)
	}

	if len(page.Elements) != 0 {
		t.Errorf("expected 0 elements on empty page, got %d", len(page.Elements))
	}
}

func TestPageOutOfRange(t *testing.T) {
	rs := buildTestPDF(t, 2, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	tests := []struct {
		name   string
		number int
	}{
		{name: "zero", number: 0},
		{name: "negative", number: -1},
		{name: "beyond count", number: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := r.Page(tc.number)
			if err == nil {
				t.Errorf("Page(%d): expected error, got nil", tc.number)
			}
		})
	}
}

func TestInvalidPDF(t *testing.T) {
	r := NewReader()

	rs := bytes.NewReader([]byte("this is not a PDF"))
	err := r.Open(rs)

	if err == nil {
		t.Fatal("expected error on invalid PDF, got nil")
	}
}

func TestPageBeforeOpen(t *testing.T) {
	r := NewReader()

	_, err := r.Page(1)
	if err == nil {
		t.Fatal("expected error from Page before Open, got nil")
	}
}

func TestClose(t *testing.T) {
	rs := buildTestPDF(t, 1, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, PageCount should return 0.
	if got := r.PageCount(); got != 0 {
		t.Errorf("PageCount after Close = %d, want 0", got)
	}
}
