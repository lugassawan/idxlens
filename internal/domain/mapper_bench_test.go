package domain

import (
	"fmt"
	"testing"

	"github.com/lugassawan/idxlens/internal/pdf"
	"github.com/lugassawan/idxlens/internal/table"
)

func BenchmarkMap(b *testing.B) {
	tests := []struct {
		name string
		rows int
	}{
		{name: "5 rows", rows: 5},
		{name: "20 rows", rows: 20},
		{name: "50 rows", rows: 50},
		{name: "100 rows", rows: 100},
	}

	for _, tc := range tests {
		tables := []table.Table{buildBenchTable(tc.rows)}

		b.Run(tc.name, func(b *testing.B) {
			m := NewMapper()

			for range b.N {
				if _, err := m.Map(DocTypeBalanceSheet, tables); err != nil {
					b.Fatalf("Map: %v", err)
				}
			}
		})
	}
}

func BenchmarkMapMultipleTables(b *testing.B) {
	tests := []struct {
		name       string
		tableCount int
		rowsPer    int
	}{
		{name: "2 tables x 10 rows", tableCount: 2, rowsPer: 10},
		{name: "5 tables x 20 rows", tableCount: 5, rowsPer: 20},
	}

	for _, tc := range tests {
		tables := make([]table.Table, tc.tableCount)
		for i := range tc.tableCount {
			tables[i] = buildBenchTable(tc.rowsPer)
			tables[i].PageNum = i + 1
		}

		b.Run(tc.name, func(b *testing.B) {
			m := NewMapper()

			for range b.N {
				if _, err := m.Map(DocTypeBalanceSheet, tables); err != nil {
					b.Fatalf("Map: %v", err)
				}
			}
		})
	}
}

func BenchmarkExtractMetadata(b *testing.B) {
	tbl := table.Table{
		Headers: []string{
			"PT Bank Central Asia Tbk",
			"31 Desember 2024",
			"dalam jutaan rupiah",
		},
		Rows: []table.Row{
			{
				Index: 0,
				Cells: []table.Cell{
					{Text: "Kas dan setara kas", Col: 0},
					{Text: "1.234.567", Col: 1},
				},
			},
		},
		PageNum: 1,
	}

	b.ResetTimer()

	for range b.N {
		stmt := &FinancialStatement{Type: DocTypeBalanceSheet}

		m := &mapper{}
		m.extractMetadata(tbl, stmt)
	}
}

// buildBenchTable creates a synthetic financial table with Indonesian-format
// numeric values for benchmarking the mapper.
func buildBenchTable(rowCount int) table.Table {
	labels := []string{
		"Kas dan setara kas",
		"Piutang usaha",
		"Persediaan",
		"Aset tetap",
		"Aset tidak berwujud",
		"Jumlah aset",
		"Utang usaha",
		"Utang bank",
		"Liabilitas jangka panjang",
		"Jumlah liabilitas",
		"Modal saham",
		"Saldo laba",
		"Jumlah ekuitas",
	}

	rows := make([]table.Row, rowCount)
	for i := range rowCount {
		label := labels[i%len(labels)]

		rows[i] = table.Row{
			Index: i,
			Cells: []table.Cell{
				{
					Text: label,
					Row:  i,
					Col:  0,
					Bounds: pdf.Rect{
						X1: 72, Y1: float64(700 - i*14),
						X2: 300, Y2: float64(710 - i*14),
					},
				},
				{
					Text: fmt.Sprintf("%d.%03d.%03d", (i+1)*10, (i*123)%1000, (i*456)%1000),
					Row:  i,
					Col:  1,
					Bounds: pdf.Rect{
						X1: 350, Y1: float64(700 - i*14),
						X2: 450, Y2: float64(710 - i*14),
					},
				},
				{
					Text: fmt.Sprintf("%d.%03d.%03d", (i+2)*8, (i*789)%1000, (i*321)%1000),
					Row:  i,
					Col:  2,
					Bounds: pdf.Rect{
						X1: 460, Y1: float64(700 - i*14),
						X2: 560, Y2: float64(710 - i*14),
					},
				},
			},
		}
	}

	return table.Table{
		Headers: []string{
			"PT Contoh Perusahaan Tbk",
			"31 Desember 2024",
			"31 Desember 2023",
		},
		Columns: []table.Column{
			{Index: 0, X1: 72, X2: 300, Alignment: "left"},
			{Index: 1, X1: 350, X2: 450, Alignment: "right"},
			{Index: 2, X1: 460, X2: 560, Alignment: "right"},
		},
		Rows:    rows,
		PageNum: 2,
		Bounds: pdf.Rect{
			X1: 72, Y1: float64(700 - rowCount*14),
			X2: 560, Y2: 710,
		},
	}
}
