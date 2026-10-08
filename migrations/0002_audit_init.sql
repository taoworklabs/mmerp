CREATE SCHEMA audit;

CREATE TABLE audit.log (
    id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at       timestamptz NOT NULL DEFAULT now(),
    actor_id bigint,
    action   text NOT NULL,
    data     jsonb
);

CREATE INDEX log_at_idx ON audit.log (at);
