-- name: GetTermination :one
SELECT t.id, t.employee_id, t.reason, t.decision_date, t.pension_eligible, t.insured_months, t.paid_months,
       t.handover, t.note, t.previous_date, d.date AS termination_date, d.org_unit_id,
       e.code AS employee_code, e.full_name AS employee_name, e.user_id AS employee_user_id, e.hire_date
FROM hrm.terminations t
JOIN record.documents d ON d.id = t.id
JOIN hrm.employees e ON e.id = t.employee_id
WHERE t.id = $1;

-- name: CreateTermination :exec
INSERT INTO hrm.terminations (id, employee_id, reason, decision_date, pension_eligible, insured_months, paid_months, handover, note)
VALUES (@id, @employee_id, @reason, @decision_date, @pension_eligible, @insured_months, @paid_months, @handover, @note);

-- name: UpdateTermination :exec
UPDATE hrm.terminations SET reason = @reason, decision_date = @decision_date, pension_eligible = @pension_eligible,
    insured_months = @insured_months, paid_months = @paid_months, handover = @handover, note = @note
WHERE id = @id;

-- name: DeleteTermination :exec
DELETE FROM hrm.terminations WHERE id = $1;

-- name: OpenTerminationOf :one
-- Another termination of the employee in one of @statuses.
SELECT d.id, d.number FROM hrm.terminations t JOIN record.documents d ON d.id = t.id
WHERE t.employee_id = @employee_id AND t.id <> @except_id AND d.status = ANY(@statuses::text[])
ORDER BY d.id LIMIT 1;

-- name: TerminationListBase :many
-- data:base
SELECT t.id, d.number, d.status, d.org_unit_id, d.date, t.employee_id, e.code AS employee_code, e.full_name AS employee_name,
       u.name AS org_unit_name, t.reason
FROM hrm.terminations t
JOIN record.documents d ON d.id = t.id
JOIN hrm.employees e ON e.id = t.employee_id
JOIN iam.org_units u ON u.id = d.org_unit_id;

-- name: SetEmployeeTerminationDate :exec
UPDATE hrm.employees SET termination_date = @termination_date WHERE id = @id;

-- name: SetTerminationPreviousDate :exec
UPDATE hrm.terminations SET previous_date = @previous_date WHERE id = @id;

-- name: TerminationContractsAfter :many
-- Posted originals of the employee still in force after @after.
SELECT d.number, c.end_date FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
WHERE c.employee_id = @employee_id AND c.parent_id IS NULL AND d.status = 'posted' AND (c.end_date IS NULL OR c.end_date > @after::date)
ORDER BY c.start_date, d.id;

-- name: TerminationLeavesAfter :many
-- Leave requests of the employee that still stand and end after @after.
SELECT d.number, d.status, r.start_date, r.end_date FROM hrm.leave_requests r JOIN record.documents d ON d.id = r.id
WHERE r.employee_id = @employee_id AND d.status <> 'cancelled' AND r.end_date > @after::date
ORDER BY r.start_date, d.id;

-- name: TerminationOvertimeAfter :many
SELECT d.number, d.status, o.date FROM hrm.overtime_requests o JOIN record.documents d ON d.id = o.id
WHERE o.employee_id = @employee_id AND d.status <> 'cancelled' AND o.date > @after::date
ORDER BY o.date, d.id;

-- name: TerminationLeaveBalance :one
SELECT coalesce((SELECT days::text FROM hrm.leave_balances WHERE employee_id = @employee_id AND year = @year), '0')::text AS days;

-- name: PayrollTerminations :many
-- Posted terminations of the employees dated in the period, with the allowance inputs.
SELECT t.employee_id, t.reason, t.pension_eligible, t.insured_months, t.paid_months, d.date AS termination_date, e.hire_date
FROM hrm.terminations t
JOIN record.documents d ON d.id = t.id
JOIN hrm.employees e ON e.id = t.employee_id
WHERE d.status = 'posted' AND t.employee_id = ANY(@ids::bigint[]) AND d.date BETWEEN @period_start::date AND @period_end::date;

-- name: AllowanceContracts :many
-- Posted contracts and appendices of the employee with the legal entity in force from @since to
-- @until, with the unit each is filed under; an appendix runs to its original's end.
SELECT c.id, d.org_unit_id, c.parent_id, c.start_date, coalesce(p.end_date, c.end_date) AS end_date, c.terms
FROM hrm.contracts c JOIN record.documents d ON d.id = c.id LEFT JOIN hrm.contracts p ON p.id = c.parent_id
WHERE d.status = 'posted' AND d.legal_entity_id = @legal_entity_id AND c.employee_id = @employee_id AND c.start_date <= @until::date
  AND (coalesce(p.end_date, c.end_date) IS NULL OR coalesce(p.end_date, c.end_date) >= @since::date)
ORDER BY c.start_date, c.id;
