-- name: ListDependents :many
SELECT id, data FROM hrm.dependents WHERE employee_id = $1 ORDER BY id;

-- name: GetDependent :one
SELECT data FROM hrm.dependents WHERE id = $1 AND employee_id = $2 FOR UPDATE;

-- name: CreateDependent :one
INSERT INTO hrm.dependents (employee_id, data) VALUES ($1, $2) RETURNING id;

-- name: UpdateDependent :exec
UPDATE hrm.dependents SET data = $3 WHERE id = $1 AND employee_id = $2;

-- name: DeleteDependent :exec
DELETE FROM hrm.dependents WHERE id = $1 AND employee_id = $2;
