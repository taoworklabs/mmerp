-- name: InsertJournalEntry :exec
INSERT INTO accounting.journal_entries (id, description) VALUES ($1, $2);

-- name: UpdateJournalEntry :exec
UPDATE accounting.journal_entries SET description = $2 WHERE id = $1;

-- name: DeleteJournalEntry :exec
DELETE FROM accounting.journal_entries WHERE id = $1;

-- name: DeleteJournalEntryLines :exec
DELETE FROM accounting.journal_entry_lines WHERE journal_entry_id = $1;

-- name: InsertJournalEntryLines :exec
-- No org unit, partner or description travels as 0 and ''.
INSERT INTO accounting.journal_entry_lines (journal_entry_id, line_no, account_id, org_unit_id, partner_id, description, debit, credit)
SELECT @journal_entry_id::bigint, generate_series(1, cardinality(@account_ids::bigint[])), unnest(@account_ids::bigint[]),
       NULLIF(unnest(@org_unit_ids::bigint[]), 0), NULLIF(unnest(@partner_ids::bigint[]), 0), NULLIF(unnest(@descriptions::text[]), ''),
       unnest(@debits::bigint[]), unnest(@credits::bigint[]);

-- name: GetJournalEntry :one
SELECT j.description, d.number, d.status, d.version, d.date, d.legal_entity_id
FROM accounting.journal_entries j JOIN record.documents d ON d.id = j.id WHERE j.id = $1;

-- name: GetJournalEntryLines :many
SELECT l.line_no, l.account_id, a.code AS account_code, a.name AS account_name, l.org_unit_id, l.description, l.debit, l.credit,
  l.partner_id, p.code AS partner_code, p.name AS partner_name
FROM accounting.journal_entry_lines l JOIN accounting.accounts a ON a.id = l.account_id
LEFT JOIN partner.partners p ON p.id = l.partner_id
WHERE l.journal_entry_id = $1 ORDER BY l.line_no;

-- name: JournalEntryListBase :many
-- data:base
SELECT j.id, d.number, d.status, d.date, d.org_unit_id, d.legal_entity_id, j.description,
       (SELECT COALESCE(sum(l.debit), 0)::bigint FROM accounting.journal_entry_lines l WHERE l.journal_entry_id = j.id) AS total
FROM accounting.journal_entries j JOIN record.documents d ON d.id = j.id;

-- name: CountAccounts :one
SELECT count(*) FROM accounting.accounts WHERE id = ANY(@ids::bigint[]);

-- name: CountOrgUnits :one
SELECT count(*) FROM iam.org_units WHERE id = ANY(@ids::bigint[]);

-- name: LiveEntryOfDoc :one
-- The entry of a document that no reversal undoes.
SELECT e.id FROM accounting.entries e
WHERE e.doc_type = $1 AND e.doc_id = $2 AND e.reverses IS NULL
  AND NOT EXISTS (SELECT 1 FROM accounting.entries r WHERE r.reverses = e.id);

-- name: CountPartners :one
SELECT count(*) FROM partner.partners WHERE id = ANY(@ids::bigint[]);
