package domain

import "github.com/lugassawan/idxlens/internal/table"

// GenericTable represents raw table data extracted from a PDF page without
// any financial domain mapping applied.
type GenericTable struct {
	PageNum int        `json:"page_num"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// ExtractGenericTables converts detected tables into generic table
// representations. Each row's cells become a string slice and all values
// remain as raw strings without number parsing.
func ExtractGenericTables(tables []table.Table) []GenericTable {
	if len(tables) == 0 {
		return nil
	}

	result := make([]GenericTable, 0, len(tables))
	for _, tbl := range tables {
		gt := GenericTable{
			PageNum: tbl.PageNum,
			Headers: tbl.Headers,
			Rows:    convertRows(tbl.Rows),
		}
		result = append(result, gt)
	}

	return result
}

func convertRows(rows []table.Row) [][]string {
	if len(rows) == 0 {
		return nil
	}

	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, cell.Text)
		}
		result = append(result, cells)
	}

	return result
}
