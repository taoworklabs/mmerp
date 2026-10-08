-- The kinds the API accepts. posting.lines.kind stays open: every product adds its own kinds.
ALTER TABLE iam.org_units ADD CHECK (kind IN ('group', 'company', 'branch', 'department'));
