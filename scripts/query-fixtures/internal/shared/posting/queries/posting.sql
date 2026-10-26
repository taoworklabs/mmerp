-- Allowed: a shared module reads its own schema and the core tier below it.
-- name: LinesOfDocument :many
SELECT l.kind, l.amount FROM posting.lines l JOIN record.documents d ON d.id = l.document_id WHERE d.id = $1;

-- Planted violation: shared sits below the products, so it may not read one.
-- name: LineItem :one
SELECT i.name FROM inventory.items i WHERE i.id = $1;
