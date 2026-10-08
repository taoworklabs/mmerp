CREATE SCHEMA printing;

-- The print data of each part of a posted document, frozen in the transaction that posted it
-- (or at its first print, for one posted before printing existed). Encrypted: a contract or a
-- payslip holds per-person amounts. Kept forever, cancelled documents included.
CREATE TABLE printing.snapshots (
    doc_type   text NOT NULL,
    doc_id     bigint NOT NULL,
    part       bigint NOT NULL,
    position   int NOT NULL,
    data       bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (doc_type, doc_id, part)
);

-- What the first print of a posted document fixed, so that every reprint draws the same.
CREATE TABLE printing.pins (
    doc_type  text NOT NULL,
    doc_id    bigint NOT NULL,
    layout    int NOT NULL CHECK (layout >= 1),
    locale    text NOT NULL CHECK (locale IN ('vi', 'en')),
    pinned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (doc_type, doc_id)
);
