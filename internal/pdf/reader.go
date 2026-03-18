package pdf

import (
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// spaceThresholdThou is the threshold in thousandths of text space units
// above which a TJ kerning displacement is treated as an inter-word space.
// A value of 300 means 30% of the font size (0.3 * 1000).
const spaceThresholdThou = 300

var (
	errNotOpened = errors.New("reader not opened")
	// btETPattern matches BT...ET text blocks in a content stream.
	btETPattern = regexp.MustCompile(`(?s)BT\s(.*?)\sET`)
	// initOnce ensures pdfcpu's global config is initialized exactly once,
	// avoiding a data race in model.NewDefaultConfiguration when called
	// concurrently from multiple goroutines.
	initOnce sync.Once
)

// NewReader creates a new PDF Reader backed by pdfcpu.
func NewReader() Reader {
	return &pdfcpuReader{}
}

type pdfcpuReader struct {
	ctx *model.Context
}

func (r *pdfcpuReader) Open(rs io.ReadSeeker) error {
	initOnce.Do(func() {
		// Pre-initialize pdfcpu's global config to avoid data race
		// when multiple goroutines call NewDefaultConfiguration concurrently.
		_ = model.NewDefaultConfiguration()
	})

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := pdfcpuapi.ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("open pdf: %w", err)
	}

	r.ctx = ctx

	return nil
}

func (r *pdfcpuReader) Metadata() (Metadata, error) {
	if r.ctx == nil {
		return Metadata{}, errNotOpened
	}

	return Metadata{
		Title:    r.ctx.Title,
		Author:   r.ctx.Author,
		Creator:  r.ctx.Creator,
		Producer: r.ctx.Producer,
		Pages:    r.ctx.PageCount,
	}, nil
}

func (r *pdfcpuReader) PageCount() int {
	if r.ctx == nil {
		return 0
	}

	return r.ctx.PageCount
}

func (r *pdfcpuReader) Page(number int) (Page, error) {
	if r.ctx == nil {
		return Page{}, errNotOpened
	}

	if number < 1 || number > r.ctx.PageCount {
		return Page{}, fmt.Errorf("page %d out of range [1, %d]", number, r.ctx.PageCount)
	}

	size, err := r.pageSize(number)
	if err != nil {
		return Page{}, fmt.Errorf("read page %d size: %w", number, err)
	}

	elements, err := r.extractElements(number)
	if err != nil {
		return Page{}, fmt.Errorf("read page %d: %w", number, err)
	}

	return Page{
		Number:   number,
		Size:     size,
		Elements: elements,
	}, nil
}

func (r *pdfcpuReader) Close() error {
	r.ctx = nil
	return nil
}

func (r *pdfcpuReader) pageSize(number int) (PageSize, error) {
	dims, err := r.ctx.PageDims()
	if err != nil {
		return PageSize{}, fmt.Errorf("get page dimensions: %w", err)
	}

	idx := number - 1
	if idx >= len(dims) {
		return PageSize{}, fmt.Errorf("page %d dimensions not found", number)
	}

	return PageSize{
		Width:  dims[idx].Width,
		Height: dims[idx].Height,
	}, nil
}

func (r *pdfcpuReader) extractElements(pageNr int) ([]TextElement, error) {
	contentReader, err := pdfcpu.ExtractPageContent(r.ctx, pageNr)
	if err != nil {
		return nil, fmt.Errorf("extract content: %w", err)
	}

	bb, err := io.ReadAll(contentReader)
	if err != nil {
		return nil, fmt.Errorf("read content stream: %w", err)
	}

	content := string(bb)
	elements := parseContentStream(content)

	// Extract text from Form XObjects referenced by Do operators.
	xObjElements, err := r.extractFormXObjects(pageNr, content)
	if err != nil {
		return nil, fmt.Errorf("extract form xobjects: %w", err)
	}

	elements = append(elements, xObjElements...)

	if len(elements) == 0 {
		return nil, nil
	}

	return elements, nil
}

// extractFormXObjects finds Do operators in the content stream and extracts
// text from any referenced Form XObjects. Pages that use Form XObjects for
// their text content would otherwise return empty.
func (r *pdfcpuReader) extractFormXObjects(pageNr int, content string) ([]TextElement, error) {
	xObjNames := findDoOperands(content)
	if len(xObjNames) == 0 {
		return nil, nil
	}

	pageDict, _, inhAttrs, err := r.ctx.PageDict(pageNr, false)
	if err != nil {
		return nil, fmt.Errorf("get page dict: %w", err)
	}

	xObjDict := resolveXObjectDict(r.ctx, pageDict, inhAttrs)
	if xObjDict == nil {
		return nil, nil
	}

	var elements []TextElement

	for _, name := range xObjNames {
		els, err := r.extractSingleFormXObject(xObjDict, name)
		if err != nil {
			continue
		}

		elements = append(elements, els...)
	}

	return elements, nil
}

// resolveXObjectDict retrieves the XObject resource dictionary from the page
// dict or inherited page attributes.
func resolveXObjectDict(ctx *model.Context, pageDict types.Dict, inhAttrs *model.InheritedPageAttrs) types.Dict {
	resDict := extractResourceDict(ctx, pageDict)
	if resDict == nil && inhAttrs != nil {
		resDict = inhAttrs.Resources
	}

	if resDict == nil {
		return nil
	}

	obj, found := resDict.Find("XObject")
	if !found {
		return nil
	}

	d, err := ctx.DereferenceDict(obj)
	if err != nil {
		return nil
	}

	return d
}

// extractResourceDict dereferences the Resources entry from a page dict.
func extractResourceDict(ctx *model.Context, pageDict types.Dict) types.Dict {
	obj, found := pageDict.Find("Resources")
	if !found {
		return nil
	}

	d, err := ctx.DereferenceDict(obj)
	if err != nil {
		return nil
	}

	return d
}

// extractSingleFormXObject extracts text elements from a single Form XObject.
func (r *pdfcpuReader) extractSingleFormXObject(xObjDict types.Dict, name string) ([]TextElement, error) {
	obj, found := xObjDict.Find(name)
	if !found {
		return nil, nil
	}

	indRef, ok := obj.(types.IndirectRef)
	if !ok {
		return nil, nil
	}

	sd, err := r.ctx.DereferenceXObjectDict(indRef)
	if err != nil || sd == nil {
		return nil, err
	}

	subType := sd.Subtype()
	if subType == nil || *subType != "Form" {
		return nil, nil
	}

	if err := sd.Decode(); err != nil {
		return nil, fmt.Errorf("decode xobject stream: %w", err)
	}

	return parseContentStream(string(sd.Content)), nil
}

// findDoOperands extracts XObject names from Do operators in a content stream.
func findDoOperands(content string) []string {
	tokens := tokenize(content)

	var names []string

	for i, tok := range tokens {
		if tok == "Do" && i >= 1 {
			name := strings.TrimPrefix(tokens[i-1], "/")
			if name != "" {
				names = append(names, name)
			}
		}
	}

	return names
}

// parseContentStream extracts text elements from a PDF content stream
// by interpreting text-related operators (BT/ET, Tf, Td, Tm, Tj, TJ).
func parseContentStream(content string) []TextElement {
	matches := btETPattern.FindAllStringSubmatch(content, -1)
	elements := make([]TextElement, 0, len(matches))

	for _, m := range matches {
		elements = append(elements, parseTextBlock(m[1])...)
	}

	return elements
}

// textState tracks the current text rendering state within a BT...ET block.
type textState struct {
	fontName    string
	fontSize    float64
	x           float64
	y           float64
	wordSpacing float64
	charSpacing float64
	textLeading float64
	// Text matrix components for Tm operator.
	tmA float64
	tmD float64
}

func newTextState() textState {
	return textState{tmA: 1, tmD: 1}
}

func parseTextBlock(block string) []TextElement {
	var elements []TextElement

	ts := newTextState()
	tokens := tokenize(block)

	for i := range tokens {
		updateTextState(tokens, i, &ts)
		elements = appendTextElements(elements, tokens, i, &ts)
	}

	return elements
}

// updateTextState applies state-modifying operators (Tf, Td, Tm, T*, TL, Tw, Tc).
func updateTextState(tokens []string, i int, ts *textState) {
	switch tokens[i] {
	case "Tf":
		parseTf(tokens, i, ts)
	case "Td", "TD":
		parseTd(tokens, i, ts)
	case "Tm":
		parseTm(tokens, i, ts)
	case "T*":
		parseQuote(ts)
	case "TL":
		parseTL(tokens, i, ts)
	case "Tw":
		parseTw(tokens, i, ts)
	case "Tc":
		parseTc(tokens, i, ts)
	}
}

// appendTextElements handles text-emitting operators (Tj, TJ, ', ").
func appendTextElements(elements []TextElement, tokens []string, i int, ts *textState) []TextElement {
	switch tokens[i] {
	case "Tj":
		el := parseTj(tokens, i, ts)
		if el != nil {
			elements = append(elements, *el)
		}
	case "TJ":
		els := parseTJArray(tokens, i, ts)
		elements = append(elements, els...)
	case "'":
		parseQuote(ts)
		el := parseTj(tokens, i, ts)
		if el != nil {
			elements = append(elements, *el)
		}
	case `"`:
		parseDoubleQuote(tokens, i, ts)
		el := parseTj(tokens, i, ts)
		if el != nil {
			elements = append(elements, *el)
		}
	}

	return elements
}

func parseTf(tokens []string, i int, ts *textState) {
	if i >= 2 {
		ts.fontSize = parseFloat(tokens[i-1])
		ts.fontName = strings.TrimPrefix(tokens[i-2], "/")
	}
}

func parseTd(tokens []string, i int, ts *textState) {
	if i >= 2 {
		tx := parseFloat(tokens[i-2])
		ty := parseFloat(tokens[i-1])
		ts.x += tx
		ts.y += ty
	}
}

func parseTm(tokens []string, i int, ts *textState) {
	if i < 6 {
		return
	}

	args := tokens[i-6 : i]
	if len(args) < 6 {
		return
	}

	ts.tmA = parseFloat(args[0])
	ts.tmD = parseFloat(args[3])
	ts.x = parseFloat(args[4])
	ts.y = parseFloat(args[5])
}

func parseTj(tokens []string, i int, ts *textState) *TextElement {
	text := findPrecedingString(tokens, i)
	if text == "" {
		return nil
	}

	fontSize := effectiveFontSize(ts)

	el := TextElement{
		Text:     text,
		FontName: ts.fontName,
		FontSize: fontSize,
		Bounds: Rect{
			X1: ts.x,
			Y1: ts.y,
			X2: ts.x,
			Y2: ts.y + fontSize,
		},
	}

	return &el
}

func effectiveFontSize(ts *textState) float64 {
	fontSize := ts.fontSize * ts.tmA
	if ts.tmD != 0 && ts.tmD != ts.tmA {
		fontSize = ts.fontSize * ts.tmD
	}

	if fontSize < 0 {
		fontSize = -fontSize
	}

	return fontSize
}

func parseTL(tokens []string, i int, ts *textState) {
	if i >= 1 {
		ts.textLeading = parseFloat(tokens[i-1])
	}
}

func parseTw(tokens []string, i int, ts *textState) {
	if i >= 1 {
		ts.wordSpacing = parseFloat(tokens[i-1])
	}
}

func parseTc(tokens []string, i int, ts *textState) {
	if i >= 1 {
		ts.charSpacing = parseFloat(tokens[i-1])
	}
}

func parseQuote(ts *textState) {
	if ts.textLeading != 0 {
		ts.y -= ts.textLeading
	} else {
		ts.y -= ts.fontSize
	}
}

// parseDoubleQuote handles the " operator: set word spacing, char spacing,
// then move to next line. The string operand is handled by parseTj.
func parseDoubleQuote(tokens []string, i int, ts *textState) {
	if i >= 3 {
		ts.wordSpacing = parseFloat(tokens[i-3])
		ts.charSpacing = parseFloat(tokens[i-2])
	}

	parseQuote(ts)
}

func parseTJArray(tokens []string, i int, ts *textState) []TextElement {
	start := findArrayStart(tokens, i)
	if start < 0 {
		return nil
	}

	fontSize := effectiveFontSize(ts)

	var combined strings.Builder

	for j := start + 1; j < i; j++ {
		if tokens[j] == "]" {
			break
		}

		if isStringToken(tokens[j]) {
			combined.WriteString(decodeStringToken(tokens[j]))
			continue
		}

		// Numeric values in TJ arrays represent glyph displacement in
		// thousandths of a unit of text space. Negative values move the
		// text position forward; large negative values indicate a space.
		kerning := parseFloat(tokens[j])
		if isKerningSpace(kerning) {
			combined.WriteByte(' ')
		}
	}

	text := combined.String()
	if text == "" {
		return nil
	}

	el := TextElement{
		Text:     text,
		FontName: ts.fontName,
		FontSize: fontSize,
		Bounds: Rect{
			X1: ts.x,
			Y1: ts.y,
			X2: ts.x,
			Y2: ts.y + fontSize,
		},
	}

	return []TextElement{el}
}

// isKerningSpace returns true when a TJ kerning displacement is large
// enough to represent an inter-word space. TJ kerning values are in
// thousandths of a unit of text space. A value of -1000 moves by one
// full font size unit. We treat values whose magnitude exceeds
// spaceThresholdThou (300, i.e. 30% of font size) as word spaces.
func isKerningSpace(kerning float64) bool {
	return math.Abs(kerning) > spaceThresholdThou
}

func findArrayStart(tokens []string, i int) int {
	for j := i - 1; j >= 0; j-- {
		if tokens[j] == "[" {
			return j
		}
	}

	return -1
}

// tokenize splits a content stream block into tokens.
// It handles parenthesized strings, hex strings, and arrays.
func tokenize(s string) []string {
	var tokens []string

	i := 0
	for i < len(s) {
		n, tok := nextToken(s, i)
		if tok != "" {
			tokens = append(tokens, tok)
		}

		i = n
	}

	return tokens
}

// nextToken reads the next token from position i in s.
// It returns the new position and the token string (empty if whitespace/comment was skipped).
func nextToken(s string, i int) (int, string) {
	ch := s[i]

	if isWhitespace(ch) {
		return i + 1, ""
	}

	if ch == '%' {
		return skipComment(s, i), ""
	}

	if ch == '(' {
		tok, end := readParenString(s, i)
		return end, tok
	}

	if ch == '<' && i+1 < len(s) && s[i+1] != '<' {
		tok, end := readHexString(s, i)
		return end, tok
	}

	if ch == '[' {
		return i + 1, "["
	}

	if ch == ']' {
		return i + 1, "]"
	}

	return readWord(s, i)
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func skipComment(s string, i int) int {
	for i < len(s) && s[i] != '\n' && s[i] != '\r' {
		i++
	}

	return i
}

func readWord(s string, start int) (int, string) {
	i := start
	for i < len(s) && !isTokenDelimiter(s[i]) {
		i++
	}

	return i, s[start:i]
}

func isTokenDelimiter(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == '(' || ch == ')' || ch == '[' || ch == ']' ||
		ch == '<' || ch == '>' || ch == '%'
}

// readParenString reads a parenthesized string literal starting at position i.
func readParenString(s string, i int) (string, int) {
	depth := 1
	i++ // skip opening '('

	var buf strings.Builder

	buf.WriteByte('(')

	for i < len(s) && depth > 0 {
		ch := s[i]

		switch {
		case ch == '\\' && i+1 < len(s):
			buf.WriteByte(ch)
			i++
			buf.WriteByte(s[i])
			i++
		case ch == '(':
			depth++
			buf.WriteByte(ch)
			i++
		case ch == ')':
			depth--
			buf.WriteByte(ch)
			i++
		default:
			buf.WriteByte(ch)
			i++
		}
	}

	return buf.String(), i
}

// readHexString reads a hex string literal starting at position i.
func readHexString(s string, i int) (string, int) {
	start := i
	i++ // skip '<'

	for i < len(s) && s[i] != '>' {
		i++
	}

	if i < len(s) {
		i++ // skip '>'
	}

	return s[start:i], i
}

func findPrecedingString(tokens []string, i int) string {
	if i < 1 {
		return ""
	}

	tok := tokens[i-1]
	if isStringToken(tok) {
		return decodeStringToken(tok)
	}

	return ""
}

func isStringToken(tok string) bool {
	return (strings.HasPrefix(tok, "(") && strings.HasSuffix(tok, ")")) ||
		(strings.HasPrefix(tok, "<") && strings.HasSuffix(tok, ">"))
}

func decodeStringToken(tok string) string {
	if strings.HasPrefix(tok, "(") && strings.HasSuffix(tok, ")") {
		return decodeParenString(tok)
	}

	if strings.HasPrefix(tok, "<") && strings.HasSuffix(tok, ">") {
		return decodeHexString(tok)
	}

	return tok
}

func decodeParenString(tok string) string {
	// Remove outer parens.
	s := tok[1 : len(tok)-1]

	var buf strings.Builder

	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i++

			switch s[i] {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case '(', ')', '\\':
				buf.WriteByte(s[i])
			default:
				// Octal escape or unknown: pass through.
				buf.WriteByte(s[i])
			}
		} else {
			buf.WriteByte(s[i])
		}

		i++
	}

	return buf.String()
}

func decodeHexString(tok string) string {
	hex := tok[1 : len(tok)-1]
	hex = strings.ReplaceAll(hex, " ", "")

	var buf strings.Builder

	for i := 0; i+1 < len(hex); i += 2 {
		b, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err != nil {
			continue
		}

		buf.WriteByte(byte(b))
	}

	return buf.String()
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
