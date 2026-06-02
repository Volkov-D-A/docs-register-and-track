# E2E Tests Review

Дата аудита: 2026-05-28
Этап: G.02.200-G.02.203

## Вывод

After `ISSUE-038`, `npm run smoke:prod` verifies the Vite production build output and referenced assets. After `ISSUE-043`, the full browser/Wails UX safety scenarios are maintained in `docs/ux_safety_smoke.md` and validated by `make ux-smoke-check`; completed target OS execution remains release evidence.

## Required E2E Scenarios Before Release

- First run with valid config and empty DB/settings.
- Admin login and organization setup.
- Create user/department/nomenclature/access.
- Register each document kind.
- Repeat submit/idempotency smoke for document registration.
- Upload/download/delete attachment.
- Create assignment, executor completes, clerk finishes/returns.
- Link documents and inspect document view/link graph.
- Run migration status and blocked/guarded rollback flow.
- Error scenarios: DB down, MinIO down, forbidden action, validation error.

## Test Data Isolation

E2E must run against disposable PostgreSQL/MinIO buckets, never production config. Existing integration tests reset schema; e2e should have explicit env/config isolation and visible safeguards.

Связанные issues: fixed `ISSUE-038`, `ISSUE-039`, `ISSUE-041`, `ISSUE-043`.
