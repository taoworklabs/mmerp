-- An exported file names the export and its params, so every download can ask the module
-- again whether the owner may still read it. Uploads have neither.
ALTER TABLE dataio.files
    ADD COLUMN export_kind   text,
    ADD COLUMN export_params jsonb,
    ADD CHECK ((export_kind IS NULL) = (export_params IS NULL));
