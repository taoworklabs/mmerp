-- name: Next :one
-- Locks the counter row until the transaction ends, so a rollback gives the number back.
INSERT INTO numbering.counters (doc_type, legal_entity_id, year, last) VALUES ($1, $2, $3, 1)
ON CONFLICT (doc_type, legal_entity_id, year) DO UPDATE SET last = numbering.counters.last + 1
RETURNING last;
