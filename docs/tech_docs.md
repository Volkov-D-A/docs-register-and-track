# Техническая документация проекта

Дата обновления: 2026-08-30
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

Этот документ фиксирует правила, которые нужно учитывать при будущих изменениях. Источником фактического поведения остаются код, миграции и manifest-файлы; при расхождении техническая документация обновляется вместе с кодом. Операционные заметки намеренно консолидированы в небольшом наборе файлов в `docs/`, отдельный каталог audit/runbook в репозитории не поддерживается.

## Технологический Стек

Backend:

- Go module `github.com/Volkov-D-A/docs-register-and-track`;
- Go `1.26.5`;
- Wails v2.13.0;
- PostgreSQL через `database/sql`, `lib/pq`;
- миграции через `golang-migrate`;
- MinIO через `minio-go`;
- structured logging через `slog` и Seq;
- тесты: Go `testing`, `testify`, `go-sqlmock`.

Frontend:

- React 19.2.7;
- TypeScript 6.0.3;
- Vite 8.1.4;
- тесты: Node test runner для utilities, Vitest + jsdom + React Testing Library
  для component/integration-сценариев;
- Ant Design 6.5.1;
- Zustand 5.0.14;
- dayjs 1.11.21;
- `@xyflow/react` 12.11.2 для графа связей;
- `@ant-design/plots` 2.6.8 для статистики.

Инфраструктура и сборка:

- Wails CLI v2;
- Makefile как основной entrypoint для dev/build/release checks;
- Docker Compose для локальных PostgreSQL, MinIO, Seq, `docflow-server` и Caddy;
- Linux `amd64` и Windows `amd64` являются production target. macOS не входит в текущий release target.

## Высокоуровневая Архитектура

```text
Wails desktop app
├── main.go
│   ├── загружает config
│   ├── инициализирует slog/Seq
│   ├── встраивает frontend и release notes
│   └── запускает Wails с options из internal/app
│
├── cmd/docflow-server/
│   └── standalone process для централизованной обработки outbox
│
├── internal/
│   ├── background/    общий schema-dependent lifecycle
│   ├── app/           composition root, Wails bindings и shutdown
│   ├── config/        config loading, encrypted secrets
│   ├── database/      PostgreSQL connection, embedded migrations
│   ├── models/        domain entities, requests, app errors
│   ├── dto/           frontend-facing mapping
│   ├── repository/    SQL persistence and transactions
│   ├── services/      auth, permissions, business workflows, Wails API
│   ├── storage/       MinIO object storage
│   ├── outbox/        delivery worker для событий и удаления файлов
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
├── docs/              технический справочник, замечания и release notes
├── scripts/           общие helpers backup/restore
└── tools/             release gate, release generator и DB performance tooling
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
- `frontend/src/utils/latestRequest.ts` - защита состояния от устаревших async-ответов;
- `frontend/src/features/settings/ReferenceDirectoriesTab.tsx` - вынесенный feature component справочников.

Журналы документов используют cursor pagination: backend возвращает `nextCursor` и `hasMore`, а hook хранит историю курсоров для кнопок «Назад» и «Вперёд». Любое изменение фильтра, поиска или размера страницы сбрасывает историю. Поле `cursorPagination` включает этот режим; старый `page`-контракт остаётся для внутренних вызовов, которые ещё нуждаются в номере страницы и `totalCount`.

Для защищённых backend-операций `AuthService.RequireAuthenticated` и `GetCurrentUserUUID` используют лёгкий `SessionPrincipal`; полный `GetCurrentUser` следует вызывать только когда действительно нужны данные профиля, подразделения или участие в документообороте.

Крупные страницы (`SettingsPage`, `StatisticsPage`, `DocumentViewModal`, `AssignmentsPage`) нужно декомпозировать постепенно при функциональных изменениях. Не делать большой refactor без поведенческой причины и smoke/test coverage.

## Слой Wails Bridge

Wails bridge:

- serializes calls между React и Go;
- exposing происходит через `Bind` в `internal/app/app.go`;
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

После изменения public Go service signatures нужно выполнить
`make wails-bindings`, проверить и закоммитить изменения в `frontend/wailsjs`,
а затем запустить `make wails-bindings-check` и frontend build.

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
- dirty/newer schema state must block unsafe use;
- startup connect/ping ограничен контекстом; обычные SQL-операции имеют deadline;
- PostgreSQL connections используют `connect_timeout`, `statement_timeout` и `lock_timeout`.

Критичные migration rules:

- migrations лежат в `internal/database/migrations`;
- runtime UI migration management сохраняется в production для пользователя с `admin`;
- schema-dependent background services запускаются общим lifecycle только при `UpToDate` и совместимой схеме;
- desktop UI не исполняет SQL миграций и обращается к management API
  `docflow-server`;
- перед изменением схемы серверный lifecycle останавливает worker и освобождает
  его session-level PostgreSQL advisory lease; handler затем получает тот же
  lease как migration lock;
- после успешного apply сервер повторно проверяет `UpToDate`/compatibility и
  запускает worker без рестарта процесса или контейнера;
- rollback считается destructive operation;
- rollback требует fresh PostgreSQL+MinIO backup, backup reference, data-loss acknowledgment, control phrase and audit entries;
- после успешного rollback сервер остаётся в maintenance, а desktop maintenance
  gate блокирует обычные защищённые операции до повторного apply;
- в режиме обслуживания администратору остаются доступны аутентификация, статус миграций и их применение для восстановления;
- older binary against newer DB schema must be blocked;
- dirty schema means stop using app and follow recovery procedure.

## Слой Storage

MinIO хранит physical attachment objects. PostgreSQL хранит attachment metadata.

Правила:

- upload сначала пишет объект, затем metadata и journal-outbox; ошибка БД запускает compensating delete объекта;
- delete сначала атомарно скрывает metadata и ставит `attachment_delete` в outbox, worker повторяет удаление MinIO и финализацию строки;
- при рассинхронизации восстанавливать PostgreSQL и MinIO только из согласованного backup-набора;
- размер и расширения задаются системными настройками; fallback: 15 MB и `.pdf,.doc,.docx,.odt,.xls,.xlsx,.ods`;
- attachment downloads to local disk must not overwrite existing files;
- MinIO startup bucket check has timeout;
- системная статистика читает persisted storage snapshot из PostgreSQL; точная
  сверка MinIO выполняется только через `RefreshStorageUsage`, координируется с
  attachment mutations и сохраняет новый snapshot после полного scan;
- file operations must participate in operation lifecycle cancellation.

## Operation Lifecycle И Shutdown

Долгие backend operations используют shared `OperationLifecycle`:

- app root context;
- per-operation timeout;
- shutdown cancel/wait coordination;
- Wails `OnShutdown` сначала останавливает schema-dependent background services,
  затем отменяет/ждет active operations и закрывает DB/logger.

Отдельный schema lifecycle управляет outbox worker и другими фоновыми задачами,
которым нужна полностью актуальная схема. Он обеспечивает единственный экземпляр
worker, останавливает его перед rollback и включает maintenance gate для обычных
защищённых операций. Повторная успешная миграция снимает gate и запускает worker
без рестарта приложения.

Lifecycle реализован в `internal/background` и используется Wails composition
root как schema maintenance gate, а standalone `docflow-server` — как lifecycle
реального worker. Desktop composition root не создаёт outbox consumer:
repositories продолжают записывать transactional events, а единственный
consumer находится в `docflow-server` и читает ту же таблицу `event_outbox`
общей PostgreSQL. Переключение production выполняется централизованно только
после закрытия уже запущенных процессов предыдущей desktop-версии.

На время работы consumer удерживает отдельный PostgreSQL advisory lease на
выделенном соединении. Management API сначала останавливает consumer и
освобождает lease, затем использует его как межпроцессную блокировку изменения
схемы. Второй server-worker или конкурентная миграция получить lease не могут.

`docflow-server` предоставляет команды `run`, `check-config`, `healthcheck` и
`version`, а также liveness/readiness и административный API миграций. Он проверяет актуальность embedded migrations,
подключение к PostgreSQL и MinIO, использует graceful shutdown и отправляет
в Seq только значимые operational events, warnings и errors. Периодические
metric snapshots в operational log не отправляются. Desktop всё ещё напрямую
работает с PostgreSQL и MinIO для бизнес-операций, но управление миграциями уже
переведено на HTTP API сервиса.

Container image собирается через `build/server/Dockerfile` на distroless runtime
под непривилегированным пользователем. В image не копируются production config
и secrets; настройки передаются при запуске через env-файл или механизм
оркестратора; JSON-конфигурацию сервер не читает. `docker-compose.yaml` всегда
загружает `hehelf/docflow-service:${DOCFLOW_SERVER_VERSION}` из Docker Hub,
запускает его рядом с PostgreSQL, MinIO, Seq и Caddy и ожидает readiness PostgreSQL.
Версия задаётся в `.env` рядом с версиями остальных контейнеров. Сборка
исходников на production host не выполняется. Пустая схема bootstrap-ится
сервером автоматически; при обновлении существующей схемы процесс остаётся
живым в maintenance и ждёт команды администратора. Docker healthcheck проверяет
`/health/live`, а `/health/ready` остаётся 503 до готовности схемы и зависимостей.
`make
docker-server-build` создаёт локальный versioned tag, а `make
docker-server-push` после отдельного `docker login` собирает и публикует image и
immutable `DOCFLOW_SERVER_VERSION` из `.env`. Repository
`hehelf/docflow-service` жёстко задан в Makefile и Compose. Makefile не принимает
Docker Hub token.

Management endpoints:

- `GET /health/live` — процесс и HTTP listener работают;
- `GET /health/ready` — схема, PostgreSQL и MinIO готовы к обычной работе;
- `GET /api/v1/admin/migrations` — status embedded/schema versions;
- `POST /api/v1/admin/migrations/apply` — apply всех ожидающих миграций;
- `POST /api/v1/admin/migrations/rollback` — rollback одной миграции с backup
  reference, data-loss acknowledgement и контрольной фразой.

Apply и rollback повторно проверяют текущий пароль активного пользователя с
системным правом `admin`; одинаковые ошибки авторизации не раскрывают причину и
ограничиваются по частоте в памяти процесса. Пароль передаётся через HTTP Basic
только на время вызова и не входит в JSON или audit. Compose размещает Caddy
перед сервисом: наружу публикуется Caddy, а порт `docflow-server` остаётся только
во внутренней сети. До подключения сертификатов допускается явно включаемый
временный режим desktop `server.allowInsecureHttp=true`. Он не является
production-защитой и допустим только в доверенной изолированной сети. Штатная
схема — HTTPS, `allowInsecureHttp=false` и проверка certificate через системный
trust store.

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
- `document_journal` и `admin_audit_log` хранятся весь жизненный цикл проекта и не удаляются приложением;
- обязательные journal/admin-audit эффекты доставляются через transactional outbox с retry, terminal failure и административным requeue.
- deduplication key идентифицирует конкретную операцию, а не только сущность и
  конечное состояние; для повторяемых переходов используется отдельный UUID;
- повторное enqueue допустимо только при совпадении key, event type и JSON
  payload; несовпадающая коллизия должна откатывать бизнес-транзакцию.
- обработанные outbox-события хранятся 90 дней; worker очищает их пакетами по
  1000 строк при запуске и затем раз в час, не затрагивая pending, processing и
  failed;
- пятисекундный мониторинг считает только активную очередь, а количество
  processed за retention-период запрашивается административным экраном;
- после очистки outbox-строки постоянную идемпотентность journal, audit и user
  events обеспечивают уникальные deduplication keys в целевых таблицах;
  диагностика несовпадающего payload ограничена 90-дневным retention-окном.

## Конфигурация И Секреты

`docflow-server` читает подключения, Seq и параметры outbox исключительно из
runtime environment. PostgreSQL и MinIO используют те же credentials, которыми
инициализируются контейнеры; отдельные service accounts на текущем этапе не
создаются. Полный перечень и локальные примеры приведены в `.envExample`.
JSON-файл серверу не требуется.

Desktop config lookup order:

```text
DOCFLOW_CONFIG_PATH
<executable directory>/config/config.json
<current working directory>/config/config.json
```

Production desktop должен использовать `DOCFLOW_CONFIG_PATH` или
executable-relative `config/config.json`. CWD fallback предназначен для local
development.

Secrets:

- production secrets never committed;
- `ENCRYPTION_KEY` читается из runtime environment или подставляется в бинарник через ldflags;
- PostgreSQL/MinIO secrets should use `ENC:` encrypted values;
- `ENCRYPTION_KEY` currently embedded through Go ldflags, so release artifacts are sensitive;
- `.env`, `config.json`, `/etc/docflow/backup.env` и CIFS credentials file должны иметь `0600` или эквивалентный строгий ACL;
- generated release evidence and logs must not contain passwords, tokens or full encrypted secret material.

Example configs:

- `.envExample`, `config.example.json`, `docker-compose.yaml` are local/dev only;
- localhost endpoints, disabled TLS and weak sample passwords are not production defaults.

## Business Rules

### Authentication

- Любая доменная операция требует authenticated user.
- Desktop login/logout/current-user и оба сценария смены пароля выполняются
  через `docflow-server`;
- server session — opaque bearer token, в PostgreSQL хранится только SHA-256
  hash, raw token живёт только в памяти desktop-процесса;
- auth API включает `login`, `logout`, `me`, `change-password` и
  `change-required-password`;
- смена или административный сброс пароля и деактивация пользователя атомарно
  отзывают все его server sessions; после собственной смены пароля desktop
  требует повторный вход;
- при административном сбросе пароль не вводится вручную: backend генерирует
  временный пароль, UI показывает его один раз, а следующий вход требует смены;
- срок сессии задаётся `DOCFLOW_AUTH_SESSION_TTL_HOURS` (по умолчанию 12 часов);
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

### Поручения и регулярные серии

Регулярное поручение хранится как шаблон `assignment_series` и обычные строки
`assignments`, связанные через `series_id`. В общем списке отображается только
итерация, на которую указывает `current_assignment_id`; принятые итерации
доступны менеджеру в истории серии.

Жизненный цикл:

1. Создание серии атомарно создаёт шаблон и первую итерацию со статусом `new`.
   Для неё `deadline` и `planned_deadline` равны первому плановому сроку.
2. Исполнитель выполняет текущую итерацию обычными переходами поручения.
3. При переходе менеджером из `completed` в `finished` текущая итерация
   принимается и, если серия активна, в той же транзакции создаётся следующая.
4. Изменение шаблона влияет только на ещё не созданные итерации. Текст,
   исполнители и срок текущего поручения автоматически не меняются.
5. Отмена серии сохраняет текущую итерацию и историю, но после её принятия не
   создаёт следующую.

Календарные правила:

- `same_day` применяется к интервалам в днях и неделях;
- `fixed` задаёт число месяца для интервалов в месяцах и годах; если такого
  числа в целевом месяце нет, используется последний день этого месяца;
- `last_day` всегда выбирает последний календарный день целевого месяца;
- следующая дата рассчитывается от `planned_deadline`, поэтому ручное
  изменение рабочего `deadline` текущей итерации не сдвигает всю серию.

Управление параметрами, отмена и просмотр полной истории требуют document
action `assign`. Исполнитель или активный заместитель работает только с
текущей итерацией. Файлы исполнения разрешены настройкой
`assignment_completion_attachments_enabled` и для исполнителя принимаются
только пока поручение имеет статус `in_progress`. Перед записью файла backend
повторно проверяет доступ, а SQL-вставка атомарно подтверждает текущую итерацию
и допустимый статус.

Создание серии, продвижение указателя, поручения, соисполнители, journal и
user-event outbox фиксируются одной PostgreSQL-транзакцией. Устаревшее
редактирование или конкурентная приёмка возвращают `409 Conflict`; клиент
должен перечитать поручение или серию и повторить осознанное действие.

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

Должны оставаться атомарными внутри PostgreSQL:

- first-run setup после применения миграций: admin user и `admin` permission;
- document registration: idempotency check, number allocation, `next_number`, `documents`, detail table, children, journal;
- document update with child data;
- assignment create/update with co-executors;
- recurring assignment create, template update, cancellation and iteration advance;
- acknowledgment create/update with user list;
- full replacement of user access profile;
- attachment metadata и journal event после успешной загрузки объекта;
- attachment delete intent и outbox event.

Операции между PostgreSQL, MinIO и SMB не могут быть общей SQL-транзакцией. Для них используются compensation, outbox/saga и атомарная публикация backup archive + manifest.

## Идемпотентность

Должны быть idempotent or safe to repeat:

- повторная доставка одной outbox-записи не создаёт второй journal/audit effect;
- отдельные повторяемые бизнес-переходы получают разные outbox keys;

- migrations `Up()` when no change;
- MinIO bucket creation/check on startup;
- organization/resolution executor find-or-create;
- document registration by `idempotency_key`;
- saving existing system setting value;
- marking release notes as viewed;
- saving theme;
- fetching lists/cards/statistics.

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

Новые термины следует сверять с существующими подписями и этим разделом; отдельный audit-глоссарий в репозитории не поддерживается.

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
npm run lint
npm run build
```

`npm test` сначала запускает типизированные utility-тесты из `frontend/test`,
затем React component/integration-тесты из `frontend/test/components`. Для
последних используются Vitest, `jsdom`, React Testing Library и mock Wails
bridge из `frontend/test/componentTestUtils.tsx`; snapshots разметки не являются
основой regression coverage.

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
make go-test
make go-vet
```

## Database Development Rules

- New schema changes require migrations in `internal/database/migrations`.
- Migrations must be embedded and compatible with release build.
- For constraints/index changes, add focused tests where practical.
- Do not add performance indexes just because a query has a seq scan on small baseline data.
- Для production-like планов используйте `make db-performance-check`; по умолчанию он создаёт 10 000 документов, сравнивает глубокие OFFSET- и cursor-страницы поиска и сохраняет полный JSON-план. Размер и pagination-сценарий можно изменить через `PERFORMANCE_DOCUMENTS`, `PERFORMANCE_PAGE_SIZE` и `PERFORMANCE_DEEP_PAGE`, например `make db-performance-check PERFORMANCE_DOCUMENTS=50000 PERFORMANCE_DEEP_PAGE=1000`.
- Any new performance index needs before/after `EXPLAIN (ANALYZE, BUFFERS)` and write-latency consideration.
- Keep rollback impact explicit for destructive `down` migrations.

## Backup, Restore И Recovery

Backup/restore contract:

- PostgreSQL and MinIO are backed up together;
- Seq logs are excluded;
- целевые RPO 1 день и RTO 1-2 дня требуют подтверждения production-процессом;
- retention: 15 days;
- offsite copy is handled by the approved production process.

Scripts:

- `backup_smb_tar.sh`;
- `restore_smb_tar.sh`.

Rules:

- scripts строго разбирают root-owned `/etc/docflow/backup.env` либо путь из `DOCFLOW_BACKUP_ENV_FILE`, не исполняя файл как shell-код;
- backup config и `SMB_CREDENTIALS_FILE` должны находиться вне Git, не быть symlink и иметь режим `0600`;
- archive и manifest сначала создаются под скрытыми временными именами на том же SMB mount, сбрасываются через `fsync` и публикуются атомарным rename;
- manifest v1 является commit marker и содержит имя архива, время, размер, SHA-256, имя БД и bucket;
- restore до распаковки проверяет безопасное имя, manifest, размер, SHA-256 и `tar -tzf`;
- legacy archive без manifest доступен только с явным `--allow-legacy-without-manifest`;
- restore must validate PostgreSQL before mirroring MinIO;
- if PostgreSQL restore/validation fails, MinIO restore must not run;
- release requires manual PostgreSQL+MinIO test restore evidence;
- текущая передача части MinIO credentials в Docker shell command остаётся известным security debt из `docs/bugs.md`.

## Release And Versioning

Version source:

- `docs/releases.yaml`;
- generated `internal/releaseassets/current_release.yaml`;
- Wails product metadata in `wails.json`.

Release must be from a clean worktree. Before production approval:

- release gate output должен быть сохранён вне репозитория;
- checksum артефактов, target OS smoke и backup/restore test должны быть подтверждены production-процессом;
- clean `git status --short` at tag.

Current release gate:

```bash
make release-gate
```

It runs/checks:

- required `ENCRYPTION_KEY`;
- generated release asset freshness;
- Go tests;
- PostgreSQL integration tests on a disposable Docker Compose database;
- Go vet;
- `govulncheck`;
- `npm ci`;
- frontend lint/test/build;
- `npm audit --audit-level=critical`;

Для `release-gate` обязательны Docker с Compose и заданный
`POSTGRES_VERSION`. Gate не запускает DB performance benchmark, target OS
smoke или backup restore: это отдельные автоматические/ручные проверки.

## Supported Make Targets

Common targets:

- `make storage-up` - start local PostgreSQL/MinIO/Seq;
- `make storage-down` - stop local services without deleting volumes;
- `make storage-reset` - destructive local reset;
- `make dev` - Wails dev;
- `make release-assets` - generate embedded release assets;
- `make release-assets-check` - verify generated release assets;
- `make check-release-env` - проверить наличие release key;
- `make check-integration-env` - проверить Docker Compose и версию PostgreSQL;
- `make go-test`;
- `make go-vet`;
- `make integration-test`;
- `make frontend-ci`;
- `make frontend-build`;
- `make frontend-lint`;
- `make frontend-test`;
- `make db-performance-check`;
- `make npm-audit`;
- `make govulncheck`;
- `make release-gate`;
- `make build-linux`;
- `make build-windows`.

`make integration-test` проверяет prerequisites, запускает изолированный
PostgreSQL из `docker-compose.integration.yaml`, передаёт безопасный
`DOCFLOW_INTEGRATION_DSN` для `docflow_test_outbox` и после тестов всегда
удаляет контейнер и volume. Этот target входит в обязательный `release-gate`.
Для ручной отладки доступны `make integration-db-up` и
`make integration-db-down`.

Integration coverage composition root собирает Wails options с реальной
тестовой PostgreSQL и поддельным object storage, без запуска GUI и MinIO. Тест
проверяет состав bindings, отсутствие outbox consumption в desktop, сохранение
pending event, закрытие database pool и logger callback через `OnShutdown`.
Отдельный server integration test проверяет обработку события без Wails.
Production `NewWailsOptions` использует те же package-private фабрики с
реальными PostgreSQL, MinIO и theme service.

## Performance Budgets

Текущие целевые ориентиры производительности (не являются автоматическими release gates):

- startup to login: <= 5 seconds;
- main list open/search/filter: <= 2 seconds;
- document registration save: <= 2 seconds typical;
- heavy statistics/report open: <= 5 seconds;
- normal desktop memory: <= 512 MB;
- binary warning threshold: 100 MB;
- route chunk warning threshold: 1.6 MB.

Расчётные эксплуатационные предположения, требующие проверки на целевом контуре:

- up to 1000 documents/year;
- up to 20 users;
- attachments proportional to documents;
- average file around 3 MB;
- storage around 1 TB;
- storage warning 80%, critical 90%.

Performance evidence:

- `make db-performance-check` формирует локальные результаты в `build/performance/`;
- target OS timings и принятые evidence хранятся вне репозитория согласно production-процессу.

## Testing Strategy

Go:

- `make go-test`;
- focused unit tests in services/repositories/database;
- PostgreSQL integration tests through обязательный release-gate stage
  `make integration-test`;
- database constraints and idempotency covered этим integration stage.

Frontend:

- TypeScript compile;
- Node test runner for helper tests;
- production build smoke for Vite `dist`;
- ESLint gate.

Release evidence:

- output `make release-gate`;
- отдельный output `make db-performance-check` при необходимости;
- ручной backup/restore test и target OS install smoke.

Rule: a change is not production-ready just because local unit tests pass. Release-impacting changes должны отражаться в поддерживаемой документации, если меняют поведение оператора, recovery procedure или состав проверок.

## Security And Dependency Rules

- Keep `govulncheck` in release gate.
- Keep `npm audit --audit-level=critical` in release gate.
- Dependency inventories и license review при необходимости выполняются отдельным production-процессом; текущий `release-gate` их не генерирует.
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

Текущая согласованная версия release metadata: `1.0.6` в `docs/releases.yaml`, generated release asset и `wails.json`.

Актуальные результаты ревью и статусы исправлений ведутся в `docs/bugs.md`; этот справочник не утверждает отсутствие открытых проблем. Production approval требует clean-worktree release gate, target OS smoke и реального backup/restore test.

## Practical Change Checklist

Before starting:

- identify affected layer: frontend, service, repository, storage, migration, release/ops;
- check this document, `docs/bugs.md` и относящиеся к изменению инструкции;
- prefer existing patterns and local helpers.

Before finishing:

- run focused tests for changed code;
- run `git diff --check`;
- for frontend behavior, run `npm test`, `npm run lint`, `npm run build`;
- for backend behavior, run `make go-test` and focused package tests;
- update maintained docs if behavior, recovery or operator actions changed;
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
