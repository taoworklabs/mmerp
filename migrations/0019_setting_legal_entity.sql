-- Settings of one legal entity; a missing key means the default declared in code.
CREATE TABLE setting.legal_entity_values (
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    key             text NOT NULL,
    value           text NOT NULL,
    PRIMARY KEY (legal_entity_id, key)
);
