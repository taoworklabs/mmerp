CREATE SCHEMA partner;

-- One company or person, shared by the whole tenant; products add their roles on top.
CREATE TABLE partner.partners (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code         text NOT NULL,
    name         text NOT NULL,
    tax_code     text,
    address      text,
    phone        text,
    email        text,
    contact_name text
);

CREATE UNIQUE INDEX partners_code_key ON partner.partners (lower(code));
CREATE UNIQUE INDEX partners_tax_code_key ON partner.partners (lower(btrim(tax_code))) WHERE tax_code IS NOT NULL;

-- Customers sharing a tax code become one partner, its identity taken from the newest
-- customer (highest id); customers without one each get their own. A partner holds at most
-- one customer per org unit, so a later customer of the same tax code and org unit gets a
-- partner of its own, without the tax code (data is simulated, nobody merges it).
-- Partner codes follow the first customer of each partner, in id order.
CREATE TEMP TABLE customer_partner ON COMMIT DROP AS
WITH k AS (
    SELECT id, org_unit_id, lower(nullif(btrim(tax_code), '')) AS tax FROM sales.customers
), g AS (
    SELECT id, tax IS NOT NULL AND row_number() OVER (PARTITION BY tax, org_unit_id ORDER BY id) = 1 AS keeps_tax,
           tax
    FROM k
), grp AS (
    SELECT id, keeps_tax, CASE WHEN keeps_tax THEN 't' || tax ELSE 'c' || id END AS key FROM g
), first AS (
    SELECT id, keeps_tax, key, min(id) OVER (PARTITION BY key) AS first_id FROM grp
)
SELECT id, keeps_tax, 'DT' || lpad(dense_rank() OVER (ORDER BY first_id)::text, 5, '0') AS code FROM first;

INSERT INTO partner.partners (code, name, tax_code, address, phone, email, contact_name)
SELECT DISTINCT ON (m.code) m.code, c.name, CASE WHEN m.keeps_tax THEN btrim(c.tax_code) END,
       c.address, c.phone, c.email, c.contact_name
FROM customer_partner m JOIN sales.customers c ON c.id = m.id
ORDER BY m.code, c.id DESC;

INSERT INTO numbering.series (code, prefix, last)
VALUES ('partner.partner', 'DT', (SELECT count(*) FROM partner.partners))
ON CONFLICT DO NOTHING;

-- The customer stays as Sales' role of a partner: same id and code, identity moved out.
ALTER TABLE sales.customers ADD COLUMN partner_id bigint REFERENCES partner.partners;

UPDATE sales.customers c SET partner_id = p.id
FROM customer_partner m JOIN partner.partners p ON p.code = m.code
WHERE m.id = c.id;

ALTER TABLE sales.customers
    ALTER COLUMN partner_id SET NOT NULL,
    DROP COLUMN name,
    DROP COLUMN tax_code,
    DROP COLUMN address,
    DROP COLUMN phone,
    DROP COLUMN email,
    DROP COLUMN contact_name,
    -- Its index also serves lookups by partner.
    ADD CONSTRAINT customers_partner_org_unit_key UNIQUE (partner_id, org_unit_id);

-- Whoever edits customers edits their partner's identity; whoever views them views it.
INSERT INTO iam.user_roles (user_id, product, role, org_unit_id)
SELECT user_id, 'partner', CASE WHEN role = 'viewer' THEN 'viewer' ELSE 'editor' END, org_unit_id
FROM iam.user_roles
WHERE product = 'sales' AND role IN ('staff', 'manager', 'viewer')
ON CONFLICT DO NOTHING;

-- Signed-in users pick up the new grants; a fresh install has none.
UPDATE iam.authz SET version = version + 1 WHERE EXISTS (SELECT 1 FROM iam.user_roles WHERE product = 'partner');
