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
       r.deducted::text AS deducted,
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
UPDATE hrm.leave_requests SET deducted = CAST(@deducted::text AS numeric) WHERE id = @id;

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
-- Locks the row until commit; a negative result fails leave_balances_days_check.
UPDATE hrm.leave_balances SET days = days + CAST(@delta::text AS numeric)
WHERE employee_id = @employee_id AND year = @year
RETURNING days::text;

-- name: LeaveBalances :many
SELECT year, days::text AS days FROM hrm.leave_balances WHERE employee_id = $1 ORDER BY year DESC;

-- name: LeaveBalanceDays :one
SELECT coalesce((SELECT days::text FROM hrm.leave_balances WHERE employee_id = $1 AND year = $2), '0')::text;
