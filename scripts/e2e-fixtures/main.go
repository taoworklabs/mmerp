// Command e2e-fixtures writes the Excel files the Playwright tests import:
// go run ./scripts/e2e-fixtures, then commit web/e2e/fixtures.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
)

const dir = "web/e2e/fixtures"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e-fixtures:", err)
		os.Exit(1)
	}
}

// March 2026 for department P: M5E01 works every weekday, M5E02 too plus half of each
// Saturday, M5E03 every weekday from 16/03, when they join.
func march() [][]any {
	header := []any{"Mã NV", "Họ tên"}
	rows := [][]any{{"M5E01", "Nhân viên M5E01"}, {"M5E02", "Nhân viên M5E02"}, {"M5E03", "Nhân viên M5E03"}}
	for d := 1; d <= 31; d++ {
		header = append(header, fmt.Sprintf("%02d/03", d))
		wd := time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC).Weekday()
		weekday := wd != time.Saturday && wd != time.Sunday
		rows[0] = append(rows[0], cell(weekday, 1))
		rows[1] = append(rows[1], cell(weekday, 1, wd == time.Saturday, 0.5))
		rows[2] = append(rows[2], cell(weekday && d >= 16, 1))
	}
	return append([][]any{header}, rows...)
}

// cell is the value of the first condition that holds, or blank.
func cell(pairs ...any) any {
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i].(bool) {
			return pairs[i+1]
		}
	}
	return nil
}

func run() error {
	good := march()
	bad := march()
	bad[1][2+2] = 2     // M5E01, day 3: not a half or a full day
	bad[2][0] = "M5X99" // no such employee
	bad[3][2+1] = 1     // M5E03, day 2: before they join
	for name, rows := range map[string][][]any{"timesheet-2026-03.xlsx": good, "timesheet-2026-03-bad.xlsx": bad} {
		if err := write(filepath.Join(dir, name), rows); err != nil {
			return err
		}
	}
	return nil
}

func write(path string, rows [][]any) error {
	x := excelize.NewFile()
	defer func() { _ = x.Close() }()
	for r, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, r+1)
		if err != nil {
			return err
		}
		if err := x.SetSheetRow("Sheet1", cell, &row); err != nil {
			return err
		}
	}
	return x.SaveAs(path)
}
