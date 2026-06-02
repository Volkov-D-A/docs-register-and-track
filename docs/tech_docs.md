# Техническая документация проекта

Дата обновления: 2026-06-02  
Статус: основной справочник для дальнейшей разработки

## Назначение

`docs-register-and-track` собирает desktop-приложение Wails `docflow` с заголовком окна "Система регистрации документов". Приложение предназначено для регистрации, поиска, просмотра и сопровождения документооборота организации.

Основные доменные зоны:

- регистрация входящих писем, исходящих писем, обращений граждан и приказов;
- номенклатура дел, подразделения, организации-корреспонденты, исполнители резолюций;
- поручения, соисполнители, статусы исполнения и контроль сроков;
- ознакомления пользователей с документами и приказами;
- связи между документами и граф связей;
- вложения документов в MinIO;
- журнал действий по документам и административный аудит;
- статистика по документам, поручениям и системе.

Этот документ фиксирует правила, которые нужно учитывать при любых будущих изменениях. Если код и этот документ расходятся, сначала проверьте актуальные runbook/audit-документы и обновите документацию вместе с изменением поведения.

## Технологический Стек

Backend:

- Go module `github.com/Volkov-D-A/docs-register-and-track`;
- Go `1.26.3`;
- Wails v2.12.0;
- PostgreSQL через `database/sql`, `lib/pq`;
- миграции через `golang-migrate`;
- MinIO через `minio-go`;
- structured logging через `slog` и Seq;
- тесты: Go `testing`, `testify`, `go-sqlmock`.

Frontend:

- React 18;
- TypeScript 6;
- Vite 8;
- Ant Design 6;
- Zustand;
- dayjs;
- `@xyflow/react` для графа связей;
- `@ant-design/plots` для статистики.

Инфраструктура и сборка:

- Wails CLI v2;
- Makefile как основной entrypoint для dev/build/release checks;
- Docker Compose для локальных PostgreSQL, MinIO и Seq;
- Linux `amd64` и Windows `amd64` являются production target. macOS не входит в текущий release target.

## Высокоуровневая Архитектура

```text
Wails desktop app
├── main.go
│   ├── загружает config
│   ├── инициализирует slog/Seq
│   ├── подключает PostgreSQL
│   ├── создает repositories
│   ├── создает services
│   ├── подключает MinIO
│   ├── настраивает lifecycle/shutdown
│   └── bind-ит Go services в Wails bridge
│
├── internal/
│   ├── config/        config loading, encrypted secrets
│   ├── database/      PostgreSQL connection, embedded migrations
│   ├── models/        domain entities, requests, app errors
│   ├── dto/           frontend-facing mapping
│   ├── repository/    SQL persistence and transactions
│   ├── services/      auth, permissions, business workflows, Wails API
│   ├── storage/       MinIO object storage
│   ├── logger/        slog, Seq, Wails adapter
│   ├── startupdiag/   startup diagnostics
│   └── releaseassets/ embedded release notes
│
├── frontend/src/
│   ├── pages/         screen-level pages
│   ├── components/    reusable widgets
│   ├── modules/       document-kind configs and mappers
│   ├── features/      feature-level extracted components
│   ├── hooks/         access/list/modal hooks
│   ├── store/         Zustand state
│   ├── constants/     document constants
│   ├── utils/         frontend helpers
│   └── theme/         Ant Design theme provider
│
├── frontend/wailsjs/  generated Wails bindings
├── docs/              maintained runbooks and technical docs
├── audit/             audit evidence and remediation trace
└── tools/             release/test/evidence tooling
```

## Слой Frontend

Frontend отвечает за:

- отображение страниц, форм, таблиц, модалок и фильтров;
- вызов generated Wails services из `frontend/wailsjs/go/services/*`;
- локальное состояние форм, фильтров, модальных окон и навигации;
- client-side UX guards: loading states, dirty form confirmation, safe error copy;
- layout и доступность действий на основании access summary от backend.

Frontend не должен:

- обращаться к PostgreSQL или MinIO напрямую;
- принимать решения авторизации вместо backend;
- зависеть от raw Go/PostgreSQL/MinIO error text;
- показывать технические детали ошибок пользователю;
- отправлять регистрацию документа без `idempotencyKey`.

Важные frontend-модули:

- `frontend/src/App.tsx` - lazy page loading и route selection;
- `frontend/src/components/DocumentKindPage.tsx` - общий shell для документных разделов;
- `frontend/src/hooks/useDocumentListPage.ts` - общий lifecycle списков документов;
- `frontend/src/modules/documentKinds/*` - конфиги форм, фильтров, колонок и mapping по видам документов;
- `frontend/src/utils/appError.ts` - единая frontend-адаптация structured backend errors;
- `frontend/src/utils/dirtyForm.ts` - подтверждение закрытия dirty forms;
- `frontend/src/features/settings/ReferenceDirectoriesTab.tsx` - вынесенный feature component справочников.

Крупные страницы (`SettingsPage`, `StatisticsPage`, `DocumentViewModal`, `AssignmentsPage`) нужно декомпозировать постепенно при функциональных изменениях. Не делать большой refactor без поведенческой причины и smoke/test coverage.

## Слой Wails Bridge

Wails bridge:

- serializes calls между React и Go;
- exposing происходит через `Bind` в `main.go`;
- frontend использует generated bindings в `frontend/wailsjs`;
- backend errors проходят через `ErrorFormatter`.

Production error envelope для frontend:

```json
{
  "code": "VALIDATION_ERROR",
  "message": "безопасное сообщение",
  "status": 400
}
```

Правило: frontend contract - стабильные `code/status/safe message`, а не `err.Error()` и не текст PostgreSQL/storage.

После изменения public Go service signatures нужно регенерировать Wails bindings и проверить frontend build.

## Слой Backend Services

`internal/services` владеет use cases:

- authentication и current-user state;
- permission checks;
- document access scope;
- validation и orchestration;
- journal/admin audit writes;
- migration/settings operations;
- Wails-facing API.

Правила для services:

- business validation должна быть в backend, frontend validation только помогает UX;
- все domain operations должны проверять authentication/permissions;
- document read scope нельзя подменять frontend state;
- user-facing errors должны быть structured app errors;
- долгие операции должны использовать lifecycle context;
- journal/admin audit writes не должны ломать privacy rules technical logs.

## Слой Repository И Database

`internal/repository` владеет SQL и транзакциями. `internal/database` владеет connection/migrations/schema status.

Правила:

- SQL не должен расползаться в services/frontend;
- multi-table domain changes должны быть атомарными;
- registration number allocation and document create must stay in one transaction;
- constraints and indexes are part of business safety, not optional decoration;
- migrations are embedded and must remain deterministic;
- dirty/newer schema state must block unsafe use.

Критичные migration rules:

- migrations лежат в `internal/database/migrations`;
- runtime UI migration management сохраняется в production для пользователя с `admin`;
- rollback считается destructive operation;
- rollback требует fresh PostgreSQL+MinIO backup, backup reference, data-loss acknowledgment, control phrase and audit entries;
- older binary against newer DB schema must be blocked;
- dirty schema means stop using app and follow recovery procedure.

## Слой Storage

MinIO хранит physical attachment objects. PostgreSQL хранит attachment metadata.

Правила:

- attachment upload/delete must keep PostgreSQL metadata and MinIO object consistent where possible;
- при рассинхронизации восстанавливать PostgreSQL и MinIO только из согласованного backup-набора;
- max production attachment size: 15 MB;
- allowed production extensions: `.pdf,.doc,.docx,.odt,.xls,.xlsx,.ods`;
- attachment downloads to local disk must not overwrite existing files;
- MinIO startup bucket check has timeout;
- file operations must participate in operation lifecycle cancellation.

## Operation Lifecycle И Shutdown

Долгие backend operations используют shared `OperationLifecycle`:

- app root context;
- per-operation timeout;
- shutdown cancel/wait coordination;
- Wails `OnShutdown` сначала отменяет/ждет active operations, затем закрывает DB/logger.

Покрытые зоны:

- attachment upload/download/delete/bulk delete;
- link create/delete/list/graph;
- journal read/write;
- storage statistics;
- document registration command wrapper.

Правило: новые потенциально долгие DB/MinIO/file/statistics operations должны использовать lifecycle-aware context или явно объяснять, почему это не нужно.

## Логирование И Audit Trail

Есть два разных контура:

- technical logs: `slog` + Seq;
- domain audit trail: PostgreSQL `document_journal` и `admin_audit_log`.

Правила:

- Seq обязателен в production как logging service, но его данные не входят в backup;
- Seq logs не являются долговременным audit trail;
- technical logs минимизируют ФИО и business identifiers;
- technical logs используют `app_user_id`, а не ФИО;
- Wails binding errors не должны писать полный raw error text;
- `document_journal` и `admin_audit_log` хранятся весь жизненный цикл проекта и не удаляются приложением.

## Конфигурация И Секреты

Config lookup order:

```text
DOCFLOW_CONFIG_PATH
<executable directory>/config/config.json
<current working directory>/config/config.json
```

Production должен использовать `DOCFLOW_CONFIG_PATH` или executable-relative `config/config.json`. CWD fallback предназначен для local development.

Secrets:

- production secrets never committed;
- `ENCRYPTION_KEY` supplied through approved release environment or restricted `.env`;
- PostgreSQL/MinIO secrets should use `ENC:` encrypted values;
- `ENCRYPTION_KEY` currently embedded through Go ldflags, so release artifacts are sensitive;
- `.env`, `config.json`, CIFS credentials file should be `0600` or strict ACL equivalent;
- generated release evidence and logs must not contain passwords, tokens or full encrypted secret material.

Example configs:

- `.envExample`, `config.example.json`, `docker-compose.yaml` are local/dev only;
- localhost endpoints, disabled TLS and weak sample passwords are not production defaults.

## Business Rules

### Authentication

- Любая доменная операция требует authenticated user.
- После 5 неверных попыток входа аккаунт деактивируется и пишется admin audit entry.
- First-run setup creates `admin` with system permission `admin`, если пользователей еще нет.

### Document Kinds

Фиксированные виды документов:

- `incoming_letter`;
- `outgoing_letter`;
- `citizen_appeal`;
- `administrative_order`.

Document kinds are fixed in code and DB. Перевод в справочник не планируется без отдельного решения.

### Document Types

Фиксированные типы:

- "Письмо";
- "Договор";
- "Акт";
- "Счёт";
- "Запрос";
- "Ответ";
- "Уведомление";
- "Обращение";
- "Приказ".

### Registration Numbering

Критичный инвариант: автоматическая нумерация строго без пропусков.

Правила:

- registration number unique within `(kind, registration_number, year(registration_date))`;
- `nomenclature.next_number` можно увеличивать только как часть успешной регистрации;
- idempotent registration uses backend `idempotency_key`;
- repeated request with same `(created_by, kind, idempotency_key)` returns existing document;
- repeated request must not create duplicate and must not increment `next_number` again.

### Nomenclature

- Номенклатурное дело уникально по `(index, year, kind_code)`.
- Modes:
  - `index_and_number`;
  - `number_only`;
  - `manual_only`.
- В автоматических режимах номер берется из `next_number`.

### Orders

- Приказ активен только если `cancelled_at IS NULL`.
- Неактивный приказ должен иметь `cancelled_at`.
- Приказные связи `order_amends` и `order_cancels` допустимы только между приказами.

### Document Links

- Связь уникальна по `(source_document_id, target_document_id)`.
- Link graph должен фильтровать документы по read access.
- Пользователь не должен видеть связанные документы вне доступного scope.

### Attachments

- Физический файл в MinIO.
- Metadata row in PostgreSQL.
- Upload validates size and extension.
- Delete should remove object and metadata consistently.
- Download-to-disk is collision-safe.

### Journals

Журналируются:

- изменения документов;
- файлы;
- поручения;
- ознакомления;
- административные настройки;
- миграции;
- rollback requests and results.

`document_journal` and `admin_audit_log` are retention-safe and must not cascade-delete through normal app operations.

## Ролевая Модель

В проекте нет отдельной таблицы бизнес-ролей. Источник прав - permission model.

Legacy/UX profile labels:

- `admin`;
- `clerk`;
- `executor`;
- `mixed`.

Эти labels являются UX-классификацией, не источником авторизации.

### System Permissions

- `admin` - управление пользователями, подразделениями, номенклатурой, системными настройками, миграциями, правами доступа;
- `references` - редактирование организаций и исполнителей резолюций;
- `stats_documents` - статистика по документам;
- `stats_assignments` - статистика по поручениям;
- `stats_system` - системная статистика.

### Document Domain Actions

По видам документов:

- `create`;
- `read`;
- `update`;
- `assign`;
- `acknowledge`;
- `upload`;
- `link`;
- `view_journal`.

### Participant Access

`is_document_participant` включает ограниченный participant model. Участник может получать доступ через:

- подразделение/номенклатуру;
- поручение;
- ознакомление.

Правило: backend authorization must be based on system permissions, document permissions and participant scope. Frontend profile labels must not grant access.

## Критичные Данные

Потеря недопустима для:

- users, password hashes, system/document permissions;
- departments and nomenclature bindings;
- nomenclature and `next_number`;
- `documents` and all document detail tables;
- correspondent registrations and resolutions;
- assignments, co-executors, status, reports and deadlines;
- acknowledgments and view/confirm marks;
- administrative orders and acknowledgment people;
- document links;
- attachment metadata and MinIO objects;
- system settings;
- `document_journal`;
- `admin_audit_log`;
- migration state.

## Атомарные Операции

Должны оставаться атомарными:

- first-run setup: migrations, admin user, `admin` permission;
- document registration: idempotency check, number allocation, `next_number`, `documents`, detail table, children, journal;
- document update with child data;
- assignment create/update with co-executors;
- acknowledgment create/update with user list;
- full replacement of user access profile;
- attachment upload: MinIO object plus metadata row;
- attachment delete: metadata row plus object;
- migration apply/rollback;
- backup/restore PostgreSQL plus MinIO from consistent backup set.

## Идемпотентность

Должны быть idempotent or safe to repeat:

- migrations `Up()` when no change;
- MinIO bucket creation/check on startup;
- organization/resolution executor find-or-create;
- document registration by `idempotency_key`;
- saving existing system setting value;
- marking release notes as viewed;
- saving theme;
- fetching lists/cards/statistics;
- backup verification through manual test restore.

## Ошибки И UX Copy

Backend:

- use stable app error codes;
- avoid leaking DB/storage/internal details to frontend;
- map not-found to structured `NOT_FOUND`;
- map conflicts/idempotency to structured conflict behavior;
- keep startup failures operator-readable through startup diagnostics.

Frontend:

- use `formatAppError`/`normalizeAppError`;
- do not show `err?.message || String(err)` raw fallback as primary UX;
- validation/forbidden/not-found/conflict/internal cases must have safe recovery copy;
- dirty long forms must ask before discard;
- destructive confirmations must name entity and consequence.

## UX Terminology

Important current terminology:

- `Тип документа`, not ambiguous "вид документа" where field means document type;
- `Дело` for user-facing nomenclature context;
- `Ответственный исполнитель` for assignment executor;
- `Исполнитель письма` where outgoing-letter executor meaning is specific;
- no user-visible `dirty`;
- no `N/A`; use `Нет данных`;
- success/error messages should name action and entity.

Future UI changes should be checked against `audit/07_ux_texts/TERMS_GLOSSARY.md` if present and current audit UX artifacts.

## Frontend Development Rules

- Keep real behavior in backend services; frontend should orchestrate UI only.
- Prefer existing hooks/components before adding new patterns.
- Keep Ant Design patterns consistent.
- Use generated Wails service bindings.
- Keep document-kind-specific code in `frontend/src/modules/documentKinds/*` or related forms/filters.
- Add feature components under `frontend/src/features/*` when extracting larger pages.
- Use `confirmDiscardFormChanges` for important forms.
- Use `formatAppError` for user-facing errors.
- Keep loading/submitting guards on mutating actions.
- After frontend behavior changes run:

```bash
cd frontend
npm test
npm run build
npm run smoke:prod
```

## Backend Development Rules

- Put SQL in repositories.
- Put business orchestration in services.
- Keep DTO mapping in `internal/dto`.
- Use structured app errors from `internal/models`.
- Use context-aware operations for long-running work.
- Do not log PII/business details in technical logs unless explicitly required.
- Keep journal/admin audit entries for domain history.
- When adding Wails methods, update generated bindings and frontend call sites.
- After backend changes run:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
go vet ./...
```

## Database Development Rules

- New schema changes require migrations in `internal/database/migrations`.
- Migrations must be embedded and compatible with release build.
- For constraints/index changes, add focused tests where practical.
- Do not add performance indexes just because a query has a seq scan on small baseline data.
- Use `docs/db_performance_evidence.md` and `make db-performance-check` for production-like plan evidence.
- Any new performance index needs before/after `EXPLAIN (ANALYZE, BUFFERS)` and write-latency consideration.
- Keep rollback impact explicit for destructive `down` migrations.

## Backup, Restore И Recovery

Backup/restore contract:

- PostgreSQL and MinIO are backed up together;
- Seq logs are excluded;
- RPO: 1 day;
- RTO: 1-2 days;
- retention: 15 days;
- offsite copy is handled by the approved production process.

Scripts:

- `backup_smb_tar.sh`;
- `restore_smb_tar.sh`.

Rules:

- scripts read `.env` from the documented absolute cron/deployment path;
- `.env` must be outside Git and `0600`;
- use `SMB_CREDENTIALS_FILE` where possible;
- password must not appear in process arguments;
- restore must validate PostgreSQL before mirroring MinIO;
- if PostgreSQL restore/validation fails, MinIO restore must not run;
- release requires manual PostgreSQL+MinIO test restore evidence.

## Release And Versioning

Version source:

- `docs/releases.yaml`;
- generated `internal/releaseassets/current_release.yaml`;
- Wails product metadata in `wails.json`.

Release must be from a clean worktree. Before production approval:

- no open critical blockers;
- release gate output attached;
- artifact checksums attached;
- target OS smoke attached;
- backup/restore smoke attached;
- DB performance evidence attached or explicitly accepted;
- UX safety smoke attached or explicitly accepted;
- long-running smoke attached or explicitly accepted;
- clean `git status --short` at tag.

Current release gate:

```bash
make release-gate
```

It runs/checks:

- required `ENCRYPTION_KEY`;
- generated release asset freshness;
- Go tests;
- Go vet;
- disposable PostgreSQL integration tests;
- `govulncheck`;
- `npm ci`;
- frontend lint/test/build;
- production build smoke;
- UX safety checklist validation;
- long-running checklist validation;
- DB performance checklist validation;
- performance baseline generation;
- `npm audit --audit-level=critical`;
- license report and dependency inventories.

## Supported Make Targets

Common targets:

- `make storage-up` - start local PostgreSQL/MinIO/Seq;
- `make storage-down` - stop local services without deleting volumes;
- `make storage-reset` - destructive local reset;
- `make dev` - Wails dev;
- `make release-assets` - generate embedded release assets;
- `make release-assets-check` - verify generated release assets;
- `make go-test`;
- `make go-vet`;
- `make integration-test`;
- `make frontend-ci`;
- `make frontend-build`;
- `make frontend-lint`;
- `make frontend-test`;
- `make frontend-smoke`;
- `make ux-smoke-check`;
- `make long-running-smoke-check`;
- `make db-performance-check`;
- `make performance-baseline`;
- `make security-gate`;
- `make release-gate`;
- `make build-linux`;
- `make build-windows`.

`make integration-test` requires non-production `DOCFLOW_INTEGRATION_ADMIN_DSN`. It creates disposable `docflow_test_*` database and refuses unsafe direct integration DSNs.

## Performance Budgets

Production SLO:

- startup to login: <= 5 seconds;
- main list open/search/filter: <= 2 seconds;
- document registration save: <= 2 seconds typical;
- heavy statistics/report open: <= 5 seconds;
- normal desktop memory: <= 512 MB;
- binary warning threshold: 100 MB;
- route chunk warning threshold: 1.6 MB.

Expected baseline:

- up to 1000 documents/year;
- up to 20 users;
- attachments proportional to documents;
- average file around 3 MB;
- storage around 1 TB;
- storage warning 80%, critical 90%.

Performance evidence:

- `make performance-baseline`;
- `docs/performance_baseline.md`;
- `docs/db_performance_evidence.md`;
- target OS manual timings before release.

## Testing Strategy

Go:

- `go test ./...`;
- focused unit tests in services/repositories/database;
- guarded PostgreSQL integration tests through `tools/integrationtest`;
- database constraints and idempotency covered in release-gated integration path.

Frontend:

- TypeScript compile;
- Node test runner for helper tests;
- production build smoke for Vite `dist`;
- ESLint gate with accepted warning debt.

Release evidence:

- UX safety checklist;
- long-running checklist;
- DB performance evidence;
- performance baseline;
- backup/restore smoke;
- target OS install smoke.

Rule: a change is not production-ready just because local unit tests pass. Release-impacting changes must be reflected in maintained runbooks/checklists when they alter operator behavior, release evidence, or recovery procedure.

## Security And Dependency Rules

- Keep `govulncheck` in release gate.
- Keep `npm audit --audit-level=critical` in release gate.
- Keep license report and dependency inventory in release gate.
- Unknown or disallowed licenses must block release until resolved or explicitly accepted by policy.
- Do not commit secrets.
- Treat release artifacts as sensitive because `ENCRYPTION_KEY` is embedded through ldflags.
- Keep technical logs free of passwords, tokens and full encrypted secret material.

## Install And Runtime Targets

Production targets:

- Linux `amd64`;
- Windows `amd64`.

Windows policy:

- per-machine admin install is accepted;
- installed app must run for ordinary user without elevated app process;
- target OS smoke must verify this.

Linux policy:

- portable binary artifact;
- target OS smoke must verify launch path, config path and ordinary user behavior.

Target OS smoke must include:

- default shortcut/cwd;
- path with spaces and Cyrillic characters;
- missing/invalid config diagnostics;
- DB/MinIO/Seq unavailable diagnostics.

## Known Release State

As of this document:

- no open critical blockers are tracked;
- no open major issues are tracked;
- no postponed minor issues are tracked;
- production approval still requires clean-clone release gate, target OS smoke, backup/restore test restore and release evidence.

Do not interpret "no open issues" as automatic production readiness. The release process is evidence-based.

## Practical Change Checklist

Before starting:

- identify affected layer: frontend, service, repository, storage, migration, release/ops;
- check this document and the relevant runbook;
- prefer existing patterns and local helpers.

Before finishing:

- run focused tests for changed code;
- run `git diff --check`;
- for frontend behavior, run `npm test`, `npm run build`, `npm run smoke:prod`;
- for backend behavior, run `GOCACHE=/tmp/go-build-cache go test ./...`;
- update docs/runbooks if behavior, release evidence, recovery or operator actions changed;
- avoid unrelated refactors.

High-risk changes requiring extra care:

- document registration and numbering;
- permissions/access scope;
- migrations and rollback;
- backup/restore;
- attachment upload/delete/download;
- structured error contract;
- release gates;
- config/secrets;
- technical logging and audit trail.
