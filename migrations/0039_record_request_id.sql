-- Create-once for every document type: a retry by the same user with the same request id
-- finds the first document. Sales kept its own copy; its keys only matter for a retry in flight.
ALTER TABLE record.documents ADD COLUMN request_id uuid, ADD COLUMN requested_by bigint REFERENCES iam.users;
CREATE UNIQUE INDEX documents_request_id_key ON record.documents (requested_by, request_id) NULLS NOT DISTINCT
    WHERE request_id IS NOT NULL;

ALTER TABLE sales.headers DROP COLUMN request_id;
