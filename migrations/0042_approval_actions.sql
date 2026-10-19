ALTER TABLE approval.rules ADD COLUMN action text NOT NULL DEFAULT 'post';
ALTER TABLE approval.rules DROP CONSTRAINT rules_pkey, ADD PRIMARY KEY (doc_type, action);
ALTER TABLE approval.instances ADD COLUMN action text NOT NULL DEFAULT 'post', ADD COLUMN params jsonb;
-- One open request per document holds it; a second one waits on this index and fails.
CREATE UNIQUE INDEX instances_one_open_idx ON approval.instances (doc_type, doc_id) WHERE status = 'open';
