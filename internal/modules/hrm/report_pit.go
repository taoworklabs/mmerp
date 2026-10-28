package hrm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// pitFrom is the first period of the Thông tư 89/2026 forms.
var pitFrom = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

// pitPerson sums one employee's taxable pay over the months they have a line.
type pitPerson struct {
	TNCT, Exempt, Insurance, Deductions, Assessable, Tax int64
	Dependents                                           int // on their last line
	DependentDeduction                                   int64
}

// sumPIT adds up each employee's pay over the months; taxable income (TNCT) is what was
// paid less the tax-free part, recognised in the month of the payroll that pays it.
func sumPIT(months []reportMonth) map[int64]*pitPerson {
	out := map[int64]*pitPerson{}
	for _, m := range months {
		for id, l := range m.Lines {
			p := out[id]
			if p == nil {
				p = &pitPerson{}
				out[id] = p
			}
			a := l.Result.PayrollAmounts
			p.TNCT += a.Gross - a.Exempt
			p.Exempt += a.Exempt
			p.Insurance += a.employeeInsurance()
			p.Deductions += a.PersonalDeduction + a.DependentDeduction
			p.DependentDeduction += a.DependentDeduction
			p.Assessable += a.Taxable
			p.Tax += a.IncomeTax
			p.Dependents = l.Input.Dependents
		}
	}
	return out
}

// withholding is the part of 05/KK and of section I of 05/QTT the two forms share, by code
// of 05/KK; every employee is a resident with a labour contract.
func withholding(people map[int64]*pitPerson) map[string]int64 {
	v := map[string]int64{"[15]": int64(len(people)), "[16]": int64(len(people))}
	for _, p := range people {
		v["[20]"] += p.TNCT
		v["[24]"] += p.Exempt
		v["[32]"] += p.Assessable
		v["[35]"] += p.Tax
		if p.Tax > 0 {
			v["[17]"]++
			v["[29]"] += p.TNCT
		}
	}
	v["[18]"], v["[21]"], v["[28]"], v["[30]"], v["[33]"], v["[42]"], v["[43]"] = v["[17]"], v["[20]"], v["[24]"], v["[29]"], v["[32]"], v["[35]"], v["[35]"]
	return v
}

// indicatorRows writes the header fields, then one row per indicator; a code without a value is 0.
func indicatorRows(entity iam.OrgUnit, period string, layout []indicator, values map[string]int64) [][]any {
	name, tax, address := entityHeader(entity)
	rows := [][]any{
		{"", "Kỳ tính thuế", "[01]", period},
		{"", "Tên người nộp thuế", "[04]", name},
		{"", "Mã số thuế", "[05]", tax},
		{"", "Địa chỉ", "[06]", address},
	}
	for _, in := range layout {
		var v any = values[in.Code]
		if in.Code == "" {
			v = "" // a section title
		}
		rows = append(rows, []any{in.STT, in.Label, in.Code, v})
	}
	return rows
}

var indicatorHeader = []string{"STT", "Chỉ tiêu", "Mã", "Giá trị"}

// pitBasis reads the months and checks the actor may see every employee they add up.
func (s *Service) pitBasis(ctx context.Context, entity int64, first time.Time, months int) (reportBasis, map[int64]*pitPerson, error) {
	b, err := s.reportBasis(ctx, entity, first, months)
	if err != nil {
		return b, nil, err
	}
	emps, err := reportEmployees(ctx, b.employeeIDs())
	if err == nil {
		err = s.requireReportSensitive(ctx, emps)
	}
	return b, sumPIT(b.Months), err
}

// exportPIT05KK is the quarterly PIT return of a legal entity.
func (s *Service) exportPIT05KK(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	p, err := parseReportParams(params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	var year, q int
	if _, err := fmt.Sscanf(p.Quarter, "%4d-Q%1d", &year, &q); err != nil || q < 1 || q > 4 {
		return dataio.Sheet{}, dataio.ErrInvalidParams
	}
	first := time.Date(year, time.Month(3*q-2), 1, 0, 0, 0, 0, time.UTC)
	if first.Before(pitFrom) {
		return dataio.Sheet{}, ErrReportFormNotSupported
	}
	b, people, err := s.pitBasis(ctx, p.LegalEntityID, first, 3)
	if err != nil {
		return dataio.Sheet{}, err
	}
	period := "Quý " + strconv.Itoa(q) + " năm " + strconv.Itoa(year)
	rows := indicatorRows(b.Entity, period, layout05KK_TT89_2026, withholding(people))
	name := fmt.Sprintf("05-KK-TNCN %s Q%d-%d", b.Entity.Name, q, year)
	return reportSheet(name, indicatorHeader, rows, nil, p.LegalEntityID, b.employeeIDs())
}

// waivedUpTo is the most tax still due that is forgiven to someone the company finalises for.
const waivedUpTo = 50_000

// yearBrackets turns the monthly schedule into the yearly one.
func yearBrackets(b []bracket) []bracket {
	out := make([]bracket, len(b))
	twelve := decimal.NewFromInt(12)
	for i, x := range b {
		out[i] = bracket{Rate: x.Rate}
		if x.Upper != nil {
			u := x.Upper.Mul(twelve)
			out[i].Upper = &u
		}
	}
	return out
}

// finalised is one person's year: the tax due on the year's income against what was withheld.
type finalised struct {
	Assessable, Due, Over, Under int64
	Waived                       bool
}

func finalise(p pitPerson, monthly []bracket) finalised {
	f := finalised{Assessable: max(0, p.TNCT-p.Deductions-p.Insurance)}
	f.Due = platform.Round(incomeTax(yearBrackets(monthly), decimal.NewFromInt(f.Assessable)))
	f.Over, f.Under = max(0, p.Tax-f.Due), max(0, f.Due-p.Tax)
	f.Waived = f.Under > 0 && f.Under <= waivedUpTo
	return f
}

// pitYear is a year's posted payrolls, each employee's sums, who the company finalises for
// (everyone on December's payroll) and their result under the schedule in force on 31/12.
type pitYear struct {
	reportBasis
	year       int
	emps       []store.ReportEmployeesRow
	people     map[int64]*pitPerson
	authorised map[int64]finalised
}

func (s *Service) pitYear(ctx context.Context, params json.RawMessage) (pitYear, error) {
	p, err := parseReportParams(params)
	if err != nil {
		return pitYear{}, err
	}
	if p.Year < pitFrom.Year() {
		return pitYear{}, ErrReportFormNotSupported
	}
	y := pitYear{year: p.Year}
	if y.reportBasis, err = s.reportBasis(ctx, p.LegalEntityID, time.Date(p.Year, 1, 1, 0, 0, 0, 0, time.UTC), 12); err != nil {
		return y, err
	}
	if y.emps, err = reportEmployees(ctx, y.employeeIDs()); err != nil {
		return y, err
	}
	if err := s.requireReportSensitive(ctx, y.emps); err != nil {
		return y, err
	}
	lp, err := legalParamsAt(ctx, fmt.Sprintf("%d-12-31", p.Year))
	if err != nil {
		return y, err
	}
	brackets, ok := parseBrackets(lp["pit_brackets"].Value)
	if !ok {
		return y, fmt.Errorf("hrm: stored legal parameters are malformed")
	}
	y.people, y.authorised = sumPIT(y.Months), map[int64]finalised{}
	for id := range y.Months[11].Lines {
		y.authorised[id] = finalise(*y.people[id], brackets)
	}
	return y, nil
}

// exportPIT05_1BK lists every employee paid in the year, with the result of the finalisation
// for those the company finalises for, then the totals.
func (s *Service) exportPIT05_1BK(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	y, err := s.pitYear(ctx, params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	people, err := s.reportPeople(ctx, y.emps, "tax_code", "national_id")
	if err != nil {
		return dataio.Sheet{}, err
	}
	rows := make([][]any, 0, len(people)+1)
	total := make([]any, len(layout05_1BK_TT89_2026))
	for i := range total {
		total[i] = ""
	}
	total[1] = "Tổng"
	add := func(col int, v int64) {
		n, _ := total[col].(int64)
		total[col] = n + v
	}
	for i, e := range people {
		p := y.people[e.ID]
		row := make([]any, len(layout05_1BK_TT89_2026))
		for j := range row {
			row[j] = ""
		}
		row[0], row[1], row[2] = int64(i+1), e.FullName, e.TaxCode
		if e.TaxCode == "" {
			row[3] = e.NationalID
		}
		f, authorised := y.authorised[e.ID]
		if !authorised {
			f = finalise(*p, nil)
		}
		amounts := map[int]int64{6: p.TNCT, 7: 0, 8: 0, 9: int64(p.Dependents), 10: p.Deductions, 11: 0, 12: p.Insurance, 13: 0, 14: f.Assessable, 15: p.Tax}
		if authorised {
			row[4] = "x"
			amounts[16], amounts[17], amounts[18] = f.Due, f.Over, f.Under
			if f.Waived {
				row[19] = "x"
			}
		}
		for col, v := range amounts {
			row[col] = v
			add(col, v)
		}
		rows = append(rows, row)
	}
	rows = append(rows, total)
	name := fmt.Sprintf("05-1BK-QTT-TNCN %s %d", y.Entity.Name, y.year)
	return reportSheet(name, layout05_1BK_TT89_2026, rows, []int{2, 3}, y.Entity.ID, personIDs(people))
}

// exportPIT05QTT is the year's finalisation return: what was withheld, then what the
// finalisation for the authorising employees comes to.
func (s *Service) exportPIT05QTT(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	y, err := s.pitYear(ctx, params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	w := withholding(y.people)
	v := map[string]int64{"[15]": w["[15]"], "[16]": w["[16]"], "[17]": w["[17]"], "[18]": w["[18]"],
		"[22]": w["[20]"], "[23]": w["[20]"], "[30]": w["[29]"], "[31]": w["[29]"], "[33]": w["[35]"], "[34]": w["[35]"]}
	for _, p := range y.people {
		if p.DependentDeduction > 0 {
			v["[21]"]++
		}
	}
	for id, f := range y.authorised {
		v["[37]"]++
		v["[38]"] += y.people[id].Tax
		v["[40]"] += f.Due
		if f.Waived {
			v["[41]"] += f.Under
		}
	}
	net := v["[40]"] - v["[38]"] - v["[41]"]
	v["[44]"], v["[45]"] = max(0, net), max(0, -net)
	rows := indicatorRows(y.Entity, "Năm "+strconv.Itoa(y.year), layout05QTT_TT89_2026, v)
	name := fmt.Sprintf("05-QTT-TNCN %s %d", y.Entity.Name, y.year)
	return reportSheet(name, indicatorHeader, rows, nil, y.Entity.ID, y.employeeIDs())
}

// clipMonths bounds a dependant's YYYY-MM deduction months to the year, as MM/YYYY; ok is
// false when they miss it.
func clipMonths(from, to *string, year int) (first, last string, ok bool) {
	lo, hi := fmt.Sprintf("%d-01", year), fmt.Sprintf("%d-12", year)
	if from != nil && *from > lo {
		lo = *from
	}
	if to != nil && *to < hi {
		hi = *to
	}
	if lo > hi {
		return "", "", false
	}
	return lo[5:] + "/" + lo[:4], hi[5:] + "/" + hi[:4], true
}

// isTaxCode tells a personal tax code (10 or 13 digits) from another ID number.
func isTaxCode(v string) bool {
	v = strings.ReplaceAll(v, "-", "")
	if len(v) != 10 && len(v) != 13 {
		return false
	}
	return strings.Trim(v, "0123456789") == ""
}

// exportPIT05_3BK lists the dependants deducted in the year of every employee paid in it.
func (s *Service) exportPIT05_3BK(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	y, err := s.pitYear(ctx, params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	deps, err := store.PayrollDependents.Run(ctx, y.employeeIDs())
	if err != nil {
		return dataio.Sheet{}, err
	}
	byEmployee := map[int64][]DependentInput{}
	for _, d := range deps {
		in, err := openDependent(ctx, d.Data)
		if err != nil {
			return dataio.Sheet{}, err
		}
		byEmployee[d.EmployeeID] = append(byEmployee[d.EmployeeID], in)
	}
	var withDeps []store.ReportEmployeesRow
	for _, e := range y.emps {
		if len(byEmployee[e.ID]) > 0 {
			withDeps = append(withDeps, e)
		}
	}
	people, err := s.reportPeople(ctx, withDeps, "tax_code")
	if err != nil {
		return dataio.Sheet{}, err
	}
	var rows [][]any
	for _, e := range people {
		if err := platform.InTx(ctx, func(ctx context.Context) error {
			return s.d.Audit.RecordFor(ctx, "hrm.dependents_viewed", audit.Ref{Type: employeeType, ID: e.ID}, nil)
		}); err != nil {
			return dataio.Sheet{}, err
		}
		for _, d := range byEmployee[e.ID] {
			first, last, ok := clipMonths(d.DeductionFrom, d.DeductionTo, y.year)
			if !ok {
				continue
			}
			row := []any{int64(len(rows) + 1), e.FullName, e.TaxCode, d.FullName, "", "", "", "", relationships05_3BK[d.Relationship], first, last}
			if d.DateOfBirth != nil {
				if t, err := time.Parse(time.DateOnly, *d.DateOfBirth); err == nil {
					row[4] = t.Format(dateLayout)
				}
			}
			id := ""
			if d.IDNumber != nil {
				id = *d.IDNumber
			}
			if isTaxCode(id) {
				row[5] = id
			} else if id != "" {
				row[6], row[7] = "Số ĐDCN", id
			}
			rows = append(rows, row)
		}
	}
	name := fmt.Sprintf("05-3BK-QTT-TNCN %s %d", y.Entity.Name, y.year)
	return reportSheet(name, layout05_3BK_TT89_2026, rows, []int{2, 5, 7}, y.Entity.ID, personIDs(people))
}
