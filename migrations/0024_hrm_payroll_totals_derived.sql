-- Department totals were stored in clear, and a one-person department's totals are that
-- person's pay; they are now summed from the encrypted lines when read.
ALTER TABLE hrm.payrolls DROP COLUMN totals;
