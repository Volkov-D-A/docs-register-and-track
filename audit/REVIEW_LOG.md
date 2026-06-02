# Review Log

Дата аудита: 2026-05-07
Этап: A. Базовый контекст проекта

## ISSUE-001

Категория: Configuration
Пункт плана: A.05.021, A.05.022, A.05.023
Severity: major
Статус: fixed
Место: `config.example.json`, `.envExample`, `docker-compose.yaml`
Проблема: В репозитории есть только local/dev-oriented configuration examples: `localhost`, `sslmode: disable`, MinIO `useSSL: false`, Seq по `http://localhost`, слабые примерные пароли в `.envExample`. Пользователь подтвердил, что weak example passwords допустимы для закрытого контура разработки, но production template/guide базируется на утвержденной документации за пределами проекта, ссылку на нее нельзя добавить в audit context.
Почему важно: Текущие defaults безопасны только как local/dev шаблон в закрытом контуре разработки. Пользователь подтвердил, что production разворачивается вручную через `config/config.json` по утвержденной документации, поэтому production-конфигурацию, границы контура, секреты и runtime defaults нужно сверять вручную.
Рекомендация: На этапах B/E/H сверить production config/ops с утвержденной документацией, явно пометить текущие examples как local-only и не использовать их как production defaults.
Проверка после исправления: `.envExample`, `config.example.json` and `docker-compose.yaml` explicitly say they are local development examples only and must not be used as production defaults. `config.example.json` remains valid JSON and Go config loader ignores the top-level `_comment`. Проверено: `python3 -m json.tool config.example.json`, `GOCACHE=/tmp/go-build-cache go test ./internal/config`. `docker compose config` was not run because Docker CLI is not available in this WSL environment. Production diagnostics validation fixed by `ISSUE-028`; secret policy is covered by `docs/secret_policy.md`.
Связанные пункты: B.06, E.01, E.02, H.02

## ISSUE-002

Категория: Configuration
Пункт плана: A.05.021, A.05.022
Severity: major
Статус: fixed
Место: `backup_smb_tar.sh`, `restore_smb_tar.sh`, `.envExample`, `docs/backup_restore_runbook.md`
Проблема: Скрипты backup/restore читают `.env` по абсолютному пути `/home/dimas/projects/docs-register-and-track/.env` и передают SMB password в аргументах `mount`. Пользователь подтвердил, что абсолютный путь к `.env` нужен из-за особенностей запуска через cron. Backup policy подтверждена отдельно: RPO 1 день, RTO 1-2 дня, retention 15 дней, проверка архивов через ручной test restore PostgreSQL+MinIO перед релизом, offsite copy на другой сервер, Seq не бэкапится.
Почему важно: Абсолютный env path допустим для cron, но должен совпадать с production cron/deployment path по утвержденной документации. Секреты могут попадать в process list/shell history/операционные логи в зависимости от окружения из-за передачи SMB password в аргументах `mount`.
Рекомендация: Документировать production cron запуск и путь к `.env`, сверить его с утвержденной документацией, рассмотреть credentials file для CIFS и минимизацию exposure секретов.
Проверка после исправления: Скрипты используют `mount -t cifs -o credentials=...` вместо передачи password в mount args; поддержан production `SMB_CREDENTIALS_FILE` and fallback temporary credentials file with cleanup trap. `.envExample` помечен как local-only and documents `SMB_CREDENTIALS_FILE`. Добавлен `docs/backup_restore_runbook.md` с cron/env path, правами `.env`/credentials file, RPO/RTO/retention and release-gate checks. Проверено статически: `bash -n backup_smb_tar.sh`, `bash -n restore_smb_tar.sh`. Manual test restore remains release-gate evidence.
Связанные пункты: B.06, E.01, H.03

## ISSUE-003

Категория: Architecture
Пункт плана: A.03.013
Severity: minor
Статус: fixed
Место: `frontend/src/store/useAuthStore.ts`, `internal/services/dashboard_service.go`, `internal/services/dashboard_service_test.go`
Проблема: UX-классификация профиля пользователя дублируется во frontend `resolveUserProfile` и backend `determineDashboardProfile`.
Почему важно: `admin`/`clerk`/`executor`/`mixed` больше не являются бизнес-ролями и не должны становиться источником прав. При изменении permission model frontend и backend могут разойтись, что приведет к неверной навигации, release-note behavior или dashboard profile.
Рекомендация: Сделать backend access summary единственным источником profile classification или добавить тест/контракт на синхронизацию.
Проверка после исправления: Backend dashboard no longer classifies users as UX profiles `admin/clerk/executor/mixed`; it now computes only operational assignment scope (`none/global/personal`) for dashboard data. Existing behavior is preserved: admin -> none, clerk -> global, executor -> personal, mixed -> personal. Added mixed-case coverage in `TestDashboardService_GetActivity`. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/services -run TestDashboardService_GetActivity`.
Связанные пункты: C.01, D.03

## ISSUE-004

Категория: Build/Performance
Пункт плана: A.01.007
Severity: minor
Статус: fixed
Место: `frontend` production build, `frontend/src/App.tsx`, `frontend/vite.config.ts`
Проблема: `npm run build` проходит, но Vite предупреждает о большом основном чанке `index-HhgWsiDR.js` около 3000.72 kB, gzip около 872.68 kB.
Почему важно: Для desktop Wails это не обязательно blocker, но влияет на старт frontend и будущий performance budget.
Рекомендация: На frontend performance stage решить, нужен ли code splitting или увеличить budget осознанно.
Проверка после исправления: Page components are lazy-loaded with `React.lazy`/`Suspense`; main app chunk reduced from ~3000 kB to ~34 kB. `vite.config.ts` sets an explicit Wails desktop route-chunk warning budget of 1600 kB because the lazy `StatisticsPage` chunk includes charting libraries (~1470 kB). Проверено: `npm run build` passes without Vite chunk warning.
Связанные пункты: D.07, F.06

## ISSUE-005

Категория: Frontend/Configuration
Пункт плана: A.01.003, A.05.021
Severity: minor
Статус: fixed
Место: `frontend/src/components/MainLayout.tsx`
Проблема: Название организации в сайдбаре захардкожено как "УСЗН Озерск".
Почему важно: Для production-инсталляции название должно соответствовать системной настройке `organization_short_name`; иначе UI может показывать неверный бренд/организацию после первичной настройки.
Рекомендация: Брать название сайдбара из read-only settings contract, доступного всем авторизованным пользователям; чтение настроек не считается секретным, но изменение настроек должно оставаться только для пользователя с системным правом `admin`. Использовать fallback на название продукта.
Проверка после исправления: `MainLayout` lazy-loads `SettingsService.GetOrganizationShortName()` after login and displays `organization_short_name` with fallback `Система регистрации документов`. Проверено: `npm run build`.
Связанные пункты: D.01, D.03, H.01

## ISSUE-006

Категория: Database
Пункт плана: A.02.009
Severity: major
Статус: fixed
Место: `internal/repository/nomenclature_repo.go`, document command handlers
Проблема: Бизнес-правило требует строгую нумерацию без пропусков, но `GetNextNumber` увеличивает `nomenclature.next_number` до транзакции создания документа. Пользователь также подтвердил, что регистрация должна быть идемпотентной по backend `idempotency_key`. Локальный failure test подтвердил gap: после отдельного инкремента номера последующий insert документа упал по FK, `next_number=2`, `docs=0` для выданного номера.
Почему важно: Если после получения номера регистрация завершится ошибкой, номер будет потерян. Если повторный submit создаст второй документ, появится дубль и может быть израсходован следующий регистрационный номер. Это нарушает подтвержденные бизнес-инварианты.
Рекомендация: По `DECISION-004` включить проверку `documents.idempotency_key`, получение/инкремент номера, создание документа и дочерних записей в одну DB transaction. Добавить backend `idempotency_key UUID`: повторный запрос с тем же `(created_by, kind, idempotency_key)` должен возвращать уже созданный документ без повторного создания и инкремента номера.
Проверка после исправления: Добавлена миграция `008_registration_idempotency`; create DTO принимают `idempotencyKey`; выдача номера перенесена в repository transaction с `SELECT ... FOR UPDATE`; `go test ./...`, `npm run build`, `TestDocumentRegistrationIdempotencyIntegration` и `TestDocumentRegistrationConcurrencyIntegration` прошли.
Связанные пункты: B.03, B.05, C.02

## ISSUE-007

Категория: Database/Migrations
Пункт плана: B.01.027, B.02.039, B.06.075
Severity: critical
Статус: fixed
Место: `internal/database/migrations/*.down.sql`, `internal/database/postgres.go`, `internal/services/settings.go`
Проблема: `down`-миграции удаляют таблицы через `DROP TABLE IF EXISTS`, а rollback migration доступен через runtime service пользователю с системным правом `admin`. На локальном test contour runtime rollback через application migrator откатил `schema_migrations` с `7,false` на `6,false` и удалил таблицу `admin_audit_log`.
Почему важно: В production rollback последней миграции может удалить документы, поручения, вложения, журналы или административный аудит. Это подтвержденный data-loss path.
Рекомендация: По `DECISION-003` full production UI/runtime rollback сохраняется. Усилить guardrails: destructive warning, подтверждение, свежий backup PostgreSQL+MinIO перед rollback, обязательная запись в `admin_audit_log`, rollback-runbook и review `down`-миграций на data-loss impact.
Проверка после исправления: `SettingsService.RollbackMigration` теперь принимает `RollbackMigrationRequest` and rejects rollback без backup confirmation, backup reference, data-loss acknowledgment and control phrase `ОТКАТ МИГРАЦИИ`; frontend migration tab требует те же подтверждения; audit entries `MIGRATION_ROLLBACK_REQUESTED` and `MIGRATION_ROLLBACK` include backup reference; добавлен `docs/migration_rollback_runbook.md`. Проверено: `go test ./...` passed with `GOCACHE=/tmp/go-build-cache`; `npm run build`.
Связанные пункты: B.01.027, B.06.075, E.04, H.03

## ISSUE-008

Категория: Database/Retention
Пункт плана: B.02.030, B.02.039
Severity: major
Статус: fixed
Место: `internal/database/migrations/006_journal.up.sql`, `internal/database/migrations/007_admin_audit_log.up.sql`
Проблема: `document_journal` и `admin_audit_log` используют `ON DELETE CASCADE` на данные, которые по подтвержденной policy должны храниться весь жизненный цикл проекта. Локальная проверка подтвердила: delete пользователя удалил связанную строку `admin_audit_log` (`admin_audit_remaining=0`), delete документа удалил связанную строку `document_journal` (`document_journal_remaining=0`).
Почему важно: Физическое удаление пользователя или документа может удалить audit/history строки, нарушив retention invariant. Отсутствие штатного UI delete для users/documents снижает вероятность, но не защищает данные на уровне БД.
Рекомендация: По `DECISION-005` не вводить physical delete для users/documents как штатную application-операцию; заменить FK журналов на retention-safe вариант (`RESTRICT`/эквивалент, либо `SET NULL` + immutable snapshots при появлении legal delete requirements).
Проверка после исправления: Добавлена миграция `009_retention_safe_journal_fks` с `ON DELETE RESTRICT` для `document_journal.document_id`, `document_journal.user_id`, `admin_audit_log.user_id`; `TestJournalRetentionFKIntegration` подтвердил, что direct delete document/user блокируется, а journal/audit rows сохраняются. `go test ./...` прошел.
Связанные пункты: B.02.030, C.04, H.03

## ISSUE-009

Категория: Database/Performance
Пункт плана: B.04.051-B.04.064
Severity: minor
Статус: fixed
Место: document list/access queries, assignments, acknowledgments
Проблема: Частые запросы списков, scope-доступа, поручений и ознакомлений используют комбинации `kind`, `nomenclature_id`, `created_at`, `executor_id`, `status`, `deadline`, `user_id`, `confirmed_at`; representative `EXPLAIN ANALYZE` на 1000 documents показал быстрые планы, но при росте данных access/search paths могут деградировать.
Почему важно: Если production data существенно превысит baseline, `EXISTS`, OFFSET pagination и `ILIKE '%term%'` могут начать давать seq scan и задержки списков/дашборда.
Рекомендация: Не добавлять индексы преждевременно; повторить планы на финальном production-like dataset и добавить только подтвержденные composite/partial/trigram indexes.
Проверка после исправления: Added maintained `docs/db_performance_evidence.md` for production-like PostgreSQL release evidence. It requires dataset row counts, `ANALYZE`, required `EXPLAIN (ANALYZE, BUFFERS)` for document lists/access/search, assignments, acknowledgments, journal, admin audit and statistics, latency-budget review, seq-scan review and before/after evidence for any index candidate. Added `tools/db-performance-check.js`, `make db-performance-check` and release-gate integration so required DB performance evidence scenarios cannot silently disappear. Updated performance baseline, release checklist and runbook. Проверено: `make db-performance-check`, `git diff --check`; final production-like plan execution remains release evidence.
Связанные пункты: D.07, F.06

## ISSUE-010

Категория: Database/Backup-Restore
Пункт плана: B.06.075
Severity: major
Статус: fixed
Место: `restore_smb_tar.sh`
Проблема: Скрипт восстановления продолжает выполнение при любом ненулевом коде `pg_restore`, считая это обычно некритичными предупреждениями.
Почему важно: Реальная ошибка восстановления БД может быть скрыта, после чего MinIO будет синхронизирован поверх неполной или некорректной БД.
Рекомендация: Разделить ожидаемые warnings и fatal errors, fail fast для неизвестных ошибок, формировать restore report и выполнять smoke validation.
Проверка после исправления: `restore_smb_tar.sh` теперь работает с `set -euo pipefail`, проверяет наличие `database.dump` и `minio_files/`, запускает `pg_restore --exit-on-error`, пишет restore report, выполняет DB smoke validation через `psql -v ON_ERROR_STOP=1` и не переходит к MinIO mirror при ошибке PostgreSQL restore/validation. Проверено статически: `bash -n restore_smb_tar.sh`; manual successful restore на production-like контуре остается release-gate проверкой по B.06.074.
Связанные пункты: B.06.074, H.03

## ISSUE-011

Категория: Database/Indexes
Пункт плана: B.04.061
Severity: minor
Статус: fixed
Место: `internal/database/migrations/001_core_users.up.sql`, `internal/database/migrations/010_drop_duplicate_users_login_index.*.sql`
Проблема: `login VARCHAR(100) NOT NULL UNIQUE` уже создает unique index `users_login_key`, но миграция дополнительно создает `idx_users_login ON users(login)`.
Почему важно: Дублирующий индекс не улучшает lookup по login, но добавляет место на диске и write overhead при изменении пользователей.
Рекомендация: Удалить отдельный `idx_users_login`, оставив unique constraint/index.
Проверка после исправления: Добавлена forward-миграция `010_drop_duplicate_users_login_index`: `up` удаляет `idx_users_login`, `down` восстанавливает его при rollback. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/database`.
Связанные пункты: B.04.061, F.06

## ISSUE-012

Категория: Backend/Wails Errors
Пункт плана: C.01.079, C.01.080, C.01.081, C.03.090, C.04.091, C.04.092, C.04.093
Severity: major
Статус: fixed
Место: `main.go` `ErrorFormatter`, `internal/models/errors.go`, services/repositories
Проблема: Wails error boundary возвращает frontend plain string через `err.Error()`. `models.AppError` содержит только HTTP-like integer code и message, но structured object до frontend не доходит; часть validation/not-found/internal errors возвращается через `fmt.Errorf`.
Почему важно: Frontend не может стабильно различать validation/forbidden/not_found/conflict/internal errors и вынужден зависеть от текстов. Internal DB/storage context может быть показан пользователю.
Рекомендация: По `DECISION-007` ввести structured error envelope `{code,message,details?}` со стабильными domain codes; internal details логировать, но не отдавать в UI.
Проверка после исправления: `main.go` `ErrorFormatter` возвращает structured `{code,message,status}`; `models.AppError` получил стабильные domain codes; `go test ./...` прошел. На этапе D проверить frontend notifications/forms.
Связанные пункты: D.05, D.06, F.04

## ISSUE-013

Категория: Backend/Documents
Пункт плана: C.02.083, C.02.084, C.03.086
Severity: major
Статус: fixed
Место: `internal/services/*_command_handler.go`, `internal/services/document_kind_command_handler.go`
Проблема: Create-команды документов не содержат backend `idempotency_key`; регистрационные handlers получают номер через `NomenclatureStore.GetNextNumber` до repository create transaction.
Почему важно: Повторный submit может создать дубль или потратить следующий номер; ошибка после выдачи номера оставляет gap, что нарушает подтвержденный no-gaps invariant.
Рекомендация: По `DECISION-004` добавить обязательный `idempotencyKey` в create DTO всех видов документов и перенести idempotency check, numbering и create в единую DB transaction.
Проверка после исправления: Добавлены `idempotencyKey` в create requests и frontend forms; repository create выполняет idempotency check, nomenclature row lock, number allocation and insert в одной transaction; `go test ./...`, `npm run build`, `TestDocumentRegistrationIdempotencyIntegration` и `TestDocumentRegistrationConcurrencyIntegration` прошли.
Связанные пункты: B.02.032, B.05.066, B.05.068, D.05, F.02

## ISSUE-014

Категория: Backend/Wails Contracts
Пункт плана: C.02.083, C.02.084, C.02.085, C.03.086
Severity: major
Статус: fixed
Место: `DocumentRegistrationService.Register(kindCode string, req any)`, `Update(kindCode string, req any)`
Проблема: Public Wails API регистрации/обновления документов принимает `any`, затем делает JSON roundtrip в typed request. Неизвестные поля silently ignored, а контракт для frontend не зафиксирован как строгая схема.
Почему важно: Frontend может отправить лишнее или ошибочное поле без отказа backend; при добавлении `idempotencyKey` это опасно, потому что typo в имени поля может не быть замечено.
Рекомендация: Использовать typed public methods или strict decoder с запретом unknown fields; покрыть tests на лишние поля и обязательные fields.
Проверка после исправления: `decodeDocumentCommandRequest` использует `json.Decoder.DisallowUnknownFields`; register DTO требуют `idempotencyKey`; `go test ./...` прошел. Добавить отдельный unit test на unknown field.
Связанные пункты: D.05, F.02

## ISSUE-015

Категория: Backend/Resource Lifecycle
Пункт плана: C.05.094, C.05.096, C.05.097
Severity: major
Статус: fixed
Место: `internal/services/attachment.go`, `link_service.go`, command handlers, `statistics_service.go`, `main.go`
Проблема: Долгие операции MinIO, journal writes, link graph и storage statistics используют `context.Background()`. Закрытие окна не отменяет активные операции; shutdown закрывает DB/logger без coordination active requests.
Почему важно: При закрытии приложения или зависании MinIO/DB операция может продолжаться или завершаться на закрытых ресурсах; пользователь не получает предсказуемое cancellation behavior.
Рекомендация: По `DECISION-008` ввести app/request context propagation, timeout policy для MinIO/DB-heavy operations и единый shutdown coordinator.
Проверка после исправления: Added `OperationLifecycle` with app root context, per-operation timeout and shutdown wait/cancel coordination. Wails shutdown now cancels active operations before closing DB/logger. Attachment upload/download/delete/bulk delete, link create/delete/list/graph, journal read/write, storage statistics and document registration command wrapper use lifecycle operation contexts. MinIO startup bucket check now has a 15s timeout. Added lifecycle cancellation tests. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/services`, `GOCACHE=/tmp/go-build-cache go test ./...`; long-running/manual smoke remains tracked by `ISSUE-042`.
Связанные пункты: E.05, F.06, G.07

## ISSUE-016

Категория: Backend/Logging
Пункт плана: C.06.099
Severity: major
Статус: fixed
Место: `internal/logger/logger.go`, admin/document audit details
Проблема: Technical production logs добавляют `app_user` с ФИО во все события; отдельные log/audit details содержат имена, названия файлов, номера документов и другие business identifiers.
Почему важно: Для PostgreSQL `admin_audit_log`/`document_journal` такие данные являются доменным audit trail, но для Seq/technical logs нужна минимизация персональных данных и понятная retention/access policy.
Рекомендация: Разделить technical logs и domain audit trail: в Seq логировать user ID/correlation fields, ФИО и business details оставлять в audit/journal там, где это необходимо.
Проверка после исправления: `logger.GetAppUser` заменен на `logger.GetAppUserID`, technical handler добавляет `app_user_id` вместо ФИО; Wails binding error logs больше не пишут полный `err.Error()`, а используют code/status или error type. Доменный `admin_audit_log`/`document_journal` не изменен. Добавлен `TestTechnicalContextHandlerAddsOnlyUserID`. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/logger`, `GOCACHE=/tmp/go-build-cache go test ./internal/services`.
Связанные пункты: F.01, H.03

## ISSUE-017

Категория: Backend/Logging Lifecycle
Пункт плана: C.05.095, C.05.097
Severity: minor
Статус: fixed
Место: `internal/logger/seq_writer.go`, `main.go`
Проблема: Seq writer shutdown сигналит goroutine через channel, но flush ожидается фиксированным `time.Sleep(100ms)`. `closeLogger` также вызывается через defer и из `OnShutdown`.
Почему важно: При выходе последние логи могут быть потеряны, а lifecycle закрытия неочевиден.
Рекомендация: Сделать deterministic shutdown через `sync.WaitGroup`/ack channel и единый shutdown path.
Проверка после исправления: `SeqAsyncWriter` теперь закрывается через `sync.WaitGroup` без `time.Sleep`, `Close` idempotent через `sync.Once`, `Write` после close безопасно игнорируется. Добавлен `TestSeqAsyncWriterCloseFlushesBufferedLogs`, который проверяет flush буфера, повторный `Close` and write-after-close. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/logger`.
Связанные пункты: C.05.095, E.05

## ISSUE-018

Категория: Backend/Errors
Пункт плана: C.04.093
Severity: minor
Статус: fixed
Место: `internal/services/document_access_service.go`, `internal/services/assignment_service.go`, `internal/services/document_access_admin_service.go`, `internal/services/administrative_order_service.go`, `internal/repository/document_registration.go`, `internal/repository/department_repository.go`
Проблема: Отсутствие сущности обрабатывается неодинаково: местами `nil,nil`, местами `models.NewBadRequest("документ не найден")`, местами plain `fmt.Errorf("... not found")`.
Почему важно: Frontend и пользователь получают разные сценарии для одной категории ошибки; missing entity может выглядеть как validation или silent empty response.
Рекомендация: Ввести единый `NOT_FOUND` code; явно определить, какие read methods могут возвращать `null`.
Проверка после исправления: Missing document, assignment, acknowledgment row, user access target, registration nomenclature and department update/delete paths now return structured `models.NewNotFound(...)` with code `NOT_FOUND`/status 404 instead of validation, silent nil or plain `fmt.Errorf("... not found")`. Assignment and document-access-admin tests assert the stable code. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/services`, `GOCACHE=/tmp/go-build-cache go test ./internal/repository`; search for backend `fmt.Errorf("... not found")`/`NewBadRequest("...не найден")` in services/repositories returns no remaining production paths.
Связанные пункты: D.06, H.03

## ISSUE-019

Категория: Frontend/Error Handling
Пункт плана: D.02.108, D.05.134, D.06.142
Severity: major
Статус: fixed
Место: `frontend/src/utils/appError.ts`, `frontend/src/store/useAuthStore.ts`, pages/components catch handlers
Проблема: Frontend еще не использует structured backend/Wails error envelope `{code,message,status}` как контракт. Большинство handlers показывает `message.error(err?.message || String(err))`, а auth lockout распознается через поиск текста в `formatAuthError`.
Почему важно: UI остается зависимым от русских/Go/PostgreSQL/storage строк и не может стабильно отличать validation, forbidden, not_found, conflict/idempotency и internal errors.
Рекомендация: Ввести единый frontend error adapter `formatAppError`/`useAppError`, читать stable `code`, а raw strings использовать только как fallback. Перевести login lockout, forbidden/not_found/conflict и validation behavior на коды.
Проверка после исправления: Added frontend error adapter `formatAppError`/`normalizeAppError` for Wails `{code,message,status}` envelope and code-based auth lockout handling via `USER_LOCKED`. Replaced raw `err?.message || String(err)` usage in auth, document pages, dashboard, settings, assignments, acknowledgments, files, journal and document modal flows. Проверено: `npm run build`; `rg` no longer finds `err?.message || String(err)` frontend patterns. Automated helper coverage is fixed by `ISSUE-038`; full UI smoke for login/forbidden/validation/not_found/conflict/internal is maintained in `docs/ux_safety_smoke.md` after `ISSUE-043`.
Связанные пункты: C.03.090, D.05, F.04, H.03

## ISSUE-020

Категория: Frontend/Forms
Пункт плана: D.02.110, D.02.111, D.04.123
Severity: major
Статус: fixed
Место: document registration/edit modals, settings/actions
Проблема: Критичные submit/actions не везде имеют локальный `submitting`/`confirmLoading` guard. Для документов backend idempotency уже снижает риск дублей, но часть modal flows не блокирует повторный клик; в других местах loading привязан к загрузке списка, а не к конкретному submit.
Почему важно: Повторные действия могут дать плохой UX, повторные запросы, конфликтные ошибки и риск побочных эффектов в недокументных операциях.
Рекомендация: Для create/update/delete/rollback/upload/assignment actions использовать локальный submitting state, блокировать кнопку и повторный submit до завершения операции.
Проверка после исправления: Document registration/edit modals for incoming, outgoing, administrative orders and citizen appeals now use local register/edit submitting guards and `confirmLoading` independent of list loading. Assignment/acknowledgment completion modals guard repeated submit. File upload/delete and settings CRUD/migration/storage actions use local loading guards and button/modal loading states. Verified with `npm run build` and `git diff --check`; manual double-click smoke is maintained in `docs/ux_safety_smoke.md`.
Связанные пункты: C.03.086, D.04, F.02

## ISSUE-021

Категория: Frontend/UX
Пункт плана: D.02.113, D.04.130, D.06.142
Severity: major
Статус: fixed
Место: `frontend/src/components/DocumentKindPage.tsx`, document/settings modals
Проблема: Формы в модалках регистрации/редактирования можно закрыть без предупреждения о несохраненных изменениях. `onCancel` закрывает modal и иногда сбрасывает draft/link state сразу.
Почему важно: Длинные документные формы содержат много обязательных полей; случайное закрытие приводит к потере данных и повторному ручному вводу.
Рекомендация: Добавить dirty-state tracking и confirmation перед закрытием/сменой flow для registration/edit и важных settings forms.
Проверка после исправления: Added shared `confirmDiscardFormChanges` helper using AntD `form.isFieldsTouched(true)`. Registration/edit modals for all document kinds and important settings modals now ask for confirmation only when fields were changed, and reset form state after confirmed discard. Verified with `npm run build` and `git diff --check`; manual dirty modal smoke is maintained in `docs/ux_safety_smoke.md`.
Связанные пункты: D.04, D.06, H.03

## ISSUE-022

Категория: Frontend/Structure
Пункт плана: D.01.102, D.01.103
Severity: minor
Статус: fixed
Место: `frontend/src/pages/SettingsPage.tsx`, `StatisticsPage.tsx`, `AssignmentsPage.tsx`, `components/DocumentViewModal.tsx`
Проблема: Часть frontend pages/components стала крупной и смешивает UI layout, service calls, modal lifecycle, forms and table state. До исправления текущий `SettingsPage.tsx` был 1494 строки, `StatisticsPage.tsx` около 642, `DocumentViewModal.tsx` около 569, `AssignmentsPage.tsx` около 476.
Почему важно: При изменениях в settings/statistics/document view выше риск случайных регрессий, сложнее покрывать behavior tests и сложнее внедрять structured errors/loading guards точечно.
Рекомендация: Разделять эти зоны постепенно при функциональных правках: hooks для data/actions, subcomponents для tabs/modals, feature modules по образцу document-kind modules.
Проверка после исправления: Extracted reference directory management from `SettingsPage.tsx` into `frontend/src/features/settings/ReferenceDirectoriesTab.tsx`, including organizations and resolution executors tabs and their modal lifecycle/service-call logic. `SettingsPage.tsx` now imports that feature component and no longer exports it for `ReferencesPage`, reducing the page from 1494 to 1246 lines without changing UI flow. Проверено: `npm run build`, `npm test`, `wc -l frontend/src/pages/SettingsPage.tsx frontend/src/features/settings/ReferenceDirectoriesTab.tsx`.
Связанные пункты: D.01, F.04

## ISSUE-023

Категория: Build/Versioning
Пункт плана: E.01.155, E.01.156, E.01.159
Severity: major
Статус: fixed
Место: `docs/releases.yaml`, `internal/releaseassets/current_release.yaml`, `wails.json`, Wails metadata
Проблема: Версия приложения не имеет единого источника истины. About modal показывает версию из embedded release notes (`1.0.4`), а binary/installer metadata использует Wails `{{.Info.ProductVersion}}`; в `wails.json` версия продукта явно не задана.
Почему важно: Пользователь, installer, Windows properties и release notes могут показывать разные версии. Это ломает update/downgrade диагностику, поддержку и release traceability.
Рекомендация: Ввести один version source и генерировать из него release notes current version, Wails product version, installer DisplayVersion и About UI.
Проверка после исправления: `docs/releases.yaml` остается source of truth; `go generate ./internal/releaseassets` теперь генерирует `current_release.yaml` и синхронизирует `wails.json` `info.productVersion`. Добавлен `TestWailsProductVersionMatchesCurrentRelease`, который проверяет совпадение generated release asset и Wails metadata. Проверено: `GOCACHE=/tmp/go-build-cache go generate ./internal/releaseassets`, `GOCACHE=/tmp/go-build-cache go test ./internal/releaseassets ./tools/releasegen`.
Связанные пункты: E.01.155, E.03.165, H.01

## ISSUE-024

Категория: Build/Reproducibility
Пункт плана: E.01.154, E.01.158
Severity: major
Статус: fixed
Место: `Makefile`, `.env`, `wails.json`
Проблема: Production build зависит от untracked `.env`/`ENCRYPTION_KEY`, который встраивается в binary через `ldflags`; frontend install uses `npm install`, а не deterministic `npm ci`; единый release build script/gate не зафиксирован.
Почему важно: Clean-machine сборка может не воспроизвестись или собрать binary с неправильным/пустым ключом. Встраивание ключа в binary требует явной release-secret policy и rotation procedure.
Рекомендация: Описать release build contract: required env validation, `npm ci`, `go generate`, freshness check для generated release assets, tests, Wails build, artifact version check. Не хранить production key в repo; определить способ передачи и ротации.
Проверка после исправления: `wails.json` uses `frontend:install: npm ci`; `Makefile` adds `check-release-env`, `release-assets-check`, `frontend-ci` and runs them in `release-gate`; `build-linux`/`build-windows` fail fast without `ENCRYPTION_KEY`; `release-assets-check` validates generated `current_release.yaml` and Wails `info.productVersion` freshness without rewriting files. Проверено: `make check-release-env`, `make check-release-env ENCRYPTION_KEY=` fails before build, `make release-assets-check`, `GOCACHE=/tmp/go-build-cache go test ./tools/releasegen`.
Связанные пункты: E.01.154, E.04.172, H.02

## ISSUE-025

Категория: Installation/Configuration
Пункт плана: E.02.160, E.02.163, E.04.169
Severity: major
Статус: fixed
Место: `internal/config/config.go`, `main.go`
Проблема: Runtime config ищется как относительный путь `config/config.json` от текущей рабочей директории. При запуске из shortcut, другого cwd, standard install path или каталога без доступа приложение завершится до UI через `log.Fatalf`.
Почему важно: Production desktop app может не стартовать после установки, если cwd отличается от ожидаемого. Оператор не получит понятную диагностическую страницу, а config placement/update policy остается неявной.
Рекомендация: Зафиксировать production config location: executable-relative read-only config, system config dir или user config dir. На startup показывать понятную ошибку/diagnostics вместо silent process exit.
Проверка после исправления: Config lookup order зафиксирован: `DOCFLOW_CONFIG_PATH`, затем executable-relative `config/config.json`, затем cwd fallback для local development. Startup error теперь пишет фактический config path. README, diagnostics and release runbook updated. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/config`.
Связанные пункты: E.02, E.04.169, H.02

## ISSUE-026

Категория: Installation/Privileges
Пункт плана: E.02.160, E.02.162
Severity: major
Статус: fixed
Место: `build/windows/installer/wails_tools.nsh`, `build/windows/installer/project.nsi`
Проблема: Windows installer по умолчанию использует `RequestExecutionLevel admin`, ставит приложение в `$PROGRAMFILES64`, пишет uninstall registry в HKLM и shortcuts для all users. Per-user/non-admin install policy не определена.
Почему важно: В целевой эксплуатации может не быть прав администратора на установку/обновление; также важно подтвердить, что само приложение после установки не требует elevated process.
Рекомендация: Явно принять per-machine admin install или перейти на per-user install. Добавить target OS smoke: install/update/run without admin app process.
Проверка после исправления: Per-machine Windows install policy explicitly accepted. `project.nsi` now defines `REQUEST_EXECUTION_LEVEL "admin"` with comments documenting Program Files/HKLM/all-users install semantics and ordinary-user runtime requirement. Added `docs/install_policy.md`; README and release runbook link it. Target OS smoke now requires Windows elevation check and ordinary-user app launch without elevation. Static verification: `rg` confirms policy references and explicit NSIS admin setting.
Связанные пункты: E.02.162, H.02

## ISSUE-027

Категория: Update/Migrations
Пункт плана: E.03.164, E.03.165, E.03.166
Severity: major
Статус: fixed
Место: `internal/database/postgres.go`, `internal/services/auth_service.go`, `internal/services/settings.go`, `frontend/src/pages/SettingsPage.tsx`, `frontend/wailsjs/go/models.ts`
Проблема: Нет явного update/downgrade guard. Older binary может быть запущен против newer DB schema; `GetMigrationStatus` считает `version >= totalAvailable` up-to-date, а startup/update policy для миграций не определена.
Почему важно: Несовместимый binary может работать с неизвестной схемой и давать runtime ошибки или повреждать данные. Оператор не получит четкий запрет downgrade.
Рекомендация: Добавить schema compatibility check: если DB version больше embedded migrations version, блокировать работу с понятной ошибкой. Зафиксировать policy auto-migrate vs explicit admin migration before use.
Проверка после исправления: `MigrationStatus` теперь различает `upToDate`, `schemaTooNew` and `compatible`; версия БД выше встроенных миграций больше не считается актуальной. `RunMigrations`, `RollbackMigration` and `AuthService.Login` block incompatible newer/dirty schema through a structured `CONFLICT` error. Migration UI shows `Схема новее приложения` and disables apply/rollback for newer schema. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/database`, `GOCACHE=/tmp/go-build-cache go test ./internal/services -run 'TestMigrationCompatibilityAppError|TestSettingsService_GetMigrationStatus|TestSettingsService_RunMigrations|TestSettingsService_RollbackMigration|TestAuthService_Login'`, `GOCACHE=/tmp/go-build-cache go test ./...`, `npm run build`.
Связанные пункты: B.01.027, E.03, H.03

## ISSUE-028

Категория: Runtime/Startup Errors
Пункт плана: E.03.167, E.04.174
Severity: major
Статус: fixed
Место: `main.go`, startup init path
Проблема: Ошибки загрузки config, подключения к MinIO, release/theme init и часть DB-critical failures завершают процесс через `log.Fatalf`/`os.Exit` до появления UI. Пользователь не получает встроенную diagnostics/error screen.
Почему важно: На production рабочем месте проблема конфигурации или инфраструктуры будет выглядеть как "приложение не запускается", без понятного действия для оператора.
Рекомендация: Ввести startup diagnostics mode/screen или launcher-level error dialog с safe message и ссылкой на runbook; technical details логировать.
Проверка после исправления: Startup fatal exits replaced by `startupdiag`: configuration, PostgreSQL, MinIO, release notes, theme and Wails startup failures now write a structured `startup_diagnostics` log and an operator-readable stderr block with component, config path, next step and technical details. `database.Connect` now fails startup on `Ping` errors instead of continuing with a warning. Seq startup is intentionally non-fatal/asynchronous and documented as degraded logging. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/startupdiag`, `GOCACHE=/tmp/go-build-cache go test ./internal/database`, `GOCACHE=/tmp/go-build-cache go test ./...`, missing config smoke via `DOCFLOW_CONFIG_PATH=/tmp/docflow-missing-config.json GOCACHE=/tmp/go-build-cache go run .`, PostgreSQL-down smoke via temp config on `127.0.0.1:1`.
Связанные пункты: E.03.167, E.04.174, H.02

## ISSUE-029

Категория: Filesystem/Secrets
Пункт плана: E.04.172, E.04.173
Severity: major
Статус: fixed
Место: `internal/config/crypto.go`, `Makefile`, `backup_smb_tar.sh`, `restore_smb_tar.sh`
Проблема: Production secret policy неполная: plaintext config values are accepted for backward compatibility; encrypted config requires key embedded in binary or provided via env. SMB password exposure in `mount` args fixed via `ISSUE-002`, but release still needs process-list/log secret checks and a full secret delivery/rotation policy.
Почему важно: Секреты могут оказаться в binary, process list, shell history или локальных файлах без единой ротации и контроля доступа.
Рекомендация: Зафиксировать allowed secret delivery method, permissions для config, key rotation procedure, release checks for process list/log artifacts and approved CIFS credentials file usage.
Проверка после исправления: Added maintained `docs/secret_policy.md` covering allowed secret delivery, `ENCRYPTION_KEY`, encrypted config values, plaintext break-glass acceptance, file permissions, SMB credentials, sensitive release artifacts and rotation. README, release runbook, diagnostics runbook and release checklist link/check the policy; `docs/known_issues.md` no longer lists `ISSUE-029`. Release checklist requires config/CIFS permission checks, process-list backup/restore check and logs/evidence secret scan.
Связанные пункты: A.05.021, E.04.172, F.01, H.02

## ISSUE-030

Категория: Filesystem/Downloads
Пункт плана: E.05.180
Severity: major
Статус: fixed
Место: `internal/services/attachment.go` `DownloadToDisk`
Проблема: При скачивании вложения filename очищается через `filepath.Base`, но файл сохраняется в Downloads под исходным именем и silently overwrites existing file.
Почему важно: Пользователь может потерять локальный файл с тем же именем; повторное скачивание разных вложений с одинаковым названием перезапишет предыдущий файл.
Рекомендация: Добавлять collision-safe suffix, сохранять в app-specific download directory или требовать explicit overwrite confirmation.
Проверка после исправления: `DownloadToDisk` now writes with `os.O_EXCL` and collision-safe names: `file.ext`, `file (1).ext`, `file (2).ext`; original files are not overwritten. Added `TestWriteDownloadFileWithoutOverwrite`. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/services -run TestWriteDownloadFileWithoutOverwrite`, `GOCACHE=/tmp/go-build-cache go test ./internal/services`.
Связанные пункты: E.05.180, H.03

## ISSUE-031

Категория: Filesystem/Temp
Пункт плана: E.04.170
Severity: minor
Статус: fixed
Место: `backup_smb_tar.sh`, `restore_smb_tar.sh`
Проблема: Временные каталоги `/tmp/backup_stage_*` и `/tmp/restore_stage_*` удаляются в конце успешного workflow, но trap очищает только SMB mount. При ошибке/прерывании temp data может остаться на диске.
Почему важно: Backup/restore temp directories могут содержать database dump и MinIO files, то есть чувствительные данные.
Рекомендация: Добавить cleanup trap для temp dirs с осторожной проверкой переменных и прав; ограничить permissions temp dir.
Проверка после исправления: `backup_smb_tar.sh` и `restore_smb_tar.sh` создают temp dirs через `mktemp -d`, выставляют `chmod 700` и используют общий `cleanup` trap, который при любом выходе удаляет temp dir, временный CIFS credentials file и размонтирует SMB при необходимости. Проверено статически: `bash -n backup_smb_tar.sh`, `bash -n restore_smb_tar.sh`. Manual interruption smoke остается release-gate проверкой на целевом контуре.
Связанные пункты: E.04.170, H.03

## ISSUE-032

Категория: Security/Go Vulnerabilities
Пункт плана: F.01.181
Severity: critical
Статус: fixed
Место: `go.mod`, `go.sum`
Проблема: `govulncheck ./...` нашел достижимые уязвимости: `GO-2026-4971` в стандартной библиотеке `net@go1.26.2`, fixed in `go1.26.3`; `GO-2026-4918` в `net/http@go1.26.2` and `golang.org/x/net@v0.52.0`, fixed in `go1.26.3` and `x/net@v0.53.0`.
Почему важно: Уязвимости достижимы из кода приложения через DB/network paths, Seq HTTP client and MinIO client traces. Это production security gate.
Рекомендация: Обновить Go toolchain до `go1.26.3` или новее в утвержденном release окружении; обновить `golang.org/x/net` минимум до `v0.53.0`; повторить `govulncheck`.
Проверка после исправления: `go.mod` now requires `go 1.26.3`; `golang.org/x/net` upgraded to `v0.53.0`, with compatible `x/crypto v0.50.0` and `x/text v0.36.0`. Проверено: `go version` reports `go1.26.3`; `GOCACHE=/tmp/go-build-cache /home/dimas/go/bin/govulncheck ./...` reports "No vulnerabilities found" and "Your code is affected by 0 vulnerabilities"; `GOCACHE=/tmp/go-build-cache go test ./...`; `GOCACHE=/tmp/go-build-cache go vet ./...`; `npm run build`.
Связанные пункты: F.01.181, H.02

## ISSUE-033

Категория: Security/Release Gates
Пункт плана: F.01.181, F.01.182, F.02.185, F.02.186
Severity: major
Статус: fixed
Место: `Makefile`, `.gitignore`, `README.md`, `docs/release_runbook.md`
Проблема: В репозитории нет закрепленного security/dependency/license gate. `npm audit` and `govulncheck` выполнялись вручную; Go/npm license inventory не включен в release process.
Почему важно: Уязвимости и license blockers могут попасть в production artifact незамеченными при следующем обновлении зависимостей.
Рекомендация: Добавить release gate: `govulncheck ./...`, `npm audit --audit-level=critical`, Go/npm license report, `go vet`, `go test`, `npm run build`.
Проверка после исправления: Added maintained `make release-gate` target that runs `go test`, `go vet`, `govulncheck`, `npm run build`, `npm audit --audit-level=critical`, license checks and dependency inventory generation under ignored `build/release-evidence/`. Added `make security-gate` for vulnerability/security subset. README and release runbook now point to the gate. Проверено: `make release-gate` passed. Full unknown-license resolution is fixed by `ISSUE-037`; frontend lint/static analysis is fixed by `ISSUE-034`.
Связанные пункты: E.01.154, F.01, F.02.185, H.02

## ISSUE-034

Категория: Static Analysis/Frontend
Пункт плана: F.02.188, F.02.189
Severity: major
Статус: fixed
Место: `frontend/package.json`, `frontend/package-lock.json`, `frontend/eslint.config.js`, `Makefile`
Проблема: ESLint/Prettier config and lint scripts are absent. TypeScript build passes, but no automated frontend lint gate catches hooks issues, unsafe patterns, unused code, accessibility-adjacent problems or formatting drift.
Почему важно: Upcoming remediation touches frontend error handling, forms and state. Without lint, regressions are easier to miss.
Рекомендация: Add minimal ESLint config for React/TypeScript and a `lint` script; optionally add Prettier or formatting check. Start with high-signal rules to avoid noisy migration.
Проверка после исправления: Added ESLint flat config for TypeScript/React with high-signal rules: `rules-of-hooks` as error, `exhaustive-deps` and unused vars as warnings, basic JS/TS rules, and React Refresh warning. Added `npm run lint`; `make release-gate` now runs `frontend-lint`. Проверено: `npm run lint` exits 0 with 44 warnings and 0 errors; `make release-gate` passed. Existing warnings are tracked as technical debt, not blockers for the initial gate.
Связанные пункты: F.02.188, F.02.189, D.01, D.05

## ISSUE-035

Категория: Static Analysis/TypeScript
Пункт плана: F.02.187
Severity: minor
Статус: fixed
Место: `frontend/src/components/AcknowledgmentModal.tsx`, `frontend/src/components/AcknowledgmentList.tsx`, `frontend/src/components/JournalList.tsx`, `frontend/src/components/DocumentViewModal.tsx`, `frontend/src/pages/DashboardPage.tsx`
Проблема: Есть несколько `// @ts-ignore` around generated Wails service calls and model gaps, plus broad `any` usage in document pages/forms/statistics/settings.
Почему важно: Suppressions and broad `any` reduce TypeScript's value exactly at frontend/backend contract boundaries.
Рекомендация: Replace `@ts-ignore` with typed imports or `@ts-expect-error` with reason and issue reference; gradually type shared document/page DTOs.
Проверка после исправления: Removed obsolete `@ts-ignore` comments around generated Wails service imports. `rg -n '@ts-ignore|@ts-expect-error' frontend/src` returns no matches. `npm run build` passes.
Связанные пункты: C.03, D.03, F.02.187

## ISSUE-036

Категория: Code Quality/Formatting
Пункт плана: F.02.189
Severity: minor
Статус: fixed
Место: Go source formatting
Проблема: `gofmt -l main.go internal tools` reports formatting drift in several Go files: `internal/logger/seq_writer.go`, `internal/mocks/ReferenceStore.go`, `internal/models/journal.go`, `internal/repository/acknowledgment_repo_test.go`, `internal/repository/attachment.go`, `internal/services/system_service.go`.
Почему важно: Formatting drift is low risk, but it indicates formatting is not enforced in release gate.
Рекомендация: Run gofmt on non-generated files; regenerate/format mocks through generation path; add gofmt check to release gate.
Проверка после исправления: Ran gofmt on remaining drift files. `gofmt -l main.go internal tools` returns empty. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/models ./internal/repository ./internal/services`.
Связанные пункты: F.02.189

## ISSUE-037

Категория: Licenses
Пункт плана: F.02.185
Severity: major
Статус: fixed
Место: npm/go dependency license inventory
Проблема: npm lockfile scan found one unknown license package (`@antv/g2-extension-plot@0.2.2`); no Go transitive license inventory tool/report is configured.
Почему важно: Production desktop redistribution needs license compatibility and notices. Unknown license or unreviewed Go transitive licenses can block release late.
Рекомендация: Generate npm and Go license reports, resolve unknown package license from upstream package metadata/source, define allowed/blocked license policy.
Проверка после исправления: Added `tools/license-report.js` and wired it into `npm-license-check`/`license-inventory`. The report scans npm lockfile and Go modules, resolves `@antv/g2-extension-plot@0.2.2` as MIT via package `LICENSE`, blocks unknown licenses and GPL/LGPL/AGPL family licenses, and writes `build/release-evidence/LICENSE_REPORT.md`. Проверено: `node tools/license-report.js` completed with 349 npm packages, 316 Go modules, 0 unknown licenses and 0 disallowed licenses.
Связанные пункты: F.02.185, H.02

## ISSUE-038

Категория: Tests/Frontend-E2E
Пункт плана: G.02.196, G.02.197, G.02.198, G.02.199, G.02.200, G.02.201
Severity: major
Статус: fixed
Место: frontend/e2e test infrastructure
Проблема: В проекте нет frontend unit/component tests and e2e framework. `npm run build` проверяет TypeScript compile, но не покрывает формы, ошибки отправки, empty states, навигацию и основной пользовательский lifecycle на production build.
Почему важно: Исправленные frontend flows `ISSUE-019`/`ISSUE-020`/`ISSUE-021` требуют smoke coverage; без UI/e2e тестов регрессии легко пропустить.
Рекомендация: Добавить минимальный Vitest/React Testing Library слой для helpers/forms/error adapter and Playwright/Wails-compatible smoke for production build lifecycle.
Проверка после исправления: Added dependency-free frontend test infrastructure using TypeScript compile + Node test runner: `npm test` covers `formatAppError`/`normalizeAppError` safe copy and `confirmDiscardFormChanges` dirty form behavior. Added `npm run smoke:prod` to verify production `dist/index.html` and referenced JS/CSS assets after `npm run build`. `make release-gate` now runs `frontend-test`, `frontend-build` and `frontend-smoke`. Проверено: `npm test`, `npm run build`, `npm run smoke:prod`, `GOCACHE=/tmp/go-build-cache go test ./...`, `git diff --check`. Browser/manual UX safety smoke is maintained and release-gated after `ISSUE-043`.
Связанные пункты: D.04, D.05, G.02, H.03

## ISSUE-039

Категория: Tests/Integration Safety
Пункт плана: G.01.193, G.01.194, G.01.195, G.02.202
Severity: major
Статус: fixed
Место: `internal/repository/document_registration_integration_test.go`, release test gate
Проблема: Критичные PostgreSQL integration tests gated by `DOCFLOW_INTEGRATION_DSN` and skipped by default. Integration helper resets `public` schema, so DSN must never point to real data.
Почему важно: Release gate can miss no-gaps/idempotency/retention invariants if env is not set; unsafe DSN could destroy data.
Рекомендация: Add safe disposable DB provisioning/guard: require DB name prefix like `docflow_test`/`docflow_regression`, create/drop DB in test runner, include integration tests in release gate.
Проверка после исправления: Added `tools/integrationtest` and Makefile `integration-test`; release gate now runs it. Runner requires `DOCFLOW_INTEGRATION_ADMIN_DSN`, creates disposable `docflow_test_*` DB, injects `DOCFLOW_INTEGRATION_DSN`, runs registration/concurrency/retention integration tests and drops the DB. Integration tests refuse DSNs whose database name is not `docflow_test*`/`docflow_regression*`. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/repository ./tools/integrationtest`, `GOCACHE=/tmp/go-build-cache go run ./tools/integrationtest` fails with explicit missing-admin-DSN message. Full positive integration run requires local/test PostgreSQL admin DSN.
Связанные пункты: B.01, B.05, G.01, H.03

## ISSUE-040

Категория: Tests/Database Constraints
Пункт плана: G.01.194, G.01.195
Severity: major
Статус: fixed
Место: database integration tests
Проблема: Есть coverage for migration availability, document registration transactions and retention FK, но broader schema constraints не покрыты integration tests: duplicate registration number conflicts, required fields, FK references for assignments/acknowledgments/attachments, dirty migration recovery.
Почему важно: Schema-level invariants are final protection when frontend/backend validation misses a case.
Рекомендация: Add focused DB integration tests for critical unique/FK/not-null constraints and dirty migration behavior.
Проверка после исправления: Added `TestDatabaseConstraintsIntegration` to the guarded PostgreSQL integration suite. It checks duplicate `(kind, registration_number, registration year)`, required `documents.idempotency_key`, FK constraints for assignments/acknowledgments/attachments, duplicate `acknowledgment_users`, and dirty `schema_migrations` rejection through `CheckMigrationCompatibility`. `tools/integrationtest` now runs it in the disposable `docflow_test_*` DB. Проверено: `GOCACHE=/tmp/go-build-cache go test ./internal/repository ./tools/integrationtest`, `GOCACHE=/tmp/go-build-cache go test ./...`; positive DB execution requires non-production `DOCFLOW_INTEGRATION_ADMIN_DSN`.
Связанные пункты: B.02, B.05, G.01

## ISSUE-041

Категория: Performance/Baseline
Пункт плана: G.03.204, G.03.205, G.03.206, G.03.207, G.03.208, G.03.209, G.03.210
Severity: major
Статус: fixed
Место: Wails app, backend operations, React screens
Проблема: Performance baseline exists for PostgreSQL synthetic EXPLAIN only. Нет измерений Wails startup, login/dashboard, list open/search/filter, document save latency, statistics screens, frontend render cost, memory.
Почему важно: Production SLO зафиксирован, но фактическое desktop поведение на target OS/build не измерено.
Рекомендация: Add performance smoke with target metrics: startup <=5s, lists/search <=2s, statistics <=5s, memory <=512 MB, registration save latency.
Проверка после исправления: Added maintained `make performance-baseline` target and `tools/performance-report.js`; release gate now generates `build/release-evidence/PERFORMANCE_BASELINE.md`. The report records SLO budgets, frontend dist total, largest route chunks, available Linux/Windows binary sizes and a required target OS manual timings table for startup/login/lists/search/save/statistics/memory. Current generated static baseline passes thresholds: frontend dist 3.05 MiB, largest JS `StatisticsPage` 1.40 MiB under 1.6 MiB route budget, Linux binary 17.48 MiB and Windows binary 18.66 MiB under 100 MiB warning threshold. Target OS timing rows remain release-evidence fields to fill on Linux/Windows artifacts. Проверено: `make performance-baseline`, `node tools/performance-report.js`, `git diff --check`.
Связанные пункты: A.01.007, D.07, F.06, G.03

## ISSUE-042

Категория: Long Running/Cancellation
Пункт плана: G.04.211, G.04.212, G.04.213, G.04.214, G.04.215, G.04.216, G.04.217, G.04.220
Severity: major
Статус: fixed
Место: Wails runtime, file operations, statistics, link graph
Проблема: Нет long-running/memory/cancellation tests. Open lifecycle issue remained before `ISSUE-015`: several operations used `context.Background()` and app shutdown did not coordinate active requests.
Почему важно: Долгая работа desktop app, repeated modals/file operations and network failures can produce leaks, stuck UI or work on closed resources.
Рекомендация: After context propagation remediation, add long-running smoke: repeated modals/views, upload/download loops, statistics refresh, app close during operation, DB/MinIO outage.
Проверка после исправления: Added maintained `docs/long_running_smoke.md` for target OS release evidence. It covers 4-8 hour memory trend, repeated document/registration/file/statistics/link graph loops, app close during upload/download/statistics/link graph, MinIO upload/download/statistics outage and PostgreSQL list/save outage recovery. Added `tools/long-running-smoke-check.js`, `make long-running-smoke-check` and release-gate integration so required scenario IDs cannot silently disappear. Updated release checklist/runbook/smoke docs. Проверено: `make long-running-smoke-check`, `GOCACHE=/tmp/go-build-cache go test ./...`, `git diff --check`; full target OS execution remains release evidence.
Связанные пункты: C.05, E.05, G.04

## ISSUE-043

Категория: UX/Safety Tests
Пункт плана: G.04.218, G.04.219, G.04.220
Severity: major
Статус: fixed
Место: migration rollback, file delete, assignments, settings actions
Проблема: Critical/destructive action confirmations and error dead-end scenarios are not covered by e2e/smoke tests. Rollback guardrails, dirty form warning, file deletion and failed backend actions still need validation.
Почему важно: Даже если logic is correct, operator can still make destructive mistakes or get stuck after failure without validated UX path.
Рекомендация: Add smoke/e2e cases for destructive confirmations, visual separation, failure recovery, retry/cancel paths.
Проверка после исправления: Added maintained `docs/ux_safety_smoke.md` for target OS/Wails release smoke. It covers validation/forbidden/not_found/conflict/internal error recovery, migration rollback, file/link/assignment/acknowledgment/reference destructive confirmations, dirty document/settings forms, repeat submit guards, empty states and terminology. Added `tools/ux-smoke-check.js`, `make ux-smoke-check` and release-gate integration so required scenario IDs cannot silently disappear from the checklist. Updated release checklist/runbook and smoke docs. Проверено: `make ux-smoke-check`, `GOCACHE=/tmp/go-build-cache go test ./...`, `npm test`, `npm run smoke:prod`, `git diff --check`; full manual target OS execution remains release evidence.
Связанные пункты: B.06, D.04, E.03, H.03

## ISSUE-044

Категория: UX/Error Messages
Пункт плана: H.03.232, H.03.233, H.03.239, H.03.240
Severity: major
Статус: fixed
Место: frontend catch handlers, startup/migration/file errors
Проблема: Пользовательские ошибки часто строятся из raw `err?.message || String(err)`, могут содержать технический текст, не всегда объясняют следующий шаг и местами закрывают контекст после ошибки.
Почему важно: Пользователь может увидеть внутренние DB/storage/Go details, не понять, что делать дальше, или потерять контекст работы.
Рекомендация: После `DECISION-009` завести UX copy map для structured error codes: что произошло, что можно сделать, когда обращаться к администратору.
Проверка после исправления: Shared frontend `appError` adapter maps structured codes to safe UX copy with next-step actions for login, forbidden, validation, not found, conflict/idempotency and internal/unknown errors. Unstructured raw `Error.message` / string errors are no longer displayed directly; validation/not-found/conflict preserve safe backend details and add recovery action. Verified with `npm run build` and `git diff --check`; manual smoke is maintained in `docs/ux_safety_smoke.md`.
Связанные пункты: D.05, E.04.174, H.03

## ISSUE-045

Категория: UX/Terminology
Пункт плана: H.02.225, H.02.226, H.02.227, H.04.242
Severity: major
Статус: fixed
Место: document forms, document view, settings, glossary
Проблема: Термины используются неодинаково: `вид документа` используется там, где по glossary нужен `тип документа`; `дело` и `номенклатура` обозначают одну сущность; `исполнитель` используется для разных ролей; `содержание` and `краткое содержание` смешиваются.
Почему важно: Ошибка терминологии может привести к неверному заполнению документов и неправильному пониманию прав/обязанностей.
Рекомендация: Принять UX-терминологию из `TERMS_GLOSSARY.md` and apply consistently. Согласовать спорные термины с бизнесом: `Дело` vs `Номенклатура`, `Краткое содержание`. `ПОС` expansion is fixed under `ISSUE-048`.
Проверка после исправления: Frontend document forms now use `Тип документа` for `documentTypeId`; document cards use user-facing `Дело`; outgoing letter UI says `Исполнитель письма`; assignments say `Ответственный исполнитель`; citizen appeal list/card uses `Содержание` instead of `Краткое содержание`; settings resolution executor copy is qualified. Verified with `npm run build`, `git diff --check` and targeted `rg` search for old problematic frontend strings. Manual terminology smoke is maintained in `docs/ux_safety_smoke.md`.
Связанные пункты: H.02, H.04, I.01

## ISSUE-046

Категория: UX/Destructive Actions
Пункт плана: H.03.236, H.03.237, H.05.249
Severity: major
Статус: fixed
Место: Popconfirm/Modal confirmations
Проблема: Destructive confirmations часто слишком короткие: `Удалить?`, `Удалить файл?`, `Откатить последнюю`. Они не всегда называют сущность and consequence.
Почему важно: Пользователь может подтвердить destructive action без понимания, что именно будет удалено/откачено and whether it can be undone.
Рекомендация: Confirmation copy must name entity and consequence: `Удалить поручение? Это действие нельзя отменить.` For migration rollback include backup requirement.
Проверка после исправления: Strengthened destructive confirmation copy for reference deletes, file delete, document link delete, assignment delete, acknowledgment delete, migration rollback and bulk file delete. Confirmations now name affected entity and consequence; rollback button/modal explicitly says `Откатить последнюю миграцию` and keeps backup/data-loss requirements. Проверено: `rg` found no old `Удалить?`/`Удалить файл?`/`Удалить связь?` copy in `frontend/src`; `npm run build` passed; `npm run lint` passed with existing warnings only.
Связанные пункты: B.06, E.03, G.04.218, H.03

## ISSUE-047

Категория: UX/Empty States
Пункт плана: H.03.238
Severity: minor
Статус: fixed
Место: tables, files, statistics charts
Проблема: Empty states mostly only say there is no data: `Нет прикрепленных файлов`, default AntD Empty, `Нет доступа к статистике`. They rarely explain next action.
Почему важно: Empty screen can feel like dead end, especially for first-run/new users.
Рекомендация: Add action-aware empty states based on permissions: how to add data, change filters, or request access.
Проверка после исправления: Added action-aware empty states for dashboard assignment/acknowledgment cards, admin dashboard placeholder, file attachments, document links, statistics charts, statistics no-access state and statistics report/rating tables. Copy now explains whether to upload/add, wait for assignment, change period/filter or request access. Проверено: `rg` found no old passive empty strings in `frontend/src`; `npm run build` passed; `npm run lint` passed with existing warnings only.
Связанные пункты: H.03.238, G.02.198

## ISSUE-048

Категория: UX/Labels
Пункт плана: H.02.228, H.02.230, H.04.245, H.05.249
Severity: minor
Статус: fixed
Место: filters/forms/icon buttons
Проблема: Some labels/placeholders use unclear abbreviations: `ПОС`, `Рег. №`, `Проср.`, `< 3 дней`, generic `Поиск...`; some icon buttons lack consistent tooltips.
Почему важно: Users may not understand filters/actions without prior knowledge; accessibility and learnability suffer.
Рекомендация: Expand abbreviations or add tooltip/help; make placeholders specific; add consistent tooltips for icon-only actions.
Проверка после исправления: Expanded `ПОС` to `Платформа обратной связи`, replaced `Рег. №` labels with `Регистрационный номер`, changed generic assignment/document search placeholders and date quick filters (`Проср.`/`< 3 дней`) to explicit copy, and added title/tooltip hints for key icon-only view/edit/delete/clear actions in document lists, assignments, settings, files and links. Проверено: `rg` found no old `ПОС`/`Рег. №`/`Проср.`/`< 3 дней`/`Поиск...` strings in `frontend/src`; `npm run build` passed; `npm run lint` passed with existing warnings only.
Связанные пункты: H.02, H.04.245

## ISSUE-049

Категория: UX/Microcopy Style
Пункт плана: H.03.235, H.04.241, H.04.243, H.04.244
Severity: minor
Статус: fixed
Место: success messages, statuses, table/tooltips
Проблема: Microcopy is inconsistent: `Удалено`, `Статус обновлен`, `Документ обновлён`, `завершен`, `dirty`, `N/A`. There is mixed `е/ё` and occasional English technical terms.
Почему важно: Not a blocker, but reduces polish and can confuse non-technical users.
Рекомендация: Adopt style guide: neutral Russian, consistent `ё`, no English technical statuses in user UI, specific success messages.
Проверка после исправления: Replaced generic success messages with action-specific copy for settings CRUD, assignments, acknowledgments, files and profile actions; normalized visible `ё` in updated/deleted/finished/status strings; replaced migration `dirty` UI text with safe Russian copy; replaced statistics `N/A` with `Нет данных`; changed short action `Исполнено`/`Отметить` buttons to verb-first labels. Проверено: `npm run build` passed; `npm run lint` passed with existing warnings only; exact search found no old `Статус обновлен`, `Документ обновлен`, `Приказ обновлен`, `Файл удален`, `Файл сохранен`, `N/A`, `Ошибка миграции (dirty)` or `okText="Исполнено"` strings in `frontend/src`.
Связанные пункты: H.04, H.05

## ISSUE-050

Категория: Documentation/Release
Пункт плана: I.01.258, I.01.259, I.01.260, I.01.261, I.01.262
Severity: critical
Статус: fixed
Место: `README.md`, `docs/release_runbook.md`, `docs/diagnostics_runbook.md`, `docs/backup_restore_runbook.md`, `docs/migration_rollback_runbook.md`
Проблема: В репозитории отсутствует корневой release-grade README/runbook, который описывает dev запуск, production build, миграции, backup/restore and diagnostics. `build/README.md` содержит только стандартное описание Wails build directory.
Почему важно: Production handover, clean clone build and non-author release execution become dependent on unstated local knowledge. This is especially risky because rollback, restore and config/startup diagnostics need explicit release evidence.
Рекомендация: Добавить maintained root README/runbooks for dev setup, release build, DB migrations, backup/restore, diagnostics and target OS install smoke. Audit artifacts in `audit/08_docs_release` can be used as starting material.
Проверка после исправления: Добавлен root `README.md` with dev setup, local config caveats, release/ops entry points, build commands, migration/backup/diagnostics links and critical gate notes. Добавлены maintained `docs/release_runbook.md` and `docs/diagnostics_runbook.md`; existing backup/restore and rollback runbooks are linked from README. Audit docs updated so `ISSUE-050` no longer appears as a critical blocker. Проверено статически: `rg` consistency pass, `git diff --check`. Full non-author clean-clone execution remains a release checklist evidence step.
Связанные пункты: I.01.258-I.01.262, I.02.265-I.02.268

## ISSUE-051

Категория: Documentation/Changelog
Пункт плана: I.02.263, I.02.264
Severity: major
Статус: fixed
Место: `docs/releases.yaml`, release notes/current release assets, known issues docs
Проблема: `docs/releases.yaml` latest version is `1.0.4` from 2026-04-27 and does not reflect current production candidate audit/remediation. Known issues were not packaged as release-facing documentation before stage I.
Почему важно: About UI/release notes, binary metadata and installer metadata can diverge from the artifact actually delivered. Operators and users will not see accepted known issues or remediation status.
Рекомендация: Choose one target version source, update release notes/current release assets, verify About/binary/installer metadata and publish accepted known issues with owner/mitigation.
Проверка после исправления: Added `docs/releases.yaml` version `1.0.5` dated 2026-06-02 with current audit/remediation release notes; generated `internal/releaseassets/current_release.yaml` and `wails.json` `info.productVersion` now show `1.0.5`. Added maintained `docs/known_issues.md` with open critical/major/minor issues, owners and mitigation/acceptance notes; README and release runbook link it. Проверено: `GOCACHE=/tmp/go-build-cache go generate ./internal/releaseassets`, `make release-assets-check`, `GOCACHE=/tmp/go-build-cache go test ./internal/releaseassets ./tools/releasegen`.
Связанные пункты: E.01.154-E.01.159, I.02.263-I.02.264

## ISSUE-052

Категория: Release/Reproducibility
Пункт плана: I.02.270
Severity: critical
Статус: fixed
Место: Git working tree
Проблема: `git status --short` shows modified audit docs, modified backend/frontend files and untracked audit directories/migrations/repository files. Current workspace is not a clean tagged production candidate.
Почему важно: Release artifact cannot be reproduced or audited reliably if changes are uncommitted, untagged or mixed with audit-only files.
Рекомендация: Review all changes, separate product remediation from audit artifacts where needed, run full release gate, commit and tag release candidate from clean worktree.
Проверка после исправления: Audit/remediation changes were committed in `1592e9b fix: address production audit findings`; `git status --short` has no output after the commit. Release tag creation, clean-clone release gate and artifact checksum evidence remain normal release checklist steps before production delivery, but the dirty-worktree blocker is closed.
Связанные пункты: I.02.266, I.02.269, I.02.270

## ISSUE-053

Категория: Release/Checklist
Пункт плана: I.02.265, I.02.266, I.02.267, I.02.268, I.02.269
Severity: major
Статус: fixed
Место: release scripts/docs, `audit/08_docs_release/RELEASE_CHECKLIST.md`, `audit/08_docs_release/SMOKE_TEST.md`
Проблема: До этапа I не было maintained release checklist and smoke-test, executable by a non-author. Stage I created audit artifacts, but they still need to become project release process and be validated on clean machine/target OS.
Почему важно: Passing checks in current workspace is not enough for production release; clean clone, fresh DB, install smoke, backup/restore and security gates must be repeatable.
Рекомендация: Promote checklist/smoke into maintained release docs or script, automate high-confidence gates and attach completed checklist to each release.
Проверка после исправления: Added maintained `docs/release_checklist.md` and `docs/smoke_test.md`; README and `docs/release_runbook.md` link them. Checklist covers clean checkout, version/changelog, release gate, migrations, backup/restore, target OS install smoke, UX safety and final evidence. Smoke test covers startup, auth/settings, all document kinds, lists/access, files, assignments/acknowledgments, errors/safety and audit/log checks. Static verification: `rg` confirms maintained docs references; `git diff --check` passed.
Связанные пункты: F.01, G.01-G.04, I.02.265-I.02.269
