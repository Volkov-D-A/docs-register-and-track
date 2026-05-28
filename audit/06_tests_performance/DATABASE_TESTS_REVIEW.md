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
- retention-safe FK behavior for journals/audit.

## Gaps

- Integration tests are gated by `DOCFLOW_INTEGRATION_DSN` and skipped by default.
- Integration helper resets `public` schema; release pipeline must guarantee it only points to disposable test DB.
- Fresh migration from empty DB is covered by integration helper, but release gate should run it explicitly.
- Broader schema constraints are not covered as integration tests: duplicate registration number conflicts, required fields, FK references for assignments/acknowledgments/attachments, rollback dirty-state handling.

## Рекомендации

- Add safe DSN guard: require database name/prefix like `docflow_test`/`docflow_regression`.
- Add release target that creates/drops disposable DB and runs integration tests.
- Add constraint-focused DB tests for critical unique/FK/not-null invariants.

Связанные issues: `ISSUE-039`, `ISSUE-040`.
