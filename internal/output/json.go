package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lugassawan/idxlens/internal/domain"
)

type jsonFormatter struct {
	pretty bool
}

func newJSONFormatter(cfg *formatterConfig) *jsonFormatter {
	return &jsonFormatter{pretty: cfg.Pretty}
}

func (f *jsonFormatter) Format(w io.Writer, stmt *domain.FinancialStatement) error {
	data, err := f.marshal(stmt)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	data = append(data, '\n')

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	return nil
}

func (f *jsonFormatter) marshal(stmt *domain.FinancialStatement) ([]byte, error) {
	if f.pretty {
		return json.MarshalIndent(stmt, "", "  ")
	}

	return json.Marshal(stmt)
}
