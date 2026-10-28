package dataio

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

// A text column is text-formatted as a whole, so cells typed later keep leading zeros.
func TestWriteSheetTextColumns(t *testing.T) {
	b, err := writeSheet(Sheet{Header: []string{"a", "b"}, Rows: [][]any{{"0012", "x"}}, TextColumns: []int{0}})
	if err != nil {
		t.Fatal(err)
	}
	x, err := excelize.OpenReader(bytes.NewReader(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	numFmt := func(get func() (int, error)) int {
		id, err := get()
		if err != nil {
			t.Fatal(err)
		}
		st, err := x.GetStyle(id)
		if err != nil {
			t.Fatal(err)
		}
		return st.NumFmt
	}
	if n := numFmt(func() (int, error) { return x.GetColStyle("Sheet1", "A") }); n != 49 {
		t.Fatalf("column A format %d", n)
	}
	if n := numFmt(func() (int, error) { return x.GetCellStyle("Sheet1", "A2") }); n != 49 {
		t.Fatalf("A2 format %d", n)
	}
	if n := numFmt(func() (int, error) { return x.GetColStyle("Sheet1", "B") }); n == 49 {
		t.Fatal("column B is text")
	}
	if b, _ := x.GetCellStyle("Sheet1", "A1"); b == 0 {
		t.Fatal("header lost its style")
	}
}
