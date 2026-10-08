-- name: CreateSnapshots :exec
INSERT INTO printing.snapshots (doc_type, doc_id, part, position, data)
SELECT @doc_type::text, @doc_id::bigint, unnest(@parts::bigint[]), unnest(@positions::int[]), unnest(@data::bytea[]);

-- name: Snapshots :many
SELECT part, data FROM printing.snapshots WHERE doc_type = $1 AND doc_id = $2 ORDER BY position;

-- name: CreatePin :exec
INSERT INTO printing.pins (doc_type, doc_id, layout, locale) VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: GetPin :one
SELECT layout, locale FROM printing.pins WHERE doc_type = $1 AND doc_id = $2;
