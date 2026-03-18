package pdf

import (
	"bytes"
	"testing"
)

func FuzzReaderOpen(f *testing.F) {
	// Seed corpus with minimal valid PDF header
	f.Add([]byte("%PDF-1.4\n"))
	f.Add([]byte(""))
	f.Add([]byte("not a pdf"))

	f.Fuzz(func(t *testing.T, data []byte) {
		r := NewReader()
		// Should not panic on any input
		_ = r.Open(bytes.NewReader(data))
		r.Close()
	})
}
