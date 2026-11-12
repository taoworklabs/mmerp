-- name: ShareStart :one
SELECT start_date FROM accounting.starts WHERE legal_entity_id = $1 FOR SHARE;

-- name: LockStart :exec
-- Serializes a start date change with the first entry written under it.
SELECT 1 FROM accounting.starts WHERE legal_entity_id = $1 FOR UPDATE;

-- name: HasEntries :one
SELECT EXISTS (SELECT 1 FROM accounting.entries WHERE legal_entity_id = $1);

-- name: LiveRecording :one
SELECT id FROM posting.recordings WHERE doc_type = $1 AND doc_id = $2 AND voided_at IS NULL;

-- name: LiveLines :many
SELECT kind, org_unit_id, partner_id, amount FROM posting.lines
WHERE doc_type = $1 AND doc_id = $2 AND voided_at IS NULL ORDER BY id;

-- name: LockSource :one
-- The upsert row-locks the tracking row, as FOR UPDATE would.
INSERT INTO accounting.sources (doc_type, doc_id, legal_entity_id, date) VALUES ($1, $2, $3, $4)
ON CONFLICT (doc_type, doc_id) DO UPDATE SET legal_entity_id = EXCLUDED.legal_entity_id, date = EXCLUDED.date
RETURNING posted_recording_id;

-- name: SaveSource :exec
UPDATE accounting.sources SET posted_recording_id = sqlc.narg(posted_recording_id), problem = sqlc.narg(problem), missing_kinds = @missing_kinds::text[]
WHERE doc_type = @doc_type AND doc_id = @doc_id;

-- name: RuleOf :many
SELECT kind, account_id, side FROM accounting.rules WHERE kind = ANY(@kinds::text[]);

-- name: EntryOfRecording :one
SELECT id FROM accounting.entries WHERE recording_id = $1 AND reverses IS NULL;

-- name: ShareAccounts :exec
SELECT 1 FROM accounting.accounts WHERE id = ANY(@ids::bigint[]) ORDER BY id FOR SHARE;

-- name: InsertEntry :one
INSERT INTO accounting.entries (legal_entity_id, date, doc_type, doc_id, recording_id)
VALUES (@legal_entity_id, @date, @doc_type, @doc_id, sqlc.narg(recording_id)) RETURNING id;

-- name: InsertEntryLines :exec
INSERT INTO accounting.entry_lines (entry_id, line_no, account_id, org_unit_id, partner_id, kind, description, debit, credit)
SELECT @entry_id::bigint, generate_series(1, cardinality(@account_ids::bigint[])), unnest(@account_ids::bigint[]),
       NULLIF(unnest(@org_unit_ids::bigint[]), 0), NULLIF(unnest(@partner_ids::bigint[]), 0), NULLIF(unnest(@kinds::text[]), ''), NULLIF(unnest(@descriptions::text[]), ''),
       unnest(@debits::bigint[]), unnest(@credits::bigint[]);

-- name: ReverseEntry :one
-- Same date and source as the entry it reverses; its lines are copied with sides swapped.
INSERT INTO accounting.entries (legal_entity_id, date, doc_type, doc_id, recording_id, reverses)
SELECT e.legal_entity_id, e.date, e.doc_type, e.doc_id, e.recording_id, e.id FROM accounting.entries e WHERE e.id = $1
RETURNING id;

-- name: ReverseEntryLines :exec
INSERT INTO accounting.entry_lines (entry_id, line_no, account_id, org_unit_id, partner_id, kind, description, debit, credit)
SELECT @reversal::bigint, line_no, account_id, org_unit_id, partner_id, kind, description, credit, debit
FROM accounting.entry_lines WHERE entry_id = @entry_id::bigint;
