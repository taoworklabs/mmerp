-- name: OpenTimesheetUnits :many
-- Org units with an employee employed during the period and no posted timesheet for it.
SELECT DISTINCT e.org_unit_id FROM hrm.employees e
WHERE e.hire_date <= @period_end::date AND (e.termination_date IS NULL OR e.termination_date >= @period_start::date)
  AND NOT EXISTS (SELECT 1 FROM hrm.timesheets t JOIN record.documents d ON d.id = t.id
                  WHERE t.org_unit_id = e.org_unit_id AND t.period_start = @period_start AND d.status = 'posted');

-- name: OpenPayrollEntities :many
-- Legal entities with an employee under a posted original contract over the period (as
-- PayrollEmployees) and no posted payroll for it.
SELECT DISTINCT d.legal_entity_id FROM hrm.employees e
JOIN hrm.contracts c ON c.employee_id = e.id AND c.parent_id IS NULL
JOIN record.documents d ON d.id = c.id AND d.status = 'posted'
WHERE e.hire_date <= @period_end::date AND (e.termination_date IS NULL OR e.termination_date >= @period_start::date)
  AND c.start_date <= @period_end AND (c.end_date IS NULL OR c.end_date >= @period_start)
  AND NOT EXISTS (SELECT 1 FROM hrm.payroll_periods p
                  WHERE p.legal_entity_id = d.legal_entity_id AND p.period_start = @period_start AND p.posted_payroll_id IS NOT NULL);
