# Core architecture

This directory describes the shared core of a modular business-management suite: many products (accounting, sales, inventory, HR…) on one platform and one set of data, where each organisation enables the products it uses. The documents at the root do not describe any industry's business. Each product is a set of modules built on the core and has its own document in `products/`.

| Document | Contents |
| --- | --- |
| [architecture.md](./architecture.md) | Architecture overview in diagrams; the place to start reading |
| [backend.md](./backend.md) | Layers, core modules, module structure and isolation, data ownership, transactions, extension and customisation |
| [frontend.md](./frontend.md) | Areas, manifests, isolation between areas, API calls, cache, permissions, shared document features |
| [ui.md](./ui.md) | UI rules: tokens, page templates, tables, forms, feedback, text and formatting, checklist. **Read before working on a screen** |
| [documents.md](./documents.md) | Record types, document lifecycle, shared features (approval, attachments, print templates, import/export, custom fields), posting |
| [tenancy.md](./tenancy.md) | One database per tenant; on-premise (primary) and cloud deployment; versions and migrations |
| [CONTEXT.md](./CONTEXT.md) | Core and per-product terminology |
| [docs/adr/](./docs/adr/) | Settled architecture decisions |
| [techstack.md](./techstack.md) | Technology stack, i18n (Vietnamese, English), time, testing, tooling, packaging |
| [platform.md](./platform.md) | Identity and permissions, enabled products, operations, personal data, integrations, API |

| Plan | Document |
| --- | --- |
| From core to the first HRM release, then Sales | [roadmap.md](./roadmap.md) |

| Product | Document |
| --- | --- |
| HRM | [products/hrm.md](./products/hrm.md) |
| Sales | [products/sales.md](./products/sales.md) |

## Principles

- **One codebase, one binary.** Each product is a set of modules in `internal/`, not a separate service. An organisation that uses several products still has one sign-in account, one backup, and data already linked together.
- **One database per tenant, on-premise first.** The binary runs on its own on the organisation's server, without depending on the Internet or on external services. A hosted multi-tenant deployment runs the same binary.
- **The core knows no business.** `platform` and the modules in `core/` import no business module and hold no industry concept (sales, HR, inventory…). Adding a product means adding modules, not changing the core. If a product forces a change to the core, the core is missing a general concept; it is then added through an ADR.
- **Document features built once, used everywhere.** Approval, attachments, discussion, print templates, import/export, custom fields and period lock are provided by the `core/` modules. Modules only declare them and never rebuild them (see [documents.md](./documents.md)).
- **Customise through configuration, not per-organisation code.** Every installation runs the same binary. Organisations customise through settings, custom fields, print templates and approval rules. A need that configuration cannot meet becomes a general feature for everyone, or is declined.
- **One shape, isolated by tools.** Every module follows `handler → service → store`, has its own Postgres schema and keeps business logic in the service. Isolation rules that a compiler, linter or CI can check are checked by tools; the rest have contract tests or are stated explicitly as conventions (see [backend.md](./backend.md#isolation)).
- **Money is `bigint` in the currency's minor unit** (for VND, the đồng). Every rounding goes through a single function in `platform`. The rounding rules are in [ADR-0006](./docs/adr/0006-money-rounding.md).
- **Decisions are recorded as ADRs.** Anything hard to reverse (tenancy, module boundaries, shared entities, core changes) needs an ADR in `docs/adr/` before code is written.
- **One product at a time, core first, proven by HRM.** The first product is HRM ([products/hrm.md](./products/hrm.md)). The core is built out fully first, but each core capability is only done when a real HRM flow uses it ([ADR-0023](./docs/adr/0023-build-core-first.md)). The next product starts only when the roadmap says so; Sales is the second ([ADR-0027](./docs/adr/0027-sales-product.md)).

## Not doing

- Microservices, an event bus, or a message queue for modules to message each other. Modules call each other's services directly, or call hooks defined by the other module. The job queue (River) is used only for work that runs outside a request.
- Splitting the frontend app by product. There is only one app (React + Vite); each product is its own area within the app.
- Redis, a broker, or any extra container besides the app and Postgres (see [techstack.md](./techstack.md)).
- Per-organisation code, runtime plugins, user-written scripts, letting organisations create their own tables or record types, a general workflow engine.
