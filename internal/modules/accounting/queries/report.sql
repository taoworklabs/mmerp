-- name: LedgerAccountExists :one
SELECT EXISTS (SELECT 1 FROM accounting.accounts WHERE id = $1);

-- name: LedgerOpening :one
-- Signed debit minus credit of the account and every descendant before @from_date.
WITH RECURSIVE sub AS (
  SELECT x.id FROM accounting.accounts x WHERE x.id = @account_id::bigint
  UNION
  SELECT a.id FROM accounting.accounts a JOIN sub ON a.parent_id = sub.id
)
SELECT COALESCE(sum(l.debit - l.credit), 0)::bigint AS opening
FROM accounting.entries e JOIN accounting.entry_lines l ON l.entry_id = e.id
WHERE e.legal_entity_id = @legal_entity_id AND l.account_id IN (SELECT id FROM sub) AND e.date < @from_date::date;

-- name: LedgerLines :many
WITH RECURSIVE sub AS (
  SELECT x.id FROM accounting.accounts x WHERE x.id = @account_id::bigint
  UNION
  SELECT a.id FROM accounting.accounts a JOIN sub ON a.parent_id = sub.id
)
SELECT e.date, e.id AS entry_id, e.doc_type, e.doc_id, COALESCE(d.number, '') AS doc_number, l.kind, l.description, l.org_unit_id,
  a.code AS account_code, l.debit, l.credit, l.partner_id, p.code AS partner_code, p.name AS partner_name
FROM accounting.entries e
JOIN accounting.entry_lines l ON l.entry_id = e.id
JOIN accounting.accounts a ON a.id = l.account_id
LEFT JOIN record.documents d ON d.doc_type = e.doc_type AND d.id = e.doc_id
LEFT JOIN partner.partners p ON p.id = l.partner_id
WHERE e.legal_entity_id = @legal_entity_id AND l.account_id IN (SELECT id FROM sub)
  AND e.date >= @from_date::date AND e.date <= @to_date::date
ORDER BY e.date, e.id, l.line_no;

-- name: TrialBalanceRows :many
-- Each account with the movement of itself and its descendants; rows with nothing to show are left out.
WITH RECURSIVE up(id, anc) AS (
  SELECT x.id, x.id FROM accounting.accounts x
  UNION
  SELECT up.id, a.parent_id FROM up JOIN accounting.accounts a ON a.id = up.anc WHERE a.parent_id IS NOT NULL
), mv AS (
  SELECT l.account_id,
    sum(CASE WHEN e.date < @from_date::date THEN l.debit - l.credit ELSE 0 END)::bigint AS opening,
    sum(CASE WHEN e.date >= @from_date::date THEN l.debit ELSE 0 END)::bigint AS debit,
    sum(CASE WHEN e.date >= @from_date::date THEN l.credit ELSE 0 END)::bigint AS credit
  FROM accounting.entries e JOIN accounting.entry_lines l ON l.entry_id = e.id
  WHERE e.legal_entity_id = @legal_entity_id AND e.date <= @to_date::date
  GROUP BY l.account_id
)
SELECT a.id, a.code, a.name, a.parent_id,
  NOT EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id) AS postable,
  sum(mv.opening)::bigint AS opening, sum(mv.debit)::bigint AS debit, sum(mv.credit)::bigint AS credit
FROM up JOIN mv ON mv.account_id = up.id JOIN accounting.accounts a ON a.id = up.anc
GROUP BY a.id
HAVING sum(mv.opening) <> 0 OR sum(mv.debit) <> 0 OR sum(mv.credit) <> 0
ORDER BY a.code;

-- name: PartnerBalanceRows :many
-- Per tracked account and partner; rows with nothing to show are left out.
WITH mv AS (
  SELECT l.account_id, l.partner_id,
    sum(CASE WHEN e.date < @from_date::date THEN l.debit - l.credit ELSE 0 END)::bigint AS opening,
    sum(CASE WHEN e.date >= @from_date::date THEN l.debit ELSE 0 END)::bigint AS debit,
    sum(CASE WHEN e.date >= @from_date::date THEN l.credit ELSE 0 END)::bigint AS credit
  FROM accounting.entries e JOIN accounting.entry_lines l ON l.entry_id = e.id
  WHERE e.legal_entity_id = @legal_entity_id AND e.date <= @to_date::date AND l.partner_id IS NOT NULL
    AND (sqlc.narg(account_id)::bigint IS NULL OR l.account_id = sqlc.narg(account_id)::bigint)
  GROUP BY l.account_id, l.partner_id
)
SELECT mv.account_id, a.code AS account_code, p.id AS partner_id, p.code AS partner_code, p.name AS partner_name, mv.opening, mv.debit, mv.credit
FROM mv JOIN accounting.accounts a ON a.id = mv.account_id JOIN partner.partners p ON p.id = mv.partner_id
WHERE mv.opening <> 0 OR mv.debit <> 0 OR mv.credit <> 0
ORDER BY a.code, p.code;

-- name: PartnerLedgerOpening :one
SELECT COALESCE(sum(l.debit - l.credit), 0)::bigint AS opening
FROM accounting.entries e JOIN accounting.entry_lines l ON l.entry_id = e.id
WHERE e.legal_entity_id = @legal_entity_id AND l.partner_id = @partner_id AND e.date < @from_date::date
  AND (sqlc.narg(account_id)::bigint IS NULL OR l.account_id = sqlc.narg(account_id)::bigint);

-- name: PartnerLedgerLines :many
SELECT e.date, e.id AS entry_id, e.doc_type, e.doc_id, COALESCE(d.number, '') AS doc_number, l.kind, l.description, l.org_unit_id,
  a.code AS account_code, l.debit, l.credit, p.code AS partner_code, p.name AS partner_name
FROM accounting.entries e
JOIN accounting.entry_lines l ON l.entry_id = e.id
JOIN accounting.accounts a ON a.id = l.account_id
JOIN partner.partners p ON p.id = l.partner_id
LEFT JOIN record.documents d ON d.doc_type = e.doc_type AND d.id = e.doc_id
WHERE e.legal_entity_id = @legal_entity_id AND l.partner_id = @partner_id
  AND (sqlc.narg(account_id)::bigint IS NULL OR l.account_id = sqlc.narg(account_id)::bigint)
  AND e.date >= @from_date::date AND e.date <= @to_date::date
ORDER BY e.date, e.id, l.line_no;

-- name: PartnerExists :one
SELECT EXISTS (SELECT 1 FROM partner.partners WHERE id = $1);
