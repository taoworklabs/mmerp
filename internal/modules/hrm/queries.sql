-- name: ListEmployees :many
-- Scope filter: @all_units or org_unit_id in @units. Status uses @today, in the tenant time zone.
-- manager_of drops that employee and everyone under them, who cannot become their manager.
WITH RECURSIVE under (id) AS (
    SELECT sqlc.narg(manager_of)::bigint WHERE sqlc.narg(manager_of)::bigint IS NOT NULL
    UNION
    SELECT s.id FROM hrm.employees s JOIN under ON s.manager_id = under.id
)
SELECT e.id, e.code, e.full_name, e.org_unit_id, u.name AS org_unit_name, e.email, e.phone,
       e.hire_date, e.termination_date, count(*) OVER () AS total
FROM hrm.employees e JOIN iam.org_units u ON u.id = e.org_unit_id
WHERE (@all_units::bool OR e.org_unit_id = ANY(@units::bigint[]))
  AND (@q::text = '' OR e.code ILIKE '%' || @q || '%' OR e.full_name ILIKE '%' || @q || '%')
  AND (sqlc.narg(org_unit_id)::bigint IS NULL OR e.org_unit_id = sqlc.narg(org_unit_id))
  AND (@status::text = ''
       OR (@status = 'active' AND (e.termination_date IS NULL OR e.termination_date >= @today::date))
       OR (@status = 'terminated' AND e.termination_date < @today::date))
  AND e.id NOT IN (SELECT id FROM under)
ORDER BY
    CASE WHEN @sort::text = 'code' THEN lower(e.code) END,
    CASE WHEN @sort = '-code' THEN lower(e.code) END DESC,
    CASE WHEN @sort = 'full_name' THEN e.full_name END,
    CASE WHEN @sort = '-full_name' THEN e.full_name END DESC,
    CASE WHEN @sort = 'hire_date' THEN e.hire_date END,
    CASE WHEN @sort = '-hire_date' THEN e.hire_date END DESC,
    e.id
LIMIT @lim OFFSET @off;

-- name: GetEmployee :one
SELECT e.*, u.name AS org_unit_name, m.code AS manager_code, m.full_name AS manager_name, a.login AS user_login
FROM hrm.employees e
JOIN iam.org_units u ON u.id = e.org_unit_id
LEFT JOIN hrm.employees m ON m.id = e.manager_id
LEFT JOIN iam.users a ON a.id = e.user_id
WHERE e.id = $1;

-- name: LockEmployee :one
SELECT * FROM hrm.employees WHERE id = $1 FOR UPDATE;

-- name: CreateEmployee :one
INSERT INTO hrm.employees (code, full_name, date_of_birth, gender, phone, email, address, org_unit_id,
                           manager_id, user_id, hire_date, termination_date,
                           national_id, social_insurance_no, tax_code, bank_account)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING id;

-- name: UpdateEmployee :exec
UPDATE hrm.employees SET code = $2, full_name = $3, date_of_birth = $4, gender = $5, phone = $6, email = $7,
    address = $8, org_unit_id = $9, manager_id = $10, user_id = $11, hire_date = $12, termination_date = $13,
    national_id = $14, social_insurance_no = $15, tax_code = $16, bank_account = $17
WHERE id = $1;

-- name: ListDependents :many
SELECT id, data FROM hrm.dependents WHERE employee_id = $1 ORDER BY id;

-- name: GetDependent :one
SELECT data FROM hrm.dependents WHERE id = $1 AND employee_id = $2 FOR UPDATE;

-- name: CreateDependent :one
INSERT INTO hrm.dependents (employee_id, data) VALUES ($1, $2) RETURNING id;

-- name: UpdateDependent :exec
UPDATE hrm.dependents SET data = $3 WHERE id = $1 AND employee_id = $2;

-- name: DeleteDependent :exec
DELETE FROM hrm.dependents WHERE id = $1 AND employee_id = $2;

-- name: ManagerChain :many
-- The employee and everyone above it; UNION stops on a cycle.
WITH RECURSIVE chain (employee_id, next_id) AS (
    SELECT s.id, s.manager_id FROM hrm.employees s WHERE s.id = @start::bigint
    UNION
    SELECT e.id, e.manager_id FROM hrm.employees e JOIN chain ON e.id = chain.next_id
)
SELECT employee_id FROM chain;

-- name: LockManagerTree :exec
-- Serialises manager changes so two concurrent edits cannot close a cycle.
SELECT pg_advisory_xact_lock(hashtext('hrm.manager_tree'));

-- name: ListLeaveTypes :many
SELECT * FROM hrm.leave_types ORDER BY lower(name);

-- name: GetLeaveType :one
SELECT * FROM hrm.leave_types WHERE id = $1;

-- name: CreateLeaveType :one
INSERT INTO hrm.leave_types (name, deducts_balance, paid, active) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: UpdateLeaveType :execrows
UPDATE hrm.leave_types SET name = $2, deducts_balance = $3, paid = $4, active = $5 WHERE id = $1;

-- name: GetLeaveRequest :one
SELECT r.id, r.employee_id, r.leave_type_id, r.start_date, r.end_date, r.days::text AS days, r.reason,
       r.deducted::text AS deducted,
       e.code AS employee_code, e.full_name AS employee_name, e.user_id AS employee_user_id,
       t.name AS leave_type_name, t.deducts_balance, d.org_unit_id
FROM hrm.leave_requests r
JOIN hrm.employees e ON e.id = r.employee_id
JOIN hrm.leave_types t ON t.id = r.leave_type_id
JOIN record.documents d ON d.id = r.id
WHERE r.id = $1;

-- name: CreateLeaveRequest :exec
INSERT INTO hrm.leave_requests (id, employee_id, leave_type_id, start_date, end_date, days, reason)
VALUES (@id, @employee_id, @leave_type_id, @start_date, @end_date, CAST(@days::text AS numeric), @reason);

-- name: UpdateLeaveRequest :exec
UPDATE hrm.leave_requests SET leave_type_id = @leave_type_id, start_date = @start_date, end_date = @end_date,
    days = CAST(@days::text AS numeric), reason = @reason
WHERE id = @id;

-- name: SetLeaveDeducted :exec
UPDATE hrm.leave_requests SET deducted = CAST(@deducted::text AS numeric) WHERE id = @id;

-- name: DeleteLeaveRequest :exec
DELETE FROM hrm.leave_requests WHERE id = $1;

-- name: ListLeaveRequests :many
-- Visible: org unit in scope, the actor's own requests, or requests of the employees in @reports.
SELECT r.id, d.number, d.status, r.employee_id, e.code AS employee_code, e.full_name AS employee_name,
       t.name AS leave_type_name, r.start_date, r.end_date, r.days::text AS days, count(*) OVER () AS total
FROM hrm.leave_requests r
JOIN record.documents d ON d.id = r.id
JOIN hrm.employees e ON e.id = r.employee_id
JOIN hrm.leave_types t ON t.id = r.leave_type_id
WHERE (@all_units::bool OR d.org_unit_id = ANY(@units::bigint[]) OR e.user_id = @actor::bigint OR r.employee_id = ANY(@reports::bigint[]))
  AND (@status::text = '' OR d.status = @status)
  AND (sqlc.narg(employee_id)::bigint IS NULL OR r.employee_id = sqlc.narg(employee_id))
  AND (sqlc.narg(from_date)::date IS NULL OR r.end_date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR r.start_date <= sqlc.narg(to_date))
ORDER BY
    CASE WHEN @sort::text = 'start_date' THEN r.start_date END,
    CASE WHEN @sort = '-start_date' THEN r.start_date END DESC,
    CASE WHEN @sort = 'number' THEN d.number END,
    CASE WHEN @sort = '-number' THEN d.number END DESC,
    r.id DESC
LIMIT @lim OFFSET @off;

-- name: Reports :many
-- Employees below the actor's own employee record, at any depth.
WITH RECURSIVE down AS (
    SELECT e.id FROM hrm.employees e JOIN hrm.employees me ON e.manager_id = me.id WHERE me.user_id = @actor::bigint
    UNION
    SELECT e.id FROM hrm.employees e JOIN down ON e.manager_id = down.id
)
SELECT down.id FROM down;

-- name: IsManager :one
-- Whether the user is linked to an employee above @employee on its manager chain.
WITH RECURSIVE up AS (
    SELECT e.manager_id AS id FROM hrm.employees e WHERE e.id = @employee::bigint
    UNION
    SELECT e.manager_id FROM hrm.employees e JOIN up ON e.id = up.id
)
SELECT EXISTS (SELECT 1 FROM up JOIN hrm.employees m ON m.id = up.id WHERE m.user_id = @actor::bigint);

-- name: ManagerUser :one
-- The account of the manager @level steps above @employee.
WITH RECURSIVE chain (id, depth) AS (
    SELECT e.manager_id, 1 FROM hrm.employees e WHERE e.id = @employee::bigint
    UNION ALL
    SELECT e.manager_id, chain.depth + 1 FROM hrm.employees e JOIN chain ON e.id = chain.id WHERE chain.depth < @level::int
)
SELECT m.user_id FROM chain JOIN hrm.employees m ON m.id = chain.id WHERE chain.depth = @level::int;

-- name: EmployeeByUser :one
SELECT id, full_name, org_unit_id FROM hrm.employees WHERE user_id = $1;

-- name: EnsureLeaveBalance :exec
INSERT INTO hrm.leave_balances (employee_id, year, days) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING;

-- name: AddLeaveBalance :one
-- Locks the row until commit; a negative result fails leave_balances_days_check.
UPDATE hrm.leave_balances SET days = days + CAST(@delta::text AS numeric)
WHERE employee_id = @employee_id AND year = @year
RETURNING days::text;

-- name: LeaveBalances :many
SELECT year, days::text AS days FROM hrm.leave_balances WHERE employee_id = $1 ORDER BY year DESC;

-- name: EnsurePayrollLock :exec
INSERT INTO hrm.payroll_locks (legal_entity_id) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: SharePayrollLock :exec
SELECT legal_entity_id FROM hrm.payroll_locks WHERE legal_entity_id = $1 FOR SHARE;

-- name: ClosedPeriods :many
-- Closed periods of the legal entity overlapping [@from, @to]; a null @to is open-ended.
SELECT period_start, period_end FROM hrm.payroll_periods
WHERE legal_entity_id = @legal_entity_id AND posted_payroll_id IS NOT NULL
  AND period_end >= @from_date::date AND (sqlc.narg(to_date)::date IS NULL OR period_start <= sqlc.narg(to_date))
ORDER BY period_start;

-- name: ListContractTypes :many
SELECT * FROM hrm.contract_types ORDER BY lower(name);

-- name: GetContractType :one
SELECT * FROM hrm.contract_types WHERE id = $1;

-- name: CreateContractType :one
INSERT INTO hrm.contract_types (name, fixed_term, active) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateContractType :execrows
UPDATE hrm.contract_types SET name = $2, fixed_term = $3, active = $4 WHERE id = $1;

-- name: GetContract :one
SELECT c.id, c.employee_id, c.contract_type_id, c.parent_id, c.start_date, c.end_date, c.terms,
       e.code AS employee_code, e.full_name AS employee_name, e.user_id AS employee_user_id,
       t.name AS contract_type_name, d.org_unit_id,
       p.parent_id AS parent_parent_id, p.start_date AS parent_start, p.end_date AS parent_end,
       pd.number AS parent_number, pd.status AS parent_status
FROM hrm.contracts c
JOIN hrm.employees e ON e.id = c.employee_id
JOIN hrm.contract_types t ON t.id = c.contract_type_id
JOIN record.documents d ON d.id = c.id
LEFT JOIN hrm.contracts p ON p.id = c.parent_id
LEFT JOIN record.documents pd ON pd.id = c.parent_id
WHERE c.id = $1;

-- name: CreateContract :exec
INSERT INTO hrm.contracts (id, employee_id, contract_type_id, parent_id, start_date, end_date, terms)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateContract :exec
UPDATE hrm.contracts SET contract_type_id = $2, start_date = $3, end_date = $4, terms = $5 WHERE id = $1;

-- name: DeleteContract :exec
DELETE FROM hrm.contracts WHERE id = $1;

-- name: ListEmployeeContracts :many
-- Originals each followed by their appendices; scope filter on the document's org unit.
SELECT c.id, d.number, d.status, c.parent_id, t.name AS contract_type_name, c.start_date, c.end_date
FROM hrm.contracts c
JOIN record.documents d ON d.id = c.id
JOIN hrm.contract_types t ON t.id = c.contract_type_id
WHERE c.employee_id = @employee_id AND (@all_units::bool OR d.org_unit_id = ANY(@units::bigint[]))
ORDER BY (SELECT o.start_date FROM hrm.contracts o WHERE o.id = coalesce(c.parent_id, c.id)) DESC,
         coalesce(c.parent_id, c.id) DESC, c.parent_id NULLS FIRST, c.start_date, c.id;

-- name: ListContracts :many
-- Every contract and appendix in scope. Expiring: posted originals ending from @today to @today + 30 days.
SELECT c.id, d.number, d.status, c.parent_id, p.number AS parent_number, t.name AS contract_type_name,
       c.start_date, c.end_date, c.employee_id, e.code AS employee_code, e.full_name AS employee_name,
       count(*) OVER () AS total
FROM hrm.contracts c
JOIN record.documents d ON d.id = c.id
JOIN hrm.contract_types t ON t.id = c.contract_type_id
JOIN hrm.employees e ON e.id = c.employee_id
LEFT JOIN record.documents p ON p.id = c.parent_id
WHERE (@all_units::bool OR d.org_unit_id = ANY(@units::bigint[]))
  AND (@status::text = '' OR d.status = @status)
  AND (sqlc.narg(contract_type_id)::bigint IS NULL OR c.contract_type_id = sqlc.narg(contract_type_id))
  AND (sqlc.narg(org_unit_id)::bigint IS NULL OR d.org_unit_id = sqlc.narg(org_unit_id))
  AND (sqlc.narg(employee_id)::bigint IS NULL OR c.employee_id = sqlc.narg(employee_id))
  AND (NOT sqlc.arg(expiring)::bool OR (c.parent_id IS NULL AND d.status = 'posted'
       AND c.end_date BETWEEN sqlc.arg(today)::date AND sqlc.arg(today)::date + 30))
ORDER BY
    CASE WHEN @sort::text = 'start_date' THEN c.start_date END,
    CASE WHEN @sort = '-start_date' THEN c.start_date END DESC,
    CASE WHEN @sort = 'end_date' THEN c.end_date END,
    CASE WHEN @sort = '-end_date' THEN c.end_date END DESC NULLS LAST,
    CASE WHEN @sort = 'number' THEN d.number END,
    CASE WHEN @sort = '-number' THEN d.number END DESC,
    c.id DESC
LIMIT @lim OFFSET @off;

-- name: ContractOverlaps :one
-- Whether another posted original of the employee overlaps [@start, @end] (null end: open-ended).
SELECT EXISTS (
    SELECT 1 FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
    WHERE c.employee_id = @employee_id AND c.id <> @id AND c.parent_id IS NULL AND d.status = 'posted'
      AND daterange(c.start_date, c.end_date, '[]') && daterange(@start_date::date, sqlc.narg(end_date)::date, '[]')
);

-- name: HasPostedAppendix :one
SELECT EXISTS (
    SELECT 1 FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
    WHERE c.parent_id = $1 AND d.status = 'posted'
);

-- name: ContractTermsAt :one
-- The terms in force at @at, and the contract holding them: the posted original covering
-- it, or its latest posted appendix effective by then.
SELECT coalesce(a.id, c.id)::bigint AS id, coalesce(a.terms, c.terms)::bytea AS terms
FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
LEFT JOIN LATERAL (
    SELECT x.id, x.terms FROM hrm.contracts x JOIN record.documents xd ON xd.id = x.id
    WHERE x.parent_id = c.id AND xd.status = 'posted' AND x.start_date <= @at::date
    ORDER BY x.start_date DESC, x.id DESC LIMIT 1
) a ON true
WHERE c.employee_id = @employee_id AND c.parent_id IS NULL AND d.status = 'posted'
  AND c.start_date <= @at AND (c.end_date IS NULL OR c.end_date >= @at);

-- name: GetOvertimeRequest :one
SELECT r.id, r.employee_id, r.date, r.day_kind, r.day_hours::text AS day_hours, r.night_hours::text AS night_hours, r.reason,
       e.code AS employee_code, e.full_name AS employee_name, e.user_id AS employee_user_id, d.org_unit_id
FROM hrm.overtime_requests r
JOIN hrm.employees e ON e.id = r.employee_id
JOIN record.documents d ON d.id = r.id
WHERE r.id = $1;

-- name: CreateOvertimeRequest :exec
INSERT INTO hrm.overtime_requests (id, employee_id, date, day_kind, day_hours, night_hours, reason)
VALUES (@id, @employee_id, @date, @day_kind, CAST(@day_hours::text AS numeric), CAST(@night_hours::text AS numeric), @reason);

-- name: UpdateOvertimeRequest :exec
UPDATE hrm.overtime_requests SET date = @date, day_kind = @day_kind, day_hours = CAST(@day_hours::text AS numeric),
    night_hours = CAST(@night_hours::text AS numeric), reason = @reason
WHERE id = @id;

-- name: DeleteOvertimeRequest :exec
DELETE FROM hrm.overtime_requests WHERE id = $1;

-- name: ListOvertimeRequests :many
-- Visible as for leave requests.
SELECT r.id, d.number, d.status, r.employee_id, e.code AS employee_code, e.full_name AS employee_name,
       r.date, r.day_kind, r.day_hours::text AS day_hours, r.night_hours::text AS night_hours, count(*) OVER () AS total
FROM hrm.overtime_requests r
JOIN record.documents d ON d.id = r.id
JOIN hrm.employees e ON e.id = r.employee_id
WHERE (@all_units::bool OR d.org_unit_id = ANY(@units::bigint[]) OR e.user_id = @actor::bigint OR r.employee_id = ANY(@reports::bigint[]))
  AND (@status::text = '' OR d.status = @status)
  AND (sqlc.narg(employee_id)::bigint IS NULL OR r.employee_id = sqlc.narg(employee_id))
  AND (sqlc.narg(from_date)::date IS NULL OR r.date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR r.date <= sqlc.narg(to_date))
ORDER BY
    CASE WHEN @sort::text = 'date' THEN r.date END,
    CASE WHEN @sort = '-date' THEN r.date END DESC,
    CASE WHEN @sort = 'number' THEN d.number END,
    CASE WHEN @sort = '-number' THEN d.number END DESC,
    r.id DESC
LIMIT @lim OFFSET @off;

-- name: CreateTimesheet :exec
INSERT INTO hrm.timesheets (id, org_unit_id, period_start, period_end) VALUES ($1, $2, $3, $4);

-- name: GetTimesheet :one
SELECT t.*, u.name AS org_unit_name
FROM hrm.timesheets t
JOIN iam.org_units u ON u.id = t.org_unit_id
WHERE t.id = $1;

-- name: UpdateTimesheetPeriod :exec
UPDATE hrm.timesheets SET period_start = $2, period_end = $3 WHERE id = $1;

-- name: DeactivateTimesheet :exec
UPDATE hrm.timesheets SET active = false WHERE id = $1;

-- name: DeleteTimesheet :exec
DELETE FROM hrm.timesheets WHERE id = $1;

-- name: TimesheetEmployees :many
-- The grid: employees of exactly the unit employed during the period, and anyone with lines.
SELECT e.id, e.code, e.full_name, e.hire_date, e.termination_date
FROM hrm.employees e
WHERE (e.org_unit_id = @org_unit_id AND e.hire_date <= @period_end::date
       AND (e.termination_date IS NULL OR e.termination_date >= @period_start::date))
   OR e.id IN (SELECT l.employee_id FROM hrm.timesheet_lines l WHERE l.timesheet_id = @timesheet_id)
ORDER BY lower(e.code);

-- name: TimesheetLines :many
SELECT employee_id, date, days::text AS days FROM hrm.timesheet_lines WHERE timesheet_id = $1 ORDER BY employee_id, date;

-- name: DeleteTimesheetLines :exec
DELETE FROM hrm.timesheet_lines WHERE timesheet_id = $1;

-- name: InsertTimesheetLines :exec
INSERT INTO hrm.timesheet_lines (timesheet_id, employee_id, date, days)
SELECT @timesheet_id::bigint, unnest(@employee_ids::bigint[]), unnest(@dates::date[]), unnest(@days::text[])::numeric;

-- name: ListTimesheets :many
SELECT t.id, d.number, d.status, t.org_unit_id, u.name AS org_unit_name, t.period_start, t.period_end, count(*) OVER () AS total
FROM hrm.timesheets t
JOIN record.documents d ON d.id = t.id
JOIN iam.org_units u ON u.id = t.org_unit_id
WHERE (@all_units::bool OR t.org_unit_id = ANY(@units::bigint[]))
  AND (@status::text = '' OR d.status = @status)
  AND (sqlc.narg(org_unit_id)::bigint IS NULL OR t.org_unit_id = sqlc.narg(org_unit_id))
  AND (sqlc.narg(period_start)::date IS NULL OR t.period_start = sqlc.narg(period_start))
ORDER BY t.period_start DESC, u.name, t.id DESC
LIMIT @lim OFFSET @off;

-- name: EmployeesByCodes :many
SELECT id, code, org_unit_id FROM hrm.employees WHERE lower(code) = ANY(@codes::text[]);

-- name: LeaveBalanceDays :one
SELECT coalesce((SELECT days::text FROM hrm.leave_balances WHERE employee_id = $1 AND year = $2), '0')::text;

-- name: WorkWeeks :many
SELECT effective_from, off_days FROM hrm.work_weeks WHERE legal_entity_id = $1 ORDER BY effective_from;

-- name: SaveWorkWeek :exec
INSERT INTO hrm.work_weeks (legal_entity_id, effective_from, off_days) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, effective_from) DO UPDATE SET off_days = excluded.off_days;

-- name: DeleteWorkWeek :execrows
DELETE FROM hrm.work_weeks WHERE legal_entity_id = $1 AND effective_from = $2;

-- name: Holidays :many
SELECT date, name FROM hrm.holidays
WHERE legal_entity_id = @legal_entity_id AND date BETWEEN @from_date::date AND @to_date::date ORDER BY date;

-- name: SaveHoliday :exec
INSERT INTO hrm.holidays (legal_entity_id, date, name) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, date) DO UPDATE SET name = excluded.name;

-- name: DeleteHoliday :execrows
DELETE FROM hrm.holidays WHERE legal_entity_id = $1 AND date = $2;

-- name: ListLegalParams :many
SELECT key, effective_from, value FROM hrm.legal_params ORDER BY key, effective_from;

-- name: LegalParamsAt :many
-- Each key's version in force at @at.
SELECT DISTINCT ON (key) key, effective_from, value FROM hrm.legal_params
WHERE effective_from <= @at::date ORDER BY key, effective_from DESC;

-- name: GetLegalParam :one
SELECT value FROM hrm.legal_params WHERE key = $1 AND effective_from = $2;

-- name: SaveLegalParam :exec
INSERT INTO hrm.legal_params (key, effective_from, value) VALUES ($1, $2, $3)
ON CONFLICT (key, effective_from) DO UPDATE SET value = excluded.value;

-- name: LegalEntities :many
SELECT id FROM iam.org_units WHERE kind = 'company' ORDER BY id;

-- name: CreatePayroll :exec
INSERT INTO hrm.payrolls (id, period_start, period_end) VALUES ($1, $2, $3);

-- name: GetPayroll :one
SELECT p.*, d.legal_entity_id, u.name AS legal_entity_name
FROM hrm.payrolls p
JOIN record.documents d ON d.id = p.id
JOIN iam.org_units u ON u.id = d.legal_entity_id
WHERE p.id = $1;

-- name: DeletePayroll :exec
DELETE FROM hrm.payrolls WHERE id = $1;

-- name: ListPayrolls :many
SELECT p.id, d.number, d.status, d.legal_entity_id, u.name AS legal_entity_name, p.period_start, p.period_end,
       p.computed_at, count(*) OVER () AS total
FROM hrm.payrolls p
JOIN record.documents d ON d.id = p.id
JOIN iam.org_units u ON u.id = d.legal_entity_id
WHERE (@all_units::bool OR d.legal_entity_id = ANY(@units::bigint[]))
  AND (@status::text = '' OR d.status = @status)
  AND (sqlc.narg(legal_entity_id)::bigint IS NULL OR d.legal_entity_id = sqlc.narg(legal_entity_id))
  AND (sqlc.narg(period_start)::date IS NULL OR p.period_start = sqlc.narg(period_start))
ORDER BY p.period_start DESC, u.name, p.id DESC
LIMIT @lim OFFSET @off;

-- name: SavePayrollResult :exec
UPDATE hrm.payrolls SET computed_at = now(), inputs_hash = @inputs_hash WHERE id = @id;

-- name: ClearPayrollComputed :exec
UPDATE hrm.payrolls SET computed_at = NULL WHERE id = $1;

-- name: DeletePayrollLines :exec
DELETE FROM hrm.payroll_lines WHERE payroll_id = $1;

-- name: InsertPayrollLine :exec
INSERT INTO hrm.payroll_lines (payroll_id, employee_id, org_unit_id, data) VALUES ($1, $2, $3, $4);

-- name: PayrollLines :many
SELECT l.employee_id, l.org_unit_id, l.data, e.code, e.full_name
FROM hrm.payroll_lines l JOIN hrm.employees e ON e.id = l.employee_id
WHERE l.payroll_id = $1 ORDER BY lower(e.code);

-- name: DeletePayrollSources :exec
DELETE FROM hrm.payroll_sources WHERE payroll_id = $1;

-- name: InsertPayrollSources :exec
INSERT INTO hrm.payroll_sources (payroll_id, doc_type, doc_id, version)
SELECT @payroll_id::bigint, unnest(@doc_types::text[]), unnest(@doc_ids::bigint[]), unnest(@versions::int[]);

-- name: PayrollAdjustments :many
SELECT a.id, a.employee_id, a.source_period, a.data, e.code, e.full_name
FROM hrm.payroll_adjustments a JOIN hrm.employees e ON e.id = a.employee_id
WHERE a.payroll_id = $1 ORDER BY a.id;

-- name: DeletePayrollAdjustments :exec
DELETE FROM hrm.payroll_adjustments WHERE payroll_id = $1;

-- name: InsertPayrollAdjustment :exec
INSERT INTO hrm.payroll_adjustments (payroll_id, employee_id, source_period, data) VALUES ($1, $2, $3, $4);

-- name: LockPayrollLock :exec
SELECT legal_entity_id FROM hrm.payroll_locks WHERE legal_entity_id = $1 FOR UPDATE;

-- name: EnsurePayrollPeriod :exec
INSERT INTO hrm.payroll_periods (legal_entity_id, period_start, period_end) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, period_start) DO NOTHING;

-- name: PostedPayroll :one
SELECT posted_payroll_id FROM hrm.payroll_periods WHERE legal_entity_id = $1 AND period_start = $2;

-- name: SetPostedPayroll :exec
UPDATE hrm.payroll_periods SET posted_payroll_id = $3 WHERE legal_entity_id = $1 AND period_start = $2;

-- name: PayrollEmployees :many
-- Employees of the period: a posted original contract with the legal entity overlapping it,
-- employed during it; and anyone with an adjustment on the payroll.
SELECT e.id, e.code, e.full_name, e.org_unit_id, e.hire_date, e.termination_date
FROM hrm.employees e
WHERE (e.hire_date <= @period_end::date AND (e.termination_date IS NULL OR e.termination_date >= @period_start::date)
       AND EXISTS (SELECT 1 FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
                   WHERE c.employee_id = e.id AND c.parent_id IS NULL AND d.status = 'posted' AND d.legal_entity_id = @legal_entity_id
                     AND c.start_date <= @period_end AND (c.end_date IS NULL OR c.end_date >= @period_start)))
   OR e.id IN (SELECT a.employee_id FROM hrm.payroll_adjustments a WHERE a.payroll_id = @payroll_id)
ORDER BY lower(e.code);

-- name: PayrollSourceDocs :many
-- Posted documents of the legal entity a payroll of [@period_start, @period_end] reads: contracts
-- and appendices in force over it, its timesheets and leave, and overtime since @year_start.
SELECT d.doc_type, d.id, d.version FROM record.documents d
WHERE d.legal_entity_id = @legal_entity_id AND d.status = 'posted' AND (
       (d.doc_type = 'hrm.contract' AND EXISTS (
            SELECT 1 FROM hrm.contracts c LEFT JOIN hrm.contracts p ON p.id = c.parent_id
            WHERE c.id = d.id AND c.start_date <= @period_end::date
              AND (coalesce(p.end_date, c.end_date) IS NULL OR coalesce(p.end_date, c.end_date) >= @prev_start::date)))
    OR (d.doc_type = 'hrm.timesheet' AND EXISTS (SELECT 1 FROM hrm.timesheets t WHERE t.id = d.id AND t.period_start = @period_start::date))
    OR (d.doc_type = 'hrm.leave_request' AND EXISTS (
            SELECT 1 FROM hrm.leave_requests r WHERE r.id = d.id AND r.start_date <= @period_end AND r.end_date >= @period_start))
    OR (d.doc_type = 'hrm.overtime_request' AND EXISTS (
            SELECT 1 FROM hrm.overtime_requests o WHERE o.id = d.id AND o.date BETWEEN @year_start::date AND @period_end)))
ORDER BY d.doc_type, d.id;

-- name: PostedTimesheetUnits :many
SELECT t.org_unit_id FROM hrm.timesheets t JOIN record.documents d ON d.id = t.id
WHERE d.status = 'posted' AND t.period_start = $1;

-- name: OrgUnitNames :many
SELECT id, name FROM iam.org_units WHERE id = ANY(@ids::bigint[]) ORDER BY name;

-- name: PayrollWorkedDays :many
SELECT l.employee_id, l.date, l.days::text AS days
FROM hrm.timesheet_lines l JOIN hrm.timesheets t ON t.id = l.timesheet_id JOIN record.documents d ON d.id = t.id
WHERE d.status = 'posted' AND t.period_start = @period_start AND l.employee_id = ANY(@ids::bigint[])
ORDER BY l.employee_id, l.date;

-- name: PayrollLeaves :many
SELECT r.employee_id, r.start_date, r.end_date, r.days::text AS days, t.paid
FROM hrm.leave_requests r JOIN hrm.leave_types t ON t.id = r.leave_type_id JOIN record.documents d ON d.id = r.id
WHERE d.status = 'posted' AND r.employee_id = ANY(@ids::bigint[]) AND r.start_date <= @period_end::date AND r.end_date >= @period_start::date
ORDER BY r.employee_id, r.start_date, r.id;

-- name: PayrollOvertime :many
SELECT o.employee_id, o.date, o.day_kind, o.day_hours::text AS day_hours, o.night_hours::text AS night_hours
FROM hrm.overtime_requests o JOIN record.documents d ON d.id = o.id
WHERE d.status = 'posted' AND o.employee_id = ANY(@ids::bigint[]) AND o.date BETWEEN @year_start::date AND @period_end::date
ORDER BY o.employee_id, o.date, o.id;

-- name: PayrollContracts :many
-- Posted contracts and appendices with the legal entity of the employees that may be in force
-- from @prev_start to @period_end; an appendix runs to its original's end.
SELECT c.id, c.employee_id, c.parent_id, c.start_date, coalesce(p.end_date, c.end_date) AS end_date, c.terms
FROM hrm.contracts c JOIN record.documents d ON d.id = c.id LEFT JOIN hrm.contracts p ON p.id = c.parent_id
WHERE d.status = 'posted' AND d.legal_entity_id = @legal_entity_id AND c.employee_id = ANY(@ids::bigint[]) AND c.start_date <= @period_end::date
  AND (coalesce(p.end_date, c.end_date) IS NULL OR coalesce(p.end_date, c.end_date) >= @prev_start::date)
ORDER BY c.employee_id, c.start_date, c.id;

-- name: PayrollDependents :many
SELECT employee_id, data FROM hrm.dependents WHERE employee_id = ANY(@ids::bigint[]) ORDER BY employee_id, id;

-- name: PayrollLeaveBalances :many
SELECT employee_id, days::text AS days FROM hrm.leave_balances WHERE employee_id = ANY(@ids::bigint[]) AND year = @year;

-- name: PrintLegalEntity :one
SELECT coalesce(legal_name, name)::text AS name, tax_code, address FROM iam.org_units WHERE id = $1;

-- name: AnyEmployee :one
SELECT EXISTS (SELECT 1 FROM hrm.employees);
