package domain

import "testing"

func FuzzParseNumber(f *testing.F) {
	// Seed with known valid inputs
	f.Add("1.234.567,89")
	f.Add("(1.234)")
	f.Add("-")
	f.Add("")
	f.Add("N/A")
	f.Add("0")
	f.Add("-1.234,56")

	f.Fuzz(func(t *testing.T, s string) {
		// Should not panic on any input
		_, _ = ParseNumber(s)
	})
}
