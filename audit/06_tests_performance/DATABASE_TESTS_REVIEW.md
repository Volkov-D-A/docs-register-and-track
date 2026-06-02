# Database Tests Review

Дата аудита: 2026-05-28
Этап: G.01.194-G.01.195

## Выполненные Проверки

`internal/database/postgres_test.go` checks:

- connection failure handling;
- migration path validation;
- embedded migration count/availability;
- migration status behavior.

`internal/repository/document_registration_integration_test.go` checks:

- idempotency/no-gaps;
- concurrent numbering;
- retention-safe FK behavior for journals/audit;
- duplicate registration number, required idempotency key, assignments/acknowledgments/attachments FK constraints, duplicate acknowledgment users and dirty migration rejection after `ISSUE-040`.

## Gaps

- `ISSUE-039` fixed after audit: `make integration-test` provisions a disposable `docflow_test_*` DB from `DOCFLOW_INTEGRATION_ADMIN_DSN`, runs the critical repository integration tests and drops the DB afterwards.
- Integration tests refuse unsafe direct `DOCFLOW_INTEGRATION_DSN` values whose DB name does not start with `docflow_test` or `docflow_regression`.
- Broader schema constraints covered by `TestDatabaseConstraintsIntegration`; additional future constraint tests can be added for new schema rules as they appear.

## Рекомендации

- Keep constraint-focused DB tests in `make integration-test`.

Связанные issues: none; fixed: `ISSUE-039`, `ISSUE-040`.
