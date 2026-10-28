package hrm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
)

// d02ltFrom is the first month of the QĐ 366/2026 form.
var d02ltFrom = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

// d02Row is one person's movement; kind orders the rows: increases, decreases, adjustments.
type d02Row struct {
	kind      int
	person    int
	label     string
	terms     ContractTerms
	fixed     *bool
	effective time.Time
	to        bool // the movement lasts this month only
	note      string
}

// exportD02LT lists the insurance increases, decreases and adjustments of a month, from the
// insured salary of the posted payrolls of that month and the one before.
func (s *Service) exportD02LT(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	p, err := parseReportParams(params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	month, err := reportMonthOf(p.Month)
	if err != nil {
		return dataio.Sheet{}, err
	}
	if month.Before(d02ltFrom) {
		return dataio.Sheet{}, ErrReportFormNotSupported
	}
	b, err := s.reportBasis(ctx, p.LegalEntityID, month.AddDate(0, -1, 0), 2)
	if err != nil {
		return dataio.Sheet{}, err
	}
	prev, cur := b.Months[0].Lines, b.Months[1].Lines
	ids := b.employeeIDs()
	emps, err := reportEmployees(ctx, ids)
	if err != nil {
		return dataio.Sheet{}, err
	}
	contracts, err := store.ReportContracts.Run(ctx, store.ReportContractsParams{LegalEntityID: p.LegalEntityID, Ids: ids})
	if err != nil {
		return dataio.Sheet{}, err
	}
	monthEnd := month.AddDate(0, 1, -1)
	var moves []d02Row
	maybeBack := map[int64]int{}
	var listed []store.ReportEmployeesRow
	for _, e := range emps {
		pl, inPrev := prev[e.ID]
		cl, inCur := cur[e.ID]
		pb, cb := base(pl, inPrev), base(cl, inCur)
		r := d02Row{person: len(listed)}
		switch {
		case cb > 0 && pb == 0:
			r.kind, r.terms, r.effective = 0, cl.Input.InsuredTerms, month
			r.label, r.effective = "Tăng lao động", joined(contracts, e, month, monthEnd)
			if inPrev {
				// On the payroll without contributing: a return only if once insured here.
				maybeBack[e.ID] = len(moves)
			}
			r.fixed = fixedOn(contracts, e.ID, monthEnd)
		case pb > 0 && cb == 0:
			r.kind, r.terms, r.effective = 1, pl.Input.InsuredTerms, month
			r.fixed = fixedOn(contracts, e.ID, month.AddDate(0, 0, -1))
			switch {
			case e.TerminationDate.Valid && !e.TerminationDate.Time.Before(month.AddDate(0, -1, 0)) && !e.TerminationDate.Time.After(monthEnd):
				r.label, r.effective = "Giảm lao động", e.TerminationDate.Time
			case !inCur:
				r.label, r.note, r.effective = "Giảm lao động", "Chuyển đơn vị", time.Time{}
				if c, ok := contractOn(contracts, e.ID, month.AddDate(0, 0, -1)); ok && c.EndDate.Valid {
					r.effective = c.EndDate.Time
				}
			default:
				r.label, r.to = "Nghỉ không lương, ốm đau từ 14 ngày", true
			}
		case pb > 0 && cb > 0 && pb != cb:
			r.kind, r.terms, r.label = 2, cl.Input.InsuredTerms, "Điều chỉnh tiền lương"
			r.note = "Từ " + dottedThousands(pb) + " lên " + dottedThousands(cb)
			r.fixed = fixedOn(contracts, e.ID, monthEnd)
			if c, ok := contractOn(contracts, e.ID, monthEnd); ok {
				r.effective = c.StartDate.Time
			}
		default:
			continue
		}
		moves = append(moves, r)
		listed = append(listed, e)
	}
	back, err := insuredBefore(ctx, p.LegalEntityID, month.AddDate(0, -2, 0), emps, maybeBack)
	if err != nil {
		return dataio.Sheet{}, err
	}
	for id := range back {
		moves[maybeBack[id]].label, moves[maybeBack[id]].effective = "Đi làm lại", month
	}
	people, err := s.reportPeople(ctx, listed, "social_insurance_no")
	if err != nil {
		return dataio.Sheet{}, err
	}
	rows := make([][]any, 0, len(moves))
	for kind := range 3 {
		for _, m := range moves {
			if m.kind != kind {
				continue
			}
			rows = append(rows, d02Cells(len(rows)+1, people[m.person], m, month))
		}
	}
	name := "D02-LT " + b.Entity.Name + " " + month.Format("01-2006")
	return reportSheet(name, layoutD02LT_QD366_2026, rows, []int{2}, p.LegalEntityID, personIDs(people))
}

func d02Cells(n int, p reportPerson, m d02Row, month time.Time) []any {
	row := make([]any, len(layoutD02LT_QD366_2026))
	for i := range row {
		row[i] = ""
	}
	row[0], row[1], row[2] = int64(n), p.FullName, p.SocialInsuranceNo
	if p.DateOfBirth.Valid {
		row[3] = p.DateOfBirth.Time.Format(dateLayout)
	}
	row[5] = m.terms.Salary
	var allowances int64
	for _, l := range m.terms.Lines {
		if l.Kind == "allowance" {
			allowances += l.Amount
		}
	}
	if allowances > 0 {
		row[10] = allowances
	}
	if m.fixed != nil {
		row[11+boolIndex(*m.fixed)] = "x"
	}
	if !m.effective.IsZero() {
		row[14] = m.effective.Format(dateLayout)
	}
	row[15] = month.Format(monthLayout)
	if m.to {
		row[16] = month.Format(monthLayout)
	}
	row[19], row[20] = m.label, m.note
	return row
}

func boolIndex(b bool) int {
	if b {
		return 1
	}
	return 0
}

// base is the insured salary on a payroll line; 0 without a line or when contributions were waived.
func base(l payrollLine, ok bool) int64 {
	if !ok {
		return 0
	}
	return l.Result.InsuranceBase
}

// contractOn is the posted contract row of emp in force on date: the latest started, an
// addendum winning over its original from its start.
func contractOn(cs []store.ReportContractsRow, emp int64, date time.Time) (store.ReportContractsRow, bool) {
	var out store.ReportContractsRow
	found := false
	for _, c := range cs {
		if c.EmployeeID == emp && !c.StartDate.Time.After(date) && (!c.EndDate.Valid || !c.EndDate.Time.Before(date)) {
			out, found = c, true
		}
	}
	return out, found
}

func fixedOn(cs []store.ReportContractsRow, emp int64, date time.Time) *bool {
	if c, ok := contractOn(cs, emp, date); ok {
		return &c.FixedTerm
	}
	return nil
}

// joined is when an employee new to the legal entity starts: the later of the hire date and
// the start of their first contract with it in force in the month.
func joined(cs []store.ReportContractsRow, e store.ReportEmployeesRow, start, end time.Time) time.Time {
	out := e.HireDate.Time
	for _, c := range cs {
		if c.EmployeeID == e.ID && !c.ParentID.Valid && !c.StartDate.Time.After(end) && (!c.EndDate.Valid || !c.EndDate.Time.Before(start)) {
			if c.StartDate.Time.After(out) {
				out = c.StartDate.Time
			}
			break
		}
	}
	return out
}

// insuredBefore tells which of the employees in ids contributed on a posted payroll of the legal
// entity, walking back month by month from from. Leaving the entity's payroll ends the walk for
// them; where the payroll history starts, whoever was hired by then is taken as insured
// before it.
func insuredBefore(ctx context.Context, entity int64, from time.Time, emps []store.ReportEmployeesRow, ids map[int64]int) (map[int64]bool, error) {
	open := map[int64]store.ReportEmployeesRow{}
	for _, e := range emps {
		if _, ok := ids[e.ID]; ok {
			open[e.ID] = e
		}
	}
	out := map[int64]bool{}
	for m := from; len(open) > 0; m = m.AddDate(0, -1, 0) {
		id, err := store.PostedPayroll.Run(ctx, store.PostedPayrollParams{LegalEntityID: entity, PeriodStart: pgtype.Date{Time: m, Valid: true}})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && !id.Valid {
			for i, e := range open {
				if e.HireDate.Time.Before(m.AddDate(0, 1, 0)) {
					out[i] = true
				}
			}
			break
		}
		if err != nil {
			return nil, err
		}
		rows, err := store.PayrollLines.Run(ctx, id.Int64)
		if err != nil {
			return nil, err
		}
		seen := map[int64]bool{}
		for _, r := range rows {
			if _, ok := open[r.EmployeeID]; !ok {
				continue
			}
			seen[r.EmployeeID] = true
			l, err := openLine(ctx, r.Data)
			if err != nil {
				return nil, err
			}
			if l.Result.InsuranceBase > 0 {
				out[r.EmployeeID] = true
				delete(open, r.EmployeeID)
			}
		}
		for i := range open {
			if !seen[i] {
				delete(open, i)
			}
		}
	}
	return out, nil
}
