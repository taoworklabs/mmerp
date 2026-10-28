package hrm

import (
	"time"

	"github.com/shopspring/decimal"
)

// Annual leave of a full year under normal conditions (Labour Code 2019, Article 113.1a).
const annualLeaveDays = 12

// monthCounted: the employee is employed on at least half of the month's working days
// (Decree 145/2020, Article 66.2, without subtracting unpaid leave inside the span).
func monthCounted(cal calendar, hire string, termination *string, year, month int) bool {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	from, to := first, last
	if h := dayOf(hire); h.After(from) {
		from = h
	}
	if termination != nil && dayOf(*termination).Before(to) {
		to = dayOf(*termination)
	}
	all := cal.workingDays(first, last)
	if all == 0 || to.Before(from) {
		return false
	}
	return 2*cal.workingDays(from, to) >= all
}

// seniorityDays is one more day per five full years since hire, counted at the month's end
// (Article 114).
func seniorityDays(hire string, year, month int) int {
	h := dayOf(hire)
	end := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	years := end.Year() - h.Year()
	if end.Before(h.AddDate(years, 0, 0)) {
		years--
	}
	return max(years, 0) / 5
}

// entitlementThrough is the year's entitlement earned by the end of month: (12 + seniority)/12
// per counted month (Decree 145/2020, Article 66.1), rounded down to a half day.
func entitlementThrough(cal calendar, hire string, termination *string, year, month int) decimal.Decimal {
	// Twelfths, summed as integers so 13/12 twelve times is exactly 13.
	twelfths := 0
	for m := 1; m <= month; m++ {
		if monthCounted(cal, hire, termination, year, m) {
			twelfths += annualLeaveDays + seniorityDays(hire, year, m)
		}
	}
	return decimal.NewFromInt(int64(twelfths / 6)).Div(decimal.NewFromInt(2))
}

// monthCredit is what month adds to the year's rounded entitlement.
func monthCredit(cal calendar, hire string, termination *string, year, month int) decimal.Decimal {
	return entitlementThrough(cal, hire, termination, year, month).Sub(entitlementThrough(cal, hire, termination, year, month-1))
}
