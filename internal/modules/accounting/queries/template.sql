-- name: AnyAccount :one
SELECT EXISTS (SELECT 1 FROM accounting.accounts);

-- name: InsertChartTemplate :exec
INSERT INTO accounting.chart_template (code) VALUES ($1);

-- name: GetChartTemplate :one
SELECT code FROM accounting.chart_template;

-- name: AccountIDsByCode :many
SELECT id, code, by_partner FROM accounting.accounts WHERE code = ANY(@codes::text[]);
