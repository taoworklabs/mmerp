-- Permission-scope tree. Legal-entity columns belong to company nodes only;
-- the service checks the tree shape (no cycle, no company under a company).
CREATE TABLE iam.org_units (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    parent_id  bigint REFERENCES iam.org_units,
    kind       text NOT NULL,
    name       text NOT NULL,
    tax_code   text,
    legal_name text,
    address    text,
    CHECK (kind = 'company' OR (tax_code IS NULL AND legal_name IS NULL AND address IS NULL))
);

CREATE INDEX org_units_parent_id_idx ON iam.org_units (parent_id);

-- org_unit_id NULL means the whole tenant.
CREATE TABLE iam.user_roles (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES iam.users ON DELETE CASCADE,
    product     text NOT NULL,
    role        text NOT NULL,
    org_unit_id bigint REFERENCES iam.org_units,
    UNIQUE NULLS NOT DISTINCT (user_id, product, role, org_unit_id)
);

-- Tenant-wide part of authz_version: bumped by changes that affect many users.
CREATE TABLE iam.authz (
    version bigint NOT NULL
);
INSERT INTO iam.authz VALUES (1);
