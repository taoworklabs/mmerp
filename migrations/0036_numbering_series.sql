-- One counter per code series for the whole tenant; created on first use with the default prefix.
CREATE TABLE numbering.series (
    code   text PRIMARY KEY,
    prefix text NOT NULL,
    last   bigint NOT NULL
);
