-- name: InsertClosing :exec
INSERT INTO accounting.closings (id, description) VALUES ($1, $2);

-- name: UpdateClosing :exec
UPDATE accounting.closings SET description = $2 WHERE id = $1;

-- name: DeleteClosing :exec
DELETE FROM accounting.closings WHERE id = $1;

-- name: GetClosing :one
SELECT c.description, d.number, d.status, d.version, d.date, d.legal_entity_id
FROM accounting.closings c JOIN record.documents d ON d.id = c.id WHERE c.id = $1;

-- name: ClosingListBase :many
-- data:base
SELECT c.id, d.number, d.status, d.date, d.org_unit_id, d.legal_entity_id, c.description
FROM accounting.closings c JOIN record.documents d ON d.id = c.id;

-- name: GetClosingAccounts :one
SELECT r.id AS result_id, r.code AS result_code, r.name AS result_name,
       k.id AS retained_id, k.code AS retained_code, k.name AS retained_name
FROM accounting.closing_accounts c
JOIN accounting.accounts r ON r.id = c.result_account_id
JOIN accounting.accounts k ON k.id = c.retained_account_id;

-- name: SaveClosingAccounts :exec
INSERT INTO accounting.closing_accounts (result_account_id, retained_account_id) VALUES ($1, $2)
ON CONFLICT (singleton) DO UPDATE SET result_account_id = EXCLUDED.result_account_id, retained_account_id = EXCLUDED.retained_account_id;

-- name: ProfitAndLossBalances :many
-- What is left on each revenue and expense account of a legal entity up to a date, by org unit
-- and partner; earlier closings count, so only the remainder shows.
SELECT l.account_id, a.code AS account_code, a.name AS account_name, a.active, l.org_unit_id, l.partner_id,
       sum(l.debit - l.credit)::bigint AS balance
FROM accounting.entries e
JOIN accounting.entry_lines l ON l.entry_id = e.id
JOIN accounting.accounts a ON a.id = l.account_id
WHERE e.legal_entity_id = $1 AND e.date <= $2 AND a.type IN ('revenue', 'expense')
GROUP BY l.account_id, a.code, a.name, a.active, l.org_unit_id, l.partner_id
HAVING sum(l.debit - l.credit) <> 0
ORDER BY a.code, l.org_unit_id NULLS FIRST, l.partner_id NULLS FIRST;

-- name: OriginalEntryLines :many
-- The lines of a document's first entry (not a reversal), with their accounts.
SELECT l.account_id, a.code AS account_code, a.name AS account_name, l.org_unit_id, l.partner_id, l.debit, l.credit
FROM accounting.entries e
JOIN accounting.entry_lines l ON l.entry_id = e.id
JOIN accounting.accounts a ON a.id = l.account_id
WHERE e.doc_type = $1 AND e.doc_id = $2 AND e.reverses IS NULL
  AND e.id = (SELECT min(x.id) FROM accounting.entries x WHERE x.doc_type = $1 AND x.doc_id = $2 AND x.reverses IS NULL)
ORDER BY l.line_no;
