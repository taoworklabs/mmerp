-- Planted violation: this module's product is missing from the composition root's
-- dependency map, so nobody has declared what it may read. Reads nothing out of bounds
-- on its own, to keep the one message about the map.
-- name: OrphanRow :one
SELECT o.id, d.status FROM orphan.rows o JOIN record.documents d ON d.id = o.document_id WHERE o.id = $1;
