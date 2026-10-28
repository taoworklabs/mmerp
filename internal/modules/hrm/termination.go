package hrm

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// openStatuses are the statuses of a termination that still stands.
var openStatuses = []string{string(record.Draft), string(record.PendingApproval), string(record.Posted)}

// reasonOptions labels the reasons in the actor's language.
func (s *Service) reasonOptions(ctx context.Context) ([]record.Option, error) {
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]record.Option, len(terminationReasons))
	for i, r := range terminationReasons {
		out[i] = record.Option{Value: r.Code, Label: platform.Translate(me.Locale, "hrm.termination.reason."+r.Code, nil)}
	}
	return out, nil
}

// terminationSubjects is the account of the employee, who never approves their own termination.
func (s *Service) terminationSubjects(ctx context.Context, id int64) ([]int64, error) {
	r, err := store.GetTermination.Run(ctx, id)
	return subjectOf(r.EmployeeUserID), err
}

// openTermination refuses a second termination of the employee among statuses.
func openTermination(ctx context.Context, employee, except int64, statuses []string) error {
	r, err := store.OpenTerminationOf.Run(ctx, store.OpenTerminationOfParams{EmployeeID: employee, ExceptID: except, Statuses: statuses})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return &platform.Error{Status: http.StatusConflict, Code: ErrTerminationExists.Code, Params: map[string]any{"id": r.ID, "number": r.Number}}
}

// Terminations lists the terminations in the actor's units.
func (s *Service) Terminations(ctx context.Context, f TerminationFilter) (TerminationList, error) {
	out := TerminationList{Items: []TerminationListItem{}}
	acts, err := s.d.Record.TypeActions(ctx, terminationType)
	if err != nil {
		return out, err
	}
	out.AllowedActions = acts
	sc, err := s.d.IAM.DataScope(ctx, "hrm", PermTerminationView)
	if err != nil {
		return out, err
	}
	page, err := terminationList.Run(ctx, terminationQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, TerminationListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, EmployeeID: r.EmployeeID, EmployeeCode: r.EmployeeCode,
			EmployeeName: r.EmployeeName, OrgUnitName: r.OrgUnitName, TerminationDate: *platform.DatePtr(r.Date), Reason: r.Reason,
		})
	}
	return out, nil
}

// Termination returns one termination; one the actor may not view does not exist.
func (s *Service) Termination(ctx context.Context, id int64) (Termination, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: terminationType, ID: id})
	if err != nil {
		return Termination{}, err
	}
	r, err := store.GetTermination.Run(ctx, id)
	if err != nil {
		return Termination{}, err
	}
	actions, err := s.d.Record.Actions(ctx, d)
	if err != nil {
		return Termination{}, err
	}
	return Termination{
		ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, EmployeeID: r.EmployeeID,
		EmployeeCode: r.EmployeeCode, EmployeeName: r.EmployeeName, TerminationFields: terminationFields(r), AllowedActions: actions,
	}, nil
}

// CreateTermination adds a draft termination of an employee.
func (s *Service) CreateTermination(ctx context.Context, in NewTermination) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		// An employee the actor cannot see does not exist, so ids cannot be probed.
		if err := s.d.Record.Visible(ctx, record.Ref{Type: employeeType, ID: in.EmployeeID}, record.View); err != nil {
			return err
		}
		e, err := store.GetEmployee.Run(ctx, in.EmployeeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		h, err := terminationHeader(e.OrgUnitID, *platform.DatePtr(e.HireDate), in.TerminationFields)
		if err != nil {
			return err
		}
		h.Subject, h.RequestID = e.UserID.Int64, in.RequestID
		if err := s.d.Record.MayCreate(ctx, terminationType, h); err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, terminationType, h)
		id = d.ID
		if err != nil || d.Reused {
			return err
		}
		// After Create, so a retried create returns its document instead of refusing.
		if err := openTermination(ctx, e.ID, id, openStatuses); err != nil {
			return err
		}
		f := in.TerminationFields
		if err := store.CreateTermination.Run(ctx, store.CreateTerminationParams{
			ID: id, EmployeeID: e.ID, Reason: f.Reason, DecisionDate: platform.NullDate(f.DecisionDate), PensionEligible: f.PensionEligible,
			InsuredMonths: int32(f.InsuredMonths), PaidMonths: int32(f.PaidMonths), Handover: platform.NullText(f.Handover), Note: platform.NullText(f.Note),
		}); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.termination_created", audit.Ref{Type: terminationType, ID: id}, terminationChanges(TerminationFields{}, f))
	})
	return id, err
}

// UpdateTermination replaces the fields of a draft at the version the actor saw.
func (s *Service) UpdateTermination(ctx context.Context, id int64, in TerminationUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, record.Ref{Type: terminationType, ID: id}); err != nil {
			return err
		}
		old, err := store.GetTermination.Run(ctx, id)
		if err != nil {
			return err
		}
		h, err := terminationHeader(old.OrgUnitID, *platform.DatePtr(old.HireDate), in.TerminationFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: terminationType, ID: id}, in.Version, h); err != nil {
			return err
		}
		f := in.TerminationFields
		if err := store.UpdateTermination.Run(ctx, store.UpdateTerminationParams{
			ID: id, Reason: f.Reason, DecisionDate: platform.NullDate(f.DecisionDate), PensionEligible: f.PensionEligible,
			InsuredMonths: int32(f.InsuredMonths), PaidMonths: int32(f.PaidMonths), Handover: platform.NullText(f.Handover), Note: platform.NullText(f.Note),
		}); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.termination_updated", audit.Ref{Type: terminationType, ID: id}, terminationChanges(terminationFields(old), f))
	})
}

func deleteTermination(ctx context.Context, id int64) error {
	return store.DeleteTermination.Run(ctx, id)
}

// terminationHeader checks the fields and builds the record header: the date is the
// termination date, the unit the employee's.
func terminationHeader(unit int64, hire string, in TerminationFields) (record.Header, error) {
	if _, ok := reasonAllowance(in.Reason); !ok {
		return record.Header{}, ErrTerminationReason
	}
	if in.TerminationDate < hire {
		return record.Header{}, ErrTerminationBeforeHire
	}
	if in.InsuredMonths < 0 || in.PaidMonths < 0 {
		return record.Header{}, ErrTerminationMonths
	}
	return record.Header{Date: in.TerminationDate, OrgUnitID: unit, Fields: map[string]string{"reason": in.Reason}}, nil
}

func terminationFields(r store.GetTerminationRow) TerminationFields {
	return TerminationFields{
		TerminationDate: *platform.DatePtr(r.TerminationDate), Reason: r.Reason, DecisionDate: platform.DatePtr(r.DecisionDate),
		PensionEligible: r.PensionEligible, InsuredMonths: int(r.InsuredMonths), PaidMonths: int(r.PaidMonths),
		Handover: platform.TextPtr(r.Handover), Note: platform.TextPtr(r.Note),
	}
}

func terminationChanges(old, n TerminationFields) []audit.Change {
	return []audit.Change{
		{Field: "termination_date", Old: old.TerminationDate, New: n.TerminationDate},
		{Field: "reason", Old: old.Reason, New: n.Reason},
		{Field: "decision_date", Old: old.DecisionDate, New: n.DecisionDate},
		{Field: "pension_eligible", Old: old.PensionEligible, New: n.PensionEligible},
		{Field: "insured_months", Old: strconv.Itoa(old.InsuredMonths), New: strconv.Itoa(n.InsuredMonths)},
		{Field: "paid_months", Old: strconv.Itoa(old.PaidMonths), New: strconv.Itoa(n.PaidMonths)},
		{Field: "handover", Old: old.Handover, New: n.Handover},
		{Field: "note", Old: old.Note, New: n.Note},
	}
}

// terminationTransition sets the employee's termination date when posted and restores the
// one it replaced when cancelled. The new date changes the pay of its month and every later
// one, so a closed period from there on refuses both.
func (s *Service) terminationTransition(ctx context.Context, d record.Doc, from record.Status) error {
	posted, ok := payrollEffect(d, from)
	if !ok {
		return nil
	}
	if err := checkPayrollPeriods(ctx, d.LegalEntityID, d.Date, nil); err != nil {
		return err
	}
	t, err := store.GetTermination.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	e, err := store.LockEmployee.Run(ctx, t.EmployeeID)
	if err != nil {
		return err
	}
	// Pay follows the employee's legal entity: one that moved since the termination was
	// drafted is outside this payroll lock, both ways.
	legalEntity, err := s.d.IAM.LegalEntityOf(ctx, e.OrgUnitID)
	if err != nil {
		return err
	}
	if legalEntity != d.LegalEntityID {
		return ErrTerminationLegalEntity
	}
	next := t.PreviousDate
	if posted {
		if err := openTermination(ctx, e.ID, d.ID, []string{string(record.Posted)}); err != nil {
			return err
		}
		if d.Date < *platform.DatePtr(e.HireDate) {
			return ErrTerminationBeforeHire
		}
		if err := store.SetTerminationPreviousDate.Run(ctx, store.SetTerminationPreviousDateParams{ID: d.ID, PreviousDate: e.TerminationDate}); err != nil {
			return err
		}
		next = platform.NullDate(&d.Date)
	}
	// The date replaced or restored may be earlier: the pay changes from there on. The
	// payroll lock is already held, so checking again keeps the lock order.
	if was := platform.DatePtr(e.TerminationDate); was != nil && *was < d.Date {
		if err := checkPayrollPeriods(ctx, d.LegalEntityID, *was, nil); err != nil {
			return err
		}
	}
	if now := platform.DatePtr(next); now != nil && *now < d.Date {
		if err := checkPayrollPeriods(ctx, d.LegalEntityID, *now, nil); err != nil {
			return err
		}
	}
	if err := store.SetEmployeeTerminationDate.Run(ctx, store.SetEmployeeTerminationDateParams{ID: e.ID, TerminationDate: next}); err != nil {
		return err
	}
	return s.d.Audit.RecordChanges(ctx, "hrm.employee_updated", audit.Ref{Type: employeeType, ID: e.ID},
		[]audit.Change{{Field: "termination_date", Old: platform.DatePtr(e.TerminationDate), New: platform.DatePtr(next)}})
}
