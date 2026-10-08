-- /me asks on every page load which products have documents.
CREATE INDEX documents_doc_type_idx ON record.documents (doc_type);
