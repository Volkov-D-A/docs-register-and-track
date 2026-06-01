# Tests Review

Дата аудита: 2026-05-28
Этап: G. Тесты, производительность, длительная работа

## Общий Вывод

Backend/database unit coverage is reasonably strong: `go test ./...` passes and the codebase has 522 `Test`/`t.Run` markers across services, repositories, database, config, security and release tooling. Отдельно прогнаны PostgreSQL integration tests for document registration idempotency/concurrency and retention-safe FK; they passed against local test DB.

Главные gaps G:

- frontend unit/component tests are absent;
- e2e tests are absent;
- production build smoke on installed Wails app is not automated;
- performance baseline exists only for PostgreSQL synthetic EXPLAIN, not for Wails startup, React heavy screens, backend operations or memory;
- long-running/cancellation/progress scenarios are not covered by tests.

## Выполненные Проверки

- `go test ./...`: passed.
- `DOCFLOW_INTEGRATION_DSN=... go test ./internal/repository -run 'Test(DocumentRegistration.*Integration|JournalRetentionFKIntegration)' -count=1 -v`: passed.
- `npm run build`: passed on stage F; no frontend tests exist.

## Контрольные Пункты G

| Код | Статус | Severity | Тип | Сценарий / доказательство | Вывод |
| --- | --- | --- | --- | --- | --- |
| G.01.190 | ok | none | unit/integration | Services/repositories cover auth, access, assignments, links, acknowledgments, registration helper tests. | Backend business invariants partially well-covered. |
| G.01.191 | ok | none | unit | Validation cases exist for password, UUID, assignment status, references, command decode routing. | Basic validation errors covered. |
| G.01.192 | ok | none | unit | Many service tests cover forbidden/unauthenticated paths. | Access tests are broad. |
| G.01.193 | ok | none | integration | Idempotency/no-gaps/concurrency integration tests passed. | Critical transaction invariant covered. |
| G.01.194 | issue | major | database | Retention FK integration exists; broad schema constraint integration coverage is limited. | Add DB constraint tests for unique/required/FK critical paths. |
| G.01.195 | issue | major | database | Embedded migrations availability tested; fresh migration application covered locally/integration helper, but not release-gated. | Include fresh DB migration gate in release pipeline. |
| G.02.196-G.02.199 | issue | major | frontend | No Vitest/Jest/RTL/Playwright config found. | Frontend forms/errors/empty/nav not test-covered. |
| G.02.200-G.02.203 | issue | major | e2e | No e2e framework/config found. | Main user lifecycle and production build smoke not automated. |
| G.03.204 | partial | minor | performance/db | Synthetic EXPLAIN baseline exists for 1000 docs and fast plans. | Repeat on final production-like dataset. |
| G.03.205-G.03.210 | issue | major | performance | No backend/React/startup/save/search timing harness. | Define performance baseline and measure in Wails build. |
| G.04.211-G.04.220 | issue | major | long-running | No memory/leak/cancel/progress tests; lifecycle issue open. | Add long-running smoke suite. |

## Обязательные Перед Release

- Backend/security blocker from F (`ISSUE-032`) is fixed; keep `govulncheck` in release gate.
- Add release test gate for Go unit tests and gated PostgreSQL integration tests.
- Add minimum e2e smoke for production Wails build.
- Measure performance baseline on target OS/build.

Связанные issues: `ISSUE-038`-`ISSUE-043`.
