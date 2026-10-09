-- Allowed: a core module reads its own schema and another core module's.
-- name: DocumentOwner :one
SELECT d.id, u.login FROM record.documents d JOIN iam.users u ON u.id = d.created_by WHERE d.id = $1;

-- Planted violation: core knows no business, so it may not read a product's schema.
-- name: DocumentOrder :one
SELECT o.total FROM sales.orders o WHERE o.document_id = $1;
