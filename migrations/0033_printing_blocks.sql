-- The text blocks of a print template, a row per save by a tenant administrator; texts maps
-- each block to its text in vi and en.
CREATE TABLE printing.blocks (
    template text NOT NULL,
    version  int NOT NULL CHECK (version >= 1),
    texts    jsonb NOT NULL,
    saved_by bigint NOT NULL REFERENCES iam.users,
    saved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (template, version)
);

-- The text of each block as a posted document's first print drew it, in the pinned locale.
ALTER TABLE printing.pins ADD COLUMN blocks jsonb NOT NULL DEFAULT '{}';
