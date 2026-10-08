-- Contracts of a fixed-term kind need an end date; the others may not have one.
ALTER TABLE hrm.contract_types ADD COLUMN fixed_term boolean NOT NULL DEFAULT false;
