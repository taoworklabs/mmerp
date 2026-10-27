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

-- name: PayrollListBase :many
-- data:base
SELECT p.id, d.number, d.status, d.legal_entity_id, u.name AS legal_entity_name,
       p.period_start, p.period_end, p.computed_at
FROM hrm.payrolls p
JOIN record.documents d ON d.id = p.id
JOIN iam.org_units u ON u.id = d.legal_entity_id;

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

-- name: OrgUnitParents :many
-- The units under a legal entity with their parents, not crossing into another legal entity.
WITH RECURSIVE down AS (
    SELECT o.id, o.parent_id FROM iam.org_units o WHERE o.parent_id = @legal_entity_id AND o.kind <> 'company'
    UNION ALL
    SELECT o.id, o.parent_id FROM iam.org_units o JOIN down ON o.parent_id = down.id WHERE o.kind <> 'company'
)
SELECT down.id, down.parent_id::bigint AS parent_id FROM down;
