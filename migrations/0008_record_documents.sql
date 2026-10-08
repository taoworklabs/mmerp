CREATE SCHEMA record;

-- The single source of status, version, date and legal entity of every document.
-- A module's document table uses this id as its primary key.
CREATE TABLE record.documents (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type        text NOT NULL,
    number          text NOT NULL,
    status          text NOT NULL CHECK (status IN ('draft', 'pending_approval', 'posted', 'cancelled')),
    version         int NOT NULL,
    date            date NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    org_unit_id     bigint NOT NULL REFERENCES iam.org_units,
    amount          bigint,
    fields          jsonb NOT NULL DEFAULT '{}',
    approval_ticket bigint,
    CHECK ((status = 'pending_approval') = (approval_ticket IS NOT NULL))
);

CREATE INDEX documents_pending_idx ON record.documents (legal_entity_id, date) WHERE status = 'pending_approval';
CREATE INDEX documents_org_unit_id_idx ON record.documents (org_unit_id);

-- One row per legal entity, created on first use; documents dated on or before locked_until are frozen.
CREATE TABLE record.period_locks (
    legal_entity_id bigint PRIMARY KEY REFERENCES iam.org_units,
    locked_until    date
);
