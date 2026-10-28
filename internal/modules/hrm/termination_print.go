package hrm

import (
	"context"
	"encoding/json"

	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (s *Service) registerTerminationPrint() {
	s.d.Printing.Register(printing.Template{Code: terminationType, Name: "hrm.print.termination.name", DocType: terminationType,
		Blocks: []printing.Block{
			{Key: "grounds", Label: "hrm.print.termination.block.grounds", Default: "hrm.print.termination.block.grounds.default"},
			{Key: "articles", Label: "hrm.print.termination.block.articles", Default: "hrm.print.termination.block.articles.default",
				Placeholders: []string{"employee_name", "termination_date"}},
			{Key: "footer", Label: "hrm.print.termination.block.footer", Default: "hrm.print.termination.block.footer.default"},
		},
		Data: s.terminationPrint, Layouts: []printing.Layout{terminationLayout1}})
}

// terminationPrint is what a termination's decision shows; frozen as is once posted. It
// carries no amount: the payroll pays the allowance.
type terminationPrint struct {
	Number          string               `json:"number"`
	Employer        printing.LegalEntity `json:"employer"`
	EmployeeCode    string               `json:"employee_code"`
	EmployeeName    string               `json:"employee_name"`
	DateOfBirth     *string              `json:"date_of_birth"`
	TerminationDate string               `json:"termination_date"`
	DecisionDate    *string              `json:"decision_date"`
	Reason          string               `json:"reason"`
	Allowance       allowanceKind        `json:"allowance"`
	Handover        *string              `json:"handover"`
}

func (s *Service) terminationPrint(ctx context.Context, id int64) ([]printing.Part, error) {
	r, err := store.GetTermination.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: terminationType, ID: id})
	if err != nil {
		return nil, err
	}
	e, err := store.GetEmployee.Run(ctx, r.EmployeeID)
	if err != nil {
		return nil, err
	}
	employer, err := s.d.Printing.LegalEntity(ctx, d.LegalEntityID)
	if err != nil {
		return nil, err
	}
	b := terminationBasis(r.Reason, *platform.DatePtr(r.HireDate), d.Date, r.InsuredMonths, r.PaidMonths, r.PensionEligible)
	raw, err := json.Marshal(terminationPrint{
		Number: d.Number, Employer: employer, EmployeeCode: e.Code, EmployeeName: e.FullName, DateOfBirth: platform.DatePtr(e.DateOfBirth),
		TerminationDate: d.Date, DecisionDate: platform.DatePtr(r.DecisionDate), Reason: r.Reason, Allowance: b.kind(), Handover: platform.TextPtr(r.Handover),
	})
	return []printing.Part{{Data: raw}}, err
}

// terminationLayout1 prints the decision: employer, employee, last day, reason, allowance, handover.
func terminationLayout1(p *printing.Page, raw json.RawMessage) error {
	var x terminationPrint
	if err := json.Unmarshal(raw, &x); err != nil {
		return err
	}
	t := func(k string) string { return p.T("hrm.print.termination."+k, nil) }
	p.Title(t("title"))
	p.Center(p.T("hrm.print.termination.number", map[string]any{"number": x.Number}))
	if x.DecisionDate != nil {
		p.Center(p.T("hrm.print.termination.decision_date", map[string]any{"date": p.Date(*x.DecisionDate)}))
	}
	p.LegalEntity(t("employer"), t("employer_name"), x.Employer)
	p.Heading(t("grounds"))
	p.Block("grounds", nil)
	p.Heading(t("employee"))
	p.Field(t("employee_name"), x.EmployeeName)
	p.Field(t("employee_code"), x.EmployeeCode)
	if x.DateOfBirth != nil {
		p.Field(t("date_of_birth"), p.Date(*x.DateOfBirth))
	}
	p.Field(t("termination_date"), p.Date(x.TerminationDate))
	p.Field(t("reason"), p.T("hrm.termination.reason."+x.Reason, nil))
	p.Field(t("allowance"), p.T("hrm.print.termination.allowance."+string(x.Allowance), nil))
	if x.Handover != nil {
		p.Field(t("handover"), *x.Handover)
	}
	p.Heading(t("articles"))
	p.Block("articles", map[string]string{"employee_name": x.EmployeeName, "termination_date": p.Date(x.TerminationDate)})
	p.Gap()
	p.Block("footer", nil)
	return nil
}
