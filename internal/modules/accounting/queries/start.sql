-- name: ListStarts :many
SELECT o.id AS legal_entity_id, s.start_date, EXISTS (SELECT 1 FROM accounting.entries e WHERE e.legal_entity_id = o.id) AS fixed
FROM iam.org_units o LEFT JOIN accounting.starts s ON s.legal_entity_id = o.id
WHERE o.kind = 'company' ORDER BY o.id;

-- name: IsLegalEntity :one
SELECT EXISTS (SELECT 1 FROM iam.org_units WHERE id = $1 AND kind = 'company');

-- name: StartOf :one
SELECT start_date FROM accounting.starts WHERE legal_entity_id = $1;

-- name: SetStart :exec
INSERT INTO accounting.starts (legal_entity_id, start_date) VALUES (@legal_entity_id, @start_date)
ON CONFLICT (legal_entity_id) DO UPDATE SET start_date = EXCLUDED.start_date;

-- name: ClearStart :exec
DELETE FROM accounting.starts WHERE legal_entity_id = $1;

-- name: RecordedBetween :one
-- Whether a document of the legal entity dated in [from, until] would be pending from that start:
-- it has a live recording, or one voided while the ledger still holds it.
SELECT EXISTS (SELECT 1 FROM record.documents d
WHERE d.legal_entity_id = @legal_entity_id AND d.date BETWEEN @from_date::date AND @until::date
  AND (EXISTS (SELECT 1 FROM posting.recordings r WHERE r.doc_type = d.doc_type AND r.doc_id = d.id AND r.voided_at IS NULL)
    OR EXISTS (SELECT 1 FROM accounting.sources s WHERE s.doc_type = d.doc_type AND s.doc_id = d.id AND s.posted_recording_id IS NOT NULL)));
