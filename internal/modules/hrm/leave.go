package hrm

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (s *Service) leaveTypeOptions(ctx context.Context) ([]record.Option, error) {
	rows, err := store.ListLeaveTypes.Run(ctx)
	out := make([]record.Option, len(rows))
	for i, r := range rows {
		out[i] = record.Option{Value: strconv.FormatInt(r.ID, 10), Label: r.Name}
	}
	return out, err
}

// LeaveTypes lists every kind of leave, inactive ones included.
func (s *Service) LeaveTypes(ctx context.Context) ([]LeaveType, error) {
	rows, err := store.ListLeaveTypes.Run(ctx)
	out := make([]LeaveType, len(rows))
	for i, r := range rows {
		out[i] = LeaveType{ID: r.ID, LeaveTypeInput: LeaveTypeInput{Name: r.Name, DeductsBalance: r.DeductsBalance, Paid: r.Paid, Active: r.Active,
			MaxDays: numericText(r.MaxDays), RequiresAttachment: r.RequiresAttachment}}
	}
	return out, err
}

// SaveLeaveType creates a kind of leave (id 0) or replaces one. Kinds are tenant-wide.
func (s *Service) SaveLeaveType(ctx context.Context, id int64, in LeaveTypeInput) (int64, error) {
	if err := s.d.IAM.RequireTenantWide(ctx, "hrm", PermLeaveTypeManage); err != nil {
		return 0, err
	}
	if in.MaxDays != nil {
		if m, ok := record.ParseNumber(*in.MaxDays); !ok || !m.IsPositive() {
			return 0, ErrLeaveDays
		}
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var err error
		if id == 0 {
			id, err = store.CreateLeaveType.Run(ctx, store.CreateLeaveTypeParams{Name: in.Name, DeductsBalance: in.DeductsBalance, Paid: in.Paid, Active: in.Active,
				MaxDays: platform.NullText(in.MaxDays), RequiresAttachment: in.RequiresAttachment})
		} else {
			var n int64
			n, err = store.UpdateLeaveType.Run(ctx, store.UpdateLeaveTypeParams{ID: id, Name: in.Name, DeductsBalance: in.DeductsBalance, Paid: in.Paid, Active: in.Active,
				MaxDays: platform.NullText(in.MaxDays), RequiresAttachment: in.RequiresAttachment})
			if err == nil && n == 0 {
				return platform.ErrNotFound
			}
		}
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "hrm.leave_type_saved", map[string]any{"id": id, "name": in.Name, "deducts_balance": in.DeductsBalance, "paid": in.Paid, "active": in.Active,
			"max_days": in.MaxDays, "requires_attachment": in.RequiresAttachment})
	})
	return id, err
}

func (s *Service) leaveGrant(ctx context.Context, id int64, action record.Action) (bool, error) {
	r, err := store.GetLeaveRequest.Run(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return selfGrant(ctx, r.EmployeeID, r.EmployeeUserID, action)
}

func (s *Service) leaveApprovers(ctx context.Context, ref record.Ref, level int) ([]int64, error) {
	r, err := store.GetLeaveRequest.Run(ctx, ref.ID)
	if err != nil {
		return nil, err
	}
	return managerApprover(ctx, r.EmployeeID, level)
}

func (s *Service) leaveSubjects(ctx context.Context, id int64) ([]int64, error) {
	r, err := store.GetLeaveRequest.Run(ctx, id)
	return subjectOf(r.EmployeeUserID), err
}

// leaveBeforeSubmit refuses to send a request whose kind needs a file while it has none.
func (s *Service) leaveBeforeSubmit(ctx context.Context, d record.Doc) error {
	r, err := store.GetLeaveRequest.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	return checkLeaveAttachment(ctx, r)
}

func checkLeaveAttachment(ctx context.Context, r store.GetLeaveRequestRow) error {
	if !r.RequiresAttachment {
		return nil
	}
	ok, err := store.LeaveHasAttachment.Run(ctx, r.ID)
	if err == nil && !ok {
		err = ErrLeaveAttachment
	}
	return err
}

// leaveTransition deducts the balance when a request is approved, records what it took,
// and gives exactly that back when an approved request is cancelled. Either way the
// leave's days must not fall in a closed payroll period.
func (s *Service) leaveTransition(ctx context.Context, d record.Doc, from record.Status) error {
	approved, ok := payrollEffect(d, from)
	if !ok {
		return nil
	}
	r, err := store.GetLeaveRequest.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	if err := checkPayrollPeriods(ctx, d.LegalEntityID, *platform.DatePtr(r.StartDate), platform.DatePtr(r.EndDate)); err != nil {
		return err
	}
	if approved {
		// Locking the employee serializes one person's approvals, so two requests cannot
		// both find the dates free. Contracts take the same lock at the same point.
		if _, err := store.LockEmployee.Run(ctx, r.EmployeeID); err != nil {
			return err
		}
		if err := s.checkLeaveDates(ctx, d, r); err != nil {
			return err
		}
		// The file may have been removed while the request waited, and the kind may have
		// been tightened since the draft was saved.
		if err := checkLeaveAttachment(ctx, r); err != nil {
			return err
		}
		days, err := decimal.NewFromString(r.Days)
		if err != nil {
			return err
		}
		if err := checkLeaveTypeMax(r.MaxDays, days); err != nil {
			return err
		}
	}
	if approved && !r.DeductsBalance {
		return nil
	}
	year := int32(r.StartDate.Time.Year())
	bal, err := store.LockLeaveBalance.Run(ctx, store.LockLeaveBalanceParams{EmployeeID: r.EmployeeID, Year: year})
	if errors.Is(err, pgx.ErrNoRows) {
		if approved {
			return ErrInsufficientBalance
		}
		// A cancel of a request that took nothing has nothing to give back.
		return nil
	}
	if err != nil {
		return err
	}
	days, carried := decimal.RequireFromString(bal.Days), decimal.RequireFromString(bal.Carried)
	var delta, fromCarried decimal.Decimal
	if approved {
		delta = decimal.RequireFromString(r.Days).Neg()
		// Leave dated while carried days last spends them first; leave dated after they lapse
		// may not spend them at all.
		if bal.CarriedUntil.Valid && !r.StartDate.Time.After(bal.CarriedUntil.Time) {
			fromCarried = decimal.Min(carried, delta.Neg())
		} else if days.Sub(carried).Add(delta).IsNegative() {
			return ErrInsufficientBalance
		}
		carried = carried.Sub(fromCarried)
	} else {
		delta = decimal.RequireFromString(r.Deducted)
		// Carried days come back as carried only while they last; afterwards they would have lapsed.
		if back := decimal.RequireFromString(r.DeductedCarried); back.IsPositive() {
			today, err := s.d.Setting.Today(ctx)
			if err != nil {
				return err
			}
			if bal.CarriedUntil.Valid && !today.Time.After(bal.CarriedUntil.Time) {
				carried = carried.Add(back)
			} else {
				delta = delta.Sub(back)
			}
		}
	}
	if delta.IsZero() && fromCarried.IsZero() {
		return nil
	}
	days = days.Add(delta)
	if days.IsNegative() {
		return ErrInsufficientBalance
	}
	if err := store.SetLeaveBalance.Run(ctx, store.SetLeaveBalanceParams{EmployeeID: r.EmployeeID, Year: year,
		Days: days.String(), Carried: decimal.Min(carried, days).String(), CarriedUntil: bal.CarriedUntil}); err != nil {
		return err
	}
	deducted, deductedCarried := decimal.Zero, decimal.Zero
	if approved {
		deducted, deductedCarried = delta.Neg(), fromCarried
	}
	return store.SetLeaveDeducted.Run(ctx, store.SetLeaveDeductedParams{ID: d.ID, Deducted: deducted.String(), DeductedCarried: deductedCarried.String()})
}

// LeaveActions lists what the actor may do before picking a leave request.
func (s *Service) LeaveActions(ctx context.Context) (SelfServiceActions, error) {
	return s.selfServiceActions(ctx, leaveType)
}

// Leaves lists the leave requests the actor may see: their own, their reports', and those in scope.
func (s *Service) Leaves(ctx context.Context, f LeaveFilter) (LeaveList, error) {
	out := LeaveList{Items: []LeaveListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "hrm", PermLeaveView)
	if err != nil {
		return out, err
	}
	actor, _ := platform.ActorFrom(ctx)
	page, err := leaveList.Run(ctx, leaveQuery(sc, actor, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, LeaveListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, EmployeeID: r.EmployeeID, EmployeeCode: r.EmployeeCode,
			EmployeeName: r.EmployeeName, LeaveTypeName: r.LeaveTypeName, StartDate: *platform.DatePtr(r.StartDate),
			EndDate: *platform.DatePtr(r.EndDate), Days: platform.PlainDecimal(r.Days),
		})
	}
	return out, nil
}

// Leave returns one leave request; one the actor may not view does not exist.
func (s *Service) Leave(ctx context.Context, id int64) (Leave, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: leaveType, ID: id})
	if err != nil {
		return Leave{}, err
	}
	r, err := store.GetLeaveRequest.Run(ctx, id)
	if err != nil {
		return Leave{}, err
	}
	actions, err := s.d.Record.Actions(ctx, d)
	if err != nil {
		return Leave{}, err
	}
	out := Leave{
		ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, EmployeeID: r.EmployeeID,
		EmployeeCode: r.EmployeeCode, EmployeeName: r.EmployeeName, LeaveTypeName: r.LeaveTypeName,
		LeaveFields: LeaveFields{LeaveTypeID: r.LeaveTypeID, StartDate: *platform.DatePtr(r.StartDate), EndDate: *platform.DatePtr(r.EndDate),
			Days: platform.PlainDecimal(r.Days), Reason: platform.TextPtr(r.Reason), StartHalf: r.StartHalf, EndHalf: r.EndHalf},
		AllowedActions: actions,
	}
	if ok, err := s.canSeeBalance(ctx, r.EmployeeID); err != nil {
		return Leave{}, err
	} else if ok {
		balances, err := store.LeaveBalances.Run(ctx, r.EmployeeID)
		if err != nil {
			return Leave{}, err
		}
		out.Balance = new("0")
		for _, b := range balances {
			if b.Year == int32(r.StartDate.Time.Year()) {
				out.Balance = new(platform.PlainDecimal(b.Days))
			}
		}
	}
	return out, nil
}

// CreateLeave adds a draft leave request for the actor's own employee record, or as HR for another.
func (s *Service) CreateLeave(ctx context.Context, in NewLeave) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		employee, unit, user, err := subject(ctx, in.EmployeeID)
		if err != nil {
			return err
		}
		if err := s.d.Record.MayCreate(ctx, leaveType, record.Header{OrgUnitID: unit, Subject: user.Int64}); err != nil {
			return err
		}
		h, row, typeName, err := s.leaveHeader(ctx, unit, in.LeaveFields)
		if err != nil {
			return err
		}
		h.RequestID = in.RequestID
		h.Subject = user.Int64
		d, err := s.d.Record.Create(ctx, leaveType, h)
		id = d.ID
		if err != nil || d.Reused {
			return err
		}
		row.ID, row.EmployeeID = id, employee
		if err := store.CreateLeaveRequest.Run(ctx, row); err != nil {
			return err
		}
		in.Days = row.Days
		return s.d.Audit.RecordChanges(ctx, "hrm.leave_request_created", audit.Ref{Type: leaveType, ID: id}, leaveChanges(LeaveFields{}, "", in.LeaveFields, typeName))
	})
	return id, err
}

// UpdateLeave replaces the fields of a draft at the version the actor saw.
func (s *Service) UpdateLeave(ctx context.Context, id int64, in LeaveUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, record.Ref{Type: leaveType, ID: id}); err != nil {
			return err
		}
		old, err := store.GetLeaveRequest.Run(ctx, id)
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
		if err := store.UpdateLeaveRequest.Run(ctx, store.UpdateLeaveRequestParams{
			ID: id, LeaveTypeID: row.LeaveTypeID, StartDate: row.StartDate, EndDate: row.EndDate, Days: row.Days, Reason: row.Reason,
			StartHalf: row.StartHalf, EndHalf: row.EndHalf,
		}); err != nil {
			return err
		}
		in.Days = row.Days
		before := LeaveFields{LeaveTypeID: old.LeaveTypeID, StartDate: *platform.DatePtr(old.StartDate), EndDate: *platform.DatePtr(old.EndDate), Days: platform.PlainDecimal(old.Days), Reason: platform.TextPtr(old.Reason),
			StartHalf: old.StartHalf, EndHalf: old.EndHalf}
		return s.d.Audit.RecordChanges(ctx, "hrm.leave_request_updated", audit.Ref{Type: leaveType, ID: id}, leaveChanges(before, old.LeaveTypeName, in.LeaveFields, typeName))
	})
}

func deleteLeave(ctx context.Context, id int64) error {
	return store.DeleteLeaveRequest.Run(ctx, id)
}

// checkLeaveDates re-checks on approval what the entry already checked, and refuses dates
// that would hold more than a day of absence. The entry check can be stale: a holiday added
// to the range after the draft was saved leaves fewer working days than the count assumes.
func (s *Service) checkLeaveDates(ctx context.Context, d record.Doc, r store.GetLeaveRequestRow) error {
	start, end := *platform.DatePtr(r.StartDate), *platform.DatePtr(r.EndDate)
	days, err := decimal.NewFromString(r.Days)
	if err != nil {
		return err
	}
	others, err := store.ApprovedLeaveOverlapping.Run(ctx, store.ApprovedLeaveOverlappingParams{
		EmployeeID: r.EmployeeID, ID: d.ID, StartDate: r.StartDate, EndDate: r.EndDate,
	})
	if err != nil {
		return err
	}
	// One calendar over every range involved: an overlapping request may reach outside this one.
	from, to := start, end
	for _, o := range others {
		from = min(from, *platform.DatePtr(o.StartDate))
		to = max(to, *platform.DatePtr(o.EndDate))
	}
	cal, err := workCalendar(ctx, d.LegalEntityID, from, to)
	if err != nil {
		return err
	}
	if r.Computed {
		// The calendar may have changed since the request was saved; the approved count must
		// be the one the requester saw.
		now, err := computedDays(cal, start, end, r.StartHalf, r.EndHalf)
		if err != nil {
			return err
		}
		if !now.Equal(days) {
			return leaveDaysChanged(now)
		}
	} else if err := checkLeaveDaysOn(cal, start, end, days); err != nil {
		// Saved before counts were computed: the typed count under the calendar cap.
		return err
	}
	// Add every other request up per date first: three half days on one date must fail
	// even though no single pair of them exceeds a day.
	taken := map[string]decimal.Decimal{}
	culprit := map[string]string{}
	for _, o := range others {
		oDays, err := decimal.NewFromString(o.Days)
		if err != nil {
			return err
		}
		for date, v := range halfAllocation(cal, *platform.DatePtr(o.StartDate), *platform.DatePtr(o.EndDate), oDays, o.StartHalf) {
			taken[date] = taken[date].Add(v)
			culprit[date] = o.Number
		}
	}
	for date, v := range halfAllocation(cal, start, end, days, r.StartHalf) {
		if v.Add(taken[date]).GreaterThan(one) {
			return &platform.Error{Status: http.StatusConflict, Code: "leave_overlaps",
				Params: map[string]any{"date": date, "number": culprit[date]}}
		}
	}
	return nil
}

// LeaveWorkingDays is how many days a leave request over [start, end] may claim: the
// ceiling the form shows before the employee enters a number. employee is whose calendar
// it is; nil means the actor's own record.
func (s *Service) LeaveWorkingDays(ctx context.Context, employee *int64, start, end string) (int, error) {
	// The same bounds the entry has, and for the same reason a request may not span two
	// years: without them a range of centuries would walk day by day through all of it.
	from, to := dayOf(start), dayOf(end)
	switch {
	case to.Before(from):
		return 0, ErrLeaveDates
	case from.Year() != to.Year():
		return 0, ErrLeaveSpansYears
	}
	_, unit, err := s.filer(ctx, employee, PermLeaveEdit)
	if err != nil {
		return 0, err
	}
	legalEntity, err := s.d.IAM.LegalEntityOf(ctx, unit)
	if err != nil {
		return 0, err
	}
	cal, err := workCalendar(ctx, legalEntity, start, end)
	if err != nil {
		return 0, err
	}
	return cal.workingDays(from, to), nil
}

// checkLeaveDaysOn holds a typed day count to the working days of [start, end]: the days the
// allocation can place it on. Without the cap the remainder is dropped from the paid days
// while the balance still loses it.
func checkLeaveDaysOn(cal calendar, start, end string, days decimal.Decimal) error {
	available := cal.workingDays(dayOf(start), dayOf(end))
	switch {
	case available == 0:
		return ErrLeaveNoWorkingDays
	case days.GreaterThan(decimal.NewFromInt(int64(available))):
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_days_exceed_working_days",
			Params: map[string]any{"working_days": available}}
	}
	return nil
}

// checkLeaveTypeMax refuses a request over its kind's cap, if the kind has one.
func checkLeaveTypeMax(maxDays pgtype.Numeric, days decimal.Decimal) error {
	if m := numericText(maxDays); m != nil && days.GreaterThan(decimal.RequireFromString(*m)) {
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_days_exceed_type_max",
			Params: map[string]any{"max_days": *m}}
	}
	return nil
}

func numericText(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	s := decimal.NewFromBigInt(n.Int, n.Exp).String()
	return &s
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
	sent, ok := record.ParseNumber(in.Days)
	if in.Days != "" && (!ok || !sent.IsPositive() || !sent.Mul(decimal.NewFromInt(2)).IsInteger()) {
		return h, row, "", ErrLeaveDays
	}
	// The legal entity is read under the tree lock, so the calendar cannot move under us.
	if err := s.d.IAM.ShareTree(ctx); err != nil {
		return h, row, "", err
	}
	legalEntity, err := s.d.IAM.LegalEntityOf(ctx, unit)
	if err != nil {
		return h, row, "", err
	}
	cal, err := workCalendar(ctx, legalEntity, in.StartDate, in.EndDate)
	if err != nil {
		return h, row, "", err
	}
	days, err := computedDays(cal, in.StartDate, in.EndDate, in.StartHalf, in.EndHalf)
	if err != nil {
		return h, row, "", err
	}
	if in.Days != "" && !sent.Equal(days) {
		return h, row, "", leaveDaysChanged(days)
	}
	t, err := store.GetLeaveType.Run(ctx, in.LeaveTypeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, row, "", platform.ErrNotFound
	}
	if err != nil {
		return h, row, "", err
	}
	if !t.Active {
		return h, row, "", ErrLeaveTypeInactive
	}
	if err := checkLeaveTypeMax(t.MaxDays, days); err != nil {
		return h, row, "", err
	}
	row = store.CreateLeaveRequestParams{LeaveTypeID: t.ID, StartDate: start, EndDate: end, Days: days.String(), Reason: platform.NullText(in.Reason),
		StartHalf: in.StartHalf, EndHalf: in.EndHalf}
	h = record.Header{Date: in.StartDate, OrgUnitID: unit, Fields: map[string]string{"days": days.String(), "leave_type": strconv.FormatInt(t.ID, 10)}}
	return h, row, t.Name, nil
}

// leaveChanges names the kind of leave rather than its id, so history reads without lookups.
func leaveChanges(old LeaveFields, oldType string, n LeaveFields, newType string) []audit.Change {
	return []audit.Change{
		{Field: "leave_type", Old: oldType, New: newType},
		{Field: "start_date", Old: old.StartDate, New: n.StartDate},
		{Field: "end_date", Old: old.EndDate, New: n.EndDate},
		{Field: "days", Old: old.Days, New: platform.PlainDecimal(n.Days)},
		{Field: "reason", Old: old.Reason, New: n.Reason},
		{Field: "start_half", Old: old.StartHalf, New: n.StartHalf},
		{Field: "end_half", Old: old.EndHalf, New: n.EndHalf},
	}
}

// canSeeBalance: the employee themself, or leave viewers and balance adjusters at their unit.
func (s *Service) canSeeBalance(ctx context.Context, employeeID int64) (bool, error) {
	e, err := store.GetEmployee.Run(ctx, employeeID)
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
	rows, err := store.LeaveBalances.Run(ctx, employeeID)
	out := make([]LeaveBalance, len(rows))
	for i, r := range rows {
		out[i] = LeaveBalance{Year: r.Year, Days: platform.PlainDecimal(r.Days), Carried: platform.PlainDecimal(r.Carried), CarriedUntil: platform.DatePtr(r.CarriedUntil)}
	}
	return out, err
}

// AdjustLeaveBalance grants or takes away leave days of a year, with a reason. It
// locks the balance row like approving a request does, so the two never interleave.
func (s *Service) AdjustLeaveBalance(ctx context.Context, employeeID int64, year int32, in BalanceAdjustment) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		e, err := store.GetEmployee.Run(ctx, employeeID)
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
		if err := store.EnsureLeaveBalance.Run(ctx, store.EnsureLeaveBalanceParams{EmployeeID: employeeID, Year: year}); err != nil {
			return err
		}
		after, err := store.AddLeaveBalance.Run(ctx, store.AddLeaveBalanceParams{EmployeeID: employeeID, Year: year, Delta: in.Delta})
		if platform.Violates(err, "leave_balances_days_check") {
			return ErrNegativeBalance
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "hrm.leave_balance_adjusted", audit.Ref{Type: employeeType, ID: employeeID},
			map[string]any{"year": year, "delta": in.Delta, "days": platform.PlainDecimal(after), "reason": in.Reason})
	})
}

// ErrLeaveHalfDays: both halves of the only working date of a request.
var ErrLeaveHalfDays = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "leave_half_days"}

// computedDays is a request's day count: the working days of [start, end] less half a day
// for each half flag, which fall on the first and last working dates.
func computedDays(c calendar, start, end string, startHalf, endHalf bool) (decimal.Decimal, error) {
	n := c.workingDays(dayOf(start), dayOf(end))
	switch {
	case n == 0:
		return decimal.Zero, ErrLeaveNoWorkingDays
	case n == 1 && startHalf && endHalf:
		return decimal.Zero, ErrLeaveHalfDays
	}
	days := decimal.NewFromInt(int64(n))
	for _, half := range []bool{startHalf, endHalf} {
		if half {
			days = days.Sub(decimal.New(5, -1))
		}
	}
	return days, nil
}

// leaveDaysChanged: the calendar gives another count than the one sent or saved.
func leaveDaysChanged(days decimal.Decimal) error {
	return &platform.Error{Status: http.StatusConflict, Code: "leave_days_changed", Params: map[string]any{"days": days.String()}}
}
