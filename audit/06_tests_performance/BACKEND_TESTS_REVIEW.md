# Backend Tests Review

Дата аудита: 2026-05-28
Этап: G.01.190-G.01.193

## Что Хорошо Покрыто

- Auth/profile/password and brute-force paths.
- Access/forbidden paths across references, nomenclature, departments, assignments, acknowledgments and links.
- Assignment status transitions and validation.
- Link graph access filtering.
- Document registration routing and strict decode behavior.
- Release/theme/config helpers.
- Repository CRUD/query behavior via `sqlmock`.

## Critical Invariants

Document registration no-gaps/idempotency/concurrency is now covered by PostgreSQL integration tests:

- repeated idempotency key returns existing document;
- failed create does not advance numbering;
- concurrent registrations produce contiguous numbers;
- concurrent duplicate idempotency key produces one document.

These tests are not part of default `go test ./...` unless `DOCFLOW_INTEGRATION_DSN` is set.

## Gaps

- Startup failure behavior is not tested as a user-facing diagnostic flow.
- Context cancellation/shutdown behavior is not tested.
- Structured frontend/backend error behavior is not tested end-to-end.
- Seq/log redaction is not test-covered.

Связанные issues: `ISSUE-015`, `ISSUE-016`, `ISSUE-019`, `ISSUE-028`, `ISSUE-039`.
