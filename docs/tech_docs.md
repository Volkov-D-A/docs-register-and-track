# Техническая документация проекта

Дата обновления: 2026-10-14
Статус: основной справочник для дальнейшей разработки

## Назначение

`docs-register-and-track` собирает desktop-приложение Wails `docflow` с заголовком окна "Система регистрации документов". Приложение предназначено для регистрации, поиска, просмотра и сопровождения документооборота организации.

Основные доменные зоны:

- регистрация входящих писем, исходящих писем, обращений граждан и приказов;
- номенклатура дел, подразделения, организации-корреспонденты, исполнители резолюций;
- контроли сроков, соисполнители и регулярные серии;
- внешнее ознакомление с приказами;
- связи между документами и граф связей;
- журнал действий по документам и административный аудит;
- статистика по документам, контролям и системе;
- отчёты за период, экспорт XLSX/PDF на компьютер пользователя.

Этот документ фиксирует правила, которые нужно учитывать при будущих изменениях. Источником фактического поведения остаются код, миграции и manifest-файлы; при расхождении техническая документация обновляется вместе с кодом. Операционные заметки намеренно консолидированы в небольшом наборе файлов в `docs/`, отдельный каталог audit/runbook в репозитории не поддерживается.

## Технологический Стек

Актуальные версии зависимостей определяются `go.mod`, `frontend/package.json` и
`frontend/package-lock.json`.

Backend:

- Go module `github.com/Volkov-D-A/docs-register-and-track`;
- Go (версия в `go.mod`);
- Wails v2;
- PostgreSQL через `database/sql`, `lib/pq`;
- миграции через `golang-migrate`;
- structured logging через `slog` и Seq;
- тесты: Go `testing`, `testify`, `go-sqlmock`.

Frontend:

- React;
- TypeScript;
- Vite;
- тесты: Node test runner для utilities, Vitest + jsdom + React Testing Library
  для component/integration-сценариев;
- Ant Design;
- Zustand;
- dayjs;
- `@xyflow/react` для графа связей;
- `@ant-design/plots` для статистики.

Инфраструктура и сборка:

- Wails CLI v2;
- Makefile как основной entrypoint для dev/build/release checks;
- Docker Compose для локальных PostgreSQL, Seq, `docflow-server` и Caddy;
- Linux `amd64` и Windows `amd64` являются production target. macOS не входит в текущий release target.

## Высокоуровневая Архитектура

Разделение desktop и сервера завершено 14 сентября 2026 года. Desktop отвечает
за Wails API, локальные пользовательские файлы, интерфейс ОС и HTTP-клиент;
сервер — за аутентификацию, права, бизнес-операции, БД и outbox.
Desktop не импортирует server, server не импортирует desktop и Wails, общие
пакеты не зависят от обеих сторон.

Wails публикует desktop-сервисы без служебных setters, Startup и LogAction. Виды документов используют только системные коды `incoming_letter`,
`outgoing_letter`, `citizen_appeal`, `administrative_order`; сокращённые коды
`incoming`/`outgoing` остаются только ключами страниц интерфейса. Актуальный контракт хранится в
`internal/desktop/app/testdata/wails-api.json`; тесты проверяют Go/JS/TS и
совпадение регистраций runtime и генератора.

Приёмка включает Go-тесты, vet, race, PostgreSQL integration, frontend lint/test/build,
генерацию bindings, сборки server без CGO и desktop Linux/Windows, а также SMB smoke
и database-only backup/restore v4. Фактически выполненные проверки новой модели
и оставшиеся ограничения записываются в [журнал изменений](document-deadline-control-changes.md).

```text
Wails desktop app
├── main.go
│   ├── загружает config
│   ├── инициализирует slog и доставку логов через HTTP
│   ├── встраивает frontend и release notes
│   └── запускает Wails с options из internal/desktop/app
│
├── cmd/docflow-server/
│   └── HTTP API, migrations, backup и outbox worker
│
├── internal/
│   ├── desktop/
│   │   ├── app/       composition root, Wails bindings и shutdown
│   │   ├── serverclient/ HTTP, сессии, SSE и отмена запросов
│   │   ├── config/    локальный адрес HTTP API
│   │   ├── logging/   Wails adapter и HTTP-доставка логов
│   │   ├── operations/ жизненный цикл клиентских операций
│   │   └── services/  Wails API и HTTP-адаптеры
│   ├── server/
│   │   ├── background/ lifecycle фоновых задач и проверка схемы
│   │   ├── config/    PostgreSQL, backup, Seq и outbox из окружения
│   │   ├── logging/   серверная настройка slog и доставка в Seq
│   │   ├── observability/ метрики серверных операций
│   │   ├── services/  права и бизнес-операции
│   │   ├── database/  PostgreSQL connection и embedded migrations
│   │   ├── repository/ SQL persistence и транзакции
│   │   ├── backup/    архивы, восстановление и SMB
│   │   ├── liveevents/ SSE-уведомления
│   │   ├── mocks/     тестовые реализации серверных портов
│   │   ├── outbox/    доставка событий
│   │   ├── coordination/ блокировки миграций и резервирования
│   │   ├── security/  пароли и токены
│   │   └── testutil/  PostgreSQL для интеграционных тестов
│   ├── models/        domain entities, requests, app errors
│   ├── dto/           frontend-facing mapping
│   └── shared/        общие attachmentname, buildinfo, logger, releaseassets и startupdiag
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

- обращаться к PostgreSQL напрямую;
- принимать решения авторизации вместо backend;
- зависеть от raw Go/PostgreSQL error text;
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
- exposing происходит через `Bind` в `internal/desktop/app/app.go`;
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

Bindings автоматически генерируются Wails при `make dev-client`, `make build-linux`
и `make build-windows` через обычный `main.go` клиента. Для генерации нужен
корректный клиентский `config.json`; работающий сервер не требуется.
После изменения public Go service signatures проверьте и закоммитьте
полученные изменения в `frontend/wailsjs`.

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

`internal/server/repository` владеет SQL и транзакциями. `internal/server/database` владеет connection/migrations/schema status.

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

- migrations лежат в `internal/server/database/migrations`;
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
- rollback требует fresh PostgreSQL backup, backup reference, data-loss acknowledgment, control phrase and audit entries;
- после успешного rollback сервер остаётся в maintenance, а desktop maintenance
  gate блокирует обычные защищённые операции до повторного apply;
- в режиме обслуживания администратору остаются доступны аутентификация, статус миграций и их применение для восстановления;
- older binary against newer DB schema must be blocked;
- dirty schema means stop using app and follow recovery procedure.

## Пользовательские файлы

Документные вложения, маршруты загрузки/скачивания, их настройки и объектное
хранилище удалены. Сохраняется бумажный реквизит количества листов приложений.
Отчёты XLSX/PDF сохраняются desktop на выбранный пользователем путь; существующий
файл не перезаписывается, незавершённая загрузка удаляется. SMB хранит серверные
резервные копии PostgreSQL, а не вложения документов.

## Operation Lifecycle И Shutdown

Долгие backend operations используют shared `OperationLifecycle`:

- app root context;
- per-operation timeout;
- shutdown cancel/wait coordination;
- Wails `OnShutdown` останавливает приём серверных событий, отменяет/ждёт
  активные операции desktop и закрывает logger. Подключениями к БД управляет сервер.

Отдельный schema lifecycle управляет outbox worker и другими фоновыми задачами,
которым нужна полностью актуальная схема. Он обеспечивает единственный экземпляр
worker, останавливает его перед rollback и включает maintenance gate для обычных
защищённых операций. Повторная успешная миграция снимает gate и запускает worker
без рестарта приложения.

Lifecycle реализован в `internal/server/background` и используется `docflow-server`
для управления worker. Desktop проверяет готовность через `SystemService` и
`SystemBootstrapGate`; доступ к операциям во время обслуживания контролирует сервер.
Локальной копии schema maintenance gate в desktop нет.
Desktop composition root не создаёт PostgreSQL repositories,
outbox consumer. Команды и transactional events фиксируются
сервером, а единственный consumer читает `event_outbox` внутри server process.

На время работы consumer удерживает отдельный PostgreSQL advisory lease на
выделенном соединении. Management API сначала останавливает consumer и
освобождает lease, затем использует его как межпроцессную блокировку изменения
схемы. Второй server-worker или конкурентная миграция получить lease не могут.

`docflow-server` предоставляет команды `run`, `check-config`, `healthcheck` и
`version`, а также liveness/readiness и административный API миграций. Он проверяет актуальность embedded migrations,
подключение к PostgreSQL, использует graceful shutdown и отправляет
в Seq только значимые operational events, warnings и errors. Периодические
metric snapshots в operational log не отправляются. Desktop выполняет все
business и migration operations через HTTP API сервиса.

Container image собирается через `build/server/Dockerfile` на runtime PostgreSQL 18 Alpine
под непривилегированным пользователем, со штатными `pg_dump` и `pg_restore`. В image не копируются production config
и secrets; настройки передаются при запуске через env-файл или механизм
оркестратора; JSON-конфигурацию сервер не читает. Корневой `docker-compose.yaml`
собирает `docflow-service-local:dev` из исходников через `make storage-up` / `make dev-server`.
Команды автоматически передают версию и идентичность сборки, не меняя `.env`.
`make dev-client` отдельно запускает клиент после обновления стека. Production-пример загружает
`hehelf/docflow-service:${DOCFLOW_SERVER_IMAGE_TAG}` (по умолчанию: `latest`)
из Docker Hub. Оба запускают сервер рядом с PostgreSQL, Seq и Caddy
после готовности PostgreSQL.
Оба хранят настройки и логины в `.env`, а пароли PostgreSQL/Seq
и ключ шифрования настроек — в файловых secrets. Оба используют постоянный том
резервных копий. Отличие dev от production — HTTP вместо HTTPS в Caddy,
включая порт, Caddyfile и хранилище сертификатов. Локальные изменения исходников
требуют повторного `make dev-server`; тег локального образа выбирается автоматически.
`testing/compose/integration.yaml` содержит тестовые PostgreSQL и
сервер под профилем `smoke`. Пустая схема bootstrap-ится
сервером автоматически; при обновлении существующей схемы процесс остаётся
живым в maintenance и ждёт команды администратора. Docker healthcheck проверяет
`/health/live`, а `/health/ready` остаётся 503 до готовности схемы и зависимостей.
`make docker-server-push` после отдельного `docker login` проверяет соответствие
`DOCFLOW_SERVER_VERSION` встроенной версии продукта, собирает и публикует
immutable image tag `<release>.<число-коммитов>-<SHA>`. Для запуска этот тег задаётся
в `DOCFLOW_SERVER_IMAGE_TAG` в `.env` для фиксации сборки. Также публикуется тег
`latest`, который production Compose использует по умолчанию. Обновление сервера:
`docker compose up -d --pull always docflow-server`; публикация сама по себе
не перезапускает работающие контейнеры. Repository
`hehelf/docflow-service` жёстко задан в Makefile и production-примере Compose. Makefile не принимает
Docker Hub token.

Management endpoints:

- `GET /health/live` — процесс и HTTP listener работают;
- `GET /health/ready` — схема, PostgreSQL готовы к обычной работе;
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

- контроль сроков и регулярные серии;
- link create/delete/list/graph;
- journal read/write;
- предметная и техническая статистика;
- document registration command wrapper.

Правило: новые потенциально долгие DB/file/statistics operations должны использовать lifecycle-aware context или явно объяснять, почему это не нужно.

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
runtime environment. PostgreSQL используют те же credentials, которыми
инициализируются контейнеры; отдельные service accounts на текущем этапе не
создаются. Полный перечень и локальные примеры приведены в `docs/examples/.envExample`.
JSON-файл серверу не требуется.

Desktop не получает адрес или credentials Seq. После входа он отправляет
ограниченные пакеты технических событий в аутентифицированный
`POST /api/v1/telemetry/logs`. Сервис проверяет размер и структуру событий,
отбрасывает чувствительные атрибуты, устанавливает identity из bearer-сессии и
пишет события через собственный `slog`/Seq pipeline. Локальный JSON-вывод
desktop сохраняется как best-effort fallback.

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
- PostgreSQL credentials передаются только серверу через runtime environment;
- desktop `config.json` не содержит PostgreSQL credentials;
- `.env`, `/etc/docflow/backup.env` и CIFS credentials file должны иметь `0600` или эквивалентный строгий ACL;
- generated release evidence and logs must not contain passwords or tokens.

Example configs:

- `docs/examples/.envExample`, `docs/examples/config.example.json`, `docker-compose.yaml` are local/dev only;
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
- Чтение распространяется на всю неориентированную цепочку связей; изменение связи требует чтения обоих документов и права регистрации хотя бы одного вида.

### Контроли сроков и регулярные серии

`document_deadline_controls` хранит обязательный календарный срок, основного исполнителя, соисполнителей, необязательный предмет и состояния `active/completed/cancelled`. `document_deadline_series` хранит шаблон будущих итераций и указатель текущей. Управление требует `control_deadlines` по виду, независимо от автора и назначенного исполнителя.

Завершение окончательное: сервер запрещает повторное открытие, интерфейс предупреждает до подтверждения. Фактическая дата исполнения по умолчанию — сегодня в `Asia/Yekaterinburg`, может быть изменена на прошлую; пользователь и время внесения фиксируются отдельно. Перенос срока активного контроля требует причины.

Следующая итерация создаётся только после завершения предыдущей, в той же транзакции с журналом и outbox. Календарь основан на плановой дате, ручной перенос не сдвигает серию. Доступны дни, недели, фиксированное число и последний день месяца/года; короткий месяц не изменяет будущую календарную привязку. Недоступный основной исполнитель или соисполнитель останавливает продолжение серии.

Остановка серии сохраняет текущий контроль; отмена текущей итерации останавливает серию. После исправления шаблона возобновление продолжает активный контроль либо создаёт следующий после завершённого/отменённого. `resume-preview` возвращает расчёт и версию без изменений; UI показывает дату перед подтверждением. Устаревшая версия или несовпадающий расчёт возвращают конфликт.

Справочник `deadline_executors` объединяет связанные пользовательские записи и внешних людей с ФИО, учреждением и должностью. Допустимость назначения пользователя определяется отдельным признаком «Можно назначать исполнителем» и активностью. Она не ограничивает право регистрации и чтение номенклатуры подразделения.

### Journals

Журналируются:

- изменения документов;
- контроли сроков;
- ознакомления;
- административные настройки;
- миграции;
- rollback requests and results.

`document_journal` and `admin_audit_log` are retention-safe and must not cascade-delete through normal app operations.

## Контроль и ознакомление

Прежние HTTP-команды поручений, Wails `AssignmentService` и внутреннее ознакомление удалены из пользовательского процесса. Внешнее ознакомление с приказами (`administrative_order_acknowledgment_people`) остаётся отдельным реквизитом приказа; отметка требует `register` для приказов. Приказные реквизиты не создают и не завершают новые контроли автоматически.

Старые таблицы поручений и вложений удалены вместе с потребителями. Замещение удалено из API, UI и БД; новый baseline 001 создаёт только актуальную модель. Конвертации пользовательских данных не предусмотрено.

## Обновление данных без постоянного опроса

Аудит от 19 октября 2026 года: ежеминутный опрос удалён из рабочего стола и `DeadlineControls` (общий список, карточка документа, история серии). Кнопка «Обновить» в «Актуальных сроках» удалена; при ошибке загрузки доступен «Повторить».

| Область | Что вызывает обновление |
| --- | --- |
| Рабочий стол и контроли | Начальная загрузка, изменение фильтров, локальное изменение контроля, SSE `document-changed` (контроли, реквизиты документа, связи), `documents`, `resync`, `access-changed`; смена дня организации — один таймер на ближайшую полночь |
| Рабочий стол после возврата в окно | События `focus` / `visibilitychange`, без периодического запроса |
| Карточка документа, журналы, связи, поиск | Подходящие SSE-события; поиск — также действия пользователя |
| Персональные события | SSE `user-events`, `resync`, `access-changed`, отметка прочтения и открытие страниц окна событий |
| Статистика и фильтры отчётов | SSE и действия пользователя, без таймера опроса |
| Резервные копии | SSE `backups` / `operation`; повтор каталога каждые 3 секунды включается только после `MAINTENANCE` и прекращается после успешной загрузки или ухода с вкладки |

Фоновая перезагрузка сводки прав по `resync` / `access-changed` не демонтирует уже открытую страницу и карточку документа. До первой загрузки прав показывается индикатор; после подтверждённого отзыва доступа роутер переключается на доступный раздел.

Других безусловных периодических запросов списков во frontend не найдено. Задержки 100 мс объединяют серии событий, 500 мс ограничивают поиск при вводе; таймеры графа связей, начальной настройки и срока наблюдения за backup не опрашивают данные. Desktop сохраняет восстановление SSE с задержкой 1–30 секунд только при разрыве и сторожевой таймер потока 45 секунд. Сервер сохраняет heartbeat SSE и проверку статуса/доступа к наблюдаемой операции (15 секунд), планировщик backup (15 секунд), доставку и очистку outbox (интервалы из настроек), очистку сессий (час), ожидание блокировки worker (5 секунд) и пакетную отправку непустого буфера логов desktop. Эти механизмы выполняют фоновые задачи и обеспечивают доставку событий; удаление клиентского опроса их не заменяет.

## Рабочий стол

Блок «Новое для меня» показывает пять последних персональных событий. Кнопка «Все события» открывает модальное окно с постраничным списком и отмечает все события пользователя прочитанными через `UserEventService.MarkAllRead`. Индикатор рядом с кнопкой считает непрочитанные события за пределами краткого списка; точка на пункте меню «Рабочий стол» показывает наличие любых непрочитанных событий. Общая подписка в основном layout обновляет события и счётчик независимо от открытого раздела. Ошибка отметки прочтения отображается в окне с возможностью повтора.

Новый рабочий стол использует запросы списка контролей с общей областью чтения. Режим «Мои сроки» включает основного и соисполнителя; «Контроль сроков» ограничен видами с правом управления. Показатели активных, просроченных, сегодняшних, ближайших, исполненных и остановленных серий используют те же фильтры, которые передаются при открытии списка. Старый `/api/v1/workspace/overview`, его модели, SQL summary и Wails-метод удалены.

`GET /api/v1/workspace/documents` возвращает `WorkspaceDocuments` (`available`,
`items`) с четырьмя последними зарегистрированными документами. Запрос не принимает
режимы или параметры пагинации: полный список остаётся в видовых журналах. Сервер
разрешает scope чтения каждого вида через `ResolveReadScope`, применяет те же SQL-предикаты,
что журналы, и только затем сортирует по `documents.created_at DESC, documents.id DESC`
и ограничивает выборку. `available=false` означает недоступность запроса;
пустой `items` при `available=true` означает доступный пустой список. Строка содержит
вид, внутренний номер и дату документа, фактическое время регистрации `registeredAt`,
краткую подпись и упорядоченные имена корреспондентов. Полная карточка в запрос не входит.

Кнопка «Найти документ» открывает поиск. Она открывает
модальное окно с полем поиска и фокусом ввода. Enter или «Найти» показывает в этом
окне результаты, количество найденных документов и пагинацию по 20 строк; номер документа открывает его карточку поверх выдачи.
`POST /api/v1/documents/search` принимает `DocumentSearchRequest` (`query`, `page`,
`pageSize`) и возвращает `DocumentSearchResult` (`items`, `totalCount`, `page`,
`pageSize`). Поисковая строка ограничена 500 символами и 32 словами, размер страницы —
100 строками, номер страницы — 10000. Scope чтения каждого зарегистрированного вида
разрешается сервером и применяется до ранжирования, подсчёта и пагинации.

Поиск использует текст `documents.content` (для приказов —
`administrative_order_details.title`), тексты всех резолюций, названия всех
корреспондентов (для исходящих — организацию-получателя), подписанта входящего и
подписанта/адресата исходящего. Номера документов, авторы/исполнители резолюций,
исполнитель исходящего и вложения в поисковый текст не входят. PostgreSQL
`plainto_tsquery` с конфигурацией `russian` требует все значимые слова, которые могут
распределяться между полями; служебные слова обычно исключаются. Для запроса,
состоящего только из служебных слов, используется `simple`. `ё` и `е` при поиске
равнозначны. Семантического поиска, обработки опечаток и поиска по части слова нет.

Релевантность вычисляется через `ts_rank`: вес содержания/заголовка — 1.0,
резолюции — 0.6, корреспондента и подписанта/адресата — 0.3. При равном ранге
используется `created_at DESC, id DESC`. Поля читаются из актуальных таблиц в одном
SQL-запросе; отдельной сохранённой поисковой записи и GIN-индекса пока нет. Это
исключает рассинхронизацию после редактирования карточек и переименования
организаций, но стоимость построения поисковых векторов растёт с числом доступных
документов. Для большой базы следует измерить задержку и при необходимости
добавить сохраняемый поисковый вектор с индексом и согласованным обновлением.
Desktop вызывает endpoint через `serverclient` и Wails `DocumentQueryService.Search`.
При закрытии окна, смене пользователя или изменении доступа выдача сбрасывается;
поздние ответы предыдущего поиска игнорируются.

Desktop получает последние документы через HTTP/Wails `WorkspaceService`, контроли — через `DeadlineControlService`. Изменения графа доступа передаются как глобальная инвалидация без содержания карточек; после неё клиент перечитывает доступные данные.

## Ролевая Модель

В проекте нет отдельной таблицы бизнес-ролей. Источник прав - permission model.

Исторические UX-профили (в актуальном клиенте не вычисляются):

- `admin`;
- `clerk`;
- `executor`;
- `mixed`.

Эти названия не являются источником авторизации.

### System Permissions

Поддерживаются `admin` и `references`. Предметная статистика и отчёты доступны всем авторизованным пользователям в общей области чтения документов. Техническая статистика находится в настройках администратора и требует `admin`; это право само по себе не предоставляет чтение всех документов. Системные права `reports/stats_*` удалены из API и исходной схемы.

### Статистика и отчёты

Статистика документов и контролей использует scope проверенной сессии до подсчётов и группировок. Методы `GetDeadlineStatistics/GetDeadlineReport/GetDeadlineFilterOptions` и HTTP `/api/v1/statistics/controls` заменили статистику поручений. Фильтры ограничены читаемыми документами, основными исполнителями из справочника, регистраторами и номенклатурой; внешние исполнители учитываются без фиктивных пользователей. Статистический отчёт по контролям отбирается по срокам; распределение состояний включает историю и отмены.

Конструктор работает через `POST /api/v1/reports/run`, `POST /api/v1/reports/export/{format}` и scoped `GET /api/v1/reports/filters`. Четыре шаблона: входящие/исходящие по организациям, нагрузка основных исполнителей по подразделениям/месяцам создания, среднее календарное время от создания контроля до фактической даты исполнения, доля исполненных позже срока. Завершённая просрочка определяется `executed_on > due_date`; отмены не участвуют в показателях исполнения. Открытая просрочка рассчитывается на текущую дату организации отдельно от выбранного периода.

Период до трёх лет выбирается по дате регистрации документа, сроку контроля или фактической дате исполнения. Группировки и фильтры ограничиваются общей областью чтения, включая всю цепочку связей. Повторные контроли и несколько корреспондентов не размножают уникальные документы. Основные исполнители учитываются один раз; внешние люди без пользовательского подразделения показываются в «Без подразделения». Часовая зона фиксирована — `Asia/Yekaterinburg`; версия определения показателей — 2.

При изменении/отзыве доступа интерфейс очищает результаты и игнорирует поздние ответы. Статистика обновляется; отчёт пользователь формирует повторно самостоятельно. XLSX/PDF сохраняются на его ПК, сервер не формирует их по расписанию и не хранит готовые результаты. Экспорт заново проверяет текущую область чтения. PDF — читаемая выгрузка без подписи и утверждённого бланка.

### Новая модель контроля: выполняемый переход

Сервер предоставляет новую модель через `/api/v1/deadline-controls`, `/api/v1/deadline-series` и `/api/v1/deadline-executors`. Desktop использует отдельный HTTP/Wails-сервис `DeadlineControlService`. Карточки, рабочий стол и общая область чтения переключены. Статистика и отчёты переключены. Вложения, backup/restore и окончательная очистка ещё не завершены; ход перехода и ограничения описаны в [журнале изменений](document-deadline-control-changes.md).

В новых маршрутах управление требует `control_deadlines` по виду документа, независимо от автора. Контроль имеет одного исполнителя, необязательных соисполнителей и состояния `active`, `completed`, `cancelled`; конечные состояния не открываются повторно. Просрочка вычисляется по календарной дате организации `Asia/Yekaterinburg`. Фактическая дата исполнения отдельна от времени отметки и по умолчанию равна текущему дню; будущая запрещена. Редактирование передаёт версию, перенос срока — также причину.

Серия создаёт следующую итерацию после завершения предыдущей, от её планового срока. Отмена текущей останавливает серию; остановка сама по себе не закрывает текущий контроль. Возобновление требует актуальную версию и подтверждённый рассчитанный срок. Ошибка допустимости исполнителя или соисполнителя при продолжении останавливает серию с причиной. Состояние и outbox сохраняются одной транзакцией.

Область чтения новых маршрутов объединяет права `register`/`control_deadlines` по виду, номенклатуру подразделения, назначения и всю цепочку связей. Пустая область даёт пустой список. Все document-domain проверки используют ту же область чтения.

Справочник контроля показывает пользователей с отдельным признаком допустимости назначения и внешних людей без учётной записи. Связанные ФИО читаются из пользователя; внешние записи имеют ФИО, учреждение, должность и активность. Управление внешними записями требует `references` либо `admin`. Изменения версионируются и аудируются; используемые записи не удаляются. Бумажный справочник исполнителей резолюций остаётся отдельным.

### Document Domain Actions

По видам документов независимо выдаются `register` (регистрация и редактирование)
и `control_deadlines`. Любое право открывает все документы вида. Дополнительные
источники чтения — номенклатура подразделения, назначение основным/соисполнителем
активного или завершённого контроля и вся неориентированная цепочка связей.
Чтение связанных документов не добавляет права изменения.

### Допустимость назначения

`is_document_participant` означает «Можно назначать исполнителем». Он влияет
на выбор исполнителя и продолжение серии, а доступ по подразделению сохраняется
независимо от него. Внешний исполнитель хранится в справочнике без учётной записи.
Замещение и внутреннее ознакомление удалены; внешнее ознакомление с приказами
остаётся отдельным реквизитом. Профиль показывает права, которые проверяет сервер.

### Реквизиты пользователя

Пользователь хранит `last_name`, `first_name`, `patronymic`, `no_patronymic`.
Фамилия и имя обязательны; отчество обязательно при выключенном признаке
«Нет отчества». При включённом признаке сервер очищает отчество, а форма
отключает поле и снимает обязательность. Компоненты обрезаются по краям и
ограничиваются 100 Unicode-символами; снимок `admin_audit_log.user_name` хранится
как TEXT, чтобы вместить составное имя целиком. Правила находятся в независимом
`internal/shared/personalname`. Составные имена, дефисы и другие алфавиты допустимы.

`users.full_name` — generated stored column для JOIN, сортировки и статистики,
а `dto.User.fullName` — вычисляемая подпись. Независимого записываемого ФИО
пользователя нет. HTTP/Wails-команды используют четыре отдельных реквизита;
`PATCH /api/v1/profile` возвращает 200 и актуальный `dto.User`, который frontend
использует для обновления профиля и шапки. Первоначальная настройка запрашивает
те же реквизиты и пароль, создавая учётную запись с логином `admin`.
В шапке подпись сокращается из отдельных полей до «Иванов И.И.»;
при `noPatronymic=true` отображается только инициал имени — «Иванов И.».
Приветствие рабочего стола использует имя и отчество: «Добрый день, Иван Иванович!»;
при отсутствии отчества — «Добрый день, Иван!». Обращение обновляется из состояния
профиля, а выбор приветствия сохраняет зависимость от местного времени суток.

Схема изменена непосредственно в `001_core_users`, без новой версии миграции.
Для прежней тестовой БД требуется явное пересоздание; повторный запуск миграций
не изменит уже применённую 001. Архивы с прежней структурой пользователей
не совместимы с новым кодом, а проверка номера версии не выявит это различие.
Использовать архивы, созданные после пересоздания тестового окружения.
ФИО заявителей и внешних ознакомляемых приказов остаются отдельными реквизитами;
снимки имени в ранее записанном аудите и outbox не переписываются.

## Критичные Данные

Потеря недопустима для:

- users, password hashes, system/document permissions;
- departments and nomenclature bindings;
- nomenclature and `next_number`;
- `documents` and all document detail tables;
- correspondent registrations and resolutions;
- deadline controls, external executors, co-executors, series and actual execution dates;
- administrative orders and acknowledgment people;
- document links;
- system settings;
- `document_journal`;
- `admin_audit_log`;
- migration state.

## Атомарные Операции

Должны оставаться атомарными внутри PostgreSQL:

- first-run setup после применения миграций: admin user и `admin` permission;
- document registration: idempotency check, number allocation, `next_number`, `documents`, detail table, children, journal;
- document update with child data;
- control create/update with co-executors;
- deadline series create, template update, cancellation and iteration advance;
- full replacement of user access profile;
- переходы контроля, журнал и outbox.

Операции между PostgreSQL и SMB не могут быть общей SQL-транзакцией. Для них используются compensation, outbox/saga и атомарная публикация backup archive + manifest.

## Идемпотентность

Должны быть idempotent or safe to repeat:

- повторная доставка одной outbox-записи не создаёт второй journal/audit effect;
- отдельные повторяемые бизнес-переходы получают разные outbox keys;

- migrations `Up()` when no change;
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

- New schema changes require migrations in `internal/server/database/migrations`.
- Migrations must be embedded and compatible with release build.
- For constraints/index changes, add focused tests where practical.
- Do not add performance indexes just because a query has a seq scan on small baseline data.
- Для production-like планов используйте `make db-performance-check`; по умолчанию он создаёт 10 000 документов, сравнивает глубокие OFFSET- и cursor-страницы поиска и сохраняет полный JSON-план. Размер и pagination-сценарий можно изменить через `PERFORMANCE_DOCUMENTS`, `PERFORMANCE_PAGE_SIZE` и `PERFORMANCE_DEEP_PAGE`, например `make db-performance-check PERFORMANCE_DOCUMENTS=50000 PERFORMANCE_DEEP_PAGE=1000`.
- Any new performance index needs before/after `EXPLAIN (ANALYZE, BUFFERS)` and write-latency consideration.
- Keep rollback impact explicit for destructive `down` migrations.

## Backup, Restore И Recovery

Сервер создаёт снимок PostgreSQL, передаёт его напрямую по SMB и поддерживает
расписание резервирования и retention. Барьер обслуживания дожидается запросов,
outbox и очистки сессий; передача архива начинается после снятия барьера.

Настройки SMB и AES-GCM ciphertext пароля хранятся в `backup_settings`. Ключ
передаётся через Compose secret и не включается в архив. Постоянный staging volume
хранит задания и неотправленные архивы. Формат 4 содержит ровно `manifest.json`
и `database.dump`; размер и SHA-256 проверяются при подготовке. Manifest с полями
объектов, архивы прежнего формата, дополнительные/небезопасные члены отклоняются.
Каталог показывает старые копии как несовместимые, без восстановления и удаления.

Восстановление заменяет данные только выделенной PostgreSQL при полном обслуживании,
после проверки источника и локальной страховочной копии. Фазы и инициатор сохраняются
вне БД; при ошибке сервер пытается выполнить откат. Проверяются версия схемы и
существование основного/соисполнителей. Маркер незавершённой замены блокирует бизнес-API
после перезапуска. После успеха сессии отзываются, расписание резервирования и
SMB-пароль очищаются. При потере БД оператор создаёт пустую БД, инициализирует
приложение и восстанавливает новую копию через обычную админпанель.
Seq и ключи не входят в backup; шифрование архивов не реализовано.
Production RPO/RTO подтверждаются пробным восстановлением.

Подробности, установка, ограничения и проверки:
[серверное резервирование](instructions.md#серверное-резервирование-и-восстановление).

## Release And Versioning

Version source:

- `docs/releases.yaml`;
- generated `internal/shared/releaseassets/current_release.yaml`;
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

Основные команды (`make` показывает справку):

- Разработка: `make dev-client`, `make dev-server`.
- Весь локальный Compose-стек: `make storage-up`, `make storage-down`, `make storage-reset`.
- Продакшен: `make build-linux`, `make build-windows`, `make docker-server-push`.
- Основная проверка: `make release-gate`.

`storage-up` и `dev-server` собирают текущий локальный сервер и запускают весь стек.
`storage-reset` удаляет все именованные тома Compose, включая Seq и staging бэкапов.
Проверки и диагностические цели вынесены в `build/make/checks.mk`, но по-прежнему
доступны через корневой Makefile. `make help-checks` показывает их назначения.
В частности, сохранены все 11 шагов release-gate, отдельные `storage-smoke-test`
и `db-performance-check`, ручное управление интеграционной БД, `release-assets`
и `clean`. Служебные prerequisite-цели имеют префикс `_`.

`make integration-test` проверяет prerequisites, запускает изолированные
PostgreSQL из `testing/compose/integration.yaml`, передаёт безопасный
`DOCFLOW_INTEGRATION_DSN` для `docflow_test_outbox`, затем запускает отдельный
стек с Samba и инструментированным сервером для backup/verify/restore/delete.
Go-профили покрытия сохраняются в `build/release-evidence/`, а тестовые контейнеры
и тома удаляются после завершения. Этот target входит в обязательный `release-gate`.
`make storage-smoke-test` проверяет production Compose пример, доступ через Caddy
и сохранность контролей после перезапуска сервера.
Для ручной отладки доступны `make integration-db-up` и
`make integration-db-down`.

Integration coverage composition root собирает Wails options только с server
URL и локальным theme service, без PostgreSQL. Тест проверяет состав
bindings, startup/shutdown и logger callback. Отдельные server integration tests
проверяют обработку outbox, контроли, scope статистики и database-only восстановление
на реальной PostgreSQL. Production `NewWailsOptions` создаёт API adapters и
локальные UI services, но не infrastructure clients.

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
- контроли и связи пропорционально числу документов;
- размер БД и резервных копий измеряется на целевом контуре.

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
- Supply PostgreSQL credentials to `docflow-server` through the approved runtime secret-delivery mechanism.
- Keep technical logs free of passwords and tokens.

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
- DB/Seq unavailable diagnostics.

## Known Release State

Текущая согласованная версия release metadata: `1.0.7` в `docs/releases.yaml`, generated release asset и `wails.json`.

Этот справочник не утверждает отсутствие открытых проблем. Завершение архитектурного разделения не заменяет приёмку релиза: production approval требует clean-worktree release gate, target OS smoke и реального backup/restore test.

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
- формирование и экспорт отчётов;
- structured error contract;
- release gates;
- config/secrets;
- technical logging and audit trail.

Runtime secrets и backup/restore v4 описаны в
[инструкции резервирования](instructions.md#серверное-резервирование-и-восстановление).
