package domain

import "testing"

// sinkFloat prevents compiler optimization of benchmark results.
var sinkFloat float64

func BenchmarkParseNumber(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "integer", input: "1234567"},
		{name: "thousands separator", input: "1.234.567"},
		{name: "decimal", input: "1.234.567,89"},
		{name: "negative parens", input: "(1.234.567,89)"},
		{name: "negative dash", input: "-1.234.567,89"},
		{name: "dash only", input: "-"},
		{name: "small number", input: "42"},
		{name: "decimal only", input: "0,50"},
		{name: "large number", input: "999.999.999.999,99"},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			var f float64

			for range b.N {
				f, _ = ParseNumber(tc.input)
			}

			sinkFloat = f
		})
	}
}

func BenchmarkParseNumberErrors(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "whitespace", input: "   "},
		{name: "not a number", input: "N/A"},
		{name: "text", input: "Total Aset"},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			var f float64

			for range b.N {
				f, _ = ParseNumber(tc.input)
			}

			sinkFloat = f
		})
	}
}

func BenchmarkParseNumberBatch(b *testing.B) {
	// Simulate parsing a column of financial values.
	values := []string{
		"1.234.567",
		"2.345.678,90",
		"(3.456.789)",
		"-4.567.890,12",
		"-",
		"5.678.901",
		"6.789.012,34",
		"(7.890.123,45)",
		"8.901.234",
		"9.012.345,67",
	}

	b.ResetTimer()

	var f float64

	for range b.N {
		for _, v := range values {
			f, _ = ParseNumber(v)
		}
	}

	sinkFloat = f
}
