-- name: FindPartners :many
-- Partners whose code, name or tax code contains @q (case-insensitive), for a line editor.
SELECT id, code, name FROM partner.partners
WHERE @q::text = '' OR strpos(lower(code), lower(@q)) > 0 OR strpos(lower(name), lower(@q)) > 0
  OR strpos(lower(COALESCE(tax_code, '')), lower(@q)) > 0
ORDER BY code LIMIT 20;
