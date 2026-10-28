package hrm_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

// yearTax is the 2026 schedule on a year's assessable income: 5/10/20/30/35% above
// 0, 120M, 360M, 720M and 1.2 billion.
func yearTax(n int64) int64 {
	var hundredths, prev int64
	for _, b := range []struct{ upper, percent int64 }{{120_000_000, 5}, {360_000_000, 10}, {720_000_000, 20}, {1_200_000_000, 30}, {1 << 62, 35}} {
		if n <= prev {
			break
		}
		hundredths += (min(n, b.upper) - prev) * b.percent
		prev = b.upper
	}
	return (hundredths + 50) / 100
}

// worked is every working day of month from the fixture's hire date, 5 January.
func worked(month string) []string {
	if month == "2026-01" {
		return workdays("2026-01-05", "2026-01-31")
	}
	return full(month)
}

// 05-1/BK columns, 0-based: [n] at n-6.
func bk(code int) int { return code - 6 }

// The whole of 2026 in C: E with a dependant from March is there all year; F, with only a
// national ID, leaves at the end of June.
func TestPITYear(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 40_000_000}, "2026-01-01", "2026-12-31")
	if _, err := f.hrm.SaveDependent(f.pay, f.emp, 0, hrm.DependentInput{FullName: "Con E", Relationship: "child", DeductionFrom: str("2026-03")}); err != nil {
		t.Fatal(err)
	}
	gone := f.hire("F", f.a, "2026-01-05", 20_000_000)
	f.setIdentifiers(gone, "", "", "001099000002")
	f.edit(gone, func(e *hrm.EmployeeFields) { e.TerminationDate = str("2026-06-30") })
	sums := map[int64]*hrm.PayrollAmounts{f.emp: {}, gone: {}}
	for m := 1; m <= 12; m++ {
		month := "2026-" + pad2(m)
		days := map[int64][]string{f.emp: worked(month)}
		if m <= 6 {
			days[gone] = worked(month)
		}
		id := f.close(f.c, month, map[int64]map[int64][]string{f.a: days})
		for _, l := range f.payrollDoc(id).Lines {
			s := sums[l.EmployeeID]
			s.Gross += l.Gross
			s.Exempt += l.Exempt
			s.SocialInsurance += l.SocialInsurance + l.HealthInsurance + l.UnemploymentIns
			s.PersonalDeduction += l.PersonalDeduction + l.DependentDeduction
			s.DependentDeduction += l.DependentDeduction
			s.IncomeTax += l.IncomeTax
		}
	}
	e, fl := sums[f.emp], sums[gone]
	params := `{"legal_entity_id":` + itoa(f.c) + `,"year":2026}`

	rows, name := f.exportReport(f.rep, "hrm.report.pit_05_1bk", params)
	if name != "05-1BK-QTT-TNCN C 2026.xlsx" {
		t.Errorf("file name %q", name)
	}
	tnct := e.Gross - e.Exempt
	assessable := tnct - e.PersonalDeduction - e.SocialInsurance
	due := yearTax(assessable)
	er := rowOf(t, rows, 1, "Nhân viên E")
	wantCells(t, er, map[int]string{bk(8): "8000000001", bk(9): "", bk(10): "x", bk(12): itoa(tnct), bk(13): "0", bk(15): "1",
		bk(16): itoa(e.PersonalDeduction), bk(18): itoa(e.SocialInsurance), bk(20): itoa(assessable), bk(21): itoa(e.IncomeTax),
		bk(22): itoa(due), bk(23): itoa(max(0, e.IncomeTax-due)), bk(24): itoa(max(0, due-e.IncomeTax))})
	t.Run("left mid-year", func(t *testing.T) {
		wantCells(t, rowOf(t, rows, 1, "Nhân viên F"), map[int]string{bk(8): "", bk(9): "001099000002", bk(10): "",
			bk(12): itoa(fl.Gross - fl.Exempt), bk(21): itoa(fl.IncomeTax), bk(22): "", bk(23): "", bk(24): "", bk(25): ""})
	})
	total := rowOf(t, rows, 1, "Tổng")
	wantCells(t, total, map[int]string{bk(12): itoa(tnct + fl.Gross - fl.Exempt), bk(21): itoa(e.IncomeTax + fl.IncomeTax), bk(22): itoa(due)})

	rows, _ = f.exportReport(f.rep, "hrm.report.pit_05qtt", params)
	under := max(0, due-e.IncomeTax)
	waived := int64(0)
	if under > 0 && under <= 50_000 {
		waived = under
	}
	taxed := int64(0)
	if e.IncomeTax > 0 {
		taxed++
	}
	if fl.IncomeTax > 0 {
		taxed++
	}
	wantIndicators(t, indicators(t, rows), map[string]int64{"[15]": 2, "[16]": 2, "[17]": taxed, "[18]": taxed, "[21]": 1,
		"[22]": tnct + fl.Gross - fl.Exempt, "[23]": tnct + fl.Gross - fl.Exempt, "[26]": 0,
		"[33]": e.IncomeTax + fl.IncomeTax, "[34]": e.IncomeTax + fl.IncomeTax,
		"[37]": 1, "[38]": e.IncomeTax, "[39]": 0, "[40]": due, "[41]": waived,
		"[44]": max(0, due-e.IncomeTax-waived), "[45]": max(0, e.IncomeTax-due)})
}

func pad2(m int) string {
	if m < 10 {
		return "0" + itoa(int64(m))
	}
	return itoa(int64(m))
}

func TestPITYearPeriods(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 22_000_000}, "2026-01-01", "2026-12-31")
	for m := 1; m <= 12; m++ {
		if m != 11 {
			month := "2026-" + pad2(m)
			f.close(f.c, month, map[int64]map[int64][]string{f.a: {f.emp: worked(month)}})
		}
	}
	for _, kind := range []string{"hrm.report.pit_05qtt", "hrm.report.pit_05_1bk"} {
		job := f.reportJob(f.rep, kind, `{"legal_entity_id":`+itoa(f.c)+`,"year":2026}`, "report_payroll_missing")
		if out := f.jobError(job); out != `{"code":"report_payroll_missing","params":{"months":"11/2026"}}` {
			t.Errorf("%s: %s", kind, out)
		}
		f.reportJob(f.rep, kind, `{"legal_entity_id":`+itoa(f.c)+`,"year":2025}`, "report_form_not_supported")
	}
}

// 05-3/BK columns, 0-based: [n] at n-6, like 05-1/BK.

// Dependants of 2026: one deducted since 2025, one for part of the year, one ended in 2025.
func TestPIT05_3BK(t *testing.T) {
	f := newReportFixture(t)
	f.contract(f.emp, sample{salary: 30_000_000}, "2026-01-01", "2026-12-31")
	for _, d := range []hrm.DependentInput{
		{FullName: "Con A", Relationship: "child", DateOfBirth: str("2020-02-03"), IDNumber: str("0123456789"), DeductionFrom: str("2025-06")},
		{FullName: "Mẹ B", Relationship: "parent", IDNumber: str("001099012345"), DeductionFrom: str("2026-03"), DeductionTo: str("2026-08")},
		{FullName: "Con C", Relationship: "child", DeductionFrom: str("2024-01"), DeductionTo: str("2025-12")},
		// Registered without a first month: no payroll deducts them, so neither does the list.
		{FullName: "Con D", Relationship: "child"},
	} {
		if _, err := f.hrm.SaveDependent(f.pay, f.emp, 0, d); err != nil {
			t.Fatal(err)
		}
	}
	for m := 1; m <= 12; m++ {
		month := "2026-" + pad2(m)
		f.close(f.c, month, map[int64]map[int64][]string{f.a: {f.emp: worked(month)}})
	}
	before := f.auditCount("hrm.dependents_viewed")
	job := f.reportJob(f.rep, "hrm.report.pit_05_3bk", `{"legal_entity_id":`+itoa(f.c)+`,"year":2026}`, "completed")
	j, err := f.core.DataIO.Job(f.rep, job)
	f.check(err)
	// Two rows of one employee: the pin holds the employee once, and the file downloads.
	name, file, err := f.core.DataIO.File(f.rep, *j.FileID)
	f.check(err)
	f.check(file.Close())
	if name != "05-3BK-QTT-TNCN C 2026.xlsx" {
		t.Errorf("file name %q", name)
	}
	rows, _ := f.exportReport(f.rep, "hrm.report.pit_05_3bk", `{"legal_entity_id":`+itoa(f.c)+`,"year":2026}`)
	if len(rows) != 3 {
		t.Fatalf("rows %v", rows)
	}
	wantCells(t, rowOf(t, rows, bk(9), "Con A"), map[int]string{bk(7): "Nhân viên E", bk(8): "8000000001", bk(10): "03/02/2020",
		bk(11): "0123456789", bk(12): "", bk(13): "", bk(14): "Con", bk(15): "01/2026", bk(16): "12/2026"})
	wantCells(t, rowOf(t, rows, bk(9), "Mẹ B"), map[int]string{bk(11): "", bk(12): "Số ĐDCN", bk(13): "001099012345",
		bk(14): "Cha/mẹ", bk(15): "03/2026", bk(16): "08/2026"})
	if got := f.auditCount("hrm.dependents_viewed") - before; got != 2 {
		t.Errorf("%d dependants reads audited, want one per export", got)
	}
}
