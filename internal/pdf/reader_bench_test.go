package pdf

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkParseContentStream(b *testing.B) {
	tests := []struct {
		name       string
		blockCount int
	}{
		{name: "single block", blockCount: 1},
		{name: "10 blocks", blockCount: 10},
		{name: "50 blocks", blockCount: 50},
		{name: "100 blocks", blockCount: 100},
	}

	for _, tc := range tests {
		content := buildContentStream(tc.blockCount)

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				parseContentStream(content)
			}
		})
	}
}

func BenchmarkTokenize(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple text",
			input: "/F1 12 Tf 72 700 Td (Hello World) Tj",
		},
		{
			name:  "hex strings",
			input: "/F1 10 Tf 50 600 Td <48656C6C6F> Tj",
		},
		{
			name:  "TJ array",
			input: "/F1 12 Tf 72 700 Td [(Hello) 50 (World)] TJ",
		},
		{
			name:  "multiple operators",
			input: "/F1 12 Tf 72 700 Td (Line1) Tj 0 -14 Td (Line2) Tj 0 -14 Td (Line3) Tj",
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				tokenize(tc.input)
			}
		})
	}
}

func BenchmarkParseTextBlock(b *testing.B) {
	tests := []struct {
		name  string
		block string
	}{
		{
			name:  "single Tj",
			block: "/F1 12 Tf 72 700 Td (Hello World) Tj",
		},
		{
			name:  "Tm with Tj",
			block: "12 0 0 12 72 700 Tm (Scaled text) Tj",
		},
		{
			name:  "TJ array",
			block: "/F1 10 Tf 50 500 Td [(Part) -50 (One) 100 (Two)] TJ",
		},
		{
			name:  "multiple lines",
			block: "/F1 12 Tf 72 700 Td (Line 1) Tj 0 -14 Td (Line 2) Tj 0 -14 Td (Line 3) Tj",
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				parseTextBlock(tc.block)
			}
		})
	}
}

func BenchmarkOpenAndRead(b *testing.B) {
	tests := []struct {
		name     string
		pages    int
		withText bool
	}{
		{name: "1 page no text", pages: 1, withText: false},
		{name: "1 page with text", pages: 1, withText: true},
		{name: "5 pages with text", pages: 5, withText: true},
	}

	for _, tc := range tests {
		// Build PDF once outside the benchmark loop.
		rs := buildTestPDF(&testing.T{}, tc.pages, tc.withText)

		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				if _, err := rs.Seek(0, 0); err != nil {
					b.Fatalf("Seek: %v", err)
				}

				r := NewReader()

				if err := r.Open(rs); err != nil {
					b.Fatalf("Open: %v", err)
				}

				for p := 1; p <= r.PageCount(); p++ {
					if _, err := r.Page(p); err != nil {
						b.Fatalf("Page(%d): %v", p, err)
					}
				}

				r.Close()
			}
		})
	}
}

func BenchmarkDecodeStringToken(b *testing.B) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "paren string", token: "(Hello World)"},
		{name: "hex string", token: "<48656C6C6F>"},
		{name: "escaped paren", token: `(Hello \(World\))`},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				decodeStringToken(tc.token)
			}
		})
	}
}

// buildContentStream generates a synthetic PDF content stream with multiple
// BT/ET text blocks for benchmarking.
func buildContentStream(blockCount int) string {
	var sb strings.Builder

	for i := range blockCount {
		y := 700 - (i * 14)
		fmt.Fprintf(&sb, "BT /F1 12 Tf 72 %d Td (Line %d of the document) Tj ET\n", y, i+1)
	}

	return sb.String()
}
