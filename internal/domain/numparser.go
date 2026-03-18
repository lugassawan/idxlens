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

// ParseNumber converts an Indonesian-format number string to float64.
// Indonesian format uses dots for thousands separators and commas for decimal
// points, which is the opposite of US format.
//
// Examples:
//
//	"1.234.567"     -> 1234567.0
//	"1.234.567,89"  -> 1234567.89
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
	} else if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, ErrNotANumber
	}

	if negative {
		f = -f
	}

	return f, nil
}
