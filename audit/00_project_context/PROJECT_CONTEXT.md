# Project Context

Дата аудита: 2026-05-07
Этап: A. Базовый контекст проекта

## Назначение приложения

Проект `docs-register-and-track` собирает desktop-приложение Wails `docflow` с заголовком окна "Система регистрации документов". По коду и интерфейсу приложение предназначено для регистрации, поиска, просмотра и сопровождения документооборота организации.

Основные доменные зоны:

- регистрация документов: входящие письма, исходящие письма, обращения граждан, приказы;
- учет номенклатуры дел, подразделений, организаций-корреспондентов и исполнителей резолюций;
- поручения, соисполнители, статусы исполнения и контроль сроков;
- ознакомления пользователей с документами и приказами;
- связи между документами и визуализация графа связей;
- вложения документов в объектном хранилище;
- журнал действий по документам и административный аудит;
- статистика по документам, поручениям и системе.

## Production Candidate

Текущее дерево Git чистое на момент проверки `git status --short`.

Проверки, выполненные в рамках этапа A:

- `npm run build` в `frontend`: успешно, `frontend/dist` создан; есть предупреждение Vite о большом основном чанке `index-HhgWsiDR.js` около 3000.72 kB, gzip около 872.68 kB.
- `go build ./...`: успешно при разрешении Go записывать служебный cache вне workspace.
- `go test ./...`: успешно при разрешении Go записывать служебный cache вне workspace; без `frontend/dist` root package падает на embed `all:frontend/dist`.
- `go mod tidy -diff`: без diff.

Вывод: кодовая база выглядит как production candidate с важным условием сборочного порядка: frontend должен быть собран до Go-сборки/тестов пакета `main`, потому что `main.go` встраивает `frontend/dist`.

## Технологический стек

- Desktop shell: Wails v2.12.0.
- Backend: Go module `github.com/Volkov-D-A/docs-register-and-track`, `go.mod` указывает `go 1.25.0`; локально проверено Go `go1.26.2 linux/amd64`.
- Frontend: React 18, TypeScript 6, Vite 8, Ant Design 6, Zustand, dayjs, `@xyflow/react`, `@ant-design/plots`.
- Database: PostgreSQL через `database/sql`, `lib/pq`, `golang-migrate`.
- Object storage: MinIO.
- Logging: `slog` + Seq через асинхронный CLEF writer.
- Packaging/build: Wails, Makefile targets for Linux and Windows, Wails build assets for Windows and macOS.

## Внешние зависимости

Runtime-зависимости:

- PostgreSQL: основная реляционная БД.
- MinIO: хранение файлов вложений.
- Seq: обязательный production-сервис приема структурированных логов; отключение допустимо для dev/local окружения или аварийной диагностики.
- Файловая система ОС: `config/config.json`, локальное состояние темы и просмотра релиза в user config dir, выгрузка файлов в `Downloads`.
- SMB-сетевое хранилище и Docker CLI: операционные скрипты backup/restore.
- WebView runtime платформы Wails: WebView2 на Windows, WebKit/WebKitGTK на Linux, WebKit на macOS.

Build/dev-зависимости:

- Node/npm для frontend build.
- Go toolchain.
- Wails CLI.
- Docker Compose для локальных PostgreSQL, MinIO, Seq.

## Production Constraints

Зафиксированные ограничения:

- `config/config.json` не хранится в Git; пример лежит в `config.example.json`.
- Production разворачивается вручную через `config/config.json` по утвержденной production-документации; отдельные repo-конфиги для `dev`, `test`, `staging`, `production` не обязательны.
- Версия PostgreSQL для production определяется утвержденной production-документацией и не фиксируется в репозитории; `POSTGRES_VERSION` в `.envExample` является local/dev примером.
- Слабые примерные пароли в `.envExample` допустимы только для local/dev шаблона в закрытом контуре разработки без внешнего доступа к ресурсам; они не являются production defaults.
- Секреты конфигурации могут быть зашифрованы с префиксом `ENC:`; ключ AES-256 передается через `ldflags` (`rawEncryptionKey`) или `ENCRYPTION_KEY`.
- Embedded frontend assets берутся из `frontend/dist`.
- Embedded migrations берутся из `internal/database/migrations`.
- Максимальный размер одного вложения в production: 15 MB.
- Разрешенные расширения вложений в production: `.pdf,.doc,.docx,.odt,.xls,.xlsx,.ods`.
- Общий MinIO/storage quota как жесткий лимит пока не задается; monitoring thresholds: warning 80%, critical 90% от доступного storage.
- Ожидаемый объем production-данных: до 1000 документов в год, количество вложений пропорционально документам, до 20 пользователей, средний размер файла около 3 MB, доступный storage около 1 TB.
- Текущий `frontend/dist`: около 3.2 MB.
- Свежие бинарники в `build/bin`, пересобранные 2026-05-07: Linux `docflow` 18,324,384 bytes (около 18 MB), Windows `docflow.exe` 19,566,080 bytes (около 19 MB).
- Production SLO/performance budget: старт приложения до экрана входа до 5 секунд; открытие основных списков при обычной пагинации/фильтрах до 2 секунд; тяжелые отчеты/статистика до 5 секунд; память desktop-приложения в обычной работе до 512 MB; размер бинарника: 18-19 MB норма, 100 MB warning threshold.
- Backup/restore policy для PostgreSQL+MinIO: RPO до 1 дня потери данных; RTO 1-2 дня на восстановление; backup retention 15 дней; проверка backup-архивов выполняется через manual test restore перед релизом; offsite copy реализован штатно отправкой копии на другой сервер.
- Backup/restore script path policy: абсолютный путь к `.env` в `backup_smb_tar.sh` и `restore_smb_tar.sh` допустим из-за запуска через cron; скрипты не обязаны работать из произвольной рабочей директории, но путь должен совпадать с production cron/deployment path по утвержденной документации.
- Attachment recovery policy: при рассинхронизации PostgreSQL attachment metadata и MinIO objects восстанавливаются одновременно PostgreSQL и MinIO из согласованного backup-набора.
- Seq обязателен в production как сервис логирования, но его данные не входят в backup-контур и не бэкапятся.
- Network security policy: PostgreSQL, MinIO и Seq находятся в закрытом защищенном LAN-контуре; доступ к ним возможен только из LAN, по открытым каналам связи данные не передаются. TLS/SSL для внутренних соединений не является обязательным production-требованием при сохранении этих границ.
- Migration management policy: полный механизм управления миграциями через UI/runtime path сохраняется в production для пользователя с системным правом `admin`, включая status/up/rollback. Для destructive rollback нужны guardrails: явное предупреждение, подтверждение, свежий PostgreSQL+MinIO backup, запись в `admin_audit_log` и rollback-runbook.
- Log retention/access policy: `document_journal` и `admin_audit_log` в PostgreSQL хранятся весь жизненный цикл проекта и не удаляются на уровне приложения; экспорт не предполагается, отдельная фильтрация не требуется, изменение возможно только прямым доступом к БД, которого в штатной эксплуатации нет; Seq logs не являются долговременным audit trail.
- Service account policy: для PostgreSQL, MinIO и Seq допустим один общий технический аккаунт production-контура; отдельные service accounts по каждому сервису не обязательны.
- Document type policy: document types остаются фиксированными в коде; расширение и перевод в справочник в ближайшее время не планируются.
- Public app settings contract: `organization_short_name` для общего layout отдается через `GetPublicAppSettings()`, доступный всем авторизованным пользователям; настройки не считаются секретом для чтения, но изменение разрешено только пользователю с системным правом `admin`.
- Document registration idempotency policy: регистрация документа должна поддерживать backend `idempotency_key`; target schema после этапа B — `documents.idempotency_key UUID NOT NULL` и unique `(created_by, kind, idempotency_key)`. Повторный запрос с тем же ключом возвращает уже созданный документ и не создает дубль или повторный инкремент `nomenclature.next_number`.
- Stage B database audit status: завершен 2026-05-27 на локальном test contour. Проверены fresh migrations, application migrator, runtime rollback, synthetic EXPLAIN ANALYZE dataset, no-gaps failure case, FK retention behavior и backup/restore scripts. Основные remediation work items переданы на C/E/F/H.
- Backend/Wails error contract target после этапа C: frontend должен получать structured error envelope `{code,message,details?}` со стабильными domain codes; plain `err.Error()` и тексты PostgreSQL/storage не являются frontend contract.
- Backend lifecycle target после этапа C: долгие DB/MinIO/file/statistics operations должны получать request/app context с cancel/timeout; закрытие окна должно отменять активные операции предсказуемо.
- Logging/privacy target после этапа C: technical Seq logs должны минимизировать ФИО и business identifiers; доменный audit trail (`document_journal`, `admin_audit_log`) остается местом для необходимых бизнес-деталей.
- Stage C backend audit status: завершен 2026-05-27 статическим review Go backend/Wails bindings. SQL injection risk не подтвержден; основные remediation work items — structured errors, strict typed document command DTO, `idempotencyKey`, context propagation, logging PII minimization.
- Stage B/C remediation status: backend `idempotencyKey` + no-gaps registration transaction реализованы и проверены integration tests; retention-safe FK для `document_journal`/`admin_audit_log` реализованы миграцией `009_retention_safe_journal_fks` и integration test.
- Stage D frontend audit status: завершен 2026-05-28. Документные frontend forms отправляют `idempotencyKey`, но frontend еще не перешел на centralized structured error handling; submit guards/dirty confirmations и bundle/performance остаются открытыми frontend work items.
- Stage E build/install/update audit status: завершен 2026-05-28 статическим review production build/install/runtime lifecycle. Основные release gates: единый version source, deterministic release build, production config lookup/startup diagnostics, installer privilege policy, downgrade/schema compatibility guard, backup/restore temp cleanup and download overwrite protection.
- Stage F security/dependencies/quality audit status: завершен 2026-05-28. `go test ./...`, `go vet ./...`, `npm run build` прошли; `npm audit` clean; `govulncheck` нашел reachable Go vulnerabilities in `go1.26.2`/`x/net@v0.52.0`, requiring toolchain/module upgrade before production.
- Stage G tests/performance audit status: завершен 2026-05-28. `go test ./...` прошел; отдельные PostgreSQL integration tests for idempotency/concurrency/retention FK passed against local test DB. Основные gaps: нет frontend/e2e tests, performance baseline for Wails/UI/memory missing, long-running/cancellation tests missing, integration tests need safe release gate.
- Stage H UX/text audit status: завершен 2026-05-28. Основные gaps: raw backend/system errors in user UI, inconsistent terminology (`вид`/`тип`, `дело`/`номенклатура`, `исполнитель`), weak destructive confirmations, passive empty states, unexplained abbreviations and style inconsistency.
- Stage I docs/release/final readiness audit status: завершен 2026-05-28. Созданы `audit/08_docs_release/*` and `audit/FINAL_SUMMARY.md`. Финальное решение для текущего production candidate: `not_ready` because critical blockers remain open (`ISSUE-007`, `ISSUE-032`, `ISSUE-050`, `ISSUE-052`).

Остается проверить на следующих этапах:

- фактическое выполнение manual test restore PostgreSQL+MinIO перед релизом;
- restore fail-fast contract: `pg_restore` fatal/unknown errors должны останавливать workflow до MinIO mirror; требуется restore report и smoke validation.
- production-версию PostgreSQL по утвержденной production-документации;
- фактический мониторинг заполнения MinIO/storage с warning 80% и critical 90%;
- соответствие утвержденной production-документации подтвержденной backup policy.
- подтверждение LAN-границ закрытого защищенного контура по утвержденной production-документации.
- rollback guardrails для `down`-миграций.
- влияние пожизненного хранения журналов на индексы, размер БД и планы backup/restore.
- frontend обработку structured error codes.
- отдельный unit/smoke test strict DTO validation на unknown fields и missing `idempotencyKey`.
- cancellation/shutdown behavior для upload/download/link graph/statistics.
- redaction/minimization policy для Seq technical logs.
- frontend submit/loading guards и dirty confirmation для длинных форм.
- frontend bundle/startup performance на production-like данных.
- единый source of truth для версии приложения: About, Wails metadata, binary properties and installer DisplayVersion.
- deterministic release build на clean machine, включая required secret validation and generated release assets freshness.
- production config path policy and startup diagnostics for missing/invalid config.
- target OS install smoke: Program Files/portable path, path with spaces/Cyrillic, run without admin after install.
- downgrade/newer-schema guard and dirty migration recovery UX/runbook.
- backup/restore temp cleanup and secret exposure checks.
- collision-safe attachment download behavior.
- Go toolchain/module vulnerability remediation: upgrade to `go1.26.3+` and `golang.org/x/net@v0.53.0+`, then repeat `govulncheck`.
- security/license/static-analysis release gates: `govulncheck`, `npm audit`, license reports, `go vet`, `go test`, `npm run build`, frontend lint.
- resolve unknown npm license (`@antv/g2-extension-plot`) and generate Go transitive license inventory.
- add minimal frontend ESLint/static-analysis gate and reduce `@ts-ignore`/broad `any` at Wails contract boundaries.
- gofmt formatting drift cleanup.
- release test gate: Go unit tests, safe disposable PostgreSQL integration tests, frontend component tests, production-build e2e smoke.
- safe integration DSN guard/provisioning because DB integration tests reset `public` schema.
- performance baseline on target OS/build: startup, lists/search, save, statistics, memory, bundle/binary size.
- long-running smoke: repeated modals/views/files, DB/MinIO outages, close app during long operations.
- apply UX terminology rules from `TERMS_GLOSSARY.md` and confirm business-sensitive terms: `Дело` vs `Номенклатура`, `ПОС`, `Краткое содержание`.
- map structured errors to safe actionable user messages.
- strengthen destructive confirmations and empty states; include them in smoke/e2e tests.
- promote stage I release checklist/smoke/known issues into maintained project docs or scripts.
- clean worktree, commit/tag release candidate and verify reproducible release evidence.

## Контрольные Пункты

| Код | Статус | Доказательство | Вывод | Уточнить или зафиксировать |
| --- | --- | --- | --- | --- |
| A.01.001 | ok | `git status --short` чистый; `npm run build`, `go build ./...`, `go test ./...` прошли с учетом Go cache permissions. | Текущее состояние можно считать production candidate для продолжения аудита. | Зафиксировать обязательный порядок: frontend build до Go build/test root package. |
| A.01.005 | ok | `go.mod`, `frontend/package.json`, `config.example.json`, `docker-compose.yaml`, `backup_smb_tar.sh`, `restore_smb_tar.sh`. | Внешние зависимости перечислены выше. | Подтвердить, какие из них обязательны в production, а какие только dev/ops. |
| A.01.006 | ok | Makefile содержит `build-linux` для Linux `amd64` и `build-windows` для Windows `amd64`; пользователь подтвердил production target. | Поддерживаемые production-платформы: Linux `amd64` и Windows `amd64`; macOS не входит в текущий release target. | Нет. |
| A.01.007 | ok | Пользователь подтвердил production SLO/performance budget; есть размер frontend build и свежих bin artifacts; есть default max attachment size. | Базовые production-лимиты скорости, памяти и размера сборки зафиксированы для следующих этапов. | На этапе F измерить фактические показатели на production-like данных. |
| A.05.021 | issue | `.gitignore` исключает `/config` и `.env`; есть local/dev `config.example.json` и `.envExample`; backup/restore scripts читают `/home/dimas/projects/docs-register-and-track/.env`; пользователь подтвердил, что абсолютный путь к `.env` нужен из-за cron, а production разворачивается вручную через `config/config.json` по утвержденной документации вне проекта. | Секреты отделены от Git; production-проверки B/E/H должны сверяться с утвержденной документацией вручную; абсолютный env path допустим, если совпадает с production cron/deployment path. | Проверить production config/ops и cron path по утвержденной документации; отдельно оценить передачу SMB password в аргументах `mount`. |
| A.05.022 | issue | `config.example.json` использует `localhost`, `sslmode: disable`, `useSSL: false`, Seq `http://localhost:5341`; `docker-compose.yaml` помечен как dev; production guide базируется на утвержденной документации вне проекта; пользователь подтвердил закрытый защищенный LAN-контур. | Примеры в репозитории ориентированы на dev/local; production-профиль и LAN-границы закрытого контура проверяются по утвержденной документации вручную. | Проверить production config/ops и LAN-границы по утвержденной документации. |
| A.05.023 | ok | `.envExample` содержит слабые примерные пароли; `config.example.json` включает отключенный TLS для PostgreSQL/MinIO; пользователь подтвердил закрытый контур разработки и production LAN-контур. | Weak example secrets допустимы как local/dev шаблон в закрытом контуре разработки; отключенный TLS допустим только внутри подтвержденного закрытого защищенного LAN-контура; example values не являются production defaults. | Не использовать example defaults в production; production значения сверяются по утвержденной документации. |
