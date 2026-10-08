-- The record an entry is about, for the change history of a record.
ALTER TABLE audit.log ADD COLUMN doc_type text, ADD COLUMN doc_id bigint;

CREATE INDEX log_doc_idx ON audit.log (doc_type, doc_id) WHERE doc_type IS NOT NULL;
