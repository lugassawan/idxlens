package domain

import (
	"testing"

	"github.com/lugassawan/idxlens/internal/table"
)

func TestExtractGenericTables(t *testing.T) {
	tests := []struct {
		name   string
		tables []table.Table
		want   []GenericTable
	}{
		{
			name:   "empty input returns nil",
			tables: nil,
			want:   nil,
		},
		{
			name: "single table with headers and data rows",
			tables: []table.Table{
				{
					PageNum: 1,
					Headers: []string{"Name", "Value"},
					Rows: []table.Row{
						{
							Index: 0,
							Cells: []table.Cell{
								{Text: "Name", Row: 0, Col: 0},
								{Text: "Value", Row: 0, Col: 1},
							},
						},
						{
							Index: 1,
							Cells: []table.Cell{
								{Text: "Revenue", Row: 1, Col: 0},
								{Text: "1000", Row: 1, Col: 1},
							},
						},
						{
							Index: 2,
							Cells: []table.Cell{
								{Text: "Expense", Row: 2, Col: 0},
								{Text: "500", Row: 2, Col: 1},
							},
						},
					},
				},
			},
			want: []GenericTable{
				{
					PageNum: 1,
					Headers: []string{"Name", "Value"},
					Rows: [][]string{
						{"Name", "Value"},
						{"Revenue", "1000"},
						{"Expense", "500"},
					},
				},
			},
		},
		{
			name: "table with no headers",
			tables: []table.Table{
				{
					PageNum: 3,
					Headers: nil,
					Rows: []table.Row{
						{
							Index: 0,
							Cells: []table.Cell{
								{Text: "A", Row: 0, Col: 0},
								{Text: "B", Row: 0, Col: 1},
							},
						},
					},
				},
			},
			want: []GenericTable{
				{
					PageNum: 3,
					Headers: nil,
					Rows: [][]string{
						{"A", "B"},
					},
				},
			},
		},
		{
			name: "empty table with no rows",
			tables: []table.Table{
				{
					PageNum: 2,
					Headers: []string{"Col1"},
					Rows:    nil,
				},
			},
			want: []GenericTable{
				{
					PageNum: 2,
					Headers: []string{"Col1"},
					Rows:    nil,
				},
			},
		},
		{
			name: "multiple tables",
			tables: []table.Table{
				{
					PageNum: 1,
					Headers: []string{"X"},
					Rows: []table.Row{
						{
							Index: 0,
							Cells: []table.Cell{
								{Text: "X", Row: 0, Col: 0},
							},
						},
						{
							Index: 1,
							Cells: []table.Cell{
								{Text: "1", Row: 1, Col: 0},
							},
						},
					},
				},
				{
					PageNum: 5,
					Headers: []string{"Y", "Z"},
					Rows: []table.Row{
						{
							Index: 0,
							Cells: []table.Cell{
								{Text: "Y", Row: 0, Col: 0},
								{Text: "Z", Row: 0, Col: 1},
							},
						},
					},
				},
			},
			want: []GenericTable{
				{
					PageNum: 1,
					Headers: []string{"X"},
					Rows: [][]string{
						{"X"},
						{"1"},
					},
				},
				{
					PageNum: 5,
					Headers: []string{"Y", "Z"},
					Rows: [][]string{
						{"Y", "Z"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGenericTables(tt.tables)
			if !genericTablesEqual(got, tt.want) {
				t.Errorf("ExtractGenericTables() =\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func genericTablesEqual(a, b []GenericTable) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].PageNum != b[i].PageNum {
			return false
		}

		if !strSliceEqual(a[i].Headers, b[i].Headers) {
			return false
		}

		if len(a[i].Rows) != len(b[i].Rows) {
			return false
		}

		for j := range a[i].Rows {
			if !strSliceEqual(a[i].Rows[j], b[i].Rows[j]) {
				return false
			}
		}
	}

	return true
}

func strSliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
