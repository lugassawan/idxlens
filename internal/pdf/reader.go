package pdf

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var (
	errNotOpened = errors.New("reader not opened")
	// btETPattern matches BT...ET text blocks in a content stream.
	btETPattern = regexp.MustCompile(`(?s)BT\s(.*?)\sET`)
)

// NewReader creates a new PDF Reader backed by pdfcpu.
func NewReader() Reader {
	return &pdfcpuReader{}
}

type pdfcpuReader struct {
	ctx *model.Context
}

func (r *pdfcpuReader) Open(rs io.ReadSeeker) error {
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

	if len(bb) == 0 {
		return nil, nil
	}

	return parseContentStream(string(bb)), nil
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
	fontName string
	fontSize float64
	x        float64
	y        float64
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
		switch tokens[i] {
		case "Tf":
			parseTf(tokens, i, &ts)
		case "Td", "TD":
			parseTd(tokens, i, &ts)
		case "Tm":
			parseTm(tokens, i, &ts)
		case "Tj":
			el := parseTj(tokens, i, &ts)
			if el != nil {
				elements = append(elements, *el)
			}
		case "TJ":
			els := parseTJArray(tokens, i, &ts)
			elements = append(elements, els...)
		case "'":
			parseQuote(&ts)
			el := parseTj(tokens, i, &ts)
			if el != nil {
				elements = append(elements, *el)
			}
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
	if i >= 6 {
		ts.tmA = parseFloat(tokens[i-6])
		ts.tmD = parseFloat(tokens[i-3])
		ts.x = parseFloat(tokens[i-2])
		ts.y = parseFloat(tokens[i-1])
	}
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

func parseQuote(ts *textState) {
	ts.y -= ts.fontSize
}

func parseTJArray(tokens []string, i int, ts *textState) []TextElement {
	start := findArrayStart(tokens, i)
	if start < 0 {
		return nil
	}

	var combined strings.Builder

	for j := start + 1; j < i; j++ {
		if tokens[j] == "]" {
			break
		}

		if isStringToken(tokens[j]) {
			combined.WriteString(decodeStringToken(tokens[j]))
		}
	}

	text := combined.String()
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

	return []TextElement{el}
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
