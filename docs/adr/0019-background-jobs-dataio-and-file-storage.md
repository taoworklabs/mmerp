# 0019. Background jobs, dataio and file storage

Status: accepted · 2026-10-06 · the "File storage" part is amended by [ADR-0024](./0024-attachments-and-discussion.md): the file store is split into subdirectories per module, each module cleans only its own directory; the "Jobs" part is extended by [ADR-0025](./0025-notifications.md): a notification flag in `JobSpec`, administrators can see system jobs

## Context

M5 needs to import timesheets and leave balances from Excel, export timesheets to Excel, and track those tasks while they run in the background. Up to M4 there is no job queue, no file storage and no `dataio` module. River and excelize were chosen in `techstack.md`; the queue lives in the tenant database per ADR-0004. The rule "moving to the background does not raise permissions" is already in `documents.md` and `platform.md`; this ADR settles how it is done.

## Decision

### Jobs (`platform`)

- **A single job-creation function** `platform.Enqueue(ctx, args)`. Every args type implements `river.JobArgs` and `Spec() JobSpec`, with `JobSpec{Product, Class, System}`. `Enqueue` calls `ProductGate(Product, Class)`, then uses `InsertTx` when ctx holds a transaction, `Insert` otherwise.
- **The requester lives in the River job metadata**, not in args: `Enqueue` writes `{"requested_by", "product", "class"}` for jobs run on behalf of a user. A job without `requested_by` is a system job. Args carry only business data, and a user's job list is filtered directly on River metadata.
- **Worker middleware** installed into River by `internal/app`:
  - rebuilds ctx from the app environment (DB, logger, enabled products, keyring, files, queue);
  - jobs on behalf of a user: set the actor to the requester and call `ProductGate` again for the job's class. Permissions (`Can`, `Scope`, sensitive fields) are re-checked by the service as for a request;
  - system jobs: no actor (the audit `actor_id` is NULL), no permission checks, always run to completion;
  - a `*platform.Error` with a 4xx status becomes `river.JobCancel`: permission, product or data errors are not retried. The error code and parameters are written to the River job output (`metadata.output.error`) for the API to read back. Other errors are retried by River.
- **Job status lives in River**, read through the River client API; there is no separate table. Finished jobs (completed, cancelled or discarded) are kept by River for 7 days, then deleted.
- `platform.Module` gets `Workers func(*river.Workers)` and `Periodic []*river.PeriodicJob`. Periodic jobs are system jobs and must be written to catch up on their own. Up to M5 no periodic job belongs to a product, so there is no skipping a run when a product is disabled yet; added when one exists.
- Every app instance works jobs unless `RUN_JOBS=false` is set: that instance only creates jobs. Used for the second instance of the e2e tests, running with a different product list on the same database; outside tests, all instances of a tenant have the same product list.
- River migrations run with `rivermigrate.MigrateTx` in the same transaction as the app migrations, after them.

### File storage (`platform`)

- `platform.Files{Dir}` stores files on local disk: `Save` (writes a temp file then renames it, returns a random id), `Open`, `Remove`. Ids are checked for the right shape before touching the disk. There is only one way to store, so no interface; an object storage adapter comes with attachments.
- The directory comes from `FILES_DIR` (default `data/files`). Compose mounts the `files` volume at `/data/files`; this directory must be included in backups.

### `dataio` (`core`)

- Table `dataio.files (id, name, owner_id, created_at, expires_at, export_kind, export_params)`: metadata of import files and temporary export files; export files also record the export kind and its parameters. The file is written first, then the metadata.
- Business modules register through deps: `RegisterImport(Import{Kind, Product, Header, Run})`, `RegisterExport(Export{Kind, Product, Run, Check})`. `Check` answers whether the current user may still read the data of that export. An import's `Run` receives the data rows (without the header row) and returns per-row errors `RowError{Row, Code, Params}`.
- `dataio` reads and writes Excel (excelize is used only here), checks the column count of the header row, numbers rows as Excel shows them, and translates per-row errors with `platform.Translate(locale, "<product>.import.<code>")` into the requester's language.
- **Import is all or nothing**: `dataio` runs `Run` in one transaction and rolls back if even one row has an error. The job then fails with `import_rows_invalid`, with up to 100 error rows. The job is marked complete (`river.JobCompleteTx`) in that same transaction, so a crash between the write and River recording the status does not make an import run twice.
- Jobs `dataio.import` (write class) and `dataio.export` (export class) run on behalf of the user. Exports write audit `dataio.exported`. The system job `dataio.cleanup` runs every hour and at startup: it deletes expired files and files on disk without metadata older than one day, so it catches up on its own.
- Core routes: `POST /imports/{kind}` (multipart: file and `params`; writes the file, then in one transaction writes the metadata and creates the job), `GET /imports/{kind}/template` (header row, runs immediately), `POST /exports/{kind}`, `GET /jobs` and `GET /jobs/{id}` (only the caller's own jobs), `GET /files/{id}` (only the file owner, before expiry; for export files each download calls `Check` again, so a permission revoked after the export also closes the file).
- Limits: import files at most 5 MB and 10,000 rows; export files expire after 24 hours.

## Consequences

- Creating a job anywhere goes through `Enqueue`, so no job skips the product gate; running a job always goes through the middleware, so no job on behalf of a user runs with more permissions than the requester.
- No extra process or container: River workers run inside the binary itself.
- Error rows and error codes live in the River job, so they expire with the job; users can review them for 7 days.
- Importing a large file holds a transaction for the whole write; with the 10,000-row limit that is acceptable.
