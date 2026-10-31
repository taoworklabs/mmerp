-- name: CreateWarehouse :one
-- The first warehouse of a legal entity without a default becomes its default.
INSERT INTO stock.warehouses (code, name, legal_entity_id, active, is_default)
SELECT sqlc.arg(code), sqlc.arg(name), o.id, sqlc.arg(active),
  NOT EXISTS (SELECT 1 FROM stock.warehouses w WHERE w.legal_entity_id = o.id AND w.is_default)
FROM iam.org_units o WHERE o.id = sqlc.arg(legal_entity_id) AND o.kind = 'company'
RETURNING id;

-- name: UpdateWarehouse :one
UPDATE stock.warehouses SET name = $2, active = $3 WHERE id = $1 RETURNING code;

-- name: WarehouseCodeTaken :one
SELECT EXISTS (SELECT 1 FROM stock.warehouses WHERE lower(code) = lower($1));

-- name: InsertDefaultWarehouse :exec
INSERT INTO stock.warehouses (code, name, legal_entity_id, is_default) VALUES ($1, $2, $3, true)
ON CONFLICT (legal_entity_id) WHERE is_default DO NOTHING;

-- name: DefaultWarehouse :one
SELECT id FROM stock.warehouses WHERE legal_entity_id = $1 AND is_default;

-- name: WarehouseListBase :many
-- data:base
SELECT id, code, name, legal_entity_id, is_default, active FROM stock.warehouses;
