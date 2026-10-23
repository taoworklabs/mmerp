-- name: ListContractTypes :many
SELECT * FROM hrm.contract_types ORDER BY lower(name);

-- name: GetContractType :one
SELECT * FROM hrm.contract_types WHERE id = $1;

-- name: CreateContractType :one
INSERT INTO hrm.contract_types (name, fixed_term, active) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateContractType :execrows
UPDATE hrm.contract_types SET name = $2, fixed_term = $3, active = $4 WHERE id = $1;

-- name: GetContract :one
SELECT c.id, c.employee_id, c.contract_type_id, c.parent_id, c.start_date, c.end_date, c.terms,
       e.code AS employee_code, e.full_name AS employee_name, e.user_id AS employee_user_id,
       t.name AS contract_type_name, d.org_unit_id,
       p.parent_id AS parent_parent_id, p.start_date AS parent_start, p.end_date AS parent_end,
       pd.number AS parent_number, pd.status AS parent_status
FROM hrm.contracts c
JOIN hrm.employees e ON e.id = c.employee_id
JOIN hrm.contract_types t ON t.id = c.contract_type_id
JOIN record.documents d ON d.id = c.id
LEFT JOIN hrm.contracts p ON p.id = c.parent_id
LEFT JOIN record.documents pd ON pd.id = c.parent_id
WHERE c.id = $1;

-- name: CreateContract :exec
INSERT INTO hrm.contracts (id, employee_id, contract_type_id, parent_id, start_date, end_date, terms)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateContract :exec
UPDATE hrm.contracts SET contract_type_id = $2, start_date = $3, end_date = $4, terms = $5 WHERE id = $1;

-- name: DeleteContract :exec
DELETE FROM hrm.contracts WHERE id = $1;

-- name: ListEmployeeContracts :many
-- Originals each followed by their appendices; scope filter on the document's org unit.
SELECT c.id, d.number, d.status, c.parent_id, t.name AS contract_type_name, c.start_date, c.end_date
FROM hrm.contracts c
JOIN record.documents d ON d.id = c.id
JOIN hrm.contract_types t ON t.id = c.contract_type_id
WHERE c.employee_id = @employee_id AND (@all_units::bool OR d.org_unit_id = ANY(@units::bigint[]))
ORDER BY (SELECT o.start_date FROM hrm.contracts o WHERE o.id = coalesce(c.parent_id, c.id)) DESC,
         coalesce(c.parent_id, c.id) DESC, c.parent_id NULLS FIRST, c.start_date, c.id;

-- name: ContractListBase :many
-- data:base
SELECT c.id, d.number, d.status, d.org_unit_id, c.parent_id, p.number AS parent_number, c.contract_type_id,
       t.name AS contract_type_name, c.start_date, c.end_date, c.employee_id, e.code AS employee_code,
       e.full_name AS employee_name
FROM hrm.contracts c
JOIN record.documents d ON d.id = c.id
JOIN hrm.contract_types t ON t.id = c.contract_type_id
JOIN hrm.employees e ON e.id = c.employee_id
LEFT JOIN record.documents p ON p.id = c.parent_id;

-- name: ContractOverlaps :one
-- Whether another posted original of the employee overlaps [@start, @end] (null end: open-ended).
SELECT EXISTS (
    SELECT 1 FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
    WHERE c.employee_id = @employee_id AND c.id <> @id AND c.parent_id IS NULL AND d.status = 'posted'
      AND daterange(c.start_date, c.end_date, '[]') && daterange(@start_date::date, sqlc.narg(end_date)::date, '[]')
);

-- name: HasPostedAppendix :one
SELECT EXISTS (
    SELECT 1 FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
    WHERE c.parent_id = $1 AND d.status = 'posted'
);

-- name: ContractTermsAt :one
-- The terms in force at @at, and the contract holding them: the posted original covering
-- it, or its latest posted appendix effective by then.
SELECT coalesce(a.id, c.id)::bigint AS id, coalesce(a.terms, c.terms)::bytea AS terms
FROM hrm.contracts c JOIN record.documents d ON d.id = c.id
LEFT JOIN LATERAL (
    SELECT x.id, x.terms FROM hrm.contracts x JOIN record.documents xd ON xd.id = x.id
    WHERE x.parent_id = c.id AND xd.status = 'posted' AND x.start_date <= @at::date
    ORDER BY x.start_date DESC, x.id DESC LIMIT 1
) a ON true
WHERE c.employee_id = @employee_id AND c.parent_id IS NULL AND d.status = 'posted'
  AND c.start_date <= @at AND (c.end_date IS NULL OR c.end_date >= @at);
