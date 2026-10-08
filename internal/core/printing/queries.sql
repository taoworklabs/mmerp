-- name: CreateSnapshots :exec
INSERT INTO printing.snapshots (doc_type, doc_id, part, position, data)
SELECT @doc_type::text, @doc_id::bigint, unnest(@parts::bigint[]), unnest(@positions::int[]), unnest(@data::bytea[]);

-- name: Snapshots :many
SELECT part, data FROM printing.snapshots WHERE doc_type = $1 AND doc_id = $2 ORDER BY position;

-- name: CreatePin :exec
INSERT INTO printing.pins (doc_type, doc_id, layout, locale, blocks) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO NOTHING;

-- name: GetPin :one
SELECT layout, locale, blocks FROM printing.pins WHERE doc_type = $1 AND doc_id = $2;

-- name: LatestBlocks :many
SELECT DISTINCT ON (b.template) b.template, b.version, b.texts, b.saved_at, u.name AS saved_by_name
FROM printing.blocks b JOIN iam.users u ON u.id = b.saved_by
ORDER BY b.template, b.version DESC;

-- name: SaveBlocks :one
INSERT INTO printing.blocks (template, version, texts, saved_by)
SELECT @template::text, coalesce(max(version), 0) + 1, @texts::jsonb, @saved_by::bigint
FROM printing.blocks WHERE template = @template::text
RETURNING version;
