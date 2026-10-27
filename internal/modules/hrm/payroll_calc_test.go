package hrm

import (
	"maps"
	"slices"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// params2026 are the defaults of the migration for April 2026, region I.
func params2026(t *testing.T) params {
	b, ok := parseBrackets("[[10000000,0.05],[30000000,0.1],[60000000,0.2],[100000000,0.3],[null,0.35]]")
	if !ok {
		t.Fatal("brackets")
	}
	return params{
		BaseSalary: dec("2340000"), MinWage: dec("5310000"), SIEmployee: dec("0.08"), HIEmployee: dec("0.015"), UIEmployee: dec("0.01"),
		SIEmployer: dec("0.175"), HIEmployer: dec("0.03"), UIEmployer: dec("0.01"), UnionEmployer: dec("0.02"), CapMultiplier: dec("20"),
		Personal: dec("15500000"), Dependent: dec("6200000"), OTMonth: dec("40"), OTYear: dec("200"), HoursPerDay: dec("8"), SkipDays: dec("14"),
		Brackets: b,
	}
}

func full(salary int64, paid string) payInput {
	terms := ContractTerms{Salary: salary}
	return payInput{StandardDays: 22, Segments: []paySegment{{From: "2026-04-01", To: "2026-04-30", Terms: terms, PaidDays: dec(paid)}},
		InsuredTerms: terms, UnpaidDays: dec("22").Sub(dec(paid))}
}

func TestCalcItems(t *testing.T) {
	p := params2026(t)
	night := full(22_000_000, "22") // 125.000/hour
	night.Overtime = []overtimeDay{
		{Date: "2026-04-04", DayKind: "weekly_off", NightHours: dec("2")},                  // 270%
		{Date: "2026-04-30", DayKind: "holiday", NightHours: dec("2")},                     // 390%
		{Date: "2026-04-07", DayKind: "weekday", DayHours: dec("2"), NightHours: dec("2")}, // 150%, then 210%
	}
	yearly := full(22_000_000, "22")
	yearly.OvertimeHoursBefore = dec("196") // 4 of the 8 hours are still tax-free
	yearly.Overtime = []overtimeDay{{Date: "2026-04-07", DayKind: "weekday", DayHours: dec("8")}}
	skip := full(22_000_000, "8") // 14 unpaid days: no insurance
	free := full(10_000_000, "22")
	free.Segments[0].Terms.Lines = []ContractLine{{Kind: "support", Name: "Ăn ca", Amount: 730_000}}
	got := calcPayroll(p, platform.RoundLine, []payInput{night, yearly, skip, free})

	if got[0].Overtime != 125_000*(2*27+2*39+2*15+2*21)/10 || got[0].OvertimeHours.String() != "8" || len(got[0].Warnings) != 0 {
		t.Errorf("night rates: %d %s %v", got[0].Overtime, got[0].OvertimeHours, got[0].Warnings)
	}
	if got[1].Overtime != 1_500_000 || got[1].Exempt != 750_000 || !slices.Equal(got[1].Warnings, []string{"overtime_year_limit"}) {
		t.Errorf("yearly limit: %d exempt %d %v", got[1].Overtime, got[1].Exempt, got[1].Warnings)
	}
	if got[2].InsuranceBase != 0 || got[2].SocialInsurance != 0 || got[2].EmployerUnion != 0 || got[2].Earned != 8_000_000 {
		t.Errorf("14 days: %+v", got[2].PayrollAmounts)
	}
	if got[3].Earned != 10_730_000 || got[3].Exempt != 730_000 || got[3].InsuranceBase != 10_000_000 {
		t.Errorf("tax-free support: %+v", got[3].PayrollAmounts)
	}
}

// In both modes the lines add up to the document and the posting lines balance; in total
// mode each item's sum is the rounded sum of the unrounded amounts.
func TestCalcRoundingModes(t *testing.T) {
	p := params2026(t)
	var in []payInput
	for _, s := range []int64{10_000_001, 10_000_003, 23_456_789, 31_111_111, 47_000_005} {
		x := full(s, "7.5")
		x.UnpaidDays = dec("3")
		x.Overtime = []overtimeDay{{Date: "2026-04-08", DayKind: "weekday", DayHours: dec("1.5")}}
		in = append(in, x)
	}
	for _, mode := range []platform.Rounding{platform.RoundLine, platform.RoundTotal} {
		res := calcPayroll(p, mode, in)
		lines := make([]PayrollLine, len(res))
		var sum PayrollAmounts
		for i, r := range res {
			lines[i] = PayrollLine{OrgUnitID: int64(1 + i%2), PayrollAmounts: r.PayrollAmounts}
			sum.add(r.PayrollAmounts)
			if r.Gross != r.Net+r.employeeInsurance()+r.IncomeTax || r.Cost != r.Gross+r.employerInsurance() {
				t.Errorf("%s: line %d does not add up: %+v", mode, i, r.PayrollAmounts)
			}
		}
		var debit, credit int64
		for _, l := range postingLines(9, nil, lines) {
			switch l.Kind {
			case posting.SalaryExpense, posting.EmployerInsuranceExpense:
				debit += l.Amount
			default:
				credit += l.Amount
			}
		}
		if debit != sum.Cost || credit != sum.Cost {
			t.Errorf("%s: posting %d / %d, document %d", mode, debit, credit, sum.Cost)
		}
		if mode != platform.RoundTotal {
			continue
		}
		raw := decimal.Zero
		for _, x := range in {
			raw = raw.Add(decimal.NewFromInt(x.Segments[0].Terms.Salary).Mul(x.Segments[0].PaidDays).Div(dec("22")))
		}
		if sum.Earned != platform.Round(raw) {
			t.Errorf("total: earned %d, want %d", sum.Earned, platform.Round(raw))
		}
	}
}

// postingTree is le=1 → A=2 → {A1=3, A2=4}, and B=5 under le.
var postingTree = map[int64]int64{2: 1, 3: 2, 4: 2, 5: 1}

// payInUnits gives each unit n payroll lines, all with the same amounts.
func payInUnits(n map[int64]int) []PayrollLine {
	var out []PayrollLine
	for unit, k := range n {
		for range k {
			out = append(out, PayrollLine{OrgUnitID: unit, PayrollAmounts: PayrollAmounts{Gross: 100, Net: 80, Cost: 100}})
		}
	}
	return out
}

// expenseByUnit returns the salary expense posted on each unit.
func expenseByUnit(lines []posting.Line) map[int64]int64 {
	out := map[int64]int64{}
	for _, l := range lines {
		if l.Kind == posting.SalaryExpense {
			out[l.OrgUnitID] += l.Amount
		}
	}
	return out
}

func TestPostingLinesMinimumGroup(t *testing.T) {
	cases := []struct {
		name string
		pay  map[int64]int
		want map[int64]int64
	}{
		{"unit at the threshold keeps its line", map[int64]int{3: 3}, map[int64]int64{3: 300}},
		{"small unit rolls into its parent", map[int64]int{3: 2, 2: 3}, map[int64]int64{2: 500}},
		{"small siblings reach the threshold together", map[int64]int{3: 1, 4: 2}, map[int64]int64{2: 300}},
		{"rolls up to the legal entity", map[int64]int{5: 1, 3: 1}, map[int64]int64{1: 200}},
		{"employees on the legal entity itself", map[int64]int{1: 2}, map[int64]int64{1: 200}},
		{"unknown unit rolls to the legal entity", map[int64]int{99: 1}, map[int64]int64{1: 100}},
		{"large and small mixed", map[int64]int{3: 4, 4: 1, 5: 3}, map[int64]int64{3: 400, 1: 100, 5: 300}},
		{"small unit two levels down reaches a legal entity with its own people", map[int64]int{3: 1, 1: 1}, map[int64]int64{1: 200}},
	}
	for _, c := range cases {
		got := expenseByUnit(postingLines(1, postingTree, payInUnits(c.pay)))
		if !maps.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPostingLinesConserveAmounts(t *testing.T) {
	lines := payInUnits(map[int64]int{1: 1, 2: 2, 3: 1, 4: 5, 5: 2})
	for i := range lines {
		lines[i].Gross += int64(i)
		lines[i].Net += int64(2 * i)
		lines[i].IncomeTax = int64(i)
	}
	want := map[posting.Kind]int64{}
	for _, l := range postingLines(1, nil, lines) {
		want[l.Kind] += l.Amount
	}
	got := map[posting.Kind]int64{}
	for _, l := range postingLines(1, postingTree, lines) {
		got[l.Kind] += l.Amount
	}
	if !maps.Equal(got, want) {
		t.Errorf("sums by kind: got %v, want %v", got, want)
	}
}
