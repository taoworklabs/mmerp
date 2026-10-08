-- Kinds of contract, entered by the customer.
CREATE TABLE hrm.contract_types (
    id     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name   text NOT NULL,
    active boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX contract_types_name_key ON hrm.contract_types (lower(name));

-- A contract is a document (see hrm.leave_requests for the deferred FK). An appendix points to
-- its original, takes the original's kind and has no end date of its own. The money terms are
-- one encrypted JSON (AAD hrm.contracts.terms).
CREATE TABLE hrm.contracts (
    id               bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    employee_id      bigint NOT NULL REFERENCES hrm.employees,
    contract_type_id bigint NOT NULL REFERENCES hrm.contract_types,
    parent_id        bigint REFERENCES hrm.contracts,
    start_date       date NOT NULL,
    end_date         date CHECK (end_date >= start_date),
    terms            bytea NOT NULL,
    CHECK (parent_id IS NULL OR end_date IS NULL)
);

CREATE INDEX contracts_employee_id_idx ON hrm.contracts (employee_id);
CREATE INDEX contracts_parent_id_idx ON hrm.contracts (parent_id);
