# Рабочий отчёт аудита кодовой базы

План: [codebase-cleanup-audit-plan.md](codebase-cleanup-audit-plan.md)  
Состояние: этапы 0–1 выполнены; этапы 2–8 в работе; этап 9 ещё не начат  
Дата фиксации исходного состояния: 24 сентября 2026 года

## Этап 0. Исходное состояние и границы

### Репозиторий и сборка

| Параметр | Установленное значение | Источник |
| --- | --- | --- |
| Commit | `11fdb9f650a5830b8d9e119815e3a984bf927256` | `git rev-parse HEAD` |
| Ветка | `main` | `git branch --show-current` |
| Исходное рабочее дерево | Нет изменений в отслеживаемых файлах; есть созданный для аудита `docs/codebase-cleanup-audit-plan.md` | `git status --short` |
| Версия продукта в исходниках | `1.0.7` | `wails.json`, `docs/releases.yaml`, `internal/shared/releaseassets/current_release.yaml` |
| Go | `1.27.1` | `go.mod` |
| Desktop | Wails v2, React/TypeScript/Vite; выпуск для Linux `amd64` и Windows `amd64` | `go.mod`, `frontend/package.json`, `Makefile`, `docs/tech_docs.md` |
| Сервер и хранилища | `docflow-server`, PostgreSQL, SeaweedFS, Seq; примеры окружения содержат PostgreSQL `18.3`, SeaweedFS `4.46`, Seq `2025.2.16020-x64` | `README.md`, `docs/examples/.envExample`, Compose |
| Миграции | Встроены в сервер; сохранены пары `up/down` для версий 001–012 | `internal/server/database/migrations_embed.go`, `internal/server/database/migrations` |
| Backup | Создание и проверка архива с manifest формата 3; восстановление сверяет версию схемы | `internal/server/backup/archive.go`, `catalog.go`, `prepare.go`, `service.go` |

Версии зависимостей и примеры Compose описывают исходники, но не доказывают,
что именно эти версии запущены у пользователей. Локальный `config/config.json`
и `.env` не используются как свидетельство production-конфигурации.

### Поддерживаемые контракты, которые нужно проверить до удаления

1. Desktop и сервер проверяют `v1` HTTP API, версию продукта и точную
   идентичность сборки (`buildProtocol=2`). Текущие бинарники работают только
   при совпадении версии и build identity; это подтверждают
   `internal/server/system.go`, `internal/desktop/serverclient/system.go` и
   `README.md`. Пользователь подтвердил, что установлена только последняя
   версия Docflow; поддержка прежних desktop-сборок не требуется.
2. Wails `Bind` в `internal/desktop/app/app.go` публикует desktop-сервисы;
   контракт дополнительно зафиксирован в
   `internal/desktop/app/testdata/wails-api.json`. Методы нельзя считать
   неиспользуемыми без проверки generated bindings и frontend.
3. Схема PostgreSQL, применённые миграции, журналы, права доступа,
   idempotency и outbox являются persistent контрактами. Пользователь
   уточнил, что приложение в разработке и действующих архивов с данными
   нет. Исторические миграции 001–012 нельзя менять как обычный код.
4. Linux и Windows, установка вне каталога репозитория, переменные
   окружения, secret files и Compose-конфигурация являются
   эксплуатационными потребителями кода и скриптов.
5. Пользователь подтвердил отсутствие сторонних HTTP-клиентов и любых
   действующих архивов с данными. Поэтому публичные маршруты и поля можно
   сокращать после проверки текущего desktop, а восстановление ограничить
   форматом 3 и текущей схемой. Миграции уже применённых БД остаются
   защищённым контрактом.

### Карта проверок для следующих этапов

| Изменение | Минимальная проверка |
| --- | --- |
| Только документация | Просмотр diff; `make docs-links-check` при изменении ссылок |
| Go-логика без БД | Затронутый пакет `go test`, при необходимости `go vet` |
| Frontend-логика | Затронутый frontend-тест и/или `make frontend-build` для контрактов типов и импорта |
| Wails public API | Генерация bindings через `make dev-client` или целевую сборку, затем проверка diff и контрактного теста |
| SQL, транзакции, миграции, S3 | Узкий интеграционный тест на `testing/compose/integration.yaml`; если риск охватывает backup — соответствующий сценарий `make integration-test` |
| Deploy, storage, backup/restore | Целевой smoke на изолированном стенде; production release — `make release-gate` и проверки целевых ОС |

Полный gate не запускается как baseline аудита: этап 0 не изменяет поведение.
Состав gate установлен по `tools/release-gate.sh` и `build/make/checks.mk`.

### Уточнённые сведения

Приложение в разработке; установлена только последняя версия Docflow.
Сторонних HTTP-клиентов и действующих архивов с данными нет. При начале
хранения резервных копий заново оценивать политику совместимости при релизах.

## Реестр находок

| ID | Место | Наблюдение и доказательство | Статус | Решение/проверка |
| --- | --- | --- | --- | --- |
| DOC-001 | `docs/tech_docs.md`, раздел Known Release State | Было `1.0.6`, тогда как release metadata содержит `1.0.7` | Исправлено | Обновлено до `1.0.7` |
| DOC-002 | `docs/examples/.envExample`, `DOCFLOW_SERVER_VERSION` | Пример содержал `1.0.6`, при текущем `1.0.7` цель `_check-docflow-server-version` отклоняла бы публикацию | Исправлено | Обновлено до `1.0.7` |
| ARC-001 | `internal/desktop/app/composition_root_integration_test.go` | Тест desktop-пакета импортировал `internal/server/services`; тестовая сборка подтягивала пять server-пакетов | Исправлено | Удалён дублирующий типовой assertion; более строгий `TestWailsAPIContract` проверяет package path всех bindings; тестовый dependency graph очищен |
| WAILS-001 | `AuthService.GetCurrentUser` | Метод не вызывался в UI, но использовался внутри Go через `Principal` | Исправлено | Метод `AuthService` стал внутренним; HTTP `Me` и `Principal.GetCurrentUser` сохранены; bindings регенерированы |
| WAILS-002 | `AssignmentService.GetByID` | UI использует одноимённый метод только `DocumentQueryService`; Wails метод в цепочке поручений не вызывался | Исправлено | Удалены Wails метод и его `serverclient` адаптер, затем серверный HTTP маршрут и service после подтверждения отсутствия внешних клиентов |
| TEST-001 | `internal/desktop/services/assignment_service_test.go`, `internal/desktop/app/testdata/wails-api.json` | Тест missing client и Wails snapshot сохраняли `AssignmentService.GetByID` | Исправлено | Удалены устаревшее ожидание и запись контракта; `TestWailsAPIContract` прошёл |
| DATA-001 | `frontend/src/store/useAuthStore.ts` | После `Login` store копировал `isActive`, `failedLoginAttempts` и `department.nomenclatureIds`, которые далее не читались. `department.id` нужен `ProfilePage` для кандидатов на замещение — это выявила проверка TypeScript | Исправлено для локального state | Shared `dto.User` сохранён; проверка типов должна подтверждать потребителей вложенных полей |
| DATA-002 | `/api/v1/access/current`, `CurrentAccessSummary` | `documentDomainAccess`, `isDocumentParticipant`, `canReadFull` и `canOpenPage` не читаются из ответа UI; `registrationFormCode`, `registryGroup`, `supportedActions` копировались в `DocumentKindMeta` и дальше не использовались. `registrationKinds` и `canRegister` дублировали `availableActions:create` | Исправлено | Удалены лишние поля frontend и HTTP DTO, промежуточный mapper и метаданные; серверный расчёт прав сохранён, регистрация в UI выводится из `availableActions` |
| TEST-002 | `internal/server/access_summary_scenarios_test.go`, frontend auth fixtures | Серверный тест проверял поля `documentDomainAccess`, `canReadFull`, `registrationKinds`; frontend fixture `accessVisibility.test.tsx` задавал неиспользуемые поля auth state | Исправлено | Серверный тест теперь проверяет доступность разделов и фактические действия; frontend fixtures сокращены ранее |
| COMP-001 | `internal/server/database/postgres.go`, `Prepare`/`PrepareContext` | Текущих вызовов обёрток нет; однако `DB` встраивает `*sql.DB`, и удаление обёрток откроет одноимённые методы без ограниченного таймаута | Обоснованно оставить | Пересматривать только вместе с явным интерфейсом БД без небезопасного promoted method |
| HTTP-001 | `GET /api/v1/assignments/{id}` | После WAILS-002 у текущего desktop нет вызова маршрута; пользователь подтвердил отсутствие внешних клиентов | Исправлено | Удалены маршрут, handler, service и тесты мёртвого метода; `repository.GetByID` сохранён для операций изменения |
| COMP-002 | `internal/server/backup/prepare.go`, `restore.go` | Формат 3 введён вместе с миграцией 012; код допускал архив со схемой ниже текущей и содержал условную очистку `backup_settings` | Исправлено | Принимается только текущая схема, несовпадение отклоняется до PostgreSQL tools; настройки копий очищаются без ветки для старой схемы |
| TEST-003 | `internal/server/backup/catalog_test.go`, `archive_test.go`, `prepare_test.go` | Тесты содержали marker формата 2 и корректный архив схемы 11, хотя действующих архивов с данными нет | Исправлено | Удалены устаревшие fixture; сохранена проверка отказа для неизвестного формата и добавлен отказ для несовпадающей схемы до запуска PostgreSQL tools |
| COMP-003 | `internal/server/backup/replacement.go`, frontend backup labels и тест recovery status | При допуске только текущей схемы шаг `RunMigrations` после восстановления становится пустым, но публикует состояние `migrating` | Исправлено | Удалены запуск миграций, состояние и устаревшая подпись/тестовое ожидание; финальная очистка сессий и настроек открывает свежее подключение к восстановленной БД |
| DOC-003 | `frontend/src/features/settings/MigrationsTab.tsx`, `internal/desktop/services/settings_test.go` | Поле ссылки на копию и тестовые fixture использовали имя старого `.tar` архива | Исправлено | Пример и fixture заменены на ID копии текущего формата; семантика аудиторской ссылки не изменена |
| DATA-003 | `DashboardActivity.expiringAssignments` | SQL выбирал восемь полей для дашборда, но `MapAssignments` превращал каждую строку в общий DTO с полями отчёта, серии, соисполнителей и служебными датами; экран читает только восемь полей | Исправлено | Добавлен узкий `DashboardAssignment`; сервис передаёт только читаемые UI поля, общий DTO поручения сохранён для остальных сценариев; узкий тест и Wails-сборка прошли |
| TEST-004 | `dashboard_outbox_adapters_test.go`, `statistics_test.go` | Fixture после DATA-003 всё ещё создавали `[]dto.Assignment` как ответ дашборда; тестовая сборка двух пакетов обнаружила устаревшие ожидания | Исправлено | Fixture переведены на `[]dto.DashboardAssignment`, точечные тесты прошли |
| DATA-004 | `Attachment.Filepath`, `dto.Attachment`, SQL-списки вложений | `Filepath` нигде не заполнялся; `DocumentID`, `AssignmentID`, `ContentType`, `UploadedBy` проходили через DTO без потребителя в desktop/UI; два запроса списков загружали эти данные вместе с путём хранения и сразу отбрасывали | Исправлено | Удалены мёртвое поле модели и пять полей ответа, оба `SELECT`/`Scan` сокращены до данных списка; серверная модель сохраняет нужные хранилищу ID, путь и тип; Wails bindings регенерированы, узкие Go-тесты и SQL-интеграция прошли |
| DATA-005 | `UserEventRepository.CreateFromOutbox` | После `INSERT ... RETURNING id` метод вызывал `GetByID`, загружая событие и автора через JOIN; worker отбрасывал результат целиком | Исправлено | Outbox-ветка выполняет `INSERT ... ON CONFLICT DO NOTHING` через `Exec` без `RETURNING` и повторного SELECT; позже прямой тестовый путь `Create` удалён в TEST-005; узкие и PostgreSQL-тесты прошли |
| DATA-006 | `UserEvent`/`dto.UserEvent`, `UserEventRepository.GetList` | UI не читает `actorUserId`, `actorUserName`, `entityId`, `metadata`, но список загружал их из БД и делал JOIN с `users`; сохранённые события уже имеют эти значения | Исправлено | Удалены четыре поля из read-модели/DTO и SQL-списка, JOIN автора больше не выполняется; Wails-сборка и PostgreSQL-тесты прошли. Запись полей удалена в DATA-007 |
| TEST-005 | `UserEventStore.Create/GetByID`, фикстуры и прямой тест репозитория | Прямой путь создания/чтения события использовали только тесты; рабочая доставка использует `CreateFromOutbox`, а UI читает список | Исправлено | Удалены методы из порта/репозитория и тестовая заглушка; интеграционный сценарий теперь создаёт событие через outbox-путь и проверяет пользовательский scope через список |
| DATA-007 | `user_events.actor_user_id`, `entity_id`, `metadata`, индекс `idx_user_events_entity` | Поля записывались, но после DATA-006 не читались приложением; по `entity_type/entity_id` нет SQL-фильтров | Исправлено | После разрешения удалить исторические значения добавлена миграция 013; запись и генерация полей удалены. Миграция 008 сохранена. Откат восстанавливает схему, но не исторические значения: `entity_id` существующих строк получает нулевой UUID |

Промежуточный итог: 22 находки, 21 исправлена, 1 обоснованно сохранена.
Новые документы аудита не входят в статистику diff отслеживаемых файлов.

## Этап 1. Инвентаризация, первый проход

В Git на исходном commit учтено 774 файла: `internal/server` — 325,
`internal/desktop` — 108, `internal/models` — 30, `internal/dto` — 13,
`internal/shared` — 12, `frontend/src` — 135, `frontend/test` — 27,
`frontend/wailsjs` — 54. В миграциях 24 SQL-файла: 12 пар `up/down`.
Числа относятся к отслеживаемым файлам и не включают новые документы аудита.

| Контур | Обнаруженные точки входа и артефакты | Что учитывать при поиске мёртвого кода |
| --- | --- | --- |
| Desktop | `main.go`; 25 сервисов в `Bind` (`internal/desktop/app/app.go`); generated JS/TS в `frontend/wailsjs/go/services` | Wails вызывает методы динамически; контракт зафиксирован в `internal/desktop/app/testdata/wails-api.json` |
| Frontend | `frontend/src/main.tsx` → `App.tsx` → `components/AppRouter.tsx`; 11 lazy страниц и динамические импорты Wails сервисов | Обычный поиск статических импортов пропускает `import()` и вызовы через переменную модуля |
| Server HTTP | `cmd/docflow-server/main.go` → `internal/server/app.go` → `internal/server/management.go`; backup-маршруты также в `internal/server/backup.go`, особый режим в `blocked_restore.go` | Роуты через `http.ServeMux`, строковые шаблоны и доступ извне desktop |
| Фоновые операции | `internal/server/background`, `outbox`, `liveevents`, `backup`, telemetry; `server:event` и `auth:session-ended` соединяют Go и frontend | Отсутствие вызова из UI не доказывает неиспользование |
| Сборка и CLI | `tools/buildmeta`, `tools/releasegen`, `tools/dbperf`; `go:generate`, `go:embed`, Makefile, release gate, Dockerfile, Compose | Генерируемые файлы и встроенные ресурсы имеют отдельные точки входа |
| Тестовая инфраструктура | `internal/server/mocks`, `internal/server/testutil`, `frontend/test`, `testing/compose`, `testing/scripts`, `tools/*_test.go` | Mock и fixture могут использоваться только при тестовой сборке |

Поиск импортов production Go-файлов не выявил прямой зависимости
`internal/desktop` → `internal/server`, `internal/server` →
`internal/desktop`/Wails или `internal/shared`, `internal/dto`,
`internal/models` → desktop/server. `go list -deps ./internal/desktop/...`
подтвердил отсутствие server-пакетов в транзитивном production-графе desktop;
`go list -deps ./cmd/docflow-server` не включает desktop и Wails.
Тестовая сборка `internal/desktop/app` включает пять server-пакетов через
`composition_root_integration_test.go` (ARC-001).
Все 25 имён generated Wails-сервисов встречаются в `frontend/src`, поэтому
на уровне целых сервисов очевидных кандидатов пока нет. Использование
отдельных методов остаётся задачей этапа 2.

Отдельно отмечены динамические точки входа: `go:embed` для frontend assets,
release notes, миграций и CA certificate; `go:generate` для release assets;
build tag `!linux && !windows` в `internal/server/backup/space_other.go`.
Файлы `build/darwin` и нецелевые платформы требуют проверки роли в Wails
tooling, прежде чем считать их лишними.

`frontend/dist`, `frontend/.test-build`, `frontend/node_modules` и `build/bin`
являются локальными результатами или установленными зависимостями и не входят
в отслеживаемые файлы. `config/config.json` существует локально, но также не
отслеживается. Их наличие в рабочем каталоге не является находкой об
избыточности исходников.

Первый проход по зависимостям: все 8 runtime npm-зависимостей встречаются
в frontend; используемые Go-модули подтверждены `go mod why -m` для
ключевых прямых зависимостей. Отсутствие буквального импорта npm-пакета
само по себе не доказывает избыточность dev dependency: ESLint и TypeScript
запускаются через scripts и конфигурацию, `@types/*` потребляются
компилятором. `@testing-library/dom` прямо не импортируется проектом,
но объявлен peer dependency трёх используемых пакетов Testing Library,
поэтому его наличие обоснованно. Полный аудит transitives и lockfile
выполняется на этапе 5/6 при изменении цепочки.

## Этап 2. Сквозные сценарии, первый проход

На исходном commit проверка generated `frontend/wailsjs/go/services/*.js` дала 122 метода.
Поиск имён в production `frontend/src` не нашёл только
`AuthService.GetCurrentUser` (WAILS-001). Остальные 121 имя встречается в
исходниках UI; для распространённых имён вроде `GetList` это ещё не
доказывает использование метода конкретного сервиса. Проверка WAILS-001
сопоставлена с `frontend/src/store/useAuthStore.ts` и
`internal/desktop/services/principal.go`: frontend получает пользователя
после `Login`, а внутренний `Principal` использует `GetCurrentUser` для
проверки прав. Удалять `GET /api/v1/auth/me` по этой находке нельзя.
Дополнительная проверка совместного присутствия имени сервиса и метода
в одном frontend-файле выявила `AssignmentService.GetByID` (WAILS-002):
имя метода в UI относится только к `DocumentQueryService`. После исправлений
Wails API содержит 120 методов; diff generated bindings включает удаление
именно этих двух методов.
Повторная проверка всех 120 generated методов после исправления: для
каждого имя метода встречается хотя бы в одном frontend-файле, который
ссылается на соответствующий сервис. Новых кандидатов этим способом не
выявлено; совпадение имён само по себе не заменяет анализ вызова и результата.
Точный проход по TypeScript AST сопоставил импортированные функции с их
локальными символами, вызовы через переменную модуля и прямые вызовы после
`import()`: 115 методов имеют такие использования. Остальные пять вызваны
через `import().then(...)`: `AssignmentService.GetSeries` и
`GetSeriesHistory` в `AssignmentSeriesModal`, `DashboardService.GetActivity`
в `DashboardPage`, `DocumentKindService.GetCurrentAccessSummary` в
`accessSummaryCache`, `SettingsService.GetOrganizationShortName` в
`useBrandName`. Таким образом, у каждого из 120 текущих Wails-методов есть
конкретный вызов в production frontend. Это подтверждает достижимость
методов, но не полезность каждого поля их ответов.

| Сценарий | Вход UI → Wails/HTTP | Дальнейший потребитель или эффект |
| --- | --- | --- |
| Запуск, совместимость, maintenance | `SystemBootstrapGate`, `SystemService` → `/system/status`, `/system/compatibility` | Состояние миграций, readiness и ограничение входа |
| Вход, профиль, сессия | `LoginPage`, `useAuthStore`, `AuthService` → `/auth/*`, `/profile` | Сессия сервера, состояние frontend и `auth:session-ended` |
| Журналы четырёх видов и карточка | `IncomingPage`, `OutgoingPage`, `CitizenAppealsPage`, `OrdersPage` → `DocumentQueryService` → `/documents/query`, `/documents/{id}` | Document query engine и repository |
| Регистрация, изменение, черновики | Формы документов, `DocumentRegistrationService` → `/documents/{kind}*` | Command handlers, транзакции, idempotency, outbox |
| Поручения и серии | `AssignmentsPage`, `AssignmentService` → `/assignments*`, `/assignment-series*` | Assignment service, repository, journal/outbox |
| Ознакомления и события | Dashboard, карточка, `AcknowledgmentService`, `UserEventService` → `/acknowledgments*`, `/user-events*` | Repository, непрочитанные события, SSE |
| Вложения | Карточки, `AttachmentService` → `/documents/{id}/attachments`, `/assignments/{id}/attachments`, `/attachments/{id}` | Metadata, SeaweedFS, локальный выбор/открытие файла, compensation |
| Связи и журнал | Карточка, `LinkService`, `JournalService` → `/document-links*`, `/documents/{id}/links`, `/link-graph`, `/journal` | Link repository и журнал документа |
| Справочники и номенклатура | `ReferencesPage`, настройки, `ReferenceService`, `NomenclatureService` → `/references*`, `/nomenclature*` | Справочники и регистрация документов |
| Пользователи, подразделения, права | Настройки, профиль → `UserService`, `DepartmentService`, `DocumentAccessAdminService`, `UserSubstitutionService` → `/users*`, `/departments*`, `/access/current`, `/profile/substitution*` | Permission model и БД |
| Статистика | Dashboard, `StatisticsPage`, `StatisticsService` → `/dashboard/activity`, `/statistics*` | SQL агрегаты, status storage |
| Системные настройки и миграции | `SettingsPage`, `SettingsService` → `/settings*`, `/admin/migrations*` | Server settings, embedded SQL, maintenance |
| Backup и restore | `BackupTab`, `BackupCatalog` → `SettingsService` → `/admin/backups*` | PostgreSQL, SeaweedFS, SMB, jobs и SSE операции |
| Административный аудит и outbox | Вкладки настроек → `AdminAuditLogService`, `OutboxAdminService` → `/admin/audit`, `/admin/outbox*` | Аудит, очередь и повторная доставка |
| Служебные события и логи | `server:event`, `auth:session-ended`; desktop logger → `/events`, `/telemetry/logs` | SSE resync, session lifecycle, Seq pipeline |

Таблица фиксирует наличие прямых цепочек на уровне групп. Для завершения
этапа требуется сопоставить каждый зарегистрированный HTTP маршрут и
каждый Wails метод с вызывающим кодом или самостоятельным внешним
потребителем, затем проверить обратный путь ответа до поля UI или эффекта.
Отдельно требуется пройти роли и права, чтобы не ошибочно удалить редкий
сценарий администратора или участника документооборота.

В `management.go` и `backup.go` на исходном commit найдено 117 мест
регистрации API-маршрутов; после HTTP-001 осталось 116. Один цикл
регистрирует три разных backup-маршрута, поэтому фактически сейчас есть
118 пар «HTTP-метод + путь»: 115 со статическим шаблоном и три динамические.
Отдельно обслуживаются `/api/v1/events`, `/health/live`, `/health/ready` и
обработчики `blocked_restore.go`.
Первичный поиск обнаружил буквальные части каждого из 115 статических
маршрутов в production `serverclient`. Дополнительное сравнение по
функциям клиента сопоставило HTTP-метод и части пути для 99 маршрутов.
Остальные 16 проверены вручную: `doSystemGET`, `getStatistics`,
`nomenclatureItemPath`, `settingPath`, `listAttachments` и `reconnectEvents`
передают путь и метод через общие функции. Сопоставление показывает наличие
клиентского вызова, но полезность ответа требует обратной трассировки данных.
Три динамических `POST /api/v1/admin/backups/catalog/{verify,restore,delete}`
составляются в `backupRoutes`; `serverclient/backup.go` посылает запрос
`POST /admin/backups/catalog/` + `kind`, а UI вызывает все три операции через
`BackupCatalog`. Остальные маршруты операций и каталога также имеют методы
клиента в том же файле. Обратная трассировка результатов маршрутов и
проверка редких ролей остаются задачами следующего прохода.

## Этап 3. Данные и события, первый проход

| Поток | Источник → передача → потребитель | Результат проверки |
| --- | --- | --- |
| Пользователь после входа | `/auth/login` → `dto.User` → `AuthService.Login` → `useAuthStore` | `isActive`, `failedLoginAttempts` и `department.nomenclatureIds` не читались из auth state (DATA-001); `department.id` и `department.name` нужны профилю |
| Текущие права | `currentAccessSummary` → `dto.CurrentAccessSummary` → Wails → `useCurrentAccessSummary` | Ряд полей ответа и локальных метаданных не доходил до решения UI (DATA-002); UI использует `sections`, `availableActions` и `systemPermissions`, а возможность регистрации выводит из действия `create` |
| Обычные события | outbox/user changes → `liveevents.Bus` → `/api/v1/events` → desktop `LiveEvent` → `server:event` | `user-events`, `backups` и `resync` имеют frontend listeners; `heartbeat` отбрасывается намеренно |
| Backup operation | backup service → `/admin/backups/operations/{id}/events` → desktop `LiveEvent.Operation` → `BackupCatalog`/`BackupJournal` | Полезная нагрузка операции имеет потребителей; capability stream живёт отдельно от login session |
| Срочные поручения дашборда | `DashboardRepository.GetExpiringAssignments` → `DashboardService.GetActivity` → `DashboardPage.ExpiringList` | SQL уже выбирал только поля экрана, но общий `dto.Assignment` добавлял к ответу неиспользуемые поля; DATA-003 заменил его узким DTO без изменения SQL |
| Списки вложений | `AttachmentRepository.GetByDocumentID` / `GetByAssignmentID` → серверный `MapAttachment` → HTTP → Wails → `FileListComponent` | UI читает `id`, `filename`, `fileSize`, `uploadedByName`, `uploadedAt`; SQL и DTO сокращены до этих полей (DATA-004). Для скачивания отдельный путь по ID сохраняет `storagePath`, `contentType` и проверку прав на сервере |
| Доставка персональных событий | Бизнес-транзакция → outbox `user_event` → worker → `UserEventRepository.CreateFromOutbox` → `liveevents.Bus` → SSE | Worker использует только ошибку вставки и ID получателя из исходного payload; `RETURNING id` и повторный SELECT удалены (DATA-005) |
| Список персональных событий | `user_events` → `UserEventRepository.GetList` → `dto.UserEvent` → `UserEventsButton` и обработчики обновления экранов | UI использует ID, документ, вид события, заголовок, текст и даты; `actorUserId`, `actorUserName`, `entityId`, `metadata` не читаются. DATA-006 сократил чтение и ответ; сохранённые столбцы и producer отдельно оценивать на этапе хранения |

Четыре типа outbox (`user_event`, `journal_entry`, `admin_audit`,
`attachment_delete`) имеют производителей и ветки обработки в worker.
`user_event` вызывает публикацию в SSE после сохранения; `journal_entry` и
`admin_audit` пишут соответствующие записи; `attachment_delete` удаляет
объект и завершает удаление метаданных. Осиротевшего типа на этом уровне
не обнаружено. Поля записи `user_events`, которые читаются из БД, но не
используются UI, требуют следующего прохода с учётом сохранённых записей.

Проверка не охватывает ещё SQL `SELECT`/`Scan`, все DTO и все типы outbox.
Следующий проход должен сравнить фактические поля по каждой крупной цепочке,
затем выделить избыточные SQL-загрузки и отброшенные результаты.

## Этап 4. Совместимость, первый проход

| Слой | Проверка | Решение сейчас |
| --- | --- | --- |
| Build compatibility | Сервер требует `buildProtocol=2`, ту же версию продукта и build identity; desktop отвергает устаревший ответ | Сохранить: защитный контракт для согласованного обновления текущих сборок |
| Списки документов | Cursor pagination используется четырьмя журналами; `page` и `totalCount` используются другими списками, включая поручения и административный аудит | Сохранить до раздельного разбора потребителей, это не чистый переходный слой |
| Database API | `database.DB.Query`, `QueryRow`, `Exec`, `Begin` помечены legacy, но вызываются репозиториями и задают timeout; `Prepare`/`PrepareContext` сейчас не вызываются (COMP-001) | Сохранить: удаление обёртки откроет встроенный метод `*sql.DB` без ограниченного таймаута |
| Backup audit | `backup/progress.go` сохраняет ключ terminal stage для записанных результатов | Сохранить: ключ участвует в дедупликации долгоживущего аудита |
| Backup formats | Каталог и restore принимают manifest format 3; пользователь подтвердил, что действующих архивов с данными нет | Принимать только текущую схему; неизвестный формат отклонять |
| Миграция при замене БД | `backup/replacement.go` выполнял `RunMigrations` после восстановления | Удалено: восстановление теперь допускает только архив текущей схемы; миграции живой БД в `database.DB` остаются |
| Конфигурация desktop | CWD fallback и `allowInsecureHttp` описаны как режимы локальной разработки | Сохранить до отдельного изменения поддерживаемого dev flow |

Само слово `legacy` в комментарии не означает лишний код. Подтверждение,
что у пользователей установлена только последняя версия, позволяет
сокращать Wails API без сохранения интерфейса для прежних desktop-сборок.
Сторонних клиентов HTTP API нет, поэтому лишние маршруты и поля ответа
допустимо удалить после проверки текущего desktop. Действующих архивов с
данными нет; проверка восстановления требует текущую версию схемы.

## Этап 5. Хранилище и эксплуатация, первый проход

- Теперь 13 пар миграций встроены через `go:embed`; backup сохраняет manifest
  формата 3. Формат 3 и миграция 012 добавлены в одном commit `3c89942`.
  После уточнения политики копий `PrepareRestore` требует текущую версию
  схемы, ветка `restoredSchema >= 12` и миграция при замене БД удалены.
  Восстановление новой копии проверено изолированным smoke-сценарием.
- `testing/compose/integration.yaml`, backup/Samba стек и production smoke
  подключены из `build/make/checks.mk` и соответствующих скриптов. На уровне
  целых файлов не найден осиротевший тестовый Compose или script.
- Для полного вывода о неиспользуемых колонках и индексах нужен отдельный
  SQL-инвентарь по каждой миграции, repository и backup allowlist.
  DATA-007 подтверждён точечным поиском SQL-ссылок; полный инвентарь ещё не сделан.
- В 12 `up`-миграциях обнаружено 37 создаваемых таблиц. Название каждой
  встречается хотя бы в одном production Go-файле сервера. Это исключает
  только очевидную таблицу без текстовых ссылок; не доказывает чтение данных
  или полезность каждого столбца и индекса.
- После DATA-006 `user_events.actor_user_id`, `entity_id` и `metadata`
  записывались из outbox, но не читались приложением. Индекс
  `idx_user_events_entity` не соответствовал ни одному текущему SQL-фильтру.
  Пользователь разрешил удалить исторические значения; миграция 013 удалила
  поля и индекс, а запись этих данных из outbox прекращена. Миграция 008
  остаётся неизменной. Откат 013 восстанавливает схему без значений полей.

## Этап 6. Тесты и моки, первый проход

- Все 14 generated mock-файлов в `internal/server/mocks` имеют вызовы
  соответствующего конструктора `mocks.New*` в Go-тестах. Целых
  неиспользуемых файлов в этом каталоге не обнаружено; отдельные методы
  mock-типов ещё надо сравнить с действующими портами.
- `OutboxRepository.Enqueue` вызывается только интеграционными тестами
  (подготовка событий), тогда как production-операции используют
  `EnqueueTx` внутри бизнес-транзакций. Это действующая тестовая опора:
  перенос в test helper потребует общего пакета для нескольких тестовых
  пакетов и не устранит саму реализацию вставки события. Пока оставить;
  при реорганизации testutil не считать метод пользовательским API.
- `integrationdb` используется многочисленными интеграционными тестами и
  `tools/dbperf`; `integrations3` используется проверками вложений и S3.
- Найдены ожидания, связанные с неиспользуемыми контрактами: TEST-001 и
  TEST-002. При удалении следует сохранять проверки роли и доступа,
  переписав ожидания через действующее поведение.

## Этапы 7–8. Первая группа исправлений

Первой группой исправлены DOC-001/002, ARC-001, DATA-001,
WAILS-001/002 и TEST-001. `AuthService.GetCurrentUser` сохранён внутри
Go как неэкспортируемый метод. После подтверждения отсутствия сторонних
HTTP-клиентов удалена серверная цепочка HTTP-001. По DATA-002 сокращены
DTO, frontend mapping и внутренние метаданные видов документов; права и
разделы по-прежнему вычисляет сервер. TEST-002 переписан на проверку
реально используемых действий и разделов.

Проверки: `go list -deps -test ./internal/desktop/app` больше не содержит
серверных пакетов; `make build-linux DOCFLOW_LOCAL_BUILD=1` успешно
регенерировал bindings и собрал клиент; `go test` для трёх затронутых
desktop-пакетов прошёл вне sandbox (внутри sandbox `httptest` не смог открыть
локальный порт); `npx tsc --noEmit` и три затронутых Vitest-файла (16 тестов)
прошли; `make docs-links-check` прошёл. Проверка TypeScript до исправления
DATA-001 обнаружила живое использование `department.id` в `ProfilePage`;
поле было сохранено и проверка повторно прошла.
После удаления неиспользуемых frontend-метаданных видов документов
`npx tsc --noEmit` повторно прошёл; в `frontend/src` больше нет ссылок на
`registrationFormCode`, `registryGroup` и `supportedActions`.
После сокращения HTTP-контракта узкие тесты `internal/server`,
`internal/server/services`, `internal/dto` и `internal/models` прошли.
`make build-linux DOCFLOW_LOCAL_BUILD=1` штатно обновил Wails bindings и
собрал клиент. После удаления дублирующего `canRegister` повторно прошли
сценарий прав сервера, сборка клиента и 4 теста `accessVisibility`.
`make docs-links-check` и `git diff --check` прошли. Для SQL и backup-кода
в первой группе изменений не было, поэтому интеграционный стек тогда не
запускался.

Для COMP-002, COMP-003 и TEST-003 прошли узкие тесты backup и состояния
восстановления, а также проверка типов frontend. Тесты формы отката миграции
прошли после замены устаревшего примера DOC-003.
`bash testing/scripts/backup-integration.sh` повторно выполнил на финальном
коде создание копии текущей схемы 012, административное восстановление,
проверку вложений и восстановление через сброс окружения. Все сценарии
прошли; изолированные контейнеры и тома удалены.
Для DATA-003 прошёл `TestDashboardService_GetActivity`, проверяющий поля
ответа дашборда и отсутствие лишних JSON-полей. Штатная команда
`make build-linux DOCFLOW_LOCAL_BUILD=1` регенерировала Wails bindings и
собрала desktop с новым типом `DashboardAssignment`.
После DATA-004 затронутые пакеты `internal/dto`, `internal/server/services`,
`internal/desktop/services`, `internal/desktop/serverclient` и
`internal/server` прошли точечные тесты. Устаревшие fixture TEST-004 были
найдены компиляцией тестовых пакетов и исправлены. Повторная штатная сборка
Linux обновила модель `Attachment` в bindings и собрала frontend/desktop.
Для изменения `SELECT`/`Scan` прошёл
`TestAssignmentSeriesLifecycleIntegration`, проверивший оба списка на
изолированной PostgreSQL. Стек и его тома после теста удалены.
Для DATA-005 прошли точечные тесты вставки события и повторной доставки
из outbox; проверка гарантирует отсутствие лишнего SELECT в успешной
outbox-ветке. `TestUserEventOutboxInsertIntegration` подтвердил на
изолированной PostgreSQL, что два выполнения с одним ключом оставляют
ровно одно событие. Стек и тома после теста удалены.
Для DATA-006 и TEST-005 прошли точечные Go-тесты DTO, repository, service и
outbox. Штатная Linux-сборка регенерировала Wails bindings и собрала
frontend/desktop. На изолированной PostgreSQL прошли
`TestUserEventOutboxInsertIntegration` и
`TestWorkflowAPIPersistsAcknowledgmentAndScopesUserEventsIntegration`,
подтвердившие чтение сокращённого списка и изоляцию событий по получателю.
Стек и тома удалены. Колонки `actor_user_id`, `entity_id`, `metadata` и
связанный индекс БД не удалялись: их назначение как долговременных данных
следует оценить отдельно от HTTP-ответа.

## Последующие этапы

Завершить точное сопоставление всех HTTP-маршрутов и Wails-методов с
потребителями, затем проследить поля ответов до UI, SQL `SELECT`/`Scan`
и outbox payload. Отдельно сравнить mock-методы с действующими портами,
инвентаризировать колонки и индексы миграций. Перед реальными релизами со
сохранёнными архивами пересматривать политику их совместимости.
Для каждой новой находки заполнять реестр до изменения кода.
