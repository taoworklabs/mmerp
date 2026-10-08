# Tenancy and deployment

A **tenant** is an organisation (company) using the system, with its own users, org units and data. **Each tenant always has its own database.** This is a settled decision ([ADR-0001](./docs/adr/0001-one-database-per-tenant.md)).

## Two places to run, one binary

| | On-premise (primary) | Cloud |
| --- | --- | --- |
| Where it runs | The organisation's server, inside its internal network | Infrastructure of whoever runs the hosted deployment |
| Layout | One compose stack: app + Postgres | One compose stack per tenant (model A); merged into one app with many databases (model B) once there are enough tenants |
| Who operates the server | The organisation (or a contractor) | The hosting operator |
| Internet connection | Not guaranteed, may be lost for hours or blocked entirely | Always available |
| Updates | On the organisation's schedule; installations run many different versions | The hosting operator drives them |

**Single-tenant mode is the primary mode and is always kept.** The binary runs fully on its own with one database, without depending on any external service. Everything that serves only the cloud (tenant registry, tenant-selecting middleware) is an optional addition and never becomes mandatory.

## Cloud: path from A to B

1. **Use A up to a few dozen cloud tenants.** The same compose stack as the on-premise release; several tenants can share one VPS as separate compose projects. The cloud and on-premise releases run identically.
2. **Move to B when operating A in the cloud becomes too heavy.** The code still knows nothing about tenants:
   - A tenant registry (its own small database) maps subdomain → database URL, the list of enabled products and the tenant's status.
   - Middleware identifies the tenant from the `Host` header and puts the matching pool into the request context with `platform.WithDB`. Services already take the DB from the context (see [backend.md](./backend.md#transactions)), so nothing inside the services changes.
   - Sessions live in the tenant's database, so one tenant's cookie is worthless at another tenant.
   - Each tenant's connection pool is small and is created only when the tenant gets its first request. If the total number of connections hits Postgres's limit, put PgBouncer in front.
   - On deploy, migrate each tenant database before switching traffic. A tenant whose migration fails is put in the `maintenance` status (answers 503) and reported; the new binary never runs on an old schema.
   - The River queue lives in each tenant's database, so enqueuing in the same business transaction stays correct, just as in single-tenant mode. The app always runs one River client for every tenant with status `active` in the registry, even when the tenant has no requests: if clients were created only on request, retried and periodic jobs of a quiet tenant would never run. When the number of tenants exceeds what one instance can handle, split tenants across several app instances by registry ([ADR-0004](./docs/adr/0004-job-queue-in-tenant-database.md)). No queue shared across tenants: a queue in another database cannot be enqueued in the same transaction, and would need an outbox.
   - A tenant's list of enabled products comes from the registry and is checked per request, as on-premise (see [platform.md](./platform.md#enabled-products)). Tenants with different enabled products share one app without mounting separate routes.
   - Files are stored under a prefix of their own for each tenant.

## Versions and migrations

On-premise means many installations running many different versions, and some will upgrade across several versions at once.

- Migrations run at app startup, in order, from any older version up to the current one. A released migration is never edited or deleted. No down migrations.
- Migrations follow expand/contract, so cloud model B can deploy in a rolling fashion.
- **Migrations are written only in SQL.** Go logic changes with the version; an installation jumping from v1 to v9 runs v9's logic on v1's data. When data must be recomputed, the migration only adds columns, the recomputation runs as a job after the app is up, and the code must work correctly even before the job finishes.
- Migrations never touch attachment files, so the dump before migrating needs only the database.
- River's migrations (`rivermigrate`) run in the same procedure and the same transaction as the app's migrations.
- The runner records the migrations it has run in `public.schema_migrations` (file names). This is the runner's ledger, not a table of any module. A database with migrations the binary does not know (an old release on a newer schema) makes the app refuse to start with the error `schema_newer_than_binary`.
- The Postgres major version is pinned in `compose.yml`, and the app image will bundle `pg_dump` of the same major (not yet: the runtime image has no `postgresql-client` yet). Upgrading Postgres's major version is a separate procedure, not bundled with app updates.
- Versions follow semver. A number of recent releases are supported (how many is settled by an ADR). Installations outside the support window must upgrade before they get support.

### Update procedure (on-premise)

**Not fully built yet**; done in [M10](./roadmap.md#m10-on-premise-operations). Today the app has only steps 4 and 5: migrating in one transaction, locked with `pg_advisory_xact_lock` inside that transaction; there is no backup step, disk space check or `pg_restore --list` yet. The dump before an update is currently made only by `scripts/deploy.sh`, outside the app.

1. **Stop writes.** The update command stops the old app container; Postgres keeps running. From here on no request or worker writes to the database.
2. **Lock migrations.** The new app takes a `pg_advisory_lock` reserved for migrations, so two instances cannot migrate at the same time. If the schema is already at the current version, it releases the lock and runs normally.
3. **Back up.** The app runs `pg_dump` into the backup directory, naming the file with the current schema version, then checks it roughly with `pg_restore --list`. That command only confirms the dump's table of contents can be read; it does not prove all data can be restored; that is the job of the periodic restore check ([platform.md](./platform.md#operations)). Before dumping, compare `pg_database_size()` with the free space; if there is not enough, stop right away. If the backup fails (disk full, corrupt dump) the app stops **before** migrating. The database has not been touched, so running the old release again is enough.
4. **Migrate.** All missing migrations run in one transaction; Postgres allows DDL inside a transaction. On error it rolls back, the schema stays at the old version, the app stops and logs a clear error; running the old release again is enough. Migrations that cannot run inside a transaction (such as `CREATE INDEX CONCURRENTLY`) are forbidden.
5. **Run.** Only after migrating does the app open HTTP and start workers.

### Restoring from a backup

Used when data is corrupted, or when the new release runs but is wrong and must be rolled back to the old one:

1. Stop the app.
2. Create a new database, `pg_restore` the dump into it, then restore the attachment file directory from the same point in time.
3. Check that the current encryption key decrypts the dump's data (try decrypting one row with a sensitive field). If the key does not match, stop: the right key must come from the key backup recorded in the handover record.
4. Point the app at the new database and run the image whose version is recorded in the dump's file name.
5. Data written after the dump was taken is lost. The admin screen and the handover record must say so clearly.

In model B, each tenant goes through the same steps. A tenant being migrated is in the `maintenance` status (answers 503) and its workers are paused.

## Rules that apply from now on

- Business code is tenant-unaware: tables have no tenant column, services take no tenant parameter.
- Services take the database only from the context (rule in [backend.md](./backend.md#transactions)). In single-tenant mode, `internal/app` puts the only database into every request's context; in model B, the tenant middleware does that.
- Outside `cmd/server` and `internal/app`, nothing may assume there is only one global database. No package-level pool or DB variable.
- Jobs live in the tenant's database, so the payload needs no tenant key.
- No business operation may depend on the Internet or on external services. Every outbound call goes through a job (see [platform.md](./platform.md#integrations)).
