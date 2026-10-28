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
