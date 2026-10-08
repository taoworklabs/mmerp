# 0026. Printing

Status: accepted · 2026-10-08 · extends [ADR-0019](./0019-background-jobs-dataio-and-file-storage.md): an export may produce a file other than Excel

## Context

Payroll staff lay out payslips by hand from the Excel export, and HR retypes labour contracts in Word from data already in the system. `documents.md` promises print templates as a shared feature of every record type, built once by `core`, with templates that only display data. The print of a signed contract or a handed-out payslip must be reproducible exactly, years later, even after names, legal-entity details or templates change.

Constraints: one Go binary plus Postgres (no Chromium, no sidecar); Vietnamese diacritics with no Internet; payslips and contracts carry per-person amounts, so every copy needs `hrm.salary.view` and is audited.

`dataio` already runs exports as user jobs: it re-checks permission when the job runs and on every download, lets only the requester download the file, expires it after 24 hours, cleans its own directory, notifies the requester and shows the job on "my background jobs". A print needs exactly that.

## Decision

### Module `printing` in `core`

- Owns the schema `printing`:
  - `printing.snapshots (doc_type, doc_id, part, data, created_at)`, primary key `(doc_type, doc_id, part)`. `data` is the print data of one part of a posted document, column-encrypted (it carries per-person amounts).
  - `printing.pins (doc_type, doc_id, layout, locale, blocks, pinned_at)`, primary key `(doc_type, doc_id)`. What a posted document's first print fixed: the layout version, the locale (`CHECK` in `vi`, `en`) and the resolved text of every text block.
  - `printing.blocks (template, version, texts, saved_by, saved_at)`, primary key `(template, version)`. Each save of a template's text blocks by a tenant administrator; `texts` holds every block in `vi` and `en`.
- No foreign key to business tables; attaches by `(doc_type, doc_id)`, like `attachment`.
- `deps.go`: `record`, `audit`, `dataio`, `iam` (for `core.admin` on text blocks). No hooks of its own. No record types, no business lines.
- A **part** is one printable unit of a record: `0` for a record printed whole (a contract), the employee id for one payslip of a payroll.

### Templates are registered in code

A business module registers a print template for one of its record types at construction:

- `Code` (e.g. `hrm.contract`, `hrm.payslip`), `DocType`, `Product`.
- `Data(ctx, id)` returns the current print data of the record, one JSON value per part. It reads all parts; who may print what is decided before it is called.
- `Layouts`: the layout functions, version `n` at index `n-1`; the last is current. A layout draws one part from its data through the drawing functions `printing` provides (titles, label and value rows, tables, text blocks), with labels translated in the given locale. **Only `printing` imports the PDF library.**
- `Blocks`: the named text blocks the layout uses, each with its allowed placeholders. Their default text lives in the module's translation files.
- A layout never changes once released. Changing what a print shows, including a label it uses, means adding a layout version (with new label keys if a label's text changes).

Templates only display data: layouts are Go code shipped with the release, and text blocks are plain text with `{name}` placeholders from the block's declared list. Saving a block that uses another placeholder fails with `print_placeholder_unknown` (`block`, `name`).

### Permission: a `print` action

- `record` gains the action `print`, of class **export**. `Can` answers `false` for unknown actions, so only types that answer it can be printed.
- HRM answers `print` on contracts with the contract view permission plus `hrm.salary.view`, the same extra permission as `view_files`; on payrolls with the payroll view permission plus `hrm.salary.view`. Payroll permissions are scoped by legal entity, so whoever may print a payroll's payslips may print every one of them; no per-part permission.
- `allowed_actions` of a contract or payroll includes `print` when allowed.

### Freezing on posting

- `record` gains a hook: `OnPosted(Freezer)`, where `Freezer{Posted(ctx, Doc) error}` runs in the transaction that moves a document to `posted`, after the type's `OnTransition`. An error rolls the posting back.
- `printing` registers itself. For a document type with a template, it writes one snapshot per part from `Data`. Snapshots never expire and are kept when the document is cancelled. A draft has none, so deleting a draft has nothing to clean.
- **Pinning at first print.** The first print of a posted document writes its pin: the current layout version, the requester's locale and the resolved text blocks. Every later print uses the snapshot and the pin, whoever prints it. Pinning resolved texts, not a block version, keeps the print unchanged even when the default texts change with a release.
- **Documents posted before printing existed** have no snapshot; the first print writes it, under the document's row lock (`record.Lock`), together with the pin.

| Status | Data | Layout, locale, blocks | Watermark |
| --- | --- | --- | --- |
| `draft`, `pending_approval` | current (`Data`) | current, requester's, current | draft |
| `posted` | snapshot | pinned | none |
| `cancelled` with a snapshot | snapshot | pinned | cancelled |
| `cancelled` without a snapshot | current | current, requester's, current | cancelled |

### Printing is a `dataio` export

- `dataio.Export` gains `Render(ctx, params) (File, error)`, the alternative to `Run` for a file that is not a sheet: `File{Name, ContentType, Body}`. Exactly one of the two is set.
- `printing` registers one export per template, with kind `printing.<template code>`, the template's product, and params `{"id": …, "parts": […]}` (no parts: all of them). Its `Check` re-checks `print` on the record and that every part exists; it runs when the job runs and on every download, as for every export.
- Starting, following and downloading a print use the existing routes (`POST /exports/{kind}`, `GET /jobs/{id}`, `GET /files/{id}`), the jobs screen and the frontend's `ExportButton`. Expiry and cleanup are `dataio`'s. `printing` has no job, file directory or download route of its own.
- `Render` writes the pin (if any) and the audit entry in one transaction, then draws the PDF. One PDF per job; with several parts, one per page, in the order of `Data`.

### PDF engine

- [`signintech/gopdf`](https://github.com/signintech/gopdf) (MIT): pure Go, no cgo; embeds TrueType fonts and subsets them to the glyphs used.
- Font: **Noto Sans** regular and bold (SIL Open Font License 1.1), embedded in the binary with `embed`, with the licence text next to it. Full Vietnamese coverage; adds about 1.2 MB to the binary.
- The whole PDF is built in memory. Measured: 3,000 one-page payslips render in about 2 seconds, using about 110 MB, into a 2.5 MB file. Stream page by page if a tenant ever prints far more.
- Tests read PDF text back with [`ledongthuc/pdf`](https://github.com/ledongthuc/pdf) (BSD-3), imported only by tests.

### Audit

- `printing.printed` on the record, in `Render`'s transaction: template, layout version, part count, and for payslips the employee ids. Never amounts or text.
- `record.History` shows it only to actors with `print` (`record.RestrictHistory`).
- Downloads are not audited separately, like every other export file: only the requester can download, and the print is already audited.
- Saving text blocks writes `printing.blocks_saved` (template, version).

### Text blocks are edited by tenant administrators

- Routes `GET /print-templates`, `GET /print-templates/{code}`, `PUT /print-templates/{code}/blocks`, for `core.admin`. Saving is a write of the template's product and goes through the product gate.
- Each save adds a version; the screen shows the current version and when it was saved. A template with no saved version uses its default texts.

## Rejected alternatives

- **Headless Chrome or an HTML-to-PDF service**: a third container, against the constraints.
- **A `printing` job, file table and download route of its own**: `dataio` already owns all of it, with the same permission re-check on every download.
- **Reusing `view_files` for printing**: a type could then not have attachments without printing; the two questions only happen to need the same permission today.
- **Freezing the rendered PDF**: the PDF is a temporary result. Freezing the data and pinning the layout keeps a posted print identical without storing files forever.
- **Pinning a block version number**: a document with no saved version would follow the default texts, which change with releases.
- **Tenants editing the whole layout**: needs a template language, a validator and an editor; text blocks cover the wording customers ask for.
- **Freezing per-person data at first print only**: the employee's name or the legal entity's details could change between posting and the first print.

## Consequences

- A new record type gets printing by registering a template and answering `print` in `Can`.
- Posting a payroll writes one encrypted snapshot per line in the posting transaction. That adds to the time the payroll lock is held (the "close in under 10 seconds" target of the production-readiness gate); measure it there.
- Every released layout version stays in the code for as long as documents may be pinned to it.
- The database backup holds every frozen print; no file needs to be kept for reprints.
