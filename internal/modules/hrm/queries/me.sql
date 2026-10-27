-- name: MyPostedPayrollLines :many
-- The employee's lines on posted payrolls, newest period first.
SELECT p.id, d.number, p.period_start, l.data
FROM hrm.payroll_lines l JOIN hrm.payrolls p ON p.id = l.payroll_id JOIN record.documents d ON d.id = p.id
WHERE l.employee_id = $1 AND d.status = 'posted'
ORDER BY p.period_start DESC, p.id DESC;

-- name: MyPayrollLine :one
SELECT p.id, d.number, p.period_start, l.data
FROM hrm.payroll_lines l JOIN hrm.payrolls p ON p.id = l.payroll_id JOIN record.documents d ON d.id = p.id
WHERE l.employee_id = @employee_id AND p.id = @payroll_id AND d.status = 'posted';

-- name: MyTimesheetLines :many
-- The employee's worked days on the month's timesheets that are not cancelled.
SELECT l.date, l.days::text AS days, d.status
FROM hrm.timesheet_lines l JOIN hrm.timesheets t ON t.id = l.timesheet_id JOIN record.documents d ON d.id = t.id
WHERE l.employee_id = @employee_id AND t.period_start = @period_start AND d.status <> 'cancelled'
ORDER BY l.date;

-- name: MyLeaves :many
SELECT r.start_date, r.end_date, r.days::text AS days, t.name AS type_name, t.paid
FROM hrm.leave_requests r JOIN hrm.leave_types t ON t.id = r.leave_type_id JOIN record.documents d ON d.id = r.id
WHERE r.employee_id = @employee_id AND d.status = 'posted' AND r.start_date <= @to_date::date AND r.end_date >= @from_date::date
ORDER BY r.start_date, r.id;

-- name: MyOvertime :many
SELECT o.date, (o.day_hours + o.night_hours)::text AS hours
FROM hrm.overtime_requests o JOIN record.documents d ON d.id = o.id
WHERE o.employee_id = @employee_id AND d.status = 'posted' AND o.date BETWEEN @from_date::date AND @to_date::date
ORDER BY o.date, o.id;
