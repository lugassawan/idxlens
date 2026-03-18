package table

import "testing"

// assertTableResult validates the structure of detected tables against
// expected values.
func assertTableResult(
	t *testing.T,
	tables []Table,
	wantPageNum int,
	wantTables int,
	wantColumns int,
	wantRows int,
	wantHeaders []string,
) {
	t.Helper()

	if len(tables) != wantTables {
		t.Fatalf("got %d tables, want %d", len(tables), wantTables)
	}

	if wantTables == 0 {
		return
	}

	tbl := tables[0]

	if wantColumns > 0 && len(tbl.Columns) != wantColumns {
		t.Errorf("got %d columns, want %d", len(tbl.Columns), wantColumns)
	}

	if wantRows > 0 && len(tbl.Rows) != wantRows {
		t.Errorf("got %d rows, want %d", len(tbl.Rows), wantRows)
	}

	if wantHeaders != nil {
		if len(tbl.Headers) != len(wantHeaders) {
			t.Fatalf("got %d headers, want %d", len(tbl.Headers), len(wantHeaders))
		}
		for i, want := range wantHeaders {
			if tbl.Headers[i] != want {
				t.Errorf("header[%d] = %q, want %q", i, tbl.Headers[i], want)
			}
		}
	}

	if tbl.PageNum != wantPageNum {
		t.Errorf("PageNum = %d, want %d", tbl.PageNum, wantPageNum)
	}
}
