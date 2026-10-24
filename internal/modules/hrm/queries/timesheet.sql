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

-- name: TimesheetListBase :many
-- data:base
SELECT t.id, d.number, d.status, t.org_unit_id, u.name AS org_unit_name, t.period_start, t.period_end
FROM hrm.timesheets t
JOIN record.documents d ON d.id = t.id
JOIN iam.org_units u ON u.id = t.org_unit_id;
