package hrm

import (
	"time"

	"github.com/shopspring/decimal"
)

// allowanceKind is what leaving pays: the severance allowance (Art 46 Labour Code 2019),
// the job-loss allowance (Art 47) or nothing.
type allowanceKind string

const (
	allowanceNone      allowanceKind = "none"
	allowanceSeverance allowanceKind = "severance"
	allowanceJobLoss   allowanceKind = "job_loss"
)

// terminationReasons are the grounds of Art 34 Labour Code 2019, each with what it pays
// (Art 46.1, 47.1). The codes are stable: reports read them.
var terminationReasons = []struct {
	Code string
	Kind allowanceKind
}{
	{"contract_expired", allowanceSeverance},     // 34.1
	{"work_completed", allowanceSeverance},       // 34.2
	{"mutual_agreement", allowanceSeverance},     // 34.3
	{"imprisonment", allowanceSeverance},         // 34.4
	{"foreigner_expelled", allowanceNone},        // 34.5
	{"death", allowanceSeverance},                // 34.6
	{"employer_ceased", allowanceSeverance},      // 34.7
	{"dismissal", allowanceNone},                 // 34.8
	{"employee_resignation", allowanceSeverance}, // 34.9
	{"employer_unilateral", allowanceSeverance},  // 34.10
	{"absent_without_reason", allowanceNone},     // 34.10 under 36.1(e), excluded by 46.1
	{"restructuring", allowanceJobLoss},          // 34.11
	{"work_permit_expired", allowanceNone},       // 34.12
	{"probation_failed", allowanceNone},          // 34.13
}

func reasonAllowance(code string) (allowanceKind, bool) {
	for _, r := range terminationReasons {
		if r.Code == code {
			return r.Kind, true
		}
	}
	return "", false
}

// monthsWorked counts the whole months from hire to the termination date inclusive; the
// days left over are dropped.
func monthsWorked(hire, termination string) int {
	h, end := dayOf(hire), dayOf(termination).AddDate(0, 0, 1)
	n := (end.Year()-h.Year())*12 + int(end.Month()-h.Month())
	if addMonths(h, n).After(end) {
		n--
	}
	return max(n, 0)
}

// addMonths moves t by n months, clamping to the last day of the month instead of overflowing.
func addMonths(t time.Time, n int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	return first.AddDate(0, 0, min(t.Day(), last)-1)
}

// allowanceYears turns months into years: a remainder up to 6 months is half a year, more
// is a whole one (Art 8.3 Decree 145/2020).
func allowanceYears(months int) decimal.Decimal {
	years := decimal.NewFromInt(int64(months / 12))
	switch rest := months % 12; {
	case rest > 6:
		return years.Add(one)
	case rest > 0:
		return years.Add(decimal.New(5, -1))
	}
	return years
}

// allowanceBasis is a leaving employee's non-sensitive allowance inputs, in months.
type allowanceBasis struct {
	Kind    allowanceKind `json:"kind"`
	Worked  int           `json:"worked"`
	Insured int           `json:"insured" doc:"In unemployment insurance"`
	Paid    int           `json:"paid" doc:"Already paid an allowance for"`
	Pension bool          `json:"pension"`
}

// kind is what is paid: nothing under 12 months worked, and no severance to someone
// eligible for a pension (Art 46.1).
func (b allowanceBasis) kind() allowanceKind {
	if b.Worked < 12 || b.Kind == allowanceSeverance && b.Pension {
		return allowanceNone
	}
	return b.Kind
}

// years leaves out the time in unemployment insurance and the time already paid (Art 46.2).
func (b allowanceBasis) years() decimal.Decimal {
	return allowanceYears(max(0, b.Worked-b.Insured-b.Paid))
}

// allowanceAmount is half a month's average salary per year for severance, one for job
// loss with at least two months (Art 47.1). Unrounded.
func allowanceAmount(kind allowanceKind, years, average decimal.Decimal) decimal.Decimal {
	switch kind {
	case allowanceSeverance:
		return years.Mul(average).Div(decimal.NewFromInt(2))
	case allowanceJobLoss:
		return decimal.Max(years, decimal.NewFromInt(2)).Mul(average)
	}
	return decimal.Zero
}
