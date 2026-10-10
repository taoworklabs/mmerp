package hrm

import (
	"context"
	"encoding/json"

	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (s *Service) registerPrints() {
	s.d.Printing.Register(printing.Template{Code: contractType, Name: "hrm.print.contract.name", DocType: contractType,
		Blocks: []printing.Block{
			{Key: "clauses", Label: "hrm.print.contract.block.clauses", Default: "hrm.print.contract.block.clauses.default",
				Placeholders: []string{"employee_name", "employer_name"}},
			{Key: "footer", Label: "hrm.print.contract.block.footer", Default: "hrm.print.contract.block.footer.default"},
		},
		Data: s.contractPrint, Layouts: []printing.Layout{contractLayout1}})
	s.d.Printing.Register(printing.Template{Code: "hrm.payslip", Name: "hrm.print.payslip.name", DocType: payrollType,
		Data: s.payslips, Layouts: []printing.Layout{payslipLayout1}})
}

// legalEntityPrint is the employer as a print shows it.
type legalEntityPrint struct {
	Name    string  `json:"name"`
	TaxCode *string `json:"tax_code"`
	Address *string `json:"address"`
}

// contractPrint is what a contract's print shows; frozen as is once the contract is posted.
type contractPrint struct {
	Number       string           `json:"number"`
	ParentNumber *string          `json:"parent_number"`
	Employer     legalEntityPrint `json:"employer"`
	EmployeeCode string           `json:"employee_code"`
	EmployeeName string           `json:"employee_name"`
	DateOfBirth  *string          `json:"date_of_birth"`
	Address      *string          `json:"address"`
	Kind         string           `json:"kind"`
	StartDate    string           `json:"start_date"`
	EndDate      *string          `json:"end_date"`
	Terms        ContractTerms    `json:"terms"`
}

// contractPrint reads a contract as printed, salary included: printing checked the
// actor may see it.
func (s *Service) contractPrint(ctx context.Context, id int64) ([]printing.Part, error) {
	q := store.New(platform.DBFrom(ctx))
	r, err := q.GetContract(ctx, id)
	if err != nil {
		return nil, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: contractType, ID: id})
	if err != nil {
		return nil, err
	}
	e, err := q.GetEmployee(ctx, r.EmployeeID)
	if err != nil {
		return nil, err
	}
	le, err := q.PrintLegalEntity(ctx, d.LegalEntityID)
	if err != nil {
		return nil, err
	}
	terms, err := openTerms(ctx, r.Terms)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(contractPrint{
		Number: d.Number, ParentNumber: platform.TextPtr(r.ParentNumber),
		Employer:     legalEntityPrint{Name: le.Name, TaxCode: platform.TextPtr(le.TaxCode), Address: platform.TextPtr(le.Address)},
		EmployeeCode: e.Code, EmployeeName: e.FullName, DateOfBirth: platform.DatePtr(e.DateOfBirth), Address: platform.TextPtr(e.Address),
		Kind: r.ContractTypeName, StartDate: *platform.DatePtr(r.StartDate), EndDate: platform.DatePtr(r.EndDate), Terms: terms,
	})
	return []printing.Part{{Data: b}}, err
}

// contractLayout1 prints a contract or appendix: employer, employee, term and pay.
func contractLayout1(p *printing.Page, raw json.RawMessage) error {
	var c contractPrint
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	t := func(k string) string { return p.T("hrm.print.contract."+k, nil) }
	or := func(s *string, f func(string) string) string {
		if s == nil {
			return ""
		}
		return f(*s)
	}
	same := func(s string) string { return s }
	if c.ParentNumber == nil {
		p.Title(t("title"))
	} else {
		p.Title(t("appendix_title"))
	}
	p.Center(p.T("hrm.print.contract.number", map[string]any{"number": c.Number}))
	if c.ParentNumber != nil {
		p.Center(p.T("hrm.print.contract.parent", map[string]any{"number": *c.ParentNumber}))
	}
	p.Heading(t("employer"))
	p.Field(t("employer_name"), c.Employer.Name)
	p.Field(t("tax_code"), or(c.Employer.TaxCode, same))
	p.Field(t("address"), or(c.Employer.Address, same))
	p.Heading(t("employee"))
	p.Field(t("employee_name"), c.EmployeeName)
	p.Field(t("employee_code"), c.EmployeeCode)
	p.Field(t("date_of_birth"), or(c.DateOfBirth, p.Date))
	p.Field(t("address"), or(c.Address, same))
	p.Heading(t("term"))
	p.Field(t("kind"), c.Kind)
	p.Field(t("start_date"), p.Date(c.StartDate))
	end := t("no_end_date")
	if c.EndDate != nil {
		end = p.Date(*c.EndDate)
	}
	p.Field(t("end_date"), end)
	p.Heading(t("pay"))
	cols := []printing.Column{{Title: t("item"), Share: 0.5}, {Title: t("line_kind"), Share: 0.25}, {Title: t("amount"), Share: 0.25, Right: true}}
	rows := [][]string{{t("salary"), "", p.Money(c.Terms.Salary)}}
	total := c.Terms.Salary
	for _, l := range c.Terms.Lines {
		rows = append(rows, []string{l.Name, t("line_kind." + l.Kind), p.Money(l.Amount)})
		total += l.Amount
	}
	rows = append(rows, []string{t("total"), "", p.Money(total)})
	p.Table(cols, rows, true)
	p.Text(t("currency"))
	p.Heading(t("clauses"))
	p.Block("clauses", map[string]string{"employee_name": c.EmployeeName, "employer_name": c.Employer.Name})
	p.Gap()
	p.Block("footer", nil)
	return nil
}

// payslipPrint is what one employee's payslip shows; frozen as is once the payroll is posted.
type payslipPrint struct {
	Number        string                   `json:"number"`
	PeriodStart   string                   `json:"period_start"`
	Employer      legalEntityPrint         `json:"employer"`
	EmployeeCode  string                   `json:"employee_code"`
	EmployeeName  string                   `json:"employee_name"`
	OrgUnit       string                   `json:"org_unit"`
	StandardDays  int                      `json:"standard_days"`
	PaidDays      string                   `json:"paid_days"`
	OvertimeHours string                   `json:"overtime_hours"`
	Amounts       PayrollAmounts           `json:"amounts"`
	Adjustments   []PayrollAdjustmentInput `json:"adjustments"`
}

// payslips reads one payslip per employee of a computed payroll, by department then code:
// printing checked the actor may see each person's pay.
func (s *Service) payslips(ctx context.Context, id int64) ([]printing.Part, error) {
	q := store.New(platform.DBFrom(ctx))
	p, err := q.GetPayroll(ctx, id)
	if err != nil {
		return nil, err
	}
	if !p.ComputedAt.Valid {
		return nil, ErrPayrollNotComputed
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: payrollType, ID: id})
	if err != nil {
		return nil, err
	}
	le, err := q.PrintLegalEntity(ctx, p.LegalEntityID)
	if err != nil {
		return nil, err
	}
	lines, err := s.payrollLines(ctx, id)
	if err != nil {
		return nil, err
	}
	adjustments, err := s.adjustments(ctx, id)
	if err != nil {
		return nil, err
	}
	totals, err := payrollTotals(ctx, lines)
	if err != nil {
		return nil, err
	}
	units := map[int64]string{}
	for _, t := range totals {
		units[t.OrgUnitID] = t.OrgUnitName
	}
	employer := legalEntityPrint{Name: le.Name, TaxCode: platform.TextPtr(le.TaxCode), Address: platform.TextPtr(le.Address)}
	parts := make([]printing.Part, 0, len(lines))
	for _, t := range totals {
		for _, l := range lines {
			if l.OrgUnitID != t.OrgUnitID {
				continue
			}
			slip := payslipPrint{Number: d.Number, PeriodStart: *platform.DatePtr(p.PeriodStart), Employer: employer,
				EmployeeCode: l.EmployeeCode, EmployeeName: l.EmployeeName, OrgUnit: units[l.OrgUnitID], StandardDays: l.StandardDays,
				PaidDays: l.PaidDays, OvertimeHours: l.OvertimeHours, Amounts: l.PayrollAmounts, Adjustments: []PayrollAdjustmentInput{}}
			for _, a := range adjustments {
				if a.EmployeeID == l.EmployeeID {
					slip.Adjustments = append(slip.Adjustments, a.PayrollAdjustmentInput)
				}
			}
			b, err := json.Marshal(slip)
			if err != nil {
				return nil, err
			}
			parts = append(parts, printing.Part{Key: l.EmployeeID, Data: b})
		}
	}
	return parts, nil
}

// payslipLayout1 prints one employee's pay: attendance, earnings, deductions, net pay.
func payslipLayout1(p *printing.Page, raw json.RawMessage) error {
	var s payslipPrint
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	t := func(k string) string { return p.T("hrm.print.payslip."+k, nil) }
	a := s.Amounts
	p.Title(t("title"))
	p.Center(p.T("hrm.print.payslip.period", map[string]any{"month": p.Month(s.PeriodStart)}))
	p.Center(p.T("hrm.print.payslip.payroll", map[string]any{"number": s.Number}))
	p.Gap()
	p.Field(t("employer"), s.Employer.Name)
	p.Field(t("employee_name"), s.EmployeeName)
	p.Field(t("employee_code"), s.EmployeeCode)
	p.Field(t("org_unit"), s.OrgUnit)
	p.Field(t("paid_days"), p.T("hrm.print.payslip.days_of", map[string]any{"paid": p.Decimal(s.PaidDays), "standard": s.StandardDays}))
	p.Field(t("overtime_hours"), p.Decimal(s.OvertimeHours))
	cols := []printing.Column{{Title: t("item"), Share: 0.7}, {Title: t("amount"), Share: 0.3, Right: true}}
	row := func(k string, v int64) []string { return []string{t(k), p.Money(v)} }
	p.Heading(t("income"))
	p.Table(cols, [][]string{row("earned", a.Earned), row("overtime", a.Overtime), row("unused_leave", a.UnusedLeave),
		row("adjustment", a.Adjustment), row("gross", a.Gross)}, true)
	p.Heading(t("deductions"))
	p.Table(cols, [][]string{row("social_insurance", a.SocialInsurance), row("health_insurance", a.HealthInsurance),
		row("unemployment_insurance", a.UnemploymentIns), row("income_tax", a.IncomeTax),
		row("total_deductions", a.SocialInsurance+a.HealthInsurance+a.UnemploymentIns+a.IncomeTax)}, true)
	p.Heading(t("tax"))
	p.Field(t("insurance_base"), p.Money(a.InsuranceBase))
	p.Field(t("exempt"), p.Money(a.Exempt))
	p.Field(t("personal_deduction"), p.Money(a.PersonalDeduction))
	p.Field(t("dependent_deduction"), p.Money(a.DependentDeduction))
	p.Field(t("taxable"), p.Money(a.Taxable))
	if len(s.Adjustments) > 0 {
		p.Heading(t("adjustments"))
		for _, adj := range s.Adjustments {
			p.Field(p.Month(adj.SourcePeriod+"-01"), p.Money(adj.Amount)+" · "+adj.Reason)
		}
	}
	p.Table([]printing.Column{{Share: 0.7}, {Share: 0.3, Right: true}}, [][]string{row("net", a.Net)}, true)
	p.Text(t("currency"))
	return nil
}
