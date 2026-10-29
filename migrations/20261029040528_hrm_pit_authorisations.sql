-- Who authorised which legal entity to finalise their PIT for a year (form 08/UQ-QTT-TNCN,
-- kept as a file on the employee). One organisation per person per year.
CREATE TABLE hrm.pit_authorisations (
    employee_id     bigint NOT NULL REFERENCES hrm.employees,
    year            int NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    PRIMARY KEY (employee_id, year)
);

CREATE INDEX pit_authorisations_legal_entity_year_idx ON hrm.pit_authorisations (legal_entity_id, year);
