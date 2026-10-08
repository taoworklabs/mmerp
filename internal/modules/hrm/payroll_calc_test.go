package hrm

import (
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
		for _, l := range postingLines(9, lines) {
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
