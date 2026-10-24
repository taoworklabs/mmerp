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

-- name: OvertimeListBase :many
-- data:base
SELECT r.id, d.number, d.status, d.org_unit_id, r.employee_id, e.code AS employee_code, e.full_name AS employee_name,
       r.date, r.day_kind, r.day_hours::text AS day_hours, r.night_hours::text AS night_hours
FROM hrm.overtime_requests r
JOIN record.documents d ON d.id = r.id
JOIN hrm.employees e ON e.id = r.employee_id;
