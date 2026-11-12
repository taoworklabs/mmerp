-- Accounts tracked by partner: every journal line on them names a partner, no other line does.
ALTER TABLE accounting.accounts ADD COLUMN by_partner boolean NOT NULL DEFAULT false;
ALTER TABLE accounting.entry_lines ADD COLUMN partner_id bigint NULL REFERENCES partner.partners;
ALTER TABLE accounting.journal_entry_lines ADD COLUMN partner_id bigint NULL REFERENCES partner.partners;

CREATE INDEX entry_lines_partner_idx ON accounting.entry_lines (partner_id) WHERE partner_id IS NOT NULL;
