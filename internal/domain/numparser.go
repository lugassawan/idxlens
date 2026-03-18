package domain

import (
	"errors"
	"strconv"
	"strings"
)

// ErrEmptyValue indicates the input string was empty or whitespace-only.
var ErrEmptyValue = errors.New("empty value")

// ErrNotANumber indicates the input string could not be parsed as a number.
var ErrNotANumber = errors.New("not a number")

// ParseNumber converts a number string to float64, auto-detecting the format.
// Supports both Indonesian format (dots=thousands, commas=decimal) and English
// format (commas=thousands, dots=decimal).
//
// Format detection:
//   - Multiple commas -> English format (commas are thousands separators)
//   - Multiple dots -> Indonesian format (dots are thousands separators)
//   - Single comma + single dot -> last separator is decimal
//   - Single separator -> Indonesian assumed (dot=thousands, comma=decimal)
//
// Examples:
//
//	"1.234.567"     -> 1234567.0   (Indonesian)
//	"1.234.567,89"  -> 1234567.89  (Indonesian)
//	"1,234,567"     -> 1234567.0   (English)
//	"1,234,567.89"  -> 1234567.89  (English)
//	"(1.234)"       -> -1234.0
//	"-1.234,56"     -> -1234.56
//	"-"             -> 0.0
//	""              -> 0.0, ErrEmptyValue
//	"N/A"           -> 0.0, ErrNotANumber
func ParseNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, ErrEmptyValue
	}

	if s == "-" {
		return 0, nil
	}

	negative := false

	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		negative = true
		s = s[1 : len(s)-1]

		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]

		s = strings.TrimSpace(s)
	}

	s = normalizeNumberFormat(s)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, ErrNotANumber
	}

	if negative {
		f = -f
	}

	return f, nil
}

func normalizeNumberFormat(s string) string {
	commaCount := strings.Count(s, ",")
	dotCount := strings.Count(s, ".")

	switch {
	case commaCount > 1:
		// Multiple commas = English thousands separators
		s = strings.ReplaceAll(s, ",", "")
	case dotCount > 1:
		// Multiple dots = Indonesian thousands separators
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	case commaCount == 1 && dotCount == 1:
		// Mixed: last separator is the decimal point
		lastComma := strings.LastIndex(s, ",")
		lastDot := strings.LastIndex(s, ".")

		if lastComma > lastDot {
			// "1.234,56" -> Indonesian
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// "1,234.56" -> English
			s = strings.ReplaceAll(s, ",", "")
		}
	case commaCount == 0 && dotCount == 0:
		// Plain integer, nothing to do
	default:
		// Single separator: use digit grouping to disambiguate.
		// If 3 digits follow the separator, it's a thousands separator.
		if isThousandsSeparator(s) {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", "")
		} else {
			// Indonesian default: dot=thousands, comma=decimal
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		}
	}

	return s
}

// isThousandsSeparator checks if a single separator in a number string is a
// thousands separator rather than a decimal point. Returns true when exactly 3
// digits follow the separator (e.g. "1,234" or "1.234").
func isThousandsSeparator(s string) bool {
	sepIdx := strings.IndexAny(s, ",.")
	if sepIdx < 0 {
		return false
	}

	afterSep := s[sepIdx+1:]

	return len(afterSep) == 3 && isAllDigits(afterSep)
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return len(s) > 0
}
