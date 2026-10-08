-- name: InsertLines :exec
INSERT INTO posting.lines (doc_type, doc_id, legal_entity_id, date, kind, org_unit_id, amount)
SELECT @doc_type::text, @doc_id::bigint, @legal_entity_id::bigint, @date::date,
       unnest(@kinds::text[]), unnest(@org_unit_ids::bigint[]), unnest(@amounts::bigint[]);

-- name: VoidLines :exec
UPDATE posting.lines SET voided_at = now() WHERE doc_type = $1 AND doc_id = $2 AND voided_at IS NULL;
