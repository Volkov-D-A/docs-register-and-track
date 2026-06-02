# Tests Review

Дата аудита: 2026-05-28
Этап: G. Тесты, производительность, длительная работа

## Общий Вывод

Backend/database unit coverage is reasonably strong: `go test ./...` passes and the codebase has 522 `Test`/`t.Run` markers across services, repositories, database, config, security and release tooling. Отдельно прогнаны PostgreSQL integration tests for document registration idempotency/concurrency and retention-safe FK; they passed against local test DB.
After `ISSUE-039`, `make integration-test` provisions a disposable `docflow_test_*` DB from `DOCFLOW_INTEGRATION_ADMIN_DSN`, runs those PostgreSQL integration tests and is included in `make release-gate`.

Главные gaps G:

- minimal frontend helper tests are present after `ISSUE-038`;
- full browser/Wails lifecycle smoke remains manual/release-evidence work;
- production build smoke on installed Wails app is not automated;
- performance baseline exists only for PostgreSQL synthetic EXPLAIN, not for Wails startup, React heavy screens, backend operations or memory;
- long-running/cancellation/progress scenarios are not covered by tests.

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
| G.02.200-G.02.203 | partial | minor | e2e | `npm run smoke:prod` checks production build index/assets; full Wails/browser lifecycle remains manual under `ISSUE-043`. | Keep release smoke evidence for login/navigation/document flows. |
| G.03.204 | partial | minor | performance/db | Synthetic EXPLAIN baseline exists for 1000 docs and fast plans. | Repeat on final production-like dataset. |
| G.03.205-G.03.210 | issue | major | performance | No backend/React/startup/save/search timing harness. | Define performance baseline and measure in Wails build. |
| G.04.211-G.04.220 | issue | major | long-running | No memory/leak/cancel/progress tests; lifecycle issue open. | Add long-running smoke suite. |

## Обязательные Перед Release

- Backend/security blocker from F (`ISSUE-032`) is fixed; keep `govulncheck` in release gate.
- Keep Go unit tests and disposable PostgreSQL integration tests in release gate.
- Add minimum e2e smoke for production Wails build.
- Measure performance baseline on target OS/build.

Связанные issues: `ISSUE-041`-`ISSUE-043`; fixed: `ISSUE-038`, `ISSUE-039`, `ISSUE-040`.
