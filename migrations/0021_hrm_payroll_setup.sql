-- Days of leave of a paid kind count as paid days in payroll; until now only deducting kinds were paid.
ALTER TABLE hrm.leave_types ADD COLUMN paid boolean NOT NULL DEFAULT false;
UPDATE hrm.leave_types SET paid = deducts_balance;

-- The weekly days off of a legal entity, by version; with no row, Saturday and Sunday.
-- Days are 0 (Sunday) to 6 (Saturday).
CREATE TABLE hrm.work_weeks (
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    effective_from  date NOT NULL,
    off_days        smallint[] NOT NULL CHECK (off_days <@ ARRAY[0, 1, 2, 3, 4, 5, 6]::smallint[]),
    PRIMARY KEY (legal_entity_id, effective_from)
);

-- Public holidays and days off in lieu of a legal entity.
CREATE TABLE hrm.holidays (
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    date            date NOT NULL,
    name            text NOT NULL,
    PRIMARY KEY (legal_entity_id, date)
);

-- Legal parameters by version: a payroll takes each key's version in force on its last day.
-- Values are decimals, or JSON for pit_brackets ([[upper bound or null, rate], …]).
CREATE TABLE hrm.legal_params (
    key            text NOT NULL,
    effective_from date NOT NULL,
    value          text NOT NULL,
    PRIMARY KEY (key, effective_from)
);

INSERT INTO hrm.legal_params (key, effective_from, value) VALUES
    ('base_salary', '2026-01-01', '2340000'),
    ('base_salary', '2026-07-01', '2530000'),
    ('min_wage_region_1', '2026-01-01', '5310000'),
    ('min_wage_region_2', '2026-01-01', '4730000'),
    ('min_wage_region_3', '2026-01-01', '4140000'),
    ('min_wage_region_4', '2026-01-01', '3700000'),
    ('si_employee', '2026-01-01', '0.08'),
    ('hi_employee', '2026-01-01', '0.015'),
    ('ui_employee', '2026-01-01', '0.01'),
    ('si_employer', '2026-01-01', '0.175'),
    ('hi_employer', '2026-01-01', '0.03'),
    ('ui_employer', '2026-01-01', '0.01'),
    ('union_employer', '2026-01-01', '0.02'),
    ('insurance_cap_multiplier', '2026-01-01', '20'),
    ('personal_deduction', '2026-01-01', '15500000'),
    ('dependent_deduction', '2026-01-01', '6200000'),
    ('pit_brackets', '2026-01-01', '[[10000000,0.05],[30000000,0.1],[60000000,0.2],[100000000,0.3],[null,0.35]]'),
    ('ot_exempt_hours_month', '2026-01-01', '40'),
    ('ot_exempt_hours_year', '2026-01-01', '200'),
    ('ot_hours_per_day', '2026-01-01', '8'),
    ('insurance_skip_days', '2026-01-01', '14');
