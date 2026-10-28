-- name: ListLeaveTypes :many
SELECT * FROM hrm.leave_types ORDER BY lower(name);

-- name: GetLeaveType :one
SELECT * FROM hrm.leave_types WHERE id = $1;

-- name: CreateLeaveType :one
INSERT INTO hrm.leave_types (name, deducts_balance, paid, active, max_days, requires_attachment)
VALUES (@name, @deducts_balance, @paid, @active, CAST(sqlc.narg(max_days)::text AS numeric), @requires_attachment) RETURNING id;

-- name: UpdateLeaveType :execrows
UPDATE hrm.leave_types SET name = @name, deducts_balance = @deducts_balance, paid = @paid, active = @active,
    max_days = CAST(sqlc.narg(max_days)::text AS numeric), requires_attachment = @requires_attachment
WHERE id = @id;

-- name: GetLeaveRequest :one
SELECT r.id, r.employee_id, r.leave_type_id, r.start_date, r.end_date, r.days::text AS days, r.reason,
       r.deducted::text AS deducted, r.deducted_carried::text AS deducted_carried,
       e.code AS employee_code, e.full_name AS employee_name, e.user_id AS employee_user_id,
       t.name AS leave_type_name, t.deducts_balance, t.max_days, t.requires_attachment, d.org_unit_id
FROM hrm.leave_requests r
JOIN hrm.employees e ON e.id = r.employee_id
JOIN hrm.leave_types t ON t.id = r.leave_type_id
JOIN record.documents d ON d.id = r.id
WHERE r.id = $1;

-- name: LeaveHasAttachment :one
SELECT EXISTS (SELECT 1 FROM attachment.files WHERE doc_type = 'hrm.leave_request' AND doc_id = $1);

-- name: CreateLeaveRequest :exec
INSERT INTO hrm.leave_requests (id, employee_id, leave_type_id, start_date, end_date, days, reason)
VALUES (@id, @employee_id, @leave_type_id, @start_date, @end_date, CAST(@days::text AS numeric), @reason);

-- name: UpdateLeaveRequest :exec
UPDATE hrm.leave_requests SET leave_type_id = @leave_type_id, start_date = @start_date, end_date = @end_date,
    days = CAST(@days::text AS numeric), reason = @reason
WHERE id = @id;

-- name: ApprovedLeaveOverlapping :many
-- The employee's other approved leave touching [@start_date, @end_date], whatever its type:
-- overlap is about absence, so a type that neither deducts nor is paid still occupies dates.
SELECT r.id, d.number, r.start_date, r.end_date, r.days::text AS days
FROM hrm.leave_requests r
JOIN record.documents d ON d.id = r.id
WHERE r.employee_id = @employee_id AND r.id <> @id AND d.status = 'posted'
  AND daterange(r.start_date, r.end_date, '[]') && daterange(@start_date::date, @end_date::date, '[]')
ORDER BY r.start_date;

-- name: SetLeaveDeducted :exec
UPDATE hrm.leave_requests SET deducted = CAST(@deducted::text AS numeric), deducted_carried = CAST(@deducted_carried::text AS numeric) WHERE id = @id;

-- name: DeleteLeaveRequest :exec
DELETE FROM hrm.leave_requests WHERE id = $1;

-- name: LeaveListBase :many
-- data:base
SELECT r.id, d.number, d.status, d.org_unit_id, r.employee_id,
       e.code AS employee_code, e.full_name AS employee_name, t.name AS leave_type_name,
       r.start_date, r.end_date, r.days::text AS days
FROM hrm.leave_requests r
JOIN record.documents d ON d.id = r.id
JOIN hrm.employees e ON e.id = r.employee_id
JOIN hrm.leave_types t ON t.id = r.leave_type_id;

-- name: EnsureLeaveBalance :exec
INSERT INTO hrm.leave_balances (employee_id, year, days) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING;

-- name: AddLeaveBalance :one
-- Locks the row until commit; a negative result fails leave_balances_days_check. Carried
-- days are part of days, so they shrink with it.
UPDATE hrm.leave_balances SET days = days + CAST(@delta::text AS numeric),
    carried = LEAST(carried, GREATEST(days + CAST(@delta::text AS numeric), 0))
WHERE employee_id = @employee_id AND year = @year
RETURNING days::text;

-- name: LeaveBalances :many
SELECT year, days::text AS days, carried::text AS carried, carried_until FROM hrm.leave_balances WHERE employee_id = $1 ORDER BY year DESC;

-- name: LockLeaveBalance :one
SELECT days::text AS days, carried::text AS carried, carried_until FROM hrm.leave_balances
WHERE employee_id = @employee_id AND year = @year FOR UPDATE;

-- name: SetLeaveBalance :exec
-- Only on a row locked by LockLeaveBalance in the same transaction.
UPDATE hrm.leave_balances SET days = CAST(@days::text AS numeric), carried = CAST(@carried::text AS numeric), carried_until = @carried_until
WHERE employee_id = @employee_id AND year = @year;

-- name: LeaveBalanceDays :one
SELECT coalesce((SELECT days::text FROM hrm.leave_balances WHERE employee_id = $1 AND year = $2), '0')::text;

-- name: InsertLeaveAccrual :execrows
-- No row when the month was already decided: a credit is applied once.
INSERT INTO hrm.leave_accruals (employee_id, year, month, days)
VALUES (@employee_id, @year, @month, CAST(@days::text AS numeric))
ON CONFLICT DO NOTHING;

-- name: LeaveAccrualMonths :many
SELECT month FROM hrm.leave_accruals WHERE employee_id = $1 AND year = $2 ORDER BY month;

-- name: LeaveAccruedDays :one
SELECT coalesce(sum(days), 0)::text FROM hrm.leave_accruals WHERE employee_id = $1 AND year = $2;

-- name: AccrualEmployees :many
-- Everyone still employed on @since, the first day of last year.
SELECT id, org_unit_id, hire_date, termination_date FROM hrm.employees
WHERE termination_date IS NULL OR termination_date >= @since::date
ORDER BY id;
