-- The chart template a tenant applied, if any: its rules are offered for kinds left unmapped.
CREATE TABLE accounting.chart_template (
    singleton  boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    code       text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);
