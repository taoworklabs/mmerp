# 0004. The job queue lives in the tenant's database

Status: accepted · 2026-10-05

## Context

Hooks and integrations rely on enqueuing River jobs in the same business transaction (`InsertTx`): a job is written if and only if the main operation commits. Under cloud model B one could choose one queue per tenant database, or one shared queue.

## Decision

River's tables live in each tenant's database, in every run mode. Under model B, the app always runs one River client for each `active` tenant in the registry, never created lazily per request: a tenant with no requests still has retry jobs and periodic jobs to run.

## Consequences

- The same-transaction enqueue guarantee holds in every mode, without an outbox.
- Job payloads need no tenant key, and a job runs with the enabled-products list of the tenant that owns that database.
- Workers and connections grow with the number of active tenants. When the limit is reached, tenants are split across several app instances via the registry. No move to a shared queue without a new ADR that includes an outbox design.
