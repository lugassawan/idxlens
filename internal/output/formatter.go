package output

import (
	"fmt"
	"io"

	"github.com/lugassawan/idxlens/internal/domain"
)

// Format represents a supported output format.
type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// Formatter writes a FinancialStatement to a writer in a specific format.
type Formatter interface {
	Format(w io.Writer, stmt *domain.FinancialStatement) error
}

// Option configures a Formatter.
type Option func(*formatterConfig)

type formatterConfig struct {
	Pretty bool
}

// WithPretty enables indented output for formats that support it.
func WithPretty(pretty bool) Option {
	return func(c *formatterConfig) {
		c.Pretty = pretty
	}
}

// NewFormatter creates a Formatter for the given format string.
func NewFormatter(f Format, opts ...Option) (Formatter, error) {
	cfg := &formatterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	switch f {
	case FormatJSON:
		return newJSONFormatter(cfg), nil
	case FormatCSV:
		return newCSVFormatter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", f)
	}
}
