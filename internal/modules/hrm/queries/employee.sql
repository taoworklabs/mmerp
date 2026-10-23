-- name: EmployeeListBase :many
-- data:base
SELECT e.id, e.code, e.full_name, e.org_unit_id, u.name AS org_unit_name, e.email, e.phone,
       e.hire_date, e.termination_date
FROM hrm.employees e JOIN iam.org_units u ON u.id = e.org_unit_id;

-- name: GetEmployee :one
SELECT e.*, u.name AS org_unit_name, m.code AS manager_code, m.full_name AS manager_name, a.login AS user_login
FROM hrm.employees e
JOIN iam.org_units u ON u.id = e.org_unit_id
LEFT JOIN hrm.employees m ON m.id = e.manager_id
LEFT JOIN iam.users a ON a.id = e.user_id
WHERE e.id = $1;

-- name: LockEmployee :one
SELECT * FROM hrm.employees WHERE id = $1 FOR UPDATE;

-- name: CreateEmployee :one
INSERT INTO hrm.employees (code, full_name, date_of_birth, gender, phone, email, address, org_unit_id,
                           manager_id, user_id, hire_date, termination_date,
                           national_id, social_insurance_no, tax_code, bank_account)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING id;

-- name: UpdateEmployee :exec
UPDATE hrm.employees SET full_name = $2, date_of_birth = $3, gender = $4, phone = $5, email = $6,
    address = $7, org_unit_id = $8, manager_id = $9, user_id = $10, hire_date = $11, termination_date = $12,
    national_id = $13, social_insurance_no = $14, tax_code = $15, bank_account = $16
WHERE id = $1;

-- name: EmployeeCodeTaken :one
SELECT EXISTS (SELECT 1 FROM hrm.employees WHERE lower(code) = lower($1));

-- name: ManagerChain :many
-- The employee and everyone above it; UNION stops on a cycle.
WITH RECURSIVE chain (employee_id, next_id) AS (
    SELECT s.id, s.manager_id FROM hrm.employees s WHERE s.id = @start::bigint
    UNION
    SELECT e.id, e.manager_id FROM hrm.employees e JOIN chain ON e.id = chain.next_id
)
SELECT employee_id FROM chain;

-- name: LockManagerTree :exec
-- Serialises manager changes so two concurrent edits cannot close a cycle.
SELECT pg_advisory_xact_lock(hashtext('hrm.manager_tree'));

-- name: IsManager :one
-- Whether the user is linked to an employee above @employee on its manager chain.
WITH RECURSIVE up AS (
    SELECT e.manager_id AS id FROM hrm.employees e WHERE e.id = @employee::bigint
    UNION
    SELECT e.manager_id FROM hrm.employees e JOIN up ON e.id = up.id
)
SELECT EXISTS (SELECT 1 FROM up JOIN hrm.employees m ON m.id = up.id WHERE m.user_id = @actor::bigint);

-- name: ManagerUser :one
-- The account of the manager @level steps above @employee.
WITH RECURSIVE chain (id, depth) AS (
    SELECT e.manager_id, 1 FROM hrm.employees e WHERE e.id = @employee::bigint
    UNION ALL
    SELECT e.manager_id, chain.depth + 1 FROM hrm.employees e JOIN chain ON e.id = chain.id WHERE chain.depth < @level::int
)
SELECT m.user_id FROM chain JOIN hrm.employees m ON m.id = chain.id WHERE chain.depth = @level::int;

-- name: EmployeeByUser :one
SELECT id, full_name, org_unit_id FROM hrm.employees WHERE user_id = $1;

-- name: EmployeesByCodes :many
SELECT id, code, org_unit_id FROM hrm.employees WHERE lower(code) = ANY(@codes::text[]);

-- name: AnyEmployee :one
SELECT EXISTS (SELECT 1 FROM hrm.employees);
