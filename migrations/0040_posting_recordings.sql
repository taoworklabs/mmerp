-- One row per recording of a document's lines: at most one unvoided recording per document,
-- so lines are never doubled.
CREATE TABLE posting.recordings (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type    text NOT NULL,
    doc_id      bigint NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    voided_at   timestamptz
);

CREATE UNIQUE INDEX recordings_doc_once ON posting.recordings (doc_type, doc_id) WHERE voided_at IS NULL;

INSERT INTO posting.recordings (doc_type, doc_id)
SELECT DISTINCT doc_type, doc_id FROM posting.lines WHERE voided_at IS NULL;
