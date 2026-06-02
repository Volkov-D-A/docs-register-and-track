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

These tests are part of `make integration-test` after `ISSUE-039`; the runner creates a disposable `docflow_test_*` database and injects `DOCFLOW_INTEGRATION_DSN`.

## Gaps

- Startup failure behavior is not tested as a user-facing diagnostic flow.
- Context cancellation/shutdown lifecycle has unit coverage after `ISSUE-015`; long-running end-to-end cancellation smoke is maintained after `ISSUE-042`.
- Structured frontend/backend error behavior is not tested end-to-end.
- Seq technical user context minimization is covered by `TestTechnicalContextHandlerAddsOnlyUserID`; end-to-end Seq smoke remains release-gate/manual.

Связанные issues: fixed: `ISSUE-015`, `ISSUE-019`, `ISSUE-028`, `ISSUE-039`, `ISSUE-042`.
