# Tests Review

Дата аудита: 2026-05-28
Этап: G. Тесты, производительность, длительная работа

## Общий Вывод

Backend/database unit coverage is reasonably strong: `go test ./...` passes and the codebase has 522 `Test`/`t.Run` markers across services, repositories, database, config, security and release tooling. Отдельно прогнаны PostgreSQL integration tests for document registration idempotency/concurrency and retention-safe FK; they passed against local test DB.
After `ISSUE-039`, `make integration-test` provisions a disposable `docflow_test_*` DB from `DOCFLOW_INTEGRATION_ADMIN_DSN`, runs those PostgreSQL integration tests and is included in `make release-gate`.

Главные gaps G:

- minimal frontend helper tests are present after `ISSUE-038`;
- full browser/Wails UX safety checklist is maintained and release-gated after `ISSUE-043`; target OS execution remains release evidence;
- production build smoke on installed Wails app is not automated;
- performance baseline report generation exists after `ISSUE-041`; target OS timing values must be filled as release evidence;
- long-running/cancellation/progress scenarios are maintained as release smoke after `ISSUE-042`; completed target OS evidence remains required.

## Выполненные Проверки

- `go test ./...`: passed.
- `make integration-test`: added after remediation; requires non-production `DOCFLOW_INTEGRATION_ADMIN_DSN` and refuses unsafe direct `DOCFLOW_INTEGRATION_DSN`.
- `npm test`: added after remediation for frontend error adapter and dirty-form helpers.
- `npm run build`: passed.
- `npm run smoke:prod`: added after remediation for production build asset smoke.

## Контрольные Пункты G

| Код | Статус | Severity | Тип | Сценарий / доказательство | Вывод |
| --- | --- | --- | --- | --- | --- |
| G.01.190 | ok | none | unit/integration | Services/repositories cover auth, access, assignments, links, acknowledgments, registration helper tests. | Backend business invariants partially well-covered. |
| G.01.191 | ok | none | unit | Validation cases exist for password, UUID, assignment status, references, command decode routing. | Basic validation errors covered. |
| G.01.192 | ok | none | unit | Many service tests cover forbidden/unauthenticated paths. | Access tests are broad. |
| G.01.193 | ok | none | integration | Idempotency/no-gaps/concurrency integration tests passed. | Critical transaction invariant covered. |
| G.01.194 | ok | none | database | Retention FK integration exists; after `ISSUE-040`, critical unique/required/FK/dirty-state constraints are covered by `TestDatabaseConstraintsIntegration`. | Keep this test in `make integration-test`. |
| G.01.195 | ok | none | database | Embedded migrations availability tested; fresh migration application covered by integration helper and `make integration-test` release gate. | Keep disposable admin DSN evidence in release artifacts. |
| G.02.196-G.02.199 | partial | minor | frontend | `npm test` covers frontend error adapter and dirty-form helpers without new dependencies. | Broader component tests can be added when a browser/jsdom runner is introduced. |
| G.02.200-G.02.203 | partial | minor | e2e | `npm run smoke:prod` checks production build index/assets; `make ux-smoke-check` validates maintained browser/Wails UX safety scenarios in `docs/ux_safety_smoke.md`. | Keep target OS release smoke evidence for login/navigation/document flows. |
| G.03.204 | fixed | none | performance/db | Synthetic EXPLAIN baseline exists for 1000 docs and fast plans; `make db-performance-check` validates maintained production-like plan evidence checklist. | Attach final production-like DB performance evidence. |
| G.03.205-G.03.210 | ok | none | performance | `make performance-baseline` generates static metrics and target OS manual timing table for startup/save/search/statistics/memory. | Fill Linux/Windows timings in release evidence. |
| G.04.211-G.04.220 | fixed | none | long-running | `make long-running-smoke-check` validates maintained release smoke scenarios for memory, repeated workflows, shutdown cancellation and DB/MinIO outages. | Execute target OS long-running smoke and attach evidence. |

## Обязательные Перед Release

- Backend/security blocker from F (`ISSUE-032`) is fixed; keep `govulncheck` in release gate.
- Keep Go unit tests and disposable PostgreSQL integration tests in release gate.
- Add minimum e2e smoke for production Wails build.
- Fill generated performance baseline report with target OS/build timings.

Связанные issues: fixed: `ISSUE-038`, `ISSUE-039`, `ISSUE-040`, `ISSUE-041`, `ISSUE-042`, `ISSUE-043`.
