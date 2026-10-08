CREATE SCHEMA numbering;

-- One counter per document type, legal entity and year; the row lock serialises numbering.
CREATE TABLE numbering.counters (
    doc_type        text NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    year            int NOT NULL,
    last            bigint NOT NULL,
    PRIMARY KEY (doc_type, legal_entity_id, year)
);
