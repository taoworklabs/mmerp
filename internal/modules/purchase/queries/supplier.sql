-- name: AnyCatalog :one
SELECT EXISTS (SELECT 1 FROM purchase.suppliers)::bool AS any;
