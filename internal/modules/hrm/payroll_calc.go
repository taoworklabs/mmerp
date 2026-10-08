package hrm

import (
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// params are the legal parameters a payroll period uses.
type params struct {
	BaseSalary, MinWage                               decimal.Decimal
	SIEmployee, HIEmployee, UIEmployee                decimal.Decimal
	SIEmployer, HIEmployer, UIEmployer, UnionEmployer decimal.Decimal
	CapMultiplier, Personal, Dependent                decimal.Decimal
	OTMonth, OTYear, HoursPerDay, SkipDays            decimal.Decimal
	Brackets                                          []bracket
}

// paySegment is a run of days under one set of contract terms, with the paid days in it.
type paySegment struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
	Terms    ContractTerms   `json:"terms"`
	PaidDays decimal.Decimal `json:"paid_days"`
	contract int64
}

// overtimeDay is one approved overtime request.
type overtimeDay struct {
	Date       string          `json:"date"`
	DayKind    string          `json:"day_kind"`
	DayHours   decimal.Decimal `json:"day_hours"`
	NightHours decimal.Decimal `json:"night_hours"`
}

// unusedLeave is the leave left when an employee leaves during the period, paid at the
// contract salary of the month before over that month's standard days.
type unusedLeave struct {
	Salary       int64           `json:"salary"`
	StandardDays int             `json:"standard_days"`
	Days         decimal.Decimal `json:"days"`
}

// payInput is everything one employee's pay is computed from; it is kept, encrypted, with the result.
type payInput struct {
	StandardDays int          `json:"standard_days"`
	Segments     []paySegment `json:"segments"`
	// Terms in force on the last employed day of the period: they set the insured salary.
	InsuredTerms ContractTerms `json:"insured_terms"`
	// Working days neither worked nor paid, before hiring and after leaving included.
	UnpaidDays          decimal.Decimal `json:"unpaid_days"`
	Overtime            []overtimeDay   `json:"overtime"`
	OvertimeHoursBefore decimal.Decimal `json:"overtime_hours_before" doc:"earlier this year"`
	UnusedLeave         *unusedLeave    `json:"unused_leave"`
	Adjustment          int64           `json:"adjustment"`
	Dependents          int             `json:"dependents"`
}

func (a *PayrollAmounts) add(b PayrollAmounts) {
	a.Earned += b.Earned
	a.Overtime += b.Overtime
	a.UnusedLeave += b.UnusedLeave
	a.Adjustment += b.Adjustment
	a.Gross += b.Gross
	a.Exempt += b.Exempt
	a.InsuranceBase += b.InsuranceBase
	a.SocialInsurance += b.SocialInsurance
	a.HealthInsurance += b.HealthInsurance
	a.UnemploymentIns += b.UnemploymentIns
	a.PersonalDeduction += b.PersonalDeduction
	a.DependentDeduction += b.DependentDeduction
	a.Taxable += b.Taxable
	a.IncomeTax += b.IncomeTax
	a.Net += b.Net
	a.EmployerSocial += b.EmployerSocial
	a.EmployerHealth += b.EmployerHealth
	a.EmployerUnemploy += b.EmployerUnemploy
	a.EmployerUnion += b.EmployerUnion
	a.Cost += b.Cost
}

// employeeInsurance is what the employee pays: social, health and unemployment insurance.
func (a PayrollAmounts) employeeInsurance() int64 {
	return a.SocialInsurance + a.HealthInsurance + a.UnemploymentIns
}

// employerInsurance is what the company pays on top: the three insurances and the union fee.
func (a PayrollAmounts) employerInsurance() int64 {
	return a.EmployerSocial + a.EmployerHealth + a.EmployerUnemploy + a.EmployerUnion
}

// payResult is one employee's pay, with the overtime hours and the warnings it raised.
type payResult struct {
	PayrollAmounts
	PaidDays      decimal.Decimal `json:"paid_days"`
	OvertimeHours decimal.Decimal `json:"overtime_hours"`
	// overtime_month_limit, overtime_year_limit: hours past the tax-free limits.
	Warnings []string `json:"warnings"`
}

var (
	dayRates   = map[string]decimal.Decimal{"weekday": decimal.RequireFromString("1.5"), "weekly_off": decimal.NewFromInt(2), "holiday": decimal.NewFromInt(3)}
	nightRates = map[string]decimal.Decimal{"weekday": decimal.NewFromInt(2), "weekly_off": decimal.RequireFromString("2.7"), "holiday": decimal.RequireFromString("3.9")}
	// Night hours of a weekday that also had day overtime: 150% + 30% + 20% × 150%.
	weekdayNightAfterDay = decimal.RequireFromString("2.1")
)

// insured is the salary and the allowances: what insurance and the overtime rate go by.
func (t ContractTerms) insured() decimal.Decimal {
	n := t.Salary
	for _, l := range t.Lines {
		if l.Kind == "allowance" {
			n += l.Amount
		}
	}
	return decimal.NewFromInt(n)
}

// total is every amount of the terms; taxFree the supports exempt from tax.
func (t ContractTerms) total() (total, taxFree decimal.Decimal) {
	n, free := t.Salary, int64(0)
	for _, l := range t.Lines {
		n += l.Amount
		if l.Kind == "support" && !l.Taxable {
			free += l.Amount
		}
	}
	return decimal.NewFromInt(n), decimal.NewFromInt(free)
}

// calcPayroll computes every employee's pay of one payroll. In line mode each employee's
// amount of each item is rounded on its own; in total mode each item is summed unrounded
// over the payroll and allocated back with platform.Allocate, item by item, so later items
// (insurance, then tax) build on amounts already rounded.
func calcPayroll(p params, mode platform.Rounding, in []payInput) []payResult {
	n := len(in)
	out := make([]payResult, n)
	col := func(f func(i int) decimal.Decimal) []int64 {
		raw := make([]decimal.Decimal, n)
		for i := range raw {
			raw[i] = f(i)
		}
		if mode == platform.RoundTotal {
			return platform.Allocate(raw)
		}
		res := make([]int64, n)
		for i, r := range raw {
			res[i] = platform.Round(r)
		}
		return res
	}

	// Pay for the paid days: in line mode each segment rounds on its own, as payslips show it.
	std := make([]decimal.Decimal, n)
	for i, x := range in {
		std[i] = decimal.NewFromInt(int64(max(x.StandardDays, 1)))
	}
	segment := func(i int, part func(ContractTerms) decimal.Decimal) decimal.Decimal {
		sum := decimal.Zero
		for _, s := range in[i].Segments {
			v := part(s.Terms).Mul(s.PaidDays).Div(std[i])
			if mode == platform.RoundLine {
				v = decimal.NewFromInt(platform.Round(v))
			}
			sum = sum.Add(v)
		}
		return sum
	}
	earned := col(func(i int) decimal.Decimal {
		return segment(i, func(t ContractTerms) decimal.Decimal { a, _ := t.total(); return a })
	})
	taxFree := col(func(i int) decimal.Decimal {
		return segment(i, func(t ContractTerms) decimal.Decimal { _, f := t.total(); return f })
	})

	otPay, otExempt := make([]decimal.Decimal, n), make([]decimal.Decimal, n)
	for i, x := range in {
		otPay[i], otExempt[i], out[i].OvertimeHours, out[i].Warnings = overtime(p, x, std[i])
	}
	ot := col(func(i int) decimal.Decimal { return otPay[i] })
	exemptOT := col(func(i int) decimal.Decimal { return otExempt[i] })
	leave := col(func(i int) decimal.Decimal {
		l := in[i].UnusedLeave
		if l == nil || l.StandardDays == 0 {
			return decimal.Zero
		}
		return decimal.NewFromInt(l.Salary).Mul(l.Days).Div(decimal.NewFromInt(int64(l.StandardDays)))
	})

	// Insurance on the insured salary in force on the last day, capped; none for a month
	// with too many days neither worked nor paid.
	base := make([]decimal.Decimal, n)
	for i, x := range in {
		if x.UnpaidDays.LessThan(p.SkipDays) {
			base[i] = x.InsuredTerms.insured()
		}
	}
	capSI, capUI := p.BaseSalary.Mul(p.CapMultiplier), p.MinWage.Mul(p.CapMultiplier)
	on := func(capped decimal.Decimal, rate decimal.Decimal) func(int) decimal.Decimal {
		return func(i int) decimal.Decimal { return decimal.Min(base[i], capped).Mul(rate) }
	}
	si, hi, ui := col(on(capSI, p.SIEmployee)), col(on(capSI, p.HIEmployee)), col(on(capUI, p.UIEmployee))
	erSI, erHI, erUI, erUnion := col(on(capSI, p.SIEmployer)), col(on(capSI, p.HIEmployer)), col(on(capUI, p.UIEmployer)), col(on(capSI, p.UnionEmployer))

	personal := platform.Round(p.Personal)
	for i, x := range in {
		a := &out[i].PayrollAmounts
		a.Earned, a.Overtime, a.UnusedLeave, a.Adjustment = earned[i], ot[i], leave[i], x.Adjustment
		a.Gross = a.Earned + a.Overtime + a.UnusedLeave + a.Adjustment
		a.Exempt = exemptOT[i] + a.UnusedLeave + taxFree[i]
		a.InsuranceBase = platform.Round(base[i])
		a.SocialInsurance, a.HealthInsurance, a.UnemploymentIns = si[i], hi[i], ui[i]
		a.EmployerSocial, a.EmployerHealth, a.EmployerUnemploy, a.EmployerUnion = erSI[i], erHI[i], erUI[i], erUnion[i]
		a.PersonalDeduction = personal
		a.DependentDeduction = platform.Round(p.Dependent.Mul(decimal.NewFromInt(int64(x.Dependents))))
		a.Taxable = max(0, a.Gross-a.Exempt-a.employeeInsurance()-a.PersonalDeduction-a.DependentDeduction)
		for _, s := range x.Segments {
			out[i].PaidDays = out[i].PaidDays.Add(s.PaidDays)
		}
	}
	tax := col(func(i int) decimal.Decimal { return incomeTax(p.Brackets, decimal.NewFromInt(out[i].Taxable)) })
	for i := range out {
		a := &out[i].PayrollAmounts
		a.IncomeTax = tax[i]
		a.Net = a.Gross - a.employeeInsurance() - a.IncomeTax
		a.Cost = a.Gross + a.employerInsurance()
	}
	return out
}

// overtime prices an employee's overtime of the month and the part of it that is tax-free:
// the first hours in date order, up to the monthly limit and what is left of the yearly one.
func overtime(p params, x payInput, std decimal.Decimal) (pay, exempt, hours decimal.Decimal, warnings []string) {
	free := decimal.Max(decimal.Zero, decimal.Min(p.OTMonth, p.OTYear.Sub(x.OvertimeHoursBefore)))
	pay, exempt = decimal.Zero, decimal.Zero
	for _, o := range x.Overtime {
		// Hourly rate = insured / standard days / hours per day; multiplied out before dividing, so it stays exact.
		insured, perMonth := termsOn(x.Segments, o.Date).insured(), std.Mul(p.HoursPerDay)
		night := nightRates[o.DayKind]
		if o.DayKind == "weekday" && o.DayHours.IsPositive() {
			night = weekdayNightAfterDay
		}
		for _, part := range []struct{ h, rate decimal.Decimal }{{o.DayHours, dayRates[o.DayKind]}, {o.NightHours, night}} {
			pay = pay.Add(insured.Mul(part.rate).Mul(part.h).Div(perMonth))
			taken := decimal.Min(part.h, free)
			exempt = exempt.Add(insured.Mul(part.rate).Mul(taken).Div(perMonth))
			free = free.Sub(taken)
			hours = hours.Add(part.h)
		}
	}
	if hours.GreaterThan(p.OTMonth) {
		warnings = append(warnings, "overtime_month_limit")
	}
	if x.OvertimeHoursBefore.Add(hours).GreaterThan(p.OTYear) {
		warnings = append(warnings, "overtime_year_limit")
	}
	return pay, exempt, hours, warnings
}

// termsOn picks the terms of the segment holding date, else the last segment's.
func termsOn(segments []paySegment, date string) ContractTerms {
	if len(segments) == 0 {
		return ContractTerms{}
	}
	for _, s := range segments {
		if s.From <= date && date <= s.To {
			return s.Terms
		}
	}
	return segments[len(segments)-1].Terms
}

// incomeTax applies the progressive brackets to the monthly taxable income.
func incomeTax(brackets []bracket, taxable decimal.Decimal) decimal.Decimal {
	tax, prev := decimal.Zero, decimal.Zero
	for _, b := range brackets {
		if !taxable.GreaterThan(prev) {
			break
		}
		top := taxable
		if b.Upper != nil {
			top = decimal.Min(taxable, *b.Upper)
		}
		tax = tax.Add(top.Sub(prev).Mul(b.Rate))
		if b.Upper == nil {
			break
		}
		prev = *b.Upper
	}
	return tax
}
