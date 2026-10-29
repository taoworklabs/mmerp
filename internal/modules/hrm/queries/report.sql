-- name: ReportEmployees :many
SELECT id, code, full_name, date_of_birth, org_unit_id, hire_date, termination_date, national_id, social_insurance_no, tax_code
FROM hrm.employees WHERE id = ANY(@ids::bigint[]);

-- name: ReportContracts :many
-- Posted contracts and addenda of the employees with the legal entity; an addendum runs to its
-- original's end.
SELECT c.employee_id, c.parent_id, c.start_date, coalesce(p.end_date, c.end_date) AS end_date, t.fixed_term
FROM hrm.contracts c JOIN record.documents d ON d.id = c.id LEFT JOIN hrm.contracts p ON p.id = c.parent_id
  JOIN hrm.contract_types t ON t.id = c.contract_type_id
WHERE d.status = 'posted' AND d.legal_entity_id = @legal_entity_id AND c.employee_id = ANY(@ids::bigint[])
ORDER BY c.employee_id, c.start_date, c.id;

-- name: PITAuthorisationCandidates :many
-- Everyone on a posted payroll of the legal entity in the year, whether on December's, and
-- which legal entity, if any, they authorised for the year.
SELECT e.id, e.code, e.full_name, bool_or(pp.period_start = make_date(@year::int, 12, 1))::bool AS december,
  a.legal_entity_id AS authorised_to
FROM hrm.payroll_periods pp JOIN hrm.payroll_lines l ON l.payroll_id = pp.posted_payroll_id
  JOIN hrm.employees e ON e.id = l.employee_id
  LEFT JOIN hrm.pit_authorisations a ON a.employee_id = e.id AND a.year = @year::int
WHERE pp.legal_entity_id = @legal_entity_id AND pp.period_start >= make_date(@year::int, 1, 1)
  AND pp.period_start <= make_date(@year::int, 12, 1)
GROUP BY e.id, a.legal_entity_id ORDER BY lower(e.code);

-- name: PITAuthorised :many
SELECT employee_id FROM hrm.pit_authorisations WHERE legal_entity_id = $1 AND year = $2;

-- name: DeletePITAuthorisations :exec
DELETE FROM hrm.pit_authorisations WHERE legal_entity_id = $1 AND year = $2;

-- name: InsertPITAuthorisations :exec
INSERT INTO hrm.pit_authorisations (employee_id, year, legal_entity_id)
SELECT unnest(@employee_ids::bigint[]), @year::int, @legal_entity_id;
