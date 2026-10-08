# 0011. Column encryption

Status: accepted · 2026-10-05

## Context

Sensitive fields (citizen ID number (CCCD), social insurance (BHXH) book, personal tax code, bank account, salary, dependants) must be encrypted in the database, with the key kept outside the database ([platform.md](../../platform.md#personal-data)).

## Decision

- **AES-256-GCM** from the standard library. Values are stored in a `bytea` column: `[1 byte key version][12 byte nonce][ciphertext + tag]`.
- **Keys are read from the `ENCRYPTION_KEYS` environment variable**: `1:<base64 32 bytes>[,2:<base64 32 bytes>…]`. If missing or malformed the app does not start. The key with the highest number is used to encrypt; every key still in the list can decrypt, so key rotation means adding a new key and then gradually re-encrypting. On-premise, the variable is set in the container's env file, with restricted permissions, and backed up separately.
- **The AAD is the column name** (`hrm.employees.national_id`), so an encrypted value from this column cannot be copied into another column.
- The keys live in the request context like the database, set by `internal/app`; modules call `platform.Encrypt(ctx, aad, plaintext)` and `platform.Decrypt(ctx, aad, ciphertext)`.
- **Audit**: `audit.log` adds `doc_type`, `doc_id` (the record the audit row is about, document or catalog), indexed. Field change history (`audit.RecordChanges`) records only the changed fields, as `{"changes": {field: {"old", "new"}}}`; sensitive fields also carry `"sensitive": true`, and `old`/`new` are the JSON of the value, encrypted (AAD `audit.<record type>.<field>`) then base64-encoded. Every read of a sensitive value writes an audit row in the request's transaction, before the data is returned; if the audit write fails, the request fails.

## Consequences

- SQL cannot filter, sort or aggregate on encrypted columns; that is done in the service.
- Losing the key means losing the sensitive data. The handover documentation must state where the key is backed up.
- Encryption comes with no search index; searching by citizen ID number would need a separate hash column.
