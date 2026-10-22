package hrm

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

var dayKinds = []string{"weekday", "weekly_off", "holiday"}

// dayKindOptions labels the day kinds in the actor's language.
func (s *Service) dayKindOptions(ctx context.Context) ([]record.Option, error) {
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]record.Option, len(dayKinds))
	for i, k := range dayKinds {
		out[i] = record.Option{Value: k, Label: platform.Translate(me.Locale, "hrm.overtime.day_kind."+k, nil)}
	}
	return out, nil
}

func (s *Service) overtimeGrant(ctx context.Context, id int64, action record.Action) (bool, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetOvertimeRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return selfGrant(ctx, r.EmployeeID, r.EmployeeUserID, action)
}

func (s *Service) overtimeApprovers(ctx context.Context, ref record.Ref, level int) ([]int64, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetOvertimeRequest(ctx, ref.ID)
	if err != nil {
		return nil, err
	}
	return managerApprover(ctx, r.EmployeeID, level)
}

func (s *Service) overtimeSubjects(ctx context.Context, id int64) ([]int64, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetOvertimeRequest(ctx, id)
	return subjectOf(r.EmployeeUserID), err
}

// overtimeTransition only keeps approved overtime out of closed payroll periods; pay
// for it is worked out by the payroll.
func (s *Service) overtimeTransition(ctx context.Context, d record.Doc, from record.Status) error {
	posted, ok := payrollEffect(d, from)
	if !ok {
		return nil
	}
	if posted {
		// The calendar may have changed since the request was saved.
		r, err := store.New(platform.DBFrom(ctx)).GetOvertimeRequest(ctx, d.ID)
		if err != nil {
			return err
		}
		if err := dayKindOn(ctx, d.LegalEntityID, d.Date, r.DayKind); err != nil {
			return err
		}
	}
	return checkPayrollPeriods(ctx, d.LegalEntityID, d.Date, &d.Date)
}

// checkDayKind holds the day kind to the one the unit's work calendar gives the date: it
// sets the overtime multiplier, so a wrong pick changes the pay.
func (s *Service) checkDayKind(ctx context.Context, unit int64, date, kind string) error {
	// The legal entity is read under the tree lock, so the calendar cannot move under us.
	if err := s.d.IAM.ShareTree(ctx); err != nil {
		return err
	}
	legalEntity, err := s.d.IAM.LegalEntityOf(ctx, unit)
	if err != nil {
		return err
	}
	return dayKindOn(ctx, legalEntity, date, kind)
}

func dayKindOn(ctx context.Context, legalEntity int64, date, kind string) error {
	cal, err := workCalendar(ctx, legalEntity, date, date)
	if err != nil {
		return err
	}
	if want := cal.dayKind(dayOf(date)); kind != want {
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "overtime_day_kind_mismatch", Params: map[string]any{"day_kind": want}}
	}
	return nil
}

// OvertimeActions lists what the actor may do before picking an overtime request.
func (s *Service) OvertimeActions(ctx context.Context) (SelfServiceActions, error) {
	return s.selfServiceActions(ctx, overtimeType)
}

// Overtimes lists the overtime requests the actor may see: their own, their reports', and those in scope.
func (s *Service) Overtimes(ctx context.Context, f OvertimeFilter) (OvertimeList, error) {
	out := OvertimeList{Items: []OvertimeListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "hrm", PermOvertimeView)
	if err != nil {
		return out, err
	}
	page, err := overtimeList.Run(ctx, overtimeQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, OvertimeListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, EmployeeID: r.EmployeeID, EmployeeCode: r.EmployeeCode,
			EmployeeName: r.EmployeeName, Date: *platform.DatePtr(r.Date), DayKind: r.DayKind,
			DayHours: platform.PlainDecimal(r.DayHours), NightHours: platform.PlainDecimal(r.NightHours),
		})
	}
	return out, nil
}

// Overtime returns one overtime request; one the actor may not view does not exist.
func (s *Service) Overtime(ctx context.Context, id int64) (Overtime, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: overtimeType, ID: id})
	if err != nil {
		return Overtime{}, err
	}
	r, err := store.New(platform.DBFrom(ctx)).GetOvertimeRequest(ctx, id)
	if err != nil {
		return Overtime{}, err
	}
	actions, err := s.d.Record.Actions(ctx, d)
	if err != nil {
		return Overtime{}, err
	}
	return Overtime{
		ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, EmployeeID: r.EmployeeID,
		EmployeeCode: r.EmployeeCode, EmployeeName: r.EmployeeName, OvertimeFields: overtimeFields(r), AllowedActions: actions,
	}, nil
}

// CreateOvertime adds a draft overtime request for the actor's own employee record, or as HR for another.
func (s *Service) CreateOvertime(ctx context.Context, in NewOvertime) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		employee, unit, user, err := subject(ctx, in.EmployeeID)
		if err != nil {
			return err
		}
		if err := s.d.Record.MayCreate(ctx, overtimeType, record.Header{OrgUnitID: unit, Subject: user.Int64}); err != nil {
			return err
		}
		h, err := overtimeHeader(unit, in.OvertimeFields)
		if err != nil {
			return err
		}
		h.Subject = user.Int64
		if err := s.checkDayKind(ctx, unit, in.Date, in.DayKind); err != nil {
			return err
		}
		h.RequestID = in.RequestID
		d, err := s.d.Record.Create(ctx, overtimeType, h)
		id = d.ID
		if err != nil || d.Reused {
			return err
		}
		f := in.OvertimeFields
		if err := store.New(platform.DBFrom(ctx)).CreateOvertimeRequest(ctx, store.CreateOvertimeRequestParams{
			ID: id, EmployeeID: employee, Date: platform.NullDate(&f.Date), DayKind: f.DayKind,
			DayHours: f.DayHours, NightHours: f.NightHours, Reason: platform.NullText(f.Reason),
		}); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.overtime_request_created", audit.Ref{Type: overtimeType, ID: id}, overtimeChanges(OvertimeFields{}, f))
	})
	return id, err
}

// UpdateOvertime replaces the fields of a draft at the version the actor saw.
func (s *Service) UpdateOvertime(ctx context.Context, id int64, in OvertimeUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, record.Ref{Type: overtimeType, ID: id}); err != nil {
			return err
		}
		old, err := store.New(platform.DBFrom(ctx)).GetOvertimeRequest(ctx, id)
		if err != nil {
			return err
		}
		h, err := overtimeHeader(old.OrgUnitID, in.OvertimeFields)
		if err != nil {
			return err
		}
		if err := s.checkDayKind(ctx, old.OrgUnitID, in.Date, in.DayKind); err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: overtimeType, ID: id}, in.Version, h); err != nil {
			return err
		}
		f := in.OvertimeFields
		if err := store.New(platform.DBFrom(ctx)).UpdateOvertimeRequest(ctx, store.UpdateOvertimeRequestParams{
			ID: id, Date: platform.NullDate(&f.Date), DayKind: f.DayKind, DayHours: f.DayHours, NightHours: f.NightHours, Reason: platform.NullText(f.Reason),
		}); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.overtime_request_updated", audit.Ref{Type: overtimeType, ID: id}, overtimeChanges(overtimeFields(old), f))
	})
}

func deleteOvertime(ctx context.Context, id int64) error {
	return store.New(platform.DBFrom(ctx)).DeleteOvertimeRequest(ctx, id)
}

// overtimeHeader checks the hours and builds the record header; hours is day plus night.
func overtimeHeader(unit int64, in OvertimeFields) (record.Header, error) {
	day, okDay := record.ParseNumber(in.DayHours)
	night, okNight := record.ParseNumber(in.NightHours)
	two := decimal.NewFromInt(2)
	sum := day.Add(night)
	if !okDay || !okNight || day.IsNegative() || night.IsNegative() || !sum.IsPositive() || sum.GreaterThan(decimal.NewFromInt(24)) ||
		!day.Mul(two).IsInteger() || !night.Mul(two).IsInteger() {
		return record.Header{}, ErrOvertimeHours
	}
	return record.Header{Date: in.Date, OrgUnitID: unit, Fields: map[string]string{"hours": sum.String(), "day_kind": in.DayKind}}, nil
}

func overtimeFields(r store.GetOvertimeRequestRow) OvertimeFields {
	return OvertimeFields{Date: *platform.DatePtr(r.Date), DayKind: r.DayKind, DayHours: platform.PlainDecimal(r.DayHours), NightHours: platform.PlainDecimal(r.NightHours), Reason: platform.TextPtr(r.Reason)}
}

func overtimeChanges(old, n OvertimeFields) []audit.Change {
	return []audit.Change{
		{Field: "date", Old: old.Date, New: n.Date},
		{Field: "day_kind", Old: old.DayKind, New: n.DayKind},
		{Field: "day_hours", Old: old.DayHours, New: platform.PlainDecimal(n.DayHours)},
		{Field: "night_hours", Old: old.NightHours, New: platform.PlainDecimal(n.NightHours)},
		{Field: "reason", Old: old.Reason, New: n.Reason},
	}
}
