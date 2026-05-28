# Architecture Map

Дата аудита: 2026-05-07

## Общая Карта

```text
Wails desktop app
├── main.go
│   ├── loads config/config.json
│   ├── initializes slog/Seq
│   ├── connects PostgreSQL
│   ├── initializes repositories
│   ├── initializes services
│   ├── initializes MinIO
│   └── binds Go services to Wails bridge
│
├── internal/
│   ├── config/        config loading and AES-GCM encrypted secrets
│   ├── database/      PostgreSQL connection, embedded migrations
│   ├── models/        domain entities, request structs, app errors
│   ├── dto/           frontend-facing mapping
│   ├── repository/    SQL persistence and transactions
│   ├── services/      auth, permissions, business workflows, Wails API
│   ├── storage/       MinIO object storage
│   ├── logger/        slog, Seq, Wails adapter
│   └── releaseassets/ embedded release notes
│
├── frontend/src/
│   ├── pages/         screen-level React pages
│   ├── components/    reusable UI and document widgets
│   ├── modules/       document-kind page configs, filters, edit value mapping
│   ├── hooks/         access and document list hooks
│   ├── store/         Zustand auth/draft/register state
│   ├── constants/     document kind/type constants
│   └── theme/         Ant Design theme provider
│
├── frontend/wailsjs/  generated Wails bridge bindings
├── internal/database/migrations/
├── build/             Wails build assets
└── docs/              audit plan and local instructions
```

## Layer Responsibilities

Frontend:

- renders Ant Design pages, forms, tables, modals and filters;
- calls generated `frontend/wailsjs/go/services/*`;
- controls navigation based on access summary from backend;
- performs user-facing form state and date formatting;
- stores short-lived UI state in Zustand.

Wails bridge:

- exposes bound Go service methods to React;
- serializes request/response DTOs;
- formats backend errors through `ErrorFormatter`.

Backend services:

- authentication and current-user state;
- permission checks and document read scope;
- business validation and orchestration;
- journal and admin audit writes;
- migration management and system settings.

Repositories/database:

- SQL queries, inserts, updates and multi-table transactions;
- PostgreSQL constraints, indexes and migration state.

Storage:

- MinIO object upload/download/delete;
- attachment metadata remains in PostgreSQL.

## Observed Boundaries

Good boundaries:

- React calls Wails services; it does not directly access PostgreSQL or MinIO.
- SQL is concentrated in `internal/repository` and `internal/database`.
- Backend services are not coupled to Ant Design components.
- DTO mapping is isolated in `internal/dto`.
- Document-kind command/query registries reduce direct branching in the generic API.

Boundary risks:

- Frontend `resolveUserProfile` mirrors backend `DashboardService.determineDashboardProfile`. It affects UI/release-note behavior and can drift from backend profile classification.
- Frontend contains some workflow decisions such as organization setup sequence and navigation fallback. These do not directly write domain data except through services, but the first-run organization setup performs several service calls sequentially.
- Registration numbering for creates was remediated after stages B/C: idempotency check, nomenclature row lock, number allocation and document create now run in one repository transaction.
- Wails error boundary was remediated on backend side to structured `{code,message,status}`; frontend still needs centralized adapter usage instead of raw `err?.message || String(err)`.
- `DocumentRegistrationService.Register/Update` still has generic public shape, but strict decoding now rejects unknown fields and create DTOs require `idempotencyKey`.
- Long-running backend operations currently use `context.Background()` in several service paths; target: app/request context propagation from Wails lifecycle.
- Some frontend pages are large and mix service calls, forms, modal lifecycle and layout; stage D recommends gradual feature-level decomposition.
- Runtime config currently sits outside embedded assets and is loaded as cwd-relative `config/config.json`; stage E requires an explicit production config placement policy.
- Release/version information is split between embedded release notes and Wails/binary metadata; stage E requires a single version source for release artifacts.
- Security/dependency checks are currently manual; stage F requires a release gate for `govulncheck`, npm audit, license inventory and static analysis.

## Structure and Cleanliness

Structure reflects responsibilities reasonably well:

- `internal/services` owns use cases;
- `internal/repository` owns persistence;
- `frontend/src/modules/documentKinds/*` separates document-kind UI differences;
- `frontend/wailsjs` is generated and should be regenerated after Wails binding changes;
- `internal/mocks` are test doubles.

No obvious unused dependency was found in high-level checks:

- `go mod tidy -diff` returned no diff.
- Go dependencies are referenced in source/tests.
- `npm ls --depth=0` is clean; frontend dependencies are referenced in `frontend/src` or build config.

Potential cleanup candidates for later stages:

- fixed document type CRUD methods remain as non-editable stubs in service/repository interfaces;
- generated mocks include panic text by design and should not be treated as runtime panic paths;
- debug-like output found: `console.debug` in theme fallback, `fmt.Printf` startup DB ping warning, `fmt.Printf` MinIO cleanup warning. These look operational rather than temporary, but logging policy should review them in backend/frontend logging stages.

## Контрольные Пункты

| Код | Статус | Доказательство | Вывод | Уточнить или зафиксировать |
| --- | --- | --- | --- | --- |
| A.03.011 | ok | `main.go`, `internal/*`, `frontend/src/*`, `frontend/wailsjs/*`. | Карта модулей Wails + React + AntD + PostgreSQL составлена. | На этапе B дать этот файл вместе с database migrations. |
| A.03.012 | ok | React imports only Wails services; SQL only in repositories/database. | Границы frontend/backend/database в целом не размыты. | Проверить на этапе D, что новые UI changes не обходят сервисы. |
| A.03.013 | issue | `frontend/src/store/useAuthStore.ts` `resolveUserProfile`; `internal/services/dashboard_service.go` `determineDashboardProfile`. | Есть дублирование классификации профиля пользователя. | Сделать backend access summary единственным источником профиля либо зафиксировать синхронизацию. |
| A.03.014 | ok | Forms call backend command services; permissions and business validation есть в services. | UI не содержит прямой скрытой бизнес-логики, влияющей на PostgreSQL в обход backend. | Отдельно проверить sequential organization setup на атомарность, если это критично. |
| A.03.015 | ok | Backend services use DTO/domain types, no Ant Design imports. | Backend не зависит от деталей Ant Design. | Нет. |
| A.04.016 | ok | Каталоги `internal/{config,database,models,dto,repository,services,storage,logger}` и `frontend/src/{pages,components,modules,hooks,store}`. | Структура каталогов отражает назначение модулей. | Нет. |
| A.04.017 | ok | Go packages mostly singular by layer; frontend document-kind modules use consistent naming. | Имена файлов и пакетов достаточно единообразны. | На этапе code quality можно отдельно решить смешение `department_repository.go` и `*_repo.go`. |
| A.04.018 | not_applicable | High-level scan found no obvious orphan module; fixed document type CRUD is intentionally disabled. | Для этапа A достаточно зафиксировать отсутствие очевидного orphan code; глубокая проверка dead code сознательно перенесена на этап статанализа/code quality. | Выполнить dedicated dead-code/static-analysis review на этапе E/F согласно DECISION-001. |
| A.04.019 | ok | `go mod tidy -diff` no diff; `npm ls --depth=0` clean; dependency imports found. | Явных неиспользуемых зависимостей на уровне manifest нет. | На этапе dependencies выполнить dedicated audit. |
| A.04.020 | ok | `rg` found `console.debug`, `fmt.Printf`, mocks panics; no TODO/FIXME in production code scan. | Явных временных debug-фрагментов не найдено; есть logging-policy кандидаты. | Проверить уровни логирования на backend/frontend stages. |
