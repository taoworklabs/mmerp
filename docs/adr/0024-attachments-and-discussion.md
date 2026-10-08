# 0024. Attachments and discussion

Status: accepted · 2026-10-07 · amends the "File storage" part of [ADR-0019](./0019-background-jobs-dataio-and-file-storage.md)

## Context

HRM users send contract scans and leave request paperwork through shared folders, email or Zalo, and ask questions about documents in private messages. `documents.md` already promised attachments and discussion as common features of every record type, built once by `core`. The ADR-0019 file store has only one directory, and the `dataio` cleanup deletes every file older than one day that is not in `dataio.files`: putting attachment files there would get them deleted.

## Decision

### Two modules in `core`

- **`attachment`** owns `attachment.files (id, doc_type, doc_id, file_id, name, size, content_type, uploaded_by, uploaded_at)`, indexed by `(doc_type, doc_id)`. `file_id` is the id issued by `platform.Files`.
- **`discussion`** owns `discussion.comments (id, doc_type, doc_id, author_id, body, created_at)`, indexed by `(doc_type, doc_id)`; `body` has a length `CHECK` of 1 to 4,000 characters.
- Both attach by `(doc_type, doc_id)`, with no foreign key to business module tables and no knowledge of business tables. Every permission decision goes through the record type's `record.Can`; a type not registered with `record` returns not found.
- Applies to every record type, both documents and catalogues.

### Two new `record` actions

| Action | Class | Meaning |
| --- | --- | --- |
| `view_files` | read | List and download attachments |
| `attach` | write | Add attachments, and delete other people's attachments |

- A record type declares the extra permission needed to see attachments by answering `view_files` in `Can`. Labour contracts answer with the contract view permission plus `hrm.salary.view`, because the scan shows the salary. Employee profiles answer with the view permission (to add: the edit permission) plus `hrm.employee.sensitive`, because the ID card (CCCD) scan contains exactly the sensitive fields. Uploaders do not mark files as sensitive themselves.
- `Can` returns `false` for actions it does not know, so a type that does not yet answer these two actions has no attachments: nothing listed, nothing can be added.
- Discussion has no action of its own: reading needs `view`; commenting needs `view` plus the product gate for the write class.

### Common rules by status

The core applies these rules on top of `Can`, reading status from `record.documents`:

- A `cancelled` document: no adding or deleting attachments, no commenting. Listing, downloading and reading the discussion still work.
- A document whose date falls in a locked period: adding, deleting attachments and commenting still work. Attachments and discussion are accompanying records that do not change the document's figures, so they do not go through the period lock and do not increase `version`.
- Other statuses are decided by the module's `Can(attach)`. HRM allows adding attachments to a `posted` contract, because contracts are often signed after being entered into the system.
- Catalogues have no status; only `Can` decides.

### Permission to delete and edit

- Deleting an attachment needs `view_files`, the product gate for the write class and a non-cancelled document; on top of that, one of two: being the uploader, or having `attach`.
- Comments cannot be edited or deleted.

### Not revealing records

For a record the user cannot view, every route returns not found, not "forbidden". If the user can view the record but lacks `view_files`, the attachment list returns `hidden: true` and no files; that person already knows the record exists, and the frontend does not have to infer permissions from error codes. Download, delete and add still return not found. The same goes for a correctly guessed attachment id: download and delete always look up the file's record and re-check the current permission. When the product is disabled, write operations return the product gate's error code, like other core routes.

### Deleting drafts

`record` defines the interface `Keeper{Deleted(ctx, ref) error}` in `hooks.go`. `attachment` and `discussion` register themselves with `record.OnDeleted` at construction. Each module's `Deleted` runs in the `record.Delete` transaction, after the document row is deleted, and deletes the record's rows; on error the draft remains. Files on disk are removed later by the cleanup.

### File store split per module

- `platform.Files` has `Sub(module) Files`: a subdirectory named after the module, created on demand. Each module writes to and cleans only its own directory.
- `Older` already skips directories; the cleanup at the root directory never descends into subdirectories.
- `dataio` moves into the `dataio` subdirectory. Old `dataio` temp files at the root are left behind: they only live 24 hours, and nobody cleans the root any more.
- Still local disk only. An object storage adapter is built when cloud needs it, not together with attachments as ADR-0019 planned.

### Adding and deleting never lose files

- Add: write the file into the `attachment` directory first, then in one transaction write the metadata and audit.
- Delete: the transaction only deletes the metadata and writes audit.
- The system job `attachment.cleanup` (like `dataio.cleanup`, every hour and at startup) deletes files older than one day that no row in `attachment.files` points to. A rollback at any step never loses a file that is still referenced, and a same-day DB backup never points to a lost file.

### Limits

- Each file at most 20 MB; beyond that returns `file_too_large` with `max_mb`. The frontend checks before uploading; the server always checks again.
- The file type is detected from content, not from the extension or the browser header: PDF, JPEG, PNG by their leading signature; DOCX and XLSX are zip files containing `word/document.xml` or `xl/workbook.xml`. Other types return `file_type_not_allowed`. The stored `content_type` is the detected type.
- File names are kept as uploaded, at most 255 characters.
- Comments are plain text, line breaks kept, 1 to 4,000 characters; longer returns `comment_too_long` with `max`.

### Core API

| Route | Notes |
| --- | --- |
| `GET /records/{type}/{id}/attachments` | List with `allowed_actions` of the record (`attach`) and of each file (`delete`), `hidden`, and the limits (`max_mb`, `accept`) so the frontend can check before uploading |
| `POST /records/{type}/{id}/attachments` | Multipart, one file at a time |
| `GET /attachments/{id}` | Download; re-checks permission every time |
| `DELETE /attachments/{id}` | |
| `GET /records/{type}/{id}/comments` | Chronological, with the record's `allowed_actions` (`comment`) and `max_length` |
| `POST /records/{type}/{id}/comments` | |

Routes live under `/records` because they apply to catalogues too. Every route goes through the record type's product gate by class: listing and downloading are reads, the rest are writes.

### Audit

- Written on the record, in the same transaction: `attachment.added`, `attachment.removed` (with file id and name), `attachment.downloaded` (with id), `discussion.commented` (with comment id). Audit never contains file content or comment text.
- The document history (`record.History`) shows `attachment.added` and `attachment.removed` only to people with `view_files`, because file names can reveal content; it does not show `attachment.downloaded`. `attachment` declares these rules itself with `record.RestrictHistory`, so `record` does not know other modules' action names.
- Huma does not limit multipart bodies: the upload route rejects a body declared longer than the limit with its own error code (`platform.UploadLimit`), and the app cuts every body at 32 MB.

## Rejected alternatives

- **Uploaders mark files as sensitive**: depends on users remembering to mark them; one slip leaks a salary. The record type knows better.
- **One `files` action for both viewing and adding**: contracts need viewing stricter than `view` but adding by the edit permission; the two questions are of different classes.
- **Attachments go through the period lock**: contract scans often arrive after the period is locked; the period lock protects figures, not accompanying records.
- **Deleting the file on disk inside the delete transaction**: a rollback after the file is deleted leaves metadata pointing to a lost file.
- **One shared directory, with `dataio` asking `attachment` before cleaning**: every new module would have to change other modules' cleanup.

## Consequences

- A new record type gets attachments just by answering `view_files` and `attach` in `Can`; discussion is available for every record type that can be viewed.
- Attaching, deleting attachments and commenting do not change the document's `version`, so they never conflict with someone editing it.
- The file directory must still be included in backups; its size grows with attachments, and there is no total limit per tenant yet.
- Download reads the whole file into memory before responding; at 20 MB per file that is acceptable.
