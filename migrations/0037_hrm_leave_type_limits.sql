-- A kind may cap the days of one request and require a file before a request is sent.
ALTER TABLE hrm.leave_types
    ADD COLUMN max_days numeric(4, 1) CHECK (max_days > 0 AND max_days * 2 = trunc(max_days * 2)),
    ADD COLUMN requires_attachment boolean NOT NULL DEFAULT false;
