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

func TestCloseAfterClose(t *testing.T) {
	rs := buildTestPDF(t, 1, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Double close should not panic or error.
	if err := r.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestMetadataAfterClose(t *testing.T) {
	rs := buildTestPDF(t, 1, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := r.Metadata()
	if err == nil {
		t.Fatal("expected error from Metadata after Close, got nil")
	}
}

func TestPageAfterClose(t *testing.T) {
	rs := buildTestPDF(t, 1, false)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := r.Page(1)
	if err == nil {
		t.Fatal("expected error from Page after Close, got nil")
	}
}

func TestPageCountBeforeOpen(t *testing.T) {
	r := NewReader()

	if got := r.PageCount(); got != 0 {
		t.Errorf("PageCount before Open = %d, want 0", got)
	}
}

func TestParseContentStream(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLen  int
		wantText string
	}{
		{
			name:    "empty content",
			content: "",
			wantLen: 0,
		},
		{
			name:    "no BT/ET blocks",
			content: "some random content",
			wantLen: 0,
		},
		{
			name:     "simple Tj operator",
			content:  "BT /F1 12 Tf 72 700 Td (Hello) Tj ET",
			wantLen:  1,
			wantText: "Hello",
		},
		{
			name:     "TJ array operator",
			content:  "BT /F1 12 Tf 72 700 Td [(Hello) -10 (World)] TJ ET",
			wantLen:  1,
			wantText: "HelloWorld",
		},
		{
			name:     "Tm operator sets position",
			content:  "BT /F1 12 Tf 1 0 0 1 100 500 Tm (Positioned) Tj ET",
			wantLen:  1,
			wantText: "Positioned",
		},
		{
			name:     "quote operator moves to next line",
			content:  "BT /F1 12 Tf 72 700 Td (Line1) ' ET",
			wantLen:  1,
			wantText: "Line1",
		},
		{
			name:    "hex string in TJ array",
			content: "BT /F1 12 Tf 72 700 Td [<48656C6C6F>] TJ ET",
			wantLen: 1,
		},
		{
			name:     "comment in content stream",
			content:  "BT\n% this is a comment\n/F1 12 Tf 72 700 Td (Test) Tj ET",
			wantLen:  1,
			wantText: "Test",
		},
		{
			name:     "TJ with large kerning inserts space",
			content:  "BT /F1 12 Tf 72 700 Td [(Kas) -500 (Dana)] TJ ET",
			wantLen:  1,
			wantText: "Kas Dana",
		},
		{
			name:     "TJ with small kerning preserves number",
			content:  "BT /F1 12 Tf 72 700 Td [(202) -20 (5)] TJ ET",
			wantLen:  1,
			wantText: "2025",
		},
		{
			name:     "TJ number with comma kerning",
			content:  "BT /F1 12 Tf 72 700 Td [(868,) -10 (686,) -10 (210)] TJ ET",
			wantLen:  1,
			wantText: "868,686,210",
		},
		{
			name:     "T* operator moves to next line",
			content:  "BT /F1 12 Tf 72 700 Td (Line1) Tj T* (Line2) Tj ET",
			wantLen:  2,
			wantText: "Line1",
		},
		{
			name:     "TL sets text leading for T*",
			content:  "BT /F1 12 Tf 72 700 Td 14 TL (Line1) Tj T* (Line2) Tj ET",
			wantLen:  2,
			wantText: "Line1",
		},
		{
			name:     "Tw sets word spacing",
			content:  "BT /F1 12 Tf 72 700 Td 2 Tw (Hello) Tj ET",
			wantLen:  1,
			wantText: "Hello",
		},
		{
			name:     "Tc sets character spacing",
			content:  "BT /F1 12 Tf 72 700 Td 0.5 Tc (Hello) Tj ET",
			wantLen:  1,
			wantText: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseContentStream(tt.content)
			if len(got) != tt.wantLen {
				t.Errorf("parseContentStream() returned %d elements, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantText != "" && len(got) > 0 && got[0].Text != tt.wantText {
				t.Errorf("parseContentStream() text = %q, want %q", got[0].Text, tt.wantText)
			}
		})
	}
}

func TestEffectiveFontSize(t *testing.T) {
	tests := []struct {
		name string
		ts   textState
		want float64
	}{
		{
			name: "default scaling",
			ts:   textState{fontSize: 12, tmA: 1, tmD: 1},
			want: 12,
		},
		{
			name: "tmD differs from tmA",
			ts:   textState{fontSize: 12, tmA: 1, tmD: 2},
			want: 24,
		},
		{
			name: "negative font size becomes positive",
			ts:   textState{fontSize: 12, tmA: -1, tmD: -1},
			want: 12,
		},
		{
			name: "tmD is zero uses tmA",
			ts:   textState{fontSize: 10, tmA: 2, tmD: 0},
			want: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveFontSize(&tt.ts)
			if got != tt.want {
				t.Errorf("effectiveFontSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "empty string",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "whitespace only",
			input:   "   \t\n",
			wantLen: 0,
		},
		{
			name:    "simple tokens",
			input:   "/F1 12 Tf",
			wantLen: 3,
		},
		{
			name:    "parenthesized string",
			input:   "(Hello World) Tj",
			wantLen: 2,
		},
		{
			name:    "hex string",
			input:   "<48656C6C6F> Tj",
			wantLen: 2,
		},
		{
			name:    "array brackets",
			input:   "[(Hello)] TJ",
			wantLen: 4,
		},
		{
			name:    "comment is skipped",
			input:   "% comment\n/F1 12 Tf",
			wantLen: 3,
		},
		{
			name:    "nested parens",
			input:   "(Hello (nested) World) Tj",
			wantLen: 2,
		},
		{
			name:    "escaped chars in string",
			input:   "(Hello\\nWorld) Tj",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("tokenize(%q) returned %d tokens %v, want %d", tt.input, len(got), got, tt.wantLen)
			}
		})
	}
}

func TestDecodeParenString(t *testing.T) {
	tests := []struct {
		name string
		tok  string
		want string
	}{
		{
			name: "simple string",
			tok:  "(Hello)",
			want: "Hello",
		},
		{
			name: "newline escape",
			tok:  "(Hello\\nWorld)",
			want: "Hello\nWorld",
		},
		{
			name: "return escape",
			tok:  "(Hello\\rWorld)",
			want: "Hello\rWorld",
		},
		{
			name: "tab escape",
			tok:  "(Hello\\tWorld)",
			want: "Hello\tWorld",
		},
		{
			name: "escaped parens",
			tok:  "(Hello\\(World\\))",
			want: "Hello(World)",
		},
		{
			name: "escaped backslash",
			tok:  "(Hello\\\\World)",
			want: "Hello\\World",
		},
		{
			name: "unknown escape passes through",
			tok:  "(Hello\\xWorld)",
			want: "HelloxWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeParenString(tt.tok)
			if got != tt.want {
				t.Errorf("decodeParenString(%q) = %q, want %q", tt.tok, got, tt.want)
			}
		})
	}
}

func TestDecodeHexString(t *testing.T) {
	tests := []struct {
		name string
		tok  string
		want string
	}{
		{
			name: "simple hex",
			tok:  "<48656C6C6F>",
			want: "Hello",
		},
		{
			name: "hex with spaces",
			tok:  "<48 65 6C 6C 6F>",
			want: "Hello",
		},
		{
			name: "empty hex",
			tok:  "<>",
			want: "",
		},
		{
			name: "invalid hex digits skipped",
			tok:  "<ZZZZ>",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeHexString(tt.tok)
			if got != tt.want {
				t.Errorf("decodeHexString(%q) = %q, want %q", tt.tok, got, tt.want)
			}
		})
	}
}

func TestDecodeStringToken(t *testing.T) {
	tests := []struct {
		name string
		tok  string
		want string
	}{
		{
			name: "paren string",
			tok:  "(Hello)",
			want: "Hello",
		},
		{
			name: "hex string",
			tok:  "<48656C6C6F>",
			want: "Hello",
		},
		{
			name: "plain token passthrough",
			tok:  "plain",
			want: "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeStringToken(tt.tok)
			if got != tt.want {
				t.Errorf("decodeStringToken(%q) = %q, want %q", tt.tok, got, tt.want)
			}
		})
	}
}

func TestFindPrecedingString(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		index  int
		want   string
	}{
		{
			name:   "index zero returns empty",
			tokens: []string{"Tj"},
			index:  0,
			want:   "",
		},
		{
			name:   "preceding paren string",
			tokens: []string{"(Hello)", "Tj"},
			index:  1,
			want:   "Hello",
		},
		{
			name:   "preceding non-string token",
			tokens: []string{"12", "Tj"},
			index:  1,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findPrecedingString(tt.tokens, tt.index)
			if got != tt.want {
				t.Errorf("findPrecedingString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTm(t *testing.T) {
	tokens := []string{"1", "0", "0", "1.5", "100", "500", "Tm"}
	ts := newTextState()
	parseTm(tokens, 6, &ts)

	if ts.tmA != 1 {
		t.Errorf("tmA = %v, want 1", ts.tmA)
	}

	if ts.tmD != 1.5 {
		t.Errorf("tmD = %v, want 1.5", ts.tmD)
	}

	if ts.x != 100 {
		t.Errorf("x = %v, want 100", ts.x)
	}

	if ts.y != 500 {
		t.Errorf("y = %v, want 500", ts.y)
	}
}

func TestParseTmInsufficientTokens(t *testing.T) {
	tokens := []string{"1", "0", "Tm"}
	ts := newTextState()
	parseTm(tokens, 2, &ts)

	// Should not modify state when insufficient tokens.
	if ts.x != 0 {
		t.Errorf("x = %v, want 0 (unchanged)", ts.x)
	}
}

func TestParseQuote(t *testing.T) {
	ts := textState{fontSize: 12, y: 700}
	parseQuote(&ts)

	if ts.y != 688 {
		t.Errorf("y after quote = %v, want 688", ts.y)
	}
}

func TestMultipleTextBlocks(t *testing.T) {
	rs := buildTestPDF(t, 3, true)
	r := NewReader()

	if err := r.Open(rs); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	for pageNum := 1; pageNum <= 3; pageNum++ {
		page, err := r.Page(pageNum)
		if err != nil {
			t.Fatalf("Page(%d): %v", pageNum, err)
		}

		if len(page.Elements) == 0 {
			t.Errorf("Page %d: expected text elements, got 0", pageNum)
		}
	}
}

func TestReadHexString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		start   int
		wantTok string
		wantEnd int
	}{
		{
			name:    "simple hex string",
			input:   "<48656C6C6F>",
			start:   0,
			wantTok: "<48656C6C6F>",
			wantEnd: 12,
		},
		{
			name:    "empty hex string",
			input:   "<>",
			start:   0,
			wantTok: "<>",
			wantEnd: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, end := readHexString(tt.input, tt.start)
			if tok != tt.wantTok {
				t.Errorf("readHexString() tok = %q, want %q", tok, tt.wantTok)
			}

			if end != tt.wantEnd {
				t.Errorf("readHexString() end = %d, want %d", end, tt.wantEnd)
			}
		})
	}
}

func TestSkipComment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		start   int
		wantEnd int
	}{
		{
			name:    "comment ending with newline",
			input:   "% this is a comment\nnext",
			start:   0,
			wantEnd: 19,
		},
		{
			name:    "comment at end of string",
			input:   "% comment",
			start:   0,
			wantEnd: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipComment(tt.input, tt.start)
			if got != tt.wantEnd {
				t.Errorf("skipComment() = %d, want %d", got, tt.wantEnd)
			}
		})
	}
}

func TestIsKerningSpace(t *testing.T) {
	tests := []struct {
		name    string
		kerning float64
		want    bool
	}{
		{
			name:    "small negative kerning is not a space",
			kerning: -10,
			want:    false,
		},
		{
			name:    "small positive kerning is not a space",
			kerning: 50,
			want:    false,
		},
		{
			name:    "large negative kerning is a space",
			kerning: -500,
			want:    true,
		},
		{
			name:    "large positive kerning is a space",
			kerning: 400,
			want:    true,
		},
		{
			name:    "exactly at threshold is not a space",
			kerning: -300,
			want:    false,
		},
		{
			name:    "just above threshold is a space",
			kerning: -301,
			want:    true,
		},
		{
			name:    "zero kerning is not a space",
			kerning: 0,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isKerningSpace(tt.kerning)
			if got != tt.want {
				t.Errorf("isKerningSpace(%v) = %v, want %v", tt.kerning, got, tt.want)
			}
		})
	}
}

func TestFindDoOperands(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "no Do operators",
			content: "BT /F1 12 Tf (Hello) Tj ET",
			want:    nil,
		},
		{
			name:    "single Do operator",
			content: "q /Form1 Do Q",
			want:    []string{"Form1"},
		},
		{
			name:    "multiple Do operators",
			content: "q /Fm0 Do Q q /Fm1 Do Q",
			want:    []string{"Fm0", "Fm1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDoOperands(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("findDoOperands() = %v, want %v", got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("findDoOperands()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseTJArraySpacing(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantText string
	}{
		{
			name:     "small kerning keeps text together",
			content:  "BT /F1 12 Tf 72 700 Td [(25,) -10 (305,) -10 (031)] TJ ET",
			wantText: "25,305,031",
		},
		{
			name:     "large kerning inserts space between words",
			content:  "BT /F1 12 Tf 72 700 Td [(Statement) -600 (of)] TJ ET",
			wantText: "Statement of",
		},
		{
			name:     "mixed kerning values",
			content:  "BT /F1 12 Tf 72 700 Td [(Lap) -20 (oran) -500 (posisi)] TJ ET",
			wantText: "Laporan posisi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elements := parseContentStream(tt.content)
			if len(elements) == 0 {
				t.Fatal("expected at least one element")
			}

			if elements[0].Text != tt.wantText {
				t.Errorf("text = %q, want %q", elements[0].Text, tt.wantText)
			}
		})
	}
}

func TestParseDoubleQuote(t *testing.T) {
	ts := textState{fontSize: 12, y: 700, tmA: 1, tmD: 1}
	tokens := []string{"2", "0.5", "(Hello)", `"`}
	parseDoubleQuote(tokens, 3, &ts)

	if ts.wordSpacing != 2 {
		t.Errorf("wordSpacing = %v, want 2", ts.wordSpacing)
	}

	if ts.charSpacing != 0.5 {
		t.Errorf("charSpacing = %v, want 0.5", ts.charSpacing)
	}

	if ts.y != 688 {
		t.Errorf("y after double quote = %v, want 688", ts.y)
	}
}

func TestParseTL(t *testing.T) {
	ts := newTextState()
	tokens := []string{"14", "TL"}
	parseTL(tokens, 1, &ts)

	if ts.textLeading != 14 {
		t.Errorf("textLeading = %v, want 14", ts.textLeading)
	}
}

func TestParseTw(t *testing.T) {
	ts := newTextState()
	tokens := []string{"2.5", "Tw"}
	parseTw(tokens, 1, &ts)

	if ts.wordSpacing != 2.5 {
		t.Errorf("wordSpacing = %v, want 2.5", ts.wordSpacing)
	}
}

func TestParseTc(t *testing.T) {
	ts := newTextState()
	tokens := []string{"0.5", "Tc"}
	parseTc(tokens, 1, &ts)

	if ts.charSpacing != 0.5 {
		t.Errorf("charSpacing = %v, want 0.5", ts.charSpacing)
	}
}

func TestParseQuoteWithTextLeading(t *testing.T) {
	ts := textState{fontSize: 12, y: 700, textLeading: 14, tmA: 1, tmD: 1}
	parseQuote(&ts)

	if ts.y != 686 {
		t.Errorf("y after quote with leading = %v, want 686", ts.y)
	}
}
