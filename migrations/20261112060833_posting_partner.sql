-- A business line may name the partner it is owed by or to; each kind declares whether it does.
ALTER TABLE posting.lines ADD COLUMN partner_id bigint NULL REFERENCES partner.partners;
