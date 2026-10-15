-- name: Next :one
-- Locks the counter row until the transaction ends, so a rollback gives the number back.
INSERT INTO numbering.counters (doc_type, legal_entity_id, year, last) VALUES ($1, $2, $3, 1)
ON CONFLICT (doc_type, legal_entity_id, year) DO UPDATE SET last = numbering.counters.last + 1
RETURNING last;

-- name: NextInSeries :one
-- Locks the series row until the transaction ends, like Next.
INSERT INTO numbering.series (code, prefix, last) VALUES ($1, $2, 1)
ON CONFLICT (code) DO UPDATE SET last = numbering.series.last + 1
RETURNING prefix, last;

-- name: ListSeries :many
SELECT * FROM numbering.series;

-- name: LockSeries :one
SELECT * FROM numbering.series WHERE code = $1 FOR UPDATE;

-- name: SetPrefix :exec
INSERT INTO numbering.series (code, prefix, last) VALUES ($1, $2, 0)
ON CONFLICT (code) DO UPDATE SET prefix = excluded.prefix;
