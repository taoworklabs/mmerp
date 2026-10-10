# Backend

Server-side module structure. For the UI side see [frontend.md](./frontend.md).

## Layers

```
internal/
├── app/              # composition root: reads the list of enabled products, assembles modules into a Huma API under /api
├── platform/         # infrastructure, NO tables: db/tx, config, errors, logging, job queue,
│                     # file storage, currency and rounding, the platform.Module type
├── core/             # core modules, always on, business-agnostic
│   ├── iam/  setting/  numbering/  audit/
│   └── record/  approval/  attachment/  discussion/  printing/  dataio/  customfield/
├── shared/           # master data and data shared between products
│   └── partner/  item/  tax/  posting/
├── modules/          # business modules of each product, including accounting
│   └── <module>/
└── integration/      # one package per external provider (see platform.md)
migrations/           # one directory, global ordering, file names prefixed by module
```

| Layer | Always on | Knows business logic | Has tables |
| --- | --- | --- | --- |
| `platform` | Yes | No | No |
| `core` | Yes | No | Yes |
| `shared` | Yes | Yes, but belongs to no product | Yes |
| `modules` | Per enabled product | Yes | Yes |

```
modules ──► shared ──► core ──► platform
   └── modules ↔ modules: only through interfaces in deps.go / hooks.go, wired in internal/app
```

Dependencies only point down. Within the `core` or `shared` layer, a module may import another module directly if the dependencies form a DAG (for example `approval → record → audit, numbering, iam`). When a lower module needs to call up into a higher one (such as `record` asking `approval` whether approval is needed), the lower module defines an interface in `hooks.go` and `internal/app` wires it, just as between products. Because the higher module is created later, this wiring uses setters called after construction, for example `rec.SetApprovalGate(appr)`, `rec.OnDeleted(att)` (the higher module registers itself at construction), `ids.SetTreeHook(rec)`, `ids.SetDataProducts(rec)`, `set.SetAuthz(ids)`. In the `modules` layer, modules **never import each other**.

A module is created when it first starts being used, not ahead of time. The first three waves built exactly the modules HRM needed; from wave 4 on, the core is built ahead of demand but still proven by HRM ([ADR-0023](./docs/adr/0023-build-core-first.md)):

| Wave | Modules | Proven by HRM flow |
| --- | --- | --- |
| 1 | `platform`, `iam`, `setting`, `audit`, `numbering` | Employee profiles, departments, permissions per org unit, sensitive fields |
| 2 | `record` + `recordtest`, `approval` | Leave requests, overtime requests, employment contracts |
| 3 | `dataio`, `shared/posting` | Importing timesheets from Excel; posting a payroll produces business lines |
| 4 | `attachment`, `discussion`, `notification`, `printing` | Attaching scanned contracts, discussion on leave requests, approval notifications, printing payslips (M11 to M13) |
| Later | `customfield` | When there is a specific need for it |


## Core modules

| Module | Owns |
| --- | --- |
| `iam` | users, sessions, roles and permissions, org units, legal entities; API keys when needed (not yet) |
| `setting` | settings per module and legal entity; keys prefixed by the module name (`<module>.<key>`) |
| `numbering` | document number sequences; fixed numbering scope `(doc_type, legal entity, year)`, called by `record` |
| `audit` | audit log and change history; `action` prefixed by the module name |
| `record` | record type registration, document status and lifecycle, period lock |
| `approval` | approval rules and flows; the adapter for `record`'s approval gate |
| `attachment` | file attachments |
| `discussion` | comments on records; mentions work together with `notification` |
| `printing` | print templates, PDF export |
| `dataio` | Excel import/export |
| `customfield` | custom field definitions, value validation |

Details of the document features are in [documents.md](./documents.md).

## Module shape

Every module in `core`, `shared` and `modules` has the same shape:

```
internal/modules/sales/
├── module.go       # Module(svc) → platform.Module: the manifest internal/app registers
├── service.go      # exported methods other modules may call; a large service splits into service_<part>.go
├── types.go        # exported types used in those methods
├── deps.go         # Deps struct: core/shared modules passed in, and interfaces for other products
├── hooks.go        # hook interfaces this module EMITS
├── handler.go      # HTTP, not exported
├── queries.sql
└── internal/
    └── store/      # sqlc-generated code; Go forbids any package outside the module from importing it
```

A file with no content is left out, never created empty: a module without routes has no `handler.go` or `module.go`, a module that emits no hooks has no `hooks.go`. The files that do exist always have the names and roles above.

`NewService(d Deps)` (in `service.go`) creates the service and registers with the core modules received through deps; `module.go` is the module's manifest, written in code:

```go
// service.go
func NewService(d Deps) *Service {
    s := &Service{d: d}
    d.Record.Register(record.Type{Code: "sales.order", Product: "sales", Kind: record.Document, Can: s.can})
    d.IAM.RegisterRoles("sales", roles) // extra parameter: roles assignable only tenant-wide
    return s
}

// module.go
func Module(s *Service) platform.Module {
    return platform.Module{
        Name:    "sales",
        Product: "sales",
        Routes:  func(api huma.API) { registerHandlers(api, s) },
        // Middleware: only iam uses it (reads the session cookie).
        // Workers: workers for the module's jobs; Periodic: periodic jobs (system jobs).
    }
}
```

Every job's args implement `Spec() platform.JobSpec{Product, Class, System, Notify}`, and jobs are created only through `platform.Enqueue` (applies the product gate by class, records the job's requester for jobs run on behalf of a user, then inserts in the transaction of `ctx`). A job run on behalf of a user that declares `Notify` notifies the requester when the job finishes or fails; such a job must complete with `platform.CompleteJob`, so the "done" notification commits together with the job. System jobs never send notifications.

A module tells a user about an event with `notification.Send(ctx, kind, ref, users)`, called in the event's transaction. Notifications do not carry the record's content; email (if the tenant has a mail server) goes through the system job `notification.email` enqueued in the same transaction ([ADR-0025](./docs/adr/0025-notifications.md)).

`internal/app` calls each module's `NewService`, then `Module(svc)`; other modules receive that same `*Service` through `Deps`, so there is only one copy of each registry (roles, record types).

`platform.Module` contains only types from `platform` and external libraries, never `core` types (such as `record.Type`, `iam.Role`): `platform` sits below `core`, so containing them would make `platform` import `core` and create an import cycle, which Go cannot compile. Registering with core modules (record types, roles, settings…) is done by calling that core module's own registration functions, received through deps.

`internal/app` constructs **every** module regardless of which products are enabled, passes deps (with adapters when data types differ between two modules), calls setters for the bottom-up calls, and registers the manifests. Enabled products are checked per request (see [platform.md](./platform.md#enabled-products)).

## Isolation

Rules a tool can enforce are enforced by a tool. For rules no tool can enforce, the alternative safeguard is stated:

| Level | Rule | What enforces it |
| --- | --- | --- |
| Code | Other modules cannot touch the store | Go's `internal/` directory: compile error |
| Code | Exports only in `service.go`, the `service_<part>.go` files and `types.go`, plus `Deps` (`deps.go`), `Module` (`module.go`) and hook types (`hooks.go`). Go controls exports per package, not per file, so any exported identifier in another file is also visible from outside | Convention, checked in review |
| Dependencies | The layer direction above; modules in `modules/` do not import each other | `depguard` in golangci-lint, run in CI |
| Cross-product calls | Through interfaces in `deps.go` or `hooks.go`, wired in `internal/app` | If it cannot be imported, it has to go through an interface |
| Database | One Postgres schema per module, named after the module (`sales.orders`, `iam.users`). Module X's `queries.sql` may only INSERT/UPDATE/DELETE into schema X, and only read the permitted schemas ([Rules between modules](#rules-between-modules)) | A CI script scans the writes and reads in `queries.sql`. SQL written outside `queries.sql` is not caught |
| Documents | Call `record` before writing document tables | The `recordtest` contract suite (see [documents.md](./documents.md#guarantees)) |
| Transactions | Services get the DB only through `platform.DBFrom(ctx)` | `depguard` (`no-pool`) forbids importing `pgxpool` in `core`, `shared`, `modules`; `forbidigo` forbids `pgx.Connect`. Package-level DB variables: checked in review |

Dependencies between products are always interfaces, so when testing a module, the other products are replaced with fakes. Dependencies down to `core` and `shared` are direct imports, so tests use the real ones on the same Postgres.

Every tenant's database has the tables of every module, whether or not that product is enabled. Enabled products only decide which requests and hooks run, not which tables exist.

## Org units

`iam` owns the table `iam.org_units (id, parent_id, kind, name, tax_code, legal_name, address)` ([ADR-0010](./docs/adr/0010-org-tree-and-roles.md)). This is the **permission scope tree**: a node is a place where a role can be assigned to a user, and a permission assigned at a node applies to its whole subtree. `kind` is stored as a string; for now `iam` accepts `group`, `company`, `branch`, `department`. The core only understands the meaning of `company` (legal entity) and `group` (grouping node); other kinds are added to the list when a product needs them.

This tree is not a business model. Which store a warehouse belongs to, which department an employee belongs to, which project a site belongs to are relations owned by business modules, in their own tables. Those tables may reference `org_unit_id` when the object needs to fall within a permission scope, but not every object needs to be a node in the tree. For example, a warehouse that needs no permissions of its own does not need to be a node.

**Legal entity** is the only core concept in the tree: units with `kind = company` carry the tax code, legal name and address. One database may hold several legal entities (multi-company model). E-invoices, period locks and the general ledger are all separated by legal entity.

- Each node belongs to exactly one legal entity: its nearest ancestor with `kind = company`. A legal entity never sits under another legal entity; a corporate group has a grouping node as its root, not a legal entity.
- **Every document has exactly one `org_unit_id`**, at the lowest the legal entity node itself. For a business object that is not a node (for example a warehouse), the module attaches the document to the nearest node that is meaningful for permissions.
- `iam.Scope(ctx, product, permission)` returns the units the user is permitted for (already expanded to the whole subtree), computed once per request. A tenant-wide permission is returned as a separate flag, not as a list of every node. Modules filter lists with `org_unit_id = ANY($scope)`. The tree is traversed with a recursive CTE; switch to `ltree` only when it is measured to be slow.
- Moving a node to another branch changes subtree permissions immediately, including permission to view old documents, and is audited. A node that already has documents may not be moved to another legal entity, because a document cannot change legal entity.

## Data ownership

- Each table has exactly one owning module and sits in that module's schema. Only the owning module's service may write to the table.
- No module creates sign-in accounts itself. Business person records (employees, customers…) are owned by business modules and link to `iam.users.id` when needed.
- **Promoting an entity to shared.** When an entity of one product is about to be used by a second product, move it to `shared/` *before* the second product starts using it. This means moving the package and moving the table to a new schema (`ALTER TABLE … SET SCHEMA`); ids and foreign keys stay unchanged.

## Transactions

A single rule:

- A service gets a connection with `platform.DBFrom(ctx)`. If ctx holds a transaction, that transaction is returned; otherwise the pool is returned. A service never keeps the pool (`*pgxpool.Pool`) or a `platform.Conn` in a field, and never creates a sqlc store in its constructor.
- A service opens a transaction with `platform.InTx(ctx, func(ctx) error)`. If ctx already holds a transaction, that transaction is reused. Every service called inside automatically shares the transaction; only `ctx` needs to be passed.
- Never pass a `pgx.Tx`, or a sqlc store bound to a transaction, as a parameter. A helper running in a transaction takes `ctx` and calls `store.New(platform.DBFrom(ctx))` itself; otherwise an audit write inside can easily use the outer ctx by mistake and fall outside the transaction.
- Never use a ctx holding a transaction in another goroutine: a transaction can only be used sequentially, and is no longer valid after commit.
- The audit log of a change is written in the same transaction as the change.

## Rules between modules

- **Writes must go through the owning module's service.**
- **Reads may join with SQL along the dependency direction**, and the query must live in the reading module's `queries.sql`. A module may join its own schema, those of its layer and of the lower layers (`core`, `shared`). A module in `modules/` may also join the schemas of the other modules of its own product and of the products its product has declared as dependencies; joining the schema of an undeclared product fails CI. `scripts/check-queries.mjs` reads the dependency map from the composition root, so the check and the running app cannot disagree, and it fails a module whose product the map does not list. Its own rules are kept honest by `scripts/query-fixtures`, a tree with one planted violation per rule. A cross read is a real dependency, so it must be declared like any other dependency. That way, when the owning module renames or drops a table or column, `make gen` fails on every query that references it. `make gen` cannot detect a change in a column's meaning, so changing a meaning requires a new column.
- **What you need goes in `deps.go`: required dependencies.** A module calling a lower layer (`shared`, `core`) imports it directly. A module calling another product declares an interface in its own `deps.go`; `internal/app` passes the implementation in. **A dependency never has a no-op version**, because a required operation that "succeeds but does nothing" (for example reserving stock) is wrong data. A product that needs another product requires both to be enabled: dependencies between products are declared in code, and the app refuses to start when a dependency product is missing. If only one feature of the product needs the other product, the implementation returns `ErrUnavailable` when the other product is not enabled, and the frontend hides that feature.
- **What you emit goes in `hooks.go`: optional reactions.** When another module needs to react to this module's actions, this module defines a hook interface and calls it at the right moment. `internal/app` wires it. If no module reacts, or the reacting product is not enabled, the hook is a no-op. The emitting module must work correctly when the hook is a no-op.
- **A hook failure does not break the main operation, unless explicitly chosen.** By default, a hook only puts a River job on the queue with `platform.Enqueue` in the same transaction: the job is guaranteed to be written if the main operation succeeds, and a failure while processing the job is retried without rolling back the main operation. Synchronous hooks (whose failure rolls back the whole operation) are used only for invariants that must be strictly consistent, and the reason must be recorded in an ADR.
- **No events.** A hook is a direct call to a known interface, not an event bus.

## Calling the core

A product module reaches the core only through these entry points. A need none of them covers goes into the core first, proven by the product that needs it, rather than into a helper the next product would copy.

| Need | Entry point |
| --- | --- |
| Permission at org units | `iam.Allowed` / `iam.Require(ctx, product, perm, unit, more...)` (every unit must be in scope); lists filter with `iam.Scope`; tenant-wide catalogues use `iam.RequireTenantWide` |
| Who may do what with a record | The record type's `Can`; other modules ask `record.Can`, or `record.Visible` to answer not found |
| Document lifecycle | `record.Create` / `Edit` / `Delete` before the module's own rows, in the same transaction; status changes through `record`'s routes, reacting in `OnTransition` and `BeforeSubmit`; `record.Lock` before writing rows that depend on a document without changing it |
| `allowed_actions` | `record.DocumentActions` for lifecycle actions, `record.AllowedActions` for the rest |
| Writes while a product is disabled | `platform.ProductGate`; `record`, the route middleware, settings, jobs, prints, imports and exports apply it by the registered product |
| Audit | `audit.Record` / `RecordFor` / `RecordChanges` in the change's transaction; `record.RestrictHistory` keeps an entry off a record's timeline for who may not see it |
| Today and settings | `setting.Today`; `setting.GetFor`; a product's legal-entity settings registered with `RegisterLegalEntityKey(product, …)` |
| Errors | `*platform.Error` sentinels with stable codes; `platform.Violates(err, constraint)` maps a constraint to one; any other error answers `internal_error` and is logged |
| Lists | Filters embed `platform.Paging` |
| Jobs, prints, imports, exports | `platform.Enqueue`; `printing.Register`; `dataio.RegisterImport` / `RegisterExport` |
| Attachments, discussion, notifications | Come with the record type; `notification.Send` for a module's own events |
| A disabled product stays shown | Documents count by themselves; a catalog type answers `HasData` |

Registration is checked when the app is composed: a record type is named `<product>.<type>`, a print template takes its product from its record type, an import or export names its product, a product's routes sit under `/<product>/`, and its manifest names a product of the `products` map. Each mistake would otherwise skip the product gate or the frontend's per-product client without a sound.

## Extension and customisation

In order of preference, stop at the first approach that solves the problem:

1. **Tenant configuration**: settings, custom fields, print templates, approval rules (see [documents.md](./documents.md)). The tenant or the implementer does it themselves, with no new release.
2. **Extension table**: a module that needs extra information for another module's entity creates its own table in its own schema, one-to-one by that entity's id. Never add columns to another module's table.
3. **Hook**: a module needs to react to another module's actions.
4. **New module**: new business logic, released to every installation, enabled per product.

**Boundaries for runtime customisation** (custom fields and whatever tenants configure themselves later). No code yet; when it is built it must keep:

- Record types have a stable identifier (`<product>.<type>`, like `doc_type`) that never changes after release; customisations attach to that identifier.
- Extend only through explicit extension points (extension tables, hooks, registration with `core`). UI and configuration never write directly to tables.
- Every write path, including custom field values, goes through the service, `ProductGate` and permission checks just like standard fields.
- Standard business fields are typed from migration to frontend. Tenant-created fields are validated by metadata at runtime and have no compile-time type; accept this boundary, never generate code per tenant.

There is no tenant-specific code. A tenant-specific need that configuration cannot meet has two options: make it a general feature (with a setting to turn it on/off), or decline. Reason: every custom code copy is a branch that must be upgraded separately forever, and with on-premise the number of branches grows with the number of customers.

## Terminology

Terms live in [CONTEXT.md](./CONTEXT.md): one section for the core and one per product. A term used by two products must have the same meaning, or there must be two different names.

## Adding a module

1. Write an ADR in `docs/adr/`, covering:
   - the module's layer and scope;
   - the entities it owns;
   - `deps.go`: what the module needs from other modules;
   - `hooks.go`: what hooks the module emits (synchronous or via job);
   - the record types and shared features enabled;
   - the business lines used for posting.
2. Add terminology entries to `CONTEXT.md`.
3. Migration creating the schema and tables, `queries.sql`, the sqlc entry in `sqlc.yaml`.
4. Write the service, handler and `module.go`. Wire it in `internal/app`, including deps and hooks (a hook no module implements gets a no-op). Add the product's `depguard` rule, the same shape as the others (lax mode: allow the product's own package, deny `internal/modules`), so no existing rule changes. If the module has document types, run the `recordtest` suite for each.
5. Run `make gen`, then build the frontend area (see [frontend.md](./frontend.md#adding-an-area)).
6. Attach the module to a product, and declare which products that product depends on (see [platform.md](./platform.md#enabled-products)).
