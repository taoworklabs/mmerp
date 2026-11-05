-- name: PendingDocs :many
-- Documents of a legal entity whose live recording differs from the one the ledger holds.
SELECT d.doc_type, d.doc_id, doc.number, doc.date, doc.legal_entity_id, s.problem, COALESCE(s.missing_kinds, '{}')::text[] AS missing_kinds
FROM (SELECT r.doc_type, r.doc_id, r.id AS recording_id FROM posting.recordings r
      WHERE r.voided_at IS NULL
      UNION ALL
      -- Voided with nothing live: pending only if the ledger still holds a recording.
      SELECT s.doc_type, s.doc_id, NULL FROM accounting.sources s
      WHERE s.posted_recording_id IS NOT NULL
        AND NOT EXISTS (SELECT 1 FROM posting.recordings r WHERE r.doc_type = s.doc_type AND r.doc_id = s.doc_id AND r.voided_at IS NULL)) d
JOIN record.documents doc ON doc.doc_type = d.doc_type AND doc.id = d.doc_id
JOIN accounting.starts st ON st.legal_entity_id = doc.legal_entity_id AND doc.date >= st.start_date
LEFT JOIN accounting.sources s ON s.doc_type = d.doc_type AND s.doc_id = d.doc_id
WHERE doc.legal_entity_id = @legal_entity_id
  AND (sqlc.narg(until)::date IS NULL OR doc.date <= sqlc.narg(until)::date)
  AND d.recording_id IS DISTINCT FROM s.posted_recording_id
ORDER BY doc.date, d.doc_type, d.doc_id;
