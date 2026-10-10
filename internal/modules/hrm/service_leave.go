package hrm

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (s *Service) leaveTypeOptions(ctx context.Context) ([]record.Option, error) {
	rows, err := store.New(platform.DBFrom(ctx)).ListLeaveTypes(ctx)
	out := make([]record.Option, len(rows))
	for i, r := range rows {
		out[i] = record.Option{Value: strconv.FormatInt(r.ID, 10), Label: r.Name}
	}
	return out, err
}

// LeaveTypes lists every kind of leave, inactive ones included.
func (s *Service) LeaveTypes(ctx context.Context) ([]LeaveType, error) {
	rows, err := store.New(platform.DBFrom(ctx)).ListLeaveTypes(ctx)
	out := make([]LeaveType, len(rows))
	for i, r := range rows {
		out[i] = LeaveType{ID: r.ID, LeaveTypeInput: LeaveTypeInput{Name: r.Name, DeductsBalance: r.DeductsBalance, Paid: r.Paid, Active: r.Active}}
	}
	return out, err
}

// SaveLeaveType creates a kind of leave (id 0) or replaces one. Kinds are tenant-wide.
func (s *Service) SaveLeaveType(ctx context.Context, id int64, in LeaveTypeInput) (int64, error) {
	if err := s.d.IAM.RequireTenantWide(ctx, "hrm", PermLeaveTypeManage); err != nil {
		return 0, err
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		var err error
		if id == 0 {
			id, err = q.CreateLeaveType(ctx, store.CreateLeaveTypeParams{Name: in.Name, DeductsBalance: in.DeductsBalance, Paid: in.Paid, Active: in.Active})
		} else {
			var n int64
			n, err = q.UpdateLeaveType(ctx, store.UpdateLeaveTypeParams{ID: id, Name: in.Name, DeductsBalance: in.DeductsBalance, Paid: in.Paid, Active: in.Active})
			if err == nil && n == 0 {
				return platform.ErrNotFound
			}
		}
		if platform.Violates(err, "leave_types_name_key") {
			return ErrLeaveTypeTaken
		}
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "hrm.leave_type_saved", map[string]any{"id": id, "name": in.Name, "deducts_balance": in.DeductsBalance, "paid": in.Paid, "active": in.Active})
	})
	return id, err
}

func (s *Service) canLeave(ctx context.Context, id int64, action record.Action) (bool, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetLeaveRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Attachments (a medical certificate) go with the request, asking nothing more.
	action, _ = filesAction(action)
	return s.canSelfService(ctx, selfService{r.EmployeeID, r.EmployeeUserID, r.OrgUnitID}, action, PermLeaveView, PermLeaveEdit)
}

func (s *Service) leaveApprovers(ctx context.Context, ref record.Ref, level int) ([]int64, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetLeaveRequest(ctx, ref.ID)
	if err != nil {
		return nil, err
	}
	return managerApprover(ctx, r.EmployeeID, level)
}

func (s *Service) leaveSubjects(ctx context.Context, id int64) ([]int64, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetLeaveRequest(ctx, id)
	return subjectOf(r.EmployeeUserID), err
}

// leaveTransition deducts the balance when a request is approved, records what it took,
// and gives exactly that back when an approved request is cancelled. Either way the
// leave's days must not fall in a closed payroll period.
func (s *Service) leaveTransition(ctx context.Context, d record.Doc, from record.Status) error {
	approved, ok := payrollEffect(d, from)
	if !ok {
		return nil
	}
	q := store.New(platform.DBFrom(ctx))
	r, err := q.GetLeaveRequest(ctx, d.ID)
	if err != nil {
		return err
	}
	if err := checkPayrollPeriods(ctx, d.LegalEntityID, *platform.DatePtr(r.StartDate), platform.DatePtr(r.EndDate)); err != nil {
		return err
	}
	var delta decimal.Decimal
	switch {
	case approved && r.DeductsBalance:
		if delta, err = decimal.NewFromString(r.Days); err != nil {
			return err
		}
		delta = delta.Neg()
	case !approved:
		if delta, err = decimal.NewFromString(r.Deducted); err != nil {
			return err
		}
	}
	if delta.IsZero() {
		return nil
	}
	_, err = q.AddLeaveBalance(ctx, store.AddLeaveBalanceParams{EmployeeID: r.EmployeeID, Year: int32(r.StartDate.Time.Year()), Delta: delta.String()})
	if errors.Is(err, pgx.ErrNoRows) || platform.Violates(err, "leave_balances_days_check") {
		return ErrInsufficientBalance
	}
	if err != nil {
		return err
	}
	deducted := decimal.Zero
	if approved {
		deducted = delta.Neg()
	}
	return q.SetLeaveDeducted(ctx, store.SetLeaveDeductedParams{ID: d.ID, Deducted: deducted.String()})
}

// LeaveActions lists what the actor may do before picking a leave request.
func (s *Service) LeaveActions(ctx context.Context) (SelfServiceActions, error) {
	return s.selfServiceActions(ctx, PermLeaveEdit)
}

// Leaves lists the leave requests the actor may see: their own, their reports', and those in scope.
func (s *Service) Leaves(ctx context.Context, f LeaveFilter) (LeaveList, error) {
	out := LeaveList{Items: []LeaveListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermLeaveView)
	if err != nil {
		return out, err
	}
	actor, _ := platform.ActorFrom(ctx)
	q := store.New(platform.DBFrom(ctx))
	reports, err := q.Reports(ctx, actor)
	if err != nil {
		return out, err
	}
	rows, err := q.ListLeaveRequests(ctx, store.ListLeaveRequestsParams{
		AllUnits: sc.All, Units: sc.Units, Actor: actor, Reports: reports, Status: f.Status,
		EmployeeID: pgtype.Int8{Int64: f.EmployeeID, Valid: f.EmployeeID != 0},
		FromDate:   platform.NullDate(optional(f.From)), ToDate: platform.NullDate(optional(f.To)),
		Sort: f.Sort, Lim: f.Limit(), Off: f.Offset(),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, LeaveListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, EmployeeID: r.EmployeeID, EmployeeCode: r.EmployeeCode,
			EmployeeName: r.EmployeeName, LeaveTypeName: r.LeaveTypeName, StartDate: *platform.DatePtr(r.StartDate),
			EndDate: *platform.DatePtr(r.EndDate), Days: plain(r.Days),
		})
	}
	return out, err
}

// Leave returns one leave request; one the actor may not view does not exist.
func (s *Service) Leave(ctx context.Context, id int64) (Leave, error) {
	ref := record.Ref{Type: leaveType, ID: id}
	if ok, err := s.d.Record.Can(ctx, leaveType, id, record.View); err != nil || !ok {
		return Leave{}, platform.OrErr(err, platform.ErrNotFound)
	}
	q := store.New(platform.DBFrom(ctx))
	r, err := q.GetLeaveRequest(ctx, id)
	if err != nil {
		return Leave{}, err
	}
	d, err := s.d.Record.Get(ctx, ref)
	if err != nil {
		return Leave{}, err
	}
	actions, err := s.d.Record.DocumentActions(ctx, d)
	if err != nil {
		return Leave{}, err
	}
	out := Leave{
		ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, EmployeeID: r.EmployeeID,
		EmployeeCode: r.EmployeeCode, EmployeeName: r.EmployeeName, LeaveTypeName: r.LeaveTypeName,
		LeaveFields: LeaveFields{LeaveTypeID: r.LeaveTypeID, StartDate: *platform.DatePtr(r.StartDate), EndDate: *platform.DatePtr(r.EndDate),
			Days: plain(r.Days), Reason: platform.TextPtr(r.Reason)},
		AllowedActions: actions,
	}
	if ok, err := s.canSeeBalance(ctx, r.EmployeeID); err != nil {
		return Leave{}, err
	} else if ok {
		balances, err := q.LeaveBalances(ctx, r.EmployeeID)
		if err != nil {
			return Leave{}, err
		}
		out.Balance = new("0")
		for _, b := range balances {
			if b.Year == int32(r.StartDate.Time.Year()) {
				out.Balance = new(plain(b.Days))
			}
		}
	}
	return out, nil
}

// CreateLeave adds a draft leave request for the actor's own employee record, or as HR for another.
func (s *Service) CreateLeave(ctx context.Context, in NewLeave) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		employee, unit, err := s.filer(ctx, in.EmployeeID, PermLeaveEdit)
		if err != nil {
			return err
		}
		h, row, typeName, err := s.leaveHeader(ctx, unit, in.LeaveFields)
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, leaveType, h)
		if err != nil {
			return err
		}
		id = d.ID
		row.ID, row.EmployeeID = id, employee
		if err := q.CreateLeaveRequest(ctx, row); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.leave_request_created", audit.Ref{Type: leaveType, ID: id}, leaveChanges(LeaveFields{}, "", in.LeaveFields, typeName))
	})
	return id, err
}

// UpdateLeave replaces the fields of a draft at the version the actor saw.
func (s *Service) UpdateLeave(ctx context.Context, id int64, in LeaveUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		old, err := s.visibleLeave(ctx, id)
		if err != nil {
			return err
		}
		h, row, typeName, err := s.leaveHeader(ctx, old.OrgUnitID, in.LeaveFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: leaveType, ID: id}, in.Version, h); err != nil {
			return err
		}
		row.ID = id
		if err := q.UpdateLeaveRequest(ctx, store.UpdateLeaveRequestParams{
			ID: id, LeaveTypeID: row.LeaveTypeID, StartDate: row.StartDate, EndDate: row.EndDate, Days: row.Days, Reason: row.Reason,
		}); err != nil {
			return err
		}
		before := LeaveFields{LeaveTypeID: old.LeaveTypeID, StartDate: *platform.DatePtr(old.StartDate), EndDate: *platform.DatePtr(old.EndDate), Days: plain(old.Days), Reason: platform.TextPtr(old.Reason)}
		return s.d.Audit.RecordChanges(ctx, "hrm.leave_request_updated", audit.Ref{Type: leaveType, ID: id}, leaveChanges(before, old.LeaveTypeName, in.LeaveFields, typeName))
	})
}

// DeleteLeave removes a draft at the version the actor saw.
func (s *Service) DeleteLeave(ctx context.Context, id int64, version int32) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visibleLeave(ctx, id); err != nil {
			return err
		}
		if err := s.d.Record.Delete(ctx, record.Ref{Type: leaveType, ID: id}, version); err != nil {
			return err
		}
		return store.New(platform.DBFrom(ctx)).DeleteLeaveRequest(ctx, id)
	})
}

func (s *Service) visibleLeave(ctx context.Context, id int64) (store.GetLeaveRequestRow, error) {
	if ok, err := s.d.Record.Can(ctx, leaveType, id, record.View); err != nil || !ok {
		return store.GetLeaveRequestRow{}, platform.OrErr(err, platform.ErrNotFound)
	}
	return store.New(platform.DBFrom(ctx)).GetLeaveRequest(ctx, id)
}

// leaveHeader checks the fields and builds the record header and the row; the row's
// leave type name comes back in typeName for the audit entry.
func (s *Service) leaveHeader(ctx context.Context, unit int64, in LeaveFields) (h record.Header, row store.CreateLeaveRequestParams, typeName string, err error) {
	start, end := platform.NullDate(&in.StartDate), platform.NullDate(&in.EndDate)
	switch {
	case end.Time.Before(start.Time):
		return h, row, "", ErrLeaveDates
	case start.Time.Year() != end.Time.Year():
		return h, row, "", ErrLeaveSpansYears
	}
	days, ok := record.ParseNumber(in.Days)
	calendar := decimal.NewFromInt(int64(end.Time.Sub(start.Time)/(24*time.Hour)) + 1)
	if !ok || !days.IsPositive() || days.GreaterThan(calendar) || !days.Mul(decimal.NewFromInt(2)).IsInteger() {
		return h, row, "", ErrLeaveDays
	}
	t, err := store.New(platform.DBFrom(ctx)).GetLeaveType(ctx, in.LeaveTypeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, row, "", platform.ErrNotFound
	}
	if err != nil {
		return h, row, "", err
	}
	if !t.Active {
		return h, row, "", ErrLeaveTypeInactive
	}
	row = store.CreateLeaveRequestParams{LeaveTypeID: t.ID, StartDate: start, EndDate: end, Days: in.Days, Reason: platform.NullText(in.Reason)}
	h = record.Header{Date: in.StartDate, OrgUnitID: unit, Fields: map[string]string{"days": in.Days, "leave_type": strconv.FormatInt(t.ID, 10)}}
	return h, row, t.Name, nil
}

// leaveChanges names the kind of leave rather than its id, so history reads without lookups.
func leaveChanges(old LeaveFields, oldType string, n LeaveFields, newType string) []audit.Change {
	return []audit.Change{
		{Field: "leave_type", Old: oldType, New: newType},
		{Field: "start_date", Old: old.StartDate, New: n.StartDate},
		{Field: "end_date", Old: old.EndDate, New: n.EndDate},
		{Field: "days", Old: old.Days, New: n.Days},
		{Field: "reason", Old: old.Reason, New: n.Reason},
	}
}

// canSeeBalance: the employee themself, or leave viewers and balance adjusters at their unit.
func (s *Service) canSeeBalance(ctx context.Context, employeeID int64) (bool, error) {
	e, err := store.New(platform.DBFrom(ctx)).GetEmployee(ctx, employeeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if actor, _ := platform.ActorFrom(ctx); e.UserID.Valid && e.UserID.Int64 == actor {
		return true, nil
	}
	if ok, err := s.allowed(ctx, PermLeaveView, e.OrgUnitID); err != nil || ok {
		return ok, err
	}
	return s.allowed(ctx, PermBalanceAdjust, e.OrgUnitID)
}

// LeaveBalances lists an employee's remaining leave by year.
func (s *Service) LeaveBalances(ctx context.Context, employeeID int64) ([]LeaveBalance, error) {
	if ok, err := s.canSeeBalance(ctx, employeeID); err != nil || !ok {
		return nil, platform.OrErr(err, platform.ErrNotFound)
	}
	rows, err := store.New(platform.DBFrom(ctx)).LeaveBalances(ctx, employeeID)
	out := make([]LeaveBalance, len(rows))
	for i, r := range rows {
		out[i] = LeaveBalance{Year: r.Year, Days: plain(r.Days)}
	}
	return out, err
}

// AdjustLeaveBalance grants or takes away leave days of a year, with a reason. It
// locks the balance row like approving a request does, so the two never interleave.
func (s *Service) AdjustLeaveBalance(ctx context.Context, employeeID int64, year int32, in BalanceAdjustment) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		e, err := store.New(platform.DBFrom(ctx)).GetEmployee(ctx, employeeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		if ok, err := s.allowed(ctx, PermBalanceAdjust, e.OrgUnitID); err != nil || !ok {
			return platform.OrErr(err, platform.ErrNotFound)
		}
		if err := platform.ProductGate(ctx, "hrm", platform.ClassWrite); err != nil {
			return err
		}
		if d, ok := record.ParseNumber(in.Delta); !ok || !d.Mul(decimal.NewFromInt(2)).IsInteger() {
			return ErrLeaveDays
		}
		// The API checks it too, but an import reaches here without the API.
		if strings.TrimSpace(in.Reason) == "" {
			return ErrReasonRequired
		}
		q := store.New(platform.DBFrom(ctx))
		if err := q.EnsureLeaveBalance(ctx, store.EnsureLeaveBalanceParams{EmployeeID: employeeID, Year: year}); err != nil {
			return err
		}
		after, err := q.AddLeaveBalance(ctx, store.AddLeaveBalanceParams{EmployeeID: employeeID, Year: year, Delta: in.Delta})
		if platform.Violates(err, "leave_balances_days_check") {
			return ErrNegativeBalance
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "hrm.leave_balance_adjusted", audit.Ref{Type: employeeType, ID: employeeID},
			map[string]any{"year": year, "delta": in.Delta, "days": plain(after), "reason": in.Reason})
	})
}

// plain writes a numeric column's text without trailing zeros: "1.0" is "1", "2.5" stays.
func plain(s string) string {
	if d, err := decimal.NewFromString(s); err == nil {
		return d.String()
	}
	return s
}
