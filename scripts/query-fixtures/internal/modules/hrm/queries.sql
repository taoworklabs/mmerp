-- Allowed, and the regression guard for the real HRM: a product that declares no
-- dependency still reads its own schema plus the core and shared tiers.
-- name: EmployeeOfUser :one
SELECT e.id, u.login, d.status, l.amount
FROM hrm.employees e
JOIN iam.users u ON u.id = e.user_id
JOIN record.documents d ON d.id = e.document_id
JOIN posting.lines l ON l.document_id = d.id
WHERE e.id = $1;
