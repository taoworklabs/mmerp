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
