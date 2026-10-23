CREATE SCHEMA audit;

CREATE TABLE audit.log (
    id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at       timestamptz NOT NULL DEFAULT now(),
    actor_id bigint,
    action   text NOT NULL,
    data     jsonb,
    -- The record an entry is about, for the change history of a record.
    doc_type text,
    doc_id   bigint
);

CREATE INDEX log_at_idx ON audit.log (at);
CREATE INDEX log_doc_idx ON audit.log (doc_type, doc_id) WHERE doc_type IS NOT NULL;

CREATE SCHEMA iam;

CREATE TABLE iam.users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login         text NOT NULL,
    name          text NOT NULL,
    password_hash text NOT NULL,
    locale        text CHECK (locale IN ('vi', 'en')),
    authz_version bigint NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    -- Where the user's email notifications go; none means in-app only.
    email         text CHECK (char_length(email) <= 254)
);

CREATE UNIQUE INDEX users_login_key ON iam.users (lower(login));

CREATE TABLE iam.sessions (
    token_hash bytea PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES iam.users ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON iam.sessions (user_id);

-- Permission-scope tree. Legal-entity columns belong to company nodes only;
-- the service checks the tree shape (no cycle, no company under a company).
CREATE TABLE iam.org_units (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    parent_id  bigint REFERENCES iam.org_units,
    kind       text NOT NULL,
    name       text NOT NULL,
    tax_code   text,
    legal_name text,
    address    text,
    CHECK (kind = 'company' OR (tax_code IS NULL AND legal_name IS NULL AND address IS NULL)),
    -- The kinds the API accepts.
    CHECK (kind IN ('group', 'company', 'branch', 'department'))
);

CREATE INDEX org_units_parent_id_idx ON iam.org_units (parent_id);

-- org_unit_id NULL means the whole tenant.
CREATE TABLE iam.user_roles (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES iam.users ON DELETE CASCADE,
    product     text NOT NULL,
    role        text NOT NULL,
    org_unit_id bigint REFERENCES iam.org_units,
    UNIQUE NULLS NOT DISTINCT (user_id, product, role, org_unit_id)
);

-- Tenant-wide part of authz_version: bumped by changes that affect many users.
CREATE TABLE iam.authz (
    version bigint NOT NULL
);

INSERT INTO iam.authz VALUES (1);

-- A user's own settings for one list: named views (the list's query string) and the columns
-- shown with their widths. Personal display preferences, so neither audited nor shared.
CREATE TABLE iam.list_settings (
    user_id  bigint NOT NULL REFERENCES iam.users ON DELETE CASCADE,
    list_key text NOT NULL CHECK (list_key ~ '^[a-z][a-z0-9_.-]{0,99}$'),
    settings jsonb NOT NULL,
    PRIMARY KEY (user_id, list_key)
);

CREATE SCHEMA setting;

-- A missing key means the default declared in code.
CREATE TABLE setting.values (
    key   text PRIMARY KEY,
    value text NOT NULL
);

-- Settings of one legal entity; a missing key means the default declared in code.
CREATE TABLE setting.legal_entity_values (
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    key             text NOT NULL,
    value           text NOT NULL,
    PRIMARY KEY (legal_entity_id, key)
);

CREATE SCHEMA numbering;

-- One counter per document type, legal entity and year; the row lock serialises numbering.
CREATE TABLE numbering.counters (
    doc_type        text NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    year            int NOT NULL,
    last            bigint NOT NULL,
    PRIMARY KEY (doc_type, legal_entity_id, year)
);

-- One counter per code series for the whole tenant; created on first use with the default prefix.
CREATE TABLE numbering.series (
    code   text PRIMARY KEY,
    prefix text NOT NULL,
    last   bigint NOT NULL
);

CREATE SCHEMA record;

-- The single source of status, version, date and legal entity of every document.
-- A module's document table uses this id as its primary key.
CREATE TABLE record.documents (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type        text NOT NULL,
    number          text NOT NULL,
    status          text NOT NULL CHECK (status IN ('draft', 'pending_approval', 'posted', 'cancelled')),
    version         int NOT NULL,
    date            date NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    org_unit_id     bigint NOT NULL REFERENCES iam.org_units,
    amount          bigint,
    fields          jsonb NOT NULL DEFAULT '{}',
    approval_ticket bigint,
    -- Who sent a pending document; only they withdraw it.
    submitted_by    bigint REFERENCES iam.users,
    -- Create-once: a retry by the same user with the same request id finds the first document.
    request_id      uuid,
    requested_by    bigint REFERENCES iam.users,
    CHECK ((status = 'pending_approval') = (approval_ticket IS NOT NULL))
);

CREATE INDEX documents_pending_idx ON record.documents (legal_entity_id, date) WHERE status = 'pending_approval';
CREATE INDEX documents_org_unit_id_idx ON record.documents (org_unit_id);
-- /me asks on every page load which products have documents.
CREATE INDEX documents_doc_type_idx ON record.documents (doc_type);
CREATE UNIQUE INDEX documents_request_id_key ON record.documents (requested_by, request_id) NULLS NOT DISTINCT
    WHERE request_id IS NOT NULL;

-- One row per legal entity, created on first use; documents dated on or before locked_until are frozen.
CREATE TABLE record.period_locks (
    legal_entity_id bigint PRIMARY KEY REFERENCES iam.org_units,
    locked_until    date
);

CREATE SCHEMA hrm;

-- Sensitive columns are encrypted by the service (AAD = schema.table.column).
CREATE TABLE hrm.employees (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code                text NOT NULL,
    full_name           text NOT NULL,
    date_of_birth       date,
    gender              text CHECK (gender IN ('male', 'female', 'other')),
    phone               text,
    email               text,
    address             text,
    org_unit_id         bigint NOT NULL REFERENCES iam.org_units,
    manager_id          bigint REFERENCES hrm.employees,
    user_id             bigint UNIQUE REFERENCES iam.users,
    hire_date           date NOT NULL,
    termination_date    date CHECK (termination_date >= hire_date),
    national_id         bytea,
    social_insurance_no bytea,
    tax_code            bytea,
    bank_account        bytea,
    CHECK (manager_id <> id)
);

CREATE UNIQUE INDEX employees_code_key ON hrm.employees (lower(code));
CREATE INDEX employees_org_unit_id_idx ON hrm.employees (org_unit_id);

-- Every dependent field is sensitive, so the whole record is one encrypted JSON.
CREATE TABLE hrm.dependents (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    employee_id bigint NOT NULL REFERENCES hrm.employees ON DELETE CASCADE,
    data        bytea NOT NULL
);

CREATE INDEX dependents_employee_id_idx ON hrm.dependents (employee_id);

-- Kinds of leave, entered by the customer; only deducting kinds touch the leave balance, and
-- days of a paid kind count as paid days in payroll. A kind may cap the days of one request
-- and require a file before a request is sent.
CREATE TABLE hrm.leave_types (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                text NOT NULL,
    deducts_balance     boolean NOT NULL,
    active              boolean NOT NULL DEFAULT true,
    paid                boolean NOT NULL DEFAULT false,
    max_days            numeric(4, 1) CHECK (max_days > 0 AND max_days * 2 = trunc(max_days * 2)),
    requires_attachment boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX leave_types_name_key ON hrm.leave_types (lower(name));

-- A leave request is a document: its id, status, version and date live in record.documents,
-- written first in the same transaction, hence the deferred FK.
CREATE TABLE hrm.leave_requests (
    id            bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    employee_id   bigint NOT NULL REFERENCES hrm.employees,
    leave_type_id bigint NOT NULL REFERENCES hrm.leave_types,
    start_date    date NOT NULL,
    end_date      date NOT NULL,
    days          numeric(4, 1) NOT NULL,
    reason        text,
    -- Days taken from the balance when the request was approved, so a cancel gives back exactly
    -- that, whatever the kind of leave says by then.
    deducted      numeric(4, 1) NOT NULL DEFAULT 0,
    CONSTRAINT leave_requests_days_check CHECK (days > 0 AND days * 2 = trunc(days * 2) AND days <= end_date - start_date + 1),
    CONSTRAINT leave_requests_dates_check CHECK (end_date >= start_date AND date_trunc('year', start_date) = date_trunc('year', end_date))
);

CREATE INDEX leave_requests_employee_id_idx ON hrm.leave_requests (employee_id);

-- Remaining leave days of an employee in a year.
CREATE TABLE hrm.leave_balances (
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    year        int NOT NULL,
    days        numeric(5, 1) NOT NULL CONSTRAINT leave_balances_days_check CHECK (days >= 0),
    PRIMARY KEY (employee_id, year)
);

-- One row per legal entity, only to lock: source documents take it FOR SHARE, posting a payroll
-- FOR UPDATE. Rows are created on first use.
CREATE TABLE hrm.payroll_locks (
    legal_entity_id bigint PRIMARY KEY REFERENCES iam.org_units
);

-- Payroll periods of a legal entity; a period is closed while posted_payroll_id is set.
CREATE TABLE hrm.payroll_periods (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    legal_entity_id   bigint NOT NULL REFERENCES iam.org_units,
    period_start      date NOT NULL,
    period_end        date NOT NULL CHECK (period_end >= period_start),
    posted_payroll_id bigint REFERENCES record.documents,
    UNIQUE (legal_entity_id, period_start)
);

-- Kinds of contract, entered by the customer. Contracts of a fixed-term kind need an end date;
-- the others may not have one.
CREATE TABLE hrm.contract_types (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text NOT NULL,
    active     boolean NOT NULL DEFAULT true,
    fixed_term boolean NOT NULL DEFAULT false
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

-- An overtime request is a document (see hrm.leave_requests for the deferred FK): the hours
-- of one day, split into day and night shift, in half-hour steps.
CREATE TABLE hrm.overtime_requests (
    id          bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    date        date NOT NULL,
    day_kind    text NOT NULL CHECK (day_kind IN ('weekday', 'weekly_off', 'holiday')),
    day_hours   numeric(4, 1) NOT NULL,
    night_hours numeric(4, 1) NOT NULL,
    reason      text,
    CONSTRAINT overtime_requests_hours_check CHECK (
        day_hours >= 0 AND night_hours >= 0 AND day_hours + night_hours > 0 AND day_hours + night_hours <= 24
        AND day_hours * 2 = trunc(day_hours * 2) AND night_hours * 2 = trunc(night_hours * 2))
);

CREATE INDEX overtime_requests_employee_id_idx ON hrm.overtime_requests (employee_id);

-- A timesheet is a document (see hrm.leave_requests for the deferred FK): the days worked
-- in one org unit over one payroll period. Cancelling one clears active, so the unit may
-- have a new timesheet for that period.
CREATE TABLE hrm.timesheets (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    org_unit_id  bigint NOT NULL REFERENCES iam.org_units,
    period_start date NOT NULL,
    period_end   date NOT NULL CHECK (period_end >= period_start),
    active       boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX timesheets_org_unit_period_key ON hrm.timesheets (org_unit_id, period_start) WHERE active;

-- One employee, one day: a full or half day actually worked. A blank cell has no row.
CREATE TABLE hrm.timesheet_lines (
    timesheet_id bigint NOT NULL REFERENCES hrm.timesheets ON DELETE CASCADE,
    employee_id  bigint NOT NULL REFERENCES hrm.employees,
    date         date NOT NULL,
    days         numeric(2, 1) NOT NULL CHECK (days IN (0.5, 1)),
    PRIMARY KEY (timesheet_id, employee_id, date)
);

CREATE INDEX timesheet_lines_employee_id_idx ON hrm.timesheet_lines (employee_id);

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

-- A payroll is a document (see hrm.leave_requests for the deferred FK): the pay of a legal
-- entity's employees over one period. inputs_hash is the sha256 of the sources it was computed
-- from. Department totals are summed from the encrypted lines when read, never stored: a
-- one-person department's totals are that person's pay.
CREATE TABLE hrm.payrolls (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    period_start date NOT NULL,
    period_end   date NOT NULL CHECK (period_end >= period_start),
    computed_at  timestamptz,
    inputs_hash  bytea
);

-- One employee's pay: the inputs read and every amount, one encrypted JSON (AAD hrm.payroll_lines.data).
CREATE TABLE hrm.payroll_lines (
    payroll_id  bigint NOT NULL REFERENCES hrm.payrolls ON DELETE CASCADE,
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    org_unit_id bigint NOT NULL REFERENCES iam.org_units,
    data        bytea NOT NULL,
    PRIMARY KEY (payroll_id, employee_id)
);

-- The posted source documents a payroll was computed from, at their versions.
CREATE TABLE hrm.payroll_sources (
    payroll_id bigint NOT NULL REFERENCES hrm.payrolls ON DELETE CASCADE,
    doc_type   text NOT NULL,
    doc_id     bigint NOT NULL,
    version    int NOT NULL,
    PRIMARY KEY (payroll_id, doc_type, doc_id)
);

-- Back pay (+) or recovery (-) of a closed period, entered on a draft payroll; amount and
-- reason are one encrypted JSON (AAD hrm.payroll_adjustments.data).
CREATE TABLE hrm.payroll_adjustments (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payroll_id    bigint NOT NULL REFERENCES hrm.payrolls ON DELETE CASCADE,
    employee_id   bigint NOT NULL REFERENCES hrm.employees,
    source_period date NOT NULL,
    data          bytea NOT NULL
);

CREATE INDEX payroll_adjustments_payroll_id_idx ON hrm.payroll_adjustments (payroll_id);

CREATE SCHEMA approval;

-- At most one rule per document type and action; flow is a JSON tree edited as a whole.
CREATE TABLE approval.rules (
    doc_type         text NOT NULL,
    max_levels       int NOT NULL CHECK (max_levels BETWEEN 1 AND 10),
    fallback_product text NOT NULL,
    fallback_role    text NOT NULL,
    flow             jsonb NOT NULL,
    action           text NOT NULL DEFAULT 'post',
    PRIMARY KEY (doc_type, action)
);

-- One submission of a document. It copies what it needs from the rule, so editing
-- the rule never changes an open instance; plan is the flow resolved at submission.
-- No FK to the document: deleting a draft keeps its history.
CREATE TABLE approval.instances (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type         text NOT NULL,
    doc_id           bigint NOT NULL,
    version          int NOT NULL,
    submitted_by     bigint NOT NULL REFERENCES iam.users,
    submitted_at     timestamptz NOT NULL DEFAULT now(),
    status           text NOT NULL CHECK (status IN ('open', 'approved', 'rejected', 'withdrawn', 'stale')),
    max_levels       int NOT NULL,
    fallback_product text NOT NULL,
    fallback_role    text NOT NULL,
    plan             jsonb NOT NULL,
    action           text NOT NULL DEFAULT 'post',
    params           jsonb
);

CREATE INDEX instances_doc_idx ON approval.instances (doc_type, doc_id);
-- One open request per document holds it; a second one waits on record's held check via
-- the document row lock and fails there. This index is a backstop against a race.
CREATE UNIQUE INDEX instances_one_open_idx ON approval.instances (doc_type, doc_id) WHERE status = 'open';

-- The steps of an instance's plan. approvers is fixed when the step starts; each vote has its
-- own row. A step's due time is fixed when it opens; overdue_notified_at makes the notice once only.
CREATE TABLE approval.instance_steps (
    instance_id         bigint NOT NULL REFERENCES approval.instances,
    position            int NOT NULL,
    approver            jsonb NOT NULL,
    approvers           bigint[],
    fallback            boolean NOT NULL DEFAULT false,
    approvals           int NOT NULL DEFAULT 1 CHECK (approvals BETWEEN 1 AND 20),
    rejections          int NOT NULL DEFAULT 1 CHECK (rejections BETWEEN 1 AND 20),
    status              text NOT NULL CHECK (status IN ('waiting', 'open', 'approved', 'rejected', 'cancelled')),
    due_hours           int CHECK (due_hours BETWEEN 1 AND 720),
    due_at              timestamptz,
    overdue_notified_at timestamptz,
    PRIMARY KEY (instance_id, position)
);

CREATE INDEX instance_steps_open_idx ON approval.instance_steps USING gin (approvers) WHERE status = 'open';
-- The overdue job scans only open steps that have a due time and were not yet told.
CREATE INDEX instance_steps_due_idx ON approval.instance_steps (due_at) WHERE status = 'open' AND overdue_notified_at IS NULL;

-- cast_by is set when a delegate votes for user_id.
CREATE TABLE approval.step_votes (
    instance_id bigint NOT NULL,
    position    int NOT NULL,
    user_id     bigint NOT NULL REFERENCES iam.users,
    decision    text NOT NULL CHECK (decision IN ('approved', 'rejected')),
    reason      text,
    decided_at  timestamptz NOT NULL DEFAULT now(),
    cast_by     bigint REFERENCES iam.users,
    PRIMARY KEY (instance_id, position, user_id),
    FOREIGN KEY (instance_id, position) REFERENCES approval.instance_steps
);

-- One vote per person per step, whether in person or for someone else.
CREATE UNIQUE INDEX step_votes_one_per_person_idx ON approval.step_votes (instance_id, position, (coalesce(cast_by, user_id)));

CREATE TABLE approval.delegations (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES iam.users,
    delegate_id bigint NOT NULL REFERENCES iam.users,
    starts_on   date NOT NULL,
    ends_on     date NOT NULL,
    created_by  bigint NOT NULL REFERENCES iam.users,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (delegate_id <> user_id),
    CHECK (ends_on >= starts_on)
);

CREATE INDEX delegations_delegate_idx ON approval.delegations (delegate_id, starts_on, ends_on);
CREATE INDEX delegations_user_idx ON approval.delegations (user_id, starts_on, ends_on);

CREATE SCHEMA dataio;

-- Uploaded import files and temporary export files; contents live in FILES_DIR under the same id,
-- written before this row. dataio.cleanup removes both once expired.
-- An exported file names the export and its params, so every download can ask the module
-- again whether the owner may still read it. Uploads have neither.
CREATE TABLE dataio.files (
    id            text PRIMARY KEY,
    name          text NOT NULL,
    owner_id      bigint NOT NULL REFERENCES iam.users,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    export_kind   text,
    export_params jsonb,
    CHECK ((export_kind IS NULL) = (export_params IS NULL))
);

CREATE INDEX files_expires_at_idx ON dataio.files (expires_at);

CREATE SCHEMA posting;

-- Posting lines: what a posted document means economically, captured once. Amounts are signed
-- (negative reduces) and never carry one person's sensitive amounts. Voiding keeps the lines.
CREATE TABLE posting.lines (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type        text NOT NULL,
    doc_id          bigint NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    date            date NOT NULL,
    -- Open: every product adds its own kinds.
    kind            text NOT NULL,
    org_unit_id     bigint NOT NULL REFERENCES iam.org_units,
    amount          bigint NOT NULL,
    voided_at       timestamptz
);

CREATE INDEX lines_doc_idx ON posting.lines (doc_type, doc_id);

-- One row per recording of a document's lines: at most one unvoided recording per document,
-- so lines are never doubled.
CREATE TABLE posting.recordings (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type    text NOT NULL,
    doc_id      bigint NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    voided_at   timestamptz
);

CREATE UNIQUE INDEX recordings_doc_once ON posting.recordings (doc_type, doc_id) WHERE voided_at IS NULL;

CREATE SCHEMA attachment;

-- Files attached to a record of any type, named by (doc_type, doc_id) without a foreign key.
-- Contents live in the attachment directory under file_id, written before this row;
-- attachment.cleanup removes contents no row names any more.
CREATE TABLE attachment.files (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type     text NOT NULL,
    doc_id       bigint NOT NULL,
    file_id      text NOT NULL UNIQUE,
    name         text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    size         bigint NOT NULL CHECK (size BETWEEN 1 AND 20971520),
    content_type text NOT NULL CHECK (content_type IN (
        'application/pdf', 'image/jpeg', 'image/png',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')),
    uploaded_by  bigint NOT NULL REFERENCES iam.users,
    uploaded_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX files_doc_idx ON attachment.files (doc_type, doc_id);

CREATE SCHEMA discussion;

-- Comments on a record of any type, named by (doc_type, doc_id) without a foreign key.
-- Plain text, never edited or deleted, except with a deleted draft.
CREATE TABLE discussion.comments (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type   text NOT NULL,
    doc_id     bigint NOT NULL,
    author_id  bigint NOT NULL REFERENCES iam.users,
    body       text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX comments_doc_idx ON discussion.comments (doc_type, doc_id);

CREATE SCHEMA notification;

-- What a user is told of: a record (doc_type, doc_id, no foreign key) or one of their jobs.
-- No text: the reader builds it from kind, so nothing of the record is copied here.
CREATE TABLE notification.notifications (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES iam.users,
    -- Products register their own kinds; the service refuses any it does not know.
    kind       text NOT NULL,
    doc_type   text,
    doc_id     bigint,
    job_id     bigint,
    actor_id   bigint REFERENCES iam.users,
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at    timestamptz,
    CHECK ((doc_type IS NOT NULL AND doc_id IS NOT NULL AND job_id IS NULL)
        OR (doc_type IS NULL AND doc_id IS NULL AND job_id IS NOT NULL))
);

CREATE INDEX notifications_user_idx ON notification.notifications (user_id, id DESC);
CREATE INDEX notifications_unread_idx ON notification.notifications (user_id) WHERE read_at IS NULL;

-- The tenant's SMTP server, one row or none: without it no email is sent. The password is
-- sealed by the app's keyring; base_url is the address users reach mmerp at, for links.
CREATE TABLE notification.mail_server (
    id           boolean PRIMARY KEY DEFAULT true CHECK (id),
    host         text NOT NULL,
    port         int NOT NULL CHECK (port BETWEEN 1 AND 65535),
    security     text NOT NULL CHECK (security IN ('starttls', 'tls', 'none')),
    username     text NOT NULL,
    password     bytea,
    from_address text NOT NULL,
    base_url     text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE SCHEMA printing;

-- The print data of each part of a posted document, frozen in the transaction that posted it
-- (or at its first print, for one posted before printing existed). Encrypted: a contract or a
-- payslip holds per-person amounts. Kept forever, cancelled documents included.
CREATE TABLE printing.snapshots (
    doc_type   text NOT NULL,
    doc_id     bigint NOT NULL,
    part       bigint NOT NULL,
    position   int NOT NULL,
    data       bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (doc_type, doc_id, part)
);

-- What the first print of a posted document fixed, so that every reprint draws the same.
CREATE TABLE printing.pins (
    doc_type  text NOT NULL,
    doc_id    bigint NOT NULL,
    layout    int NOT NULL CHECK (layout >= 1),
    locale    text NOT NULL CHECK (locale IN ('vi', 'en')),
    pinned_at timestamptz NOT NULL DEFAULT now(),
    -- The text of each block as the first print drew it, in the pinned locale.
    blocks    jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (doc_type, doc_id)
);

-- The text blocks of a print template, a row per save by a tenant administrator; texts maps
-- each block to its text in vi and en.
CREATE TABLE printing.blocks (
    template text NOT NULL,
    version  int NOT NULL CHECK (version >= 1),
    texts    jsonb NOT NULL,
    saved_by bigint NOT NULL REFERENCES iam.users,
    saved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (template, version)
);

CREATE SCHEMA sales;

-- A customer is seen by whoever holds a sales role at its owning org unit or above.
CREATE TABLE sales.customers (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code          text NOT NULL,
    name          text NOT NULL,
    tax_code      text,
    address       text,
    phone         text,
    email         text,
    contact_name  text,
    payment_terms text,
    org_unit_id   bigint NOT NULL REFERENCES iam.org_units,
    active        boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX customers_code_key ON sales.customers (lower(code));
CREATE INDEX customers_org_unit_id_idx ON sales.customers (org_unit_id);

-- One catalogue for the tenant: goods and services alike.
CREATE TABLE sales.items (
    id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code     text NOT NULL,
    name     text NOT NULL,
    unit     text NOT NULL,
    price    bigint NOT NULL CHECK (price >= 0),
    vat_rate text NOT NULL CHECK (vat_rate IN ('none', '0', '5', '8', '10')),
    active   boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX items_code_key ON sales.items (lower(code));

-- A quotation or order; the id, number, status, date and org unit live in record.documents
-- (deferred FK: record inserts its row first in the same transaction). The customer columns
-- are copies taken on each draft save, so a document keeps what it said.
CREATE TABLE sales.headers (
    id                bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    kind              text NOT NULL CHECK (kind IN ('quote', 'order')),
    customer_id       bigint NOT NULL REFERENCES sales.customers,
    customer_code     text NOT NULL,
    customer_name     text NOT NULL,
    customer_tax_code text,
    customer_address  text,
    customer_phone    text,
    customer_email    text,
    contact_name      text,
    valid_until       date,
    delivery_date     date,
    quote_id          bigint REFERENCES sales.headers,
    payment_terms     text,
    delivery_terms    text,
    note              text,
    subtotal          bigint NOT NULL,
    discount_total    bigint NOT NULL,
    vat_total         bigint NOT NULL,
    total             bigint NOT NULL,
    CHECK (total = subtotal - discount_total + vat_total),
    CHECK (kind = 'quote' AND valid_until IS NOT NULL AND delivery_date IS NULL AND quote_id IS NULL
        OR kind = 'order' AND valid_until IS NULL)
);

CREATE INDEX headers_customer_id_idx ON sales.headers (customer_id);
CREATE INDEX headers_quote_id_idx ON sales.headers (quote_id);

CREATE TABLE sales.lines (
    doc_id           bigint NOT NULL REFERENCES sales.headers ON DELETE CASCADE,
    position         int NOT NULL CHECK (position > 0),
    item_id          bigint NOT NULL REFERENCES sales.items,
    item_code        text NOT NULL,
    description      text NOT NULL,
    unit             text NOT NULL,
    quantity         numeric(15, 3) NOT NULL CHECK (quantity > 0),
    unit_price       bigint NOT NULL CHECK (unit_price >= 0),
    discount_percent numeric(5, 2) NOT NULL CHECK (discount_percent BETWEEN 0 AND 100),
    vat_rate         text NOT NULL CHECK (vat_rate IN ('none', '0', '5', '8', '10')),
    amount           bigint NOT NULL,
    discount         bigint NOT NULL,
    vat              bigint NOT NULL,
    PRIMARY KEY (doc_id, position)
);

CREATE INDEX lines_item_id_idx ON sales.lines (item_id);
