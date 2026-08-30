# План реализации серверного сервиса Docflow

Дата подготовки: 29 августа 2026 года

Дата последнего обновления: 30 августа 2026 года

Статус: этап 1 и серверное управление миграциями реализованы в коде;
production deployment, TLS и cutover не выполнены

Реализованный первый инкремент:

- отдельный entrypoint `cmd/docflow-server`;
- команды `run`, `check-config`, `healthcheck` и `version`;
- server composition root для PostgreSQL, MinIO и outbox consumers;
- общий schema-dependent lifecycle в `internal/background`;
- встроенный outbox consumer удалён из desktop composition root;
- PostgreSQL advisory lease обеспечивает один server-worker и сериализует
  серверные изменения схемы;
- настраиваемые polling, batch, timeouts, retention и cleanup;
- graceful shutdown, значимые structured operational logs и startup diagnostics;
- unit и PostgreSQL integration tests;
- environment-only server config, versioned distroless container image и
  Makefile targets для локальной сборки и явной публикации в Docker Hub;
- management API с раздельными liveness/readiness и maintenance mode;
- пустая БД bootstrap-ится сервером, обновление существующей БД подтверждается
  администратором из desktop UI;
- desktop не исполняет migration SQL: apply/rollback выполняет сервер под
  PostgreSQL advisory lock с повторной проверкой admin password и audit.

До production-переключения остаются инфраструктурное развёртывание, проверка
реального backup/restore, production-like integration/load test и согласованное
окно остановки всех централизованно запускаемых desktop-процессов.

## 1. Цель

Создать постоянно работающий серверный процесс `docflow-server`, который
сначала возьмёт на себя обработку transactional outbox и регламентные задачи,
а затем станет единственной доверенной точкой доступа desktop-приложения к
PostgreSQL и MinIO.

Целевой результат:

- outbox обрабатывается независимо от того, запущены ли рабочие места;
- PostgreSQL и MinIO не доступны с пользовательских ПК;
- в desktop-конфигурации нет PostgreSQL/MinIO credentials;
- аутентификация, авторизация и аудит выполняются на сервере;
- React продолжает вызывать локальные Wails bindings, а локальный Go-слой
  постепенно заменяет repositories на HTTPS API client;
- сервер разворачивается одним модульным Go-приложением, а не набором
  микросервисов;
- переход выполняется поэтапно и допускает контролируемый откат.

## 2. Исходное состояние

До реализации этапа 1 каждый Wails-процесс:

- напрямую подключается к PostgreSQL;
- напрямую обращается к MinIO;
- создаёт собственный `AuthService` и хранит один `currentUserID` в памяти;
- запускает outbox worker через `internal/app/background_lifecycle.go`;
- отправляет технические логи в Seq;
- содержит полный composition root repositories и business services в
  `internal/app/app.go`.

Production executable публикуется централизованно и запускается всеми рабочими
местами из одного расположения. Поэтому длительного периода, в котором новые
запуски используют разные версии desktop, не планируется. При этом замена файла
не обновляет уже запущенные процессы: атомарное переключение требует окна
обслуживания, закрытия всех экземпляров приложения и только затем публикации
нового executable/config.

Transactional outbox уже имеет важные свойства, необходимые для выделения
worker:

- события записываются в доменной транзакции;
- claim использует `FOR UPDATE SKIP LOCKED`;
- есть bounded retry и terminal failure;
- зависшие claims освобождаются по timeout;
- обработанные строки удаляются по retention;
- есть статистика очереди и административный requeue;
- удаление вложений из MinIO уже является отдельным consumer.

Главное ограничение для HTTP-сервера: текущий `AuthService` хранит единственного
пользователя процесса. На многопользовательском сервере идентичность должна
передаваться через `context.Context` каждого запроса. Нельзя просто создать один
текущий `AuthService` и вызвать его из параллельных HTTP handlers.

## 3. Принятые архитектурные решения

### 3.1. Один модульный сервер

Первой целевой единицей развёртывания будет один бинарник:

```text
cmd/docflow-server/main.go
```

Внутри него допускаются отдельные модули API, worker, scheduler и operations,
но они собираются и разворачиваются совместно. Разделение на независимые
микросервисы не входит в план без измеренной причины.

### 3.2. Поэтапный переход

Переход выполняется в двух крупных стадиях:

```text
Стадия A
Desktop ──► PostgreSQL / MinIO
                  ▲
                  │
          docflow-server worker

Стадия B
Desktop ──HTTPS──► docflow-server ──► PostgreSQL / MinIO / Seq
```

На стадии A desktop и сервер взаимодействуют косвенно через outbox в
PostgreSQL. На стадии B desktop знает только HTTPS URL сервиса.

### 3.3. Секреты не выдаются desktop-клиенту

Сервис не должен возвращать постоянные PostgreSQL или MinIO credentials после
аутентификации. Иначе пользователь сможет обойти application-level permissions
прямым SQL или прямым вызовом MinIO.

В целевом состоянии:

- PostgreSQL credentials принадлежат только серверу;
- MinIO credentials принадлежат только серверу;
- desktop получает только серверную сессию;
- файлы сначала передаются потоково через API;
- presigned MinIO URL рассматривается позднее только как оптимизация и не
  содержит постоянных credentials.

### 3.4. Wails остаётся фасадом для React

Публичные Wails service signatures по возможности сохраняются. Для каждого
переносимого сценария локальная реализация заменяется на вызов API:

```text
React -> Wails binding -> desktop API adapter -> HTTPS -> server use case
```

Это уменьшает объём одновременных изменений frontend и позволяет переносить
вертикальные сценарии по одному.

### 3.5. Сервер владеет изменениями схемы в целевом состоянии

Административный UI миграций сохраняется, но является только клиентом
management API. Каталог embedded migrations и SQL execution принадлежат
`docflow-server`. На apply/rollback сервис останавливает worker, получает
PostgreSQL advisory lock и после операции согласует maintenance/readiness без
рестарта. Несколько экземпляров не могут выполнять миграции конкурентно.

## 4. Целевая структура кода

Предварительная структура:

```text
cmd/
└── docflow-server/
    └── main.go                 process entrypoint

internal/
├── app/                        Wails composition root
├── server/
│   ├── app.go                  server composition root
│   ├── config.go               server-only configuration
│   ├── lifecycle.go            startup/shutdown/readiness
│   └── version.go              API/client compatibility
├── httpapi/
│   ├── router.go
│   ├── middleware/
│   │   ├── authentication.go
│   │   ├── request_id.go
│   │   ├── recovery.go
│   │   ├── limits.go
│   │   └── logging.go
│   ├── auth_handler.go
│   ├── system_handler.go
│   ├── document_handler.go
│   └── attachment_handler.go
├── apiclient/                  client used by Wails desktop backend
├── identity/                   request principal and authorization helpers
├── outbox/                     existing worker and consumers
├── services/                   shared business use cases
├── repository/                 server-side persistence
└── storage/                    server-side MinIO client
```

Имена каталогов могут уточняться при реализации. Важны границы:

- `httpapi` не содержит SQL и бизнес-правил;
- `apiclient` не зависит от Wails и тестируется отдельно;
- repositories не используются в desktop composition root после завершения
  миграции;
- identity запроса не хранится в глобальном mutable state;
- server-specific lifecycle не зависит от GUI.

## 5. Этапы реализации

## Этап 0. Зафиксировать требования и измерить baseline

### Работы

1. Зафиксировать ожидаемое количество зарегистрированных и одновременно
   активных пользователей.
2. Измерить текущие:
   - количество соединений PostgreSQL;
   - p50/p95/p99 критичных SQL-операций;
   - частоту и размер загрузок вложений;
   - скорость появления и обработки outbox;
   - объём логов в Seq;
   - максимальный backlog outbox при простое клиентов.
3. Определить серверную ОС, способ развёртывания и наличие Docker Compose.
4. Определить DNS-имя сервиса и источник TLS-сертификата.
5. Зафиксировать допустимое окно обслуживания и порядок backup/restore.
6. Составить матрицу сочетаний текущей и rollback-версии
   desktop/server/schema.
7. Зафиксировать способ уведомления пользователей, закрытия всех запущенных
   Wails-процессов и проверки, что старые процессы больше не подключены к БД.

### Результат

- baseline-отчёт с измерениями;
- согласованные SLO и ограничения;
- выбранные DNS, TLS и deployment topology;
- перечень поддерживаемых версий.

### Минимальные стартовые SLO

Значения должны быть подтверждены владельцем системы:

- доступность API в рабочее время не ниже 99,5%;
- p95 обычного API-запроса не выше 500 ms без учёта загрузки файлов;
- успешное outbox-событие начинает обрабатываться не позднее 15 секунд;
- terminal outbox failure виден оператору не позднее 1 минуты;
- graceful shutdown завершается не более чем за 30 секунд;
- потеря подтверждённой доменной транзакции недопустима.

### Критерий завершения

Нет неизвестных параметров инфраструктуры, которые влияют на сетевую или
security-модель реализации.

## Этап 1. Выделить server-side outbox worker

Это первый production-инкремент. Он не меняет пользовательский протокол и не
добавляет внешний API.

### Работы с кодом

1. Добавить `cmd/docflow-server/main.go`.
2. Выделить server composition root из Wails-specific composition root.
3. Переиспользовать:
   - database connection и schema compatibility check;
   - outbox repository;
   - user event, journal и audit repositories;
   - attachment repository;
   - MinIO storage;
   - observability registry и Seq logger.
4. Перенести общий schema-dependent lifecycle из GUI-ориентированного слоя в
   пакет, доступный обоим процессам, либо создать серверный эквивалент без
   дублирования правил совместимости схемы.
5. Добавить серверную конфигурацию:
   - PostgreSQL;
   - MinIO;
   - Seq;
   - polling interval;
   - batch size;
   - claim timeout;
   - consumer timeout;
   - retention;
   - shutdown timeout.
6. Заменить compile-time constants worker на валидируемые настройки с
   безопасными default и разумными limits.
7. Удалить outbox consumer из desktop composition root. Desktop всегда остаётся
   producer: repositories продолжают атомарно добавлять события, но Wails
   lifecycle не получает worker и не может обрабатывать очередь.
8. Не удалять защиту `FOR UPDATE SKIP LOCKED`: она нужна для отказоустойчивости
   server-worker и защищает от непреднамеренно оставшегося запущенного процесса
   предыдущей desktop-версии, хотя штатный rollout не предполагает параллельных
   consumers.
9. Удерживать отдельный session-level PostgreSQL advisory lease всё время
   работы server-worker. Второй экземпляр сервиса не запускает consumer, а
   desktop UI отклоняет применение и rollback миграций до остановки сервиса.
10. Добавить server process signals: `SIGTERM`, `SIGINT`, context cancellation,
   ожидание текущего batch и закрытие DB/logger.
11. Добавить команды:

    ```text
    docflow-server run
    docflow-server check-config
    docflow-server healthcheck
    docflow-server version
    ```

### Ограничение совместимости

При централизованной поставке штатный cutover выполняется без параллельной
работы desktop-worker и server-worker. Однако уже запущенный процесс не меняется
при замене общего executable. Поэтому desktop-worker предыдущей версии и новый
server-worker всё равно должны понимать существующие `event_type` и payload на
случай ошибочно не закрытого процесса или аварийного rollback.

Новые типы событий вводятся consumer-first: сервер сначала получает поддержку
payload, и только после успешного cutover новый централизованный desktop может
начать создавать такие события.

Для каждого outbox payload нужно зафиксировать версию контракта. Рекомендуемый
путь для новых событий:

```json
{
  "schemaVersion": 1,
  "...": "..."
}
```

Существующие payload считаются версией 1 по умолчанию, чтобы не требовать
переписывания очереди.

### Тесты

- unit tests server config и lifecycle;
- worker tests с отменой context во время batch;
- integration test: desktop producer создаёт событие, server consumer его
  обрабатывает;
- integration test одновременных desktop/server consumers без двойного
  эффекта;
- restart после claim до `MarkProcessed`;
- недоступность PostgreSQL на старте и восстановление;
- недоступность MinIO при `attachment_delete` и последующий retry;
- остановка при incompatible/dirty schema;
- отсутствие credentials в logs и process arguments.

### Критерий завершения

- server worker работает без запущенного Wails-приложения;
- очередь обрабатывается после рестарта процесса;
- остановка и schema gate проверены тестами;
- desktop composition root не создаёт outbox worker;
- monitoring подтверждает, что очередь обрабатывает `docflow-server`.

## Этап 2. Подготовить production deployment и наблюдаемость

### Deployment

1. Добавить multi-stage Dockerfile для `docflow-server`.
2. Запускать процесс непривилегированным пользователем.
3. Использовать read-only root filesystem, отдельный writable temporary
   directory и `no-new-privileges`.
4. Создать отдельный production deployment manifest вне локального
   `docker-compose.yaml` либо явно разделить `compose.dev.yaml` и
   `compose.production.yaml`.
5. Подключить PostgreSQL, MinIO и Seq через внутреннюю сеть.
6. Не публиковать worker endpoint в пользовательскую сеть.
7. Передавать секреты файлами с правами `0600`, Docker secrets или утверждённым
   secret provider. Не помещать секреты в image, repository и command line.
8. Зафиксировать image digest, версию бинарника и версию конфигурации.

Если Docker недоступен, подготовить эквивалентный unit `systemd` с
`Restart=always`, отдельным пользователем и sandboxing directives.

### Health и readiness

Добавить локальные endpoints или CLI checks:

- liveness: процесс и event loop не остановлены;
- readiness: PostgreSQL доступен, схема совместима, server не в maintenance;
- dependency details доступны только оператору и не содержат secrets;
- worker state: running/stopping/stopped, last successful poll, last processed
  event time.

### Метрики

Минимальный набор:

- `outbox.pending`, `outbox.processing`, `outbox.failed`;
- возраст самого старого pending event;
- processed/retried/terminal events;
- latency consumers по `event_type`;
- DB pool open/in-use/wait count/wait duration;
- operation latency и error count;
- MinIO request latency/errors;
- process CPU/RAM/goroutines;
- build version и uptime.

Следует определить способ сбора метрик. До внедрения отдельного metrics backend
метрики остаются только во внутреннем registry и не публикуются в Seq: operational
log содержит только значимые события, warnings и errors. Для alerting требуется
система, способная строить пороги по времени.

### Alerts

- terminal outbox failures больше нуля;
- oldest pending age выше SLO;
- worker не ready;
- PostgreSQL connection saturation;
- повторяющиеся MinIO failures;
- диск PostgreSQL/MinIO/Seq приближается к лимиту;
- частые рестарты процесса.

### Критерий завершения

Сервис устанавливается и обновляется воспроизводимо, автоматически
перезапускается и наблюдается оператором без подключения к контейнеру или БД.

## Этап 3. Добавить системный HTTPS API и compatibility handshake

На этом этапе business operations ещё могут идти напрямую в БД, но desktop уже
умеет связываться с сервисом.

### Server endpoints

```text
GET /api/v1/system/status
GET /api/v1/system/compatibility?clientVersion=...
```

Пример compatibility response:

```json
{
  "status": "ready",
  "apiVersion": "v1",
  "serverVersion": "1.1.0",
  "minimumClientVersion": "1.0.6",
  "maintenance": false
}
```

### Общий HTTP contract

- HTTPS обязателен вне localhost/internal container network;
- JSON UTF-8 для metadata;
- единый error envelope совместим с текущим frontend contract;
- `X-Request-ID` принимается или создаётся сервером и возвращается клиенту;
- request body имеет жёсткий limit;
- server/read/write/idle timeouts задаются явно;
- panic recovery не раскрывает stack trace клиенту;
- timestamps передаются в RFC 3339 с timezone;
- list endpoints используют cursor pagination;
- API version входит в URL;
- неизвестные JSON fields для command requests отклоняются после согласования
  compatibility policy;
- внутренние PostgreSQL/MinIO errors не попадают в response.

### Desktop API client

Добавить `internal/apiclient`:

- base URL validation;
- TLS transport и connection pooling;
- общий request ID;
- deadlines по типу операции;
- decoding structured errors;
- ограниченный retry только для заведомо безопасных GET либо idempotent command;
- отсутствие автоматического retry для произвольного POST;
- логирование без password/token/body вложений.

Desktop startup выполняет compatibility request до показа login. UI должен
различать:

- сервис недоступен;
- недоверенный/просроченный TLS certificate;
- maintenance;
- слишком старый desktop;
- несовместимую версию API;
- обычную ошибку аутентификации.

### TLS

Предпочтительная схема:

```text
Desktop -> HTTPS :443 -> Caddy/nginx -> HTTP internal -> docflow-server
```

Допускается TLS непосредственно в Go-процессе, если эксплуатация сертификатов
описана и автоматизирована. Отключение проверки certificate в production
запрещено. Для внутреннего CA корневой сертификат должен устанавливаться в
доверенное хранилище ОС.

### Критерий завершения

Все production desktops проверяют доступность и совместимость сервера, но
отказ нового status API ещё не должен повреждать данные или запускать частично
выполненную команду.

## Этап 4. Перенести аутентификацию и request identity

Это обязательный архитектурный refactoring перед переносом защищённых use
cases.

### Модель identity

Создать immutable principal:

```go
type Principal struct {
    UserID uuid.UUID
    SessionID uuid.UUID
}
```

Principal создаётся authentication middleware и помещается в request context.
Business services получают его через явный параметр или небольшой
`identity.FromContext(ctx)` на границе use case. Нельзя:

- хранить текущего пользователя в package variable;
- менять пользователя в singleton `AuthService`;
- создавать отдельное глобальное состояние на каждый handler;
- доверять `userId`, присланному в JSON самим клиентом.

### Session model

Для первого варианта выбрать opaque bearer sessions:

- клиент получает криптографически случайный token;
- в PostgreSQL хранится только hash token;
- session содержит user ID, issued/last-used/expires/revoked timestamps и
  ограниченные device metadata;
- logout отзывает session;
- блокировка/деактивация пользователя отзывает все его sessions;
- idle и absolute lifetime задаются настройками;
- сравнение token hash не раскрывает исходный token;
- login rate limiting применяется по login и источнику запроса;
- события входа, отказа, logout и revoke входят в административный аудит.

JWT может быть рассмотрен позднее, но не нужен для одного серверного приложения
и усложняет немедленный отзыв. Session token в первом релизе хранится только в
памяти desktop-процесса; после перезапуска пользователь входит повторно. Это
исключает отдельную задачу безопасного кроссплатформенного хранения refresh
token.

### Endpoints

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
POST /api/v1/auth/change-password
POST /api/v1/auth/change-required-password
```

Login response не возвращает DB/MinIO/Seq credentials.

### Refactoring services

1. Отделить проверку credentials от текущей process-local session.
2. Вынести password policy и lockout в server-side use case.
3. Заменить вызовы `GetCurrentUserUUID` и `RequireAuthenticated` в переносимых
   services на request-scoped principal/authorizer.
4. Desktop auth adapter сразу использует server API; runtime fallback на прямой
   login не поддерживается. Пока business API переносится, локальные services
   получают user ID из подтверждённой server session.
5. Не допускать одновременного смешивания server principal и локального
   `currentUserID` внутри одной доменной транзакции.
6. Добавить новую миграцию session table и необходимые индексы/retention.

Реализованный первый auth slice:

- рабочая migration `014_server_sessions` (прежняя тестовая migration удалена
  после проверки механизма миграций, её номер переиспользован);
- opaque 256-bit token и хранение только SHA-256 hash;
- login/logout/me endpoints;
- bearer middleware с request-scoped user identity и permission gate;
- login rate limit, account lockout и обязательная password-expiry проверка;
- desktop login без production fallback на прямую проверку пароля.

Периодическая очистка истёкших и отозванных sessions выполняется сервисом при
запуске и затем раз в час. Следующими остаются change-password endpoints и
явный revoke всех sessions при административной блокировке или сбросе пароля.

### Security tests

- параллельные запросы двух пользователей не смешивают identity;
- token другого пользователя не повышает права;
- revoked/expired token отклоняется;
- блокировка пользователя завершает активные сессии;
- timing/error response не позволяет различить несуществующий login и неверный
  password;
- brute-force lockout и rate limit;
- password и token отсутствуют в logs;
- CSRF не применим к Authorization header, пока token не хранится в cookie;
- CORS по умолчанию не разрешён, поскольку клиент не является browser origin.

### Критерий завершения

Desktop входит через сервер, а server-side protected endpoint всегда получает
identity из проверенной сессии. Параллельные пользователи покрыты race и
integration tests.

## Этап 5. Переносить business API вертикальными сценариями

Перенос выполняется не по техническим слоям, а законченными пользовательскими
сценариями. Для каждой группы одновременно переносятся query, command,
authorization, audit и тесты.

### Рекомендуемый порядок

1. Пользователи, подразделения и простые справочники.
2. Системные настройки и access summary.
3. Чтение списков и карточек документов.
4. Регистрация и изменение документов.
5. Поручения, соисполнители и замещения.
6. Ознакомления и user events.
7. Связи документов и журнал.
8. Dashboard и статистика.
9. Административный audit и outbox UI.
10. Миграции и остальные административные операции.

### Шаблон миграции одного сценария

Для каждого use case:

1. Зафиксировать текущий Wails DTO и error codes.
2. Определить HTTP endpoint и request/response schema.
3. Убедиться, что repository method принимает context и deadline.
4. Выполнять authorization на сервере до чтения/изменения данных.
5. Сохранить atomic domain change + outbox.
6. Добавить server handler как тонкий adapter.
7. Добавить typed method в desktop `apiclient`.
8. Переключить Wails service на API adapter feature flag.
9. Выполнить contract, integration и UI tests.
10. После rollout удалить desktop repository path только для этого сценария.

### Idempotency

Все команды, для которых клиент может не получить ответ после commit, должны
принимать `Idempotency-Key`. Уже существующий idempotency key регистрации
документа сохраняется.

Сервер должен хранить либо доменный уникальный ключ, либо запись результата
команды. Повтор с тем же key и тем же payload возвращает исходный результат;
тот же key с другим payload возвращает conflict.

### Транзакции

HTTP request не разбивает одну доменную транзакцию на несколько клиентских
вызовов. Например, выделение регистрационного номера, создание документа,
служебные записи и outbox остаются одной server-side транзакцией.

### Pagination и payload size

- журналы используют cursor pagination;
- API не возвращает неограниченные массивы;
- тяжёлая статистика имеет отдельные deadlines и concurrency limits;
- DTO не содержит password hashes, storage paths и внутренних SQL полей;
- conditional requests/cache добавляются только после измерений.

### Критерий завершения каждой группы

- production desktop использует API path;
- direct repository path выключен конфигурацией;
- permissions и audit совпадают с прежним поведением;
- contract поддерживает текущую централизованную desktop-версию и согласованный
  rollback на предыдущую версию в течение rollback window;
- rollback feature flag проверен.

## Этап 6. Перенести вложения за API

### Первый вариант: streaming proxy

Endpoints:

```text
POST   /api/v1/documents/{id}/attachments
GET    /api/v1/attachments/{id}/content
DELETE /api/v1/attachments/{id}
```

Требования:

- multipart/body читается потоково, не целиком в RAM;
- request и decompressed size ограничены;
- имя файла нормализуется и не используется как storage path;
- extension и размер проверяются серверными settings;
- upload использует существующую compensation при ошибке metadata transaction;
- delete сохраняет модель hide metadata + outbox physical delete;
- download проверяет document access до чтения объекта;
- response задаёт безопасный `Content-Disposition`;
- client cancellation отменяет MinIO operation;
- checksum сохраняется/проверяется, если это входит в утверждённый контракт;
- reverse proxy limits согласованы с application limits.

### Возможная оптимизация: presigned URL

Вводится только после нагрузочных измерений. Сервис выдаёт короткоживущий URL
для одного object key и одной операции. Нужны отдельные состояния
`upload-issued`, `uploaded`, `confirmed`, очистка orphan objects и server-side
проверка metadata/checksum. Постоянные MinIO credentials клиенту не выдаются.

### Критерий завершения

- desktop не имеет MinIO credentials;
- MinIO endpoint закрыт от пользовательской сети;
- upload/download/delete и cancellation проверены на предельных размерах;
- backup/restore согласованности PostgreSQL+MinIO повторно протестирован.

## Этап 7. Закрыть прямой доступ desktop к инфраструктуре

### Работы

1. Убедиться, что все production Wails services используют API adapters.
2. Удалить PostgreSQL и MinIO поля из desktop production config.
3. Удалить database/storage construction из Wails runtime composition root.
4. Оставить локальными только:
   - theme и desktop preferences;
   - файловый диалог и сохранение скачанного файла;
   - API client;
   - UI-specific orchestration.
5. На firewall/network уровне запретить рабочим местам PostgreSQL и MinIO.
6. Сменить прежние общие credentials, поскольку они могли сохраниться на ПК.
7. Выдать `docflow-server` отдельные least-privilege DB и MinIO identities.
8. Перевести migration ownership на server CLI/deployment job.
9. Удалить оставшиеся transitional direct API paths после rollback window;
   embedded outbox worker уже удалён на этапе 1.
10. Обновить backup/restore, installation, diagnostics и incident runbooks.

### Критерий завершения

Чистое рабочее место с desktop-конфигурацией, содержащей только server URL, не
может непосредственно подключиться к PostgreSQL или MinIO, но поддерживает все
утверждённые пользовательские сценарии через API.

## 6. API и compatibility policy

### Версионирование

- major API version находится в URL: `/api/v1`;
- additive поля не требуют новой major version;
- удаление/переименование поля требует нового контракта или периода deprecation;
- сервер поддерживает текущую централизованную desktop-версию и предыдущую
  rollback-версию в течение согласованного rollback window; их одновременная
  штатная эксплуатация не требуется;
- `minimumClientVersion` повышается только после подтверждения обновления
  общего executable и завершения всех старых процессов;
- schema migration не должна первой ломать ещё работающий desktop.

### Error envelope

Сохранить существующий принцип:

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Безопасное сообщение",
  "status": 400,
  "requestId": "..."
}
```

Рекомендуемые инфраструктурные codes:

- `SERVER_UNAVAILABLE` формируется desktop API adapter;
- `UNAUTHENTICATED`;
- `SESSION_EXPIRED`;
- `FORBIDDEN`;
- `CLIENT_VERSION_UNSUPPORTED`;
- `MAINTENANCE_MODE`;
- `RATE_LIMITED`;
- `DEPENDENCY_UNAVAILABLE` без раскрытия конкретного секрета/DSN.

### Timeouts и retry

- каждый handler имеет deadline;
- DB/MinIO вызовы используют request context;
- GET может быть повторён desktop-клиентом ограниченно при transport error;
- command повторяется только с idempotency key;
- `429` и временные `503` могут содержать `Retry-After`;
- retry имеет exponential backoff с jitter и общий time budget;
- бесконечные retry в desktop запрещены.

## 7. Миграции базы данных

Ожидаемые новые миграции:

1. Server sessions и индексы expiry/revocation.
2. При необходимости таблица command idempotency results.
3. При необходимости версия outbox payload/producer.
4. При необходимости server instance/lease для singleton jobs.

Правила:

- сначала expand, затем code rollout, затем contract;
- destructive schema changes выполняются отдельным поздним релизом;
- новая server-версия проверяет schema до readiness;
- dirty и too-new schema блокируют business API и worker;
- health остаётся доступен в maintenance;
- backup PostgreSQL и MinIO обязателен перед rollback migration;
- rollback сервера не означает автоматический rollback схемы;
- migration lock предотвращает параллельный запуск.

## 8. Фоновые и регламентные задачи

После стабилизации outbox в сервер можно переносить:

- очистку истёкших sessions;
- очистку command idempotency records;
- storage statistics reconciliation;
- уведомления и просрочки;
- подготовку тяжёлых отчётов;
- проверку целостности и диагностические snapshots.

Для каждой задачи требуется:

- owner и расписание;
- идемпотентность;
- singleton/distributed lock при нескольких replicas;
- deadline и cancellation;
- retry policy;
- метрика последнего успеха;
- ручной безопасный запуск;
- описание влияния повторного запуска;
- ограничение batch/concurrency.

Backup/restore не следует автоматически включать в application scheduler до
устранения известных рисков передачи credentials и до утверждения процедуры
оператором. Это отдельная высокорисковая операция.

## 9. Безопасность

Обязательные требования:

- TLS с проверяемым сертификатом;
- временный remote HTTP разрешается только явным desktop-флагом
  `server.allowInsecureHttp`, в доверенной изолированной сети и не считается
  завершением production TLS gate;
- password/session/storage secrets не логируются;
- секреты отсутствуют в argv и image layers;
- отдельный непривилегированный OS/container user;
- DB role без права создания superuser/изменения инфраструктуры;
- отдельная MinIO policy только для нужного bucket/prefix;
- Seq ingestion identity без административного доступа;
- uniform authentication errors;
- login rate limiting и lockout;
- request body/header limits;
- защита от path traversal и unsafe filenames;
- security headers на reverse proxy;
- request ID во всех audit/technical logs, где это не нарушает privacy;
- server authorization для каждой защищённой операции;
- frontend permissions используются только для UX и не считаются защитой;
- журналирование административных изменений и действий с пользователями;
- регулярная ротация credentials и session cleanup.

Перед закрытием прямого доступа нужно провести отдельный threat review для:

- похищенного session token;
- скомпрометированного рабочего места;
- повторной отправки command;
- подмены сервера/DNS;
- загрузки вредоносного вложения;
- обхода document access scope;
- утечки PII через Seq;
- злоупотребления административным API.

## 10. Тестовая стратегия

### Unit tests

- config validation;
- authentication/session lifecycle;
- authorization policy;
- HTTP DTO/error mapping;
- API client retry/idempotency rules;
- outbox payload versioning;
- scheduler locking;
- filename/content disposition handling.

### Contract tests

- request/response JSON fixtures;
- backward-compatible decoding предыдущим desktop client;
- stable error codes/status;
- OpenAPI schema, если она будет принята как contract source;
- обязательные/неизвестные поля;
- cursor pagination.

### Integration tests

- disposable PostgreSQL;
- MinIO test container для attachment flows;
- login -> authorized command -> audit/outbox;
- параллельные principals;
- lost response + повтор с idempotency key;
- server restart во время request/worker claim;
- dependency outage/recovery;
- migration compatibility states;
- session revoke/expiry;
- streaming и cancellation.

### End-to-end tests

- packaged desktop против production-like server;
- вход, смена обязательного пароля и logout;
- регистрация каждого вида документа;
- поручение и ознакомление;
- attachment upload/download/delete;
- admin settings и outbox requeue;
- desktop/server version mismatch;
- maintenance mode;
- восстановление согласованного backup-набора.

### Нагрузочные тесты

- одновременный login/startup dashboard;
- поиск и cursor pagination;
- burst регистрации с idempotency keys;
- параллельные uploads/downloads;
- накопленный outbox backlog;
- медленный MinIO/Seq;
- DB pool saturation.

Результаты сравниваются с baseline этапа 0. Нельзя считать миграцию успешной
только по функциональным тестам.

## 11. Развёртывание и rollout

### Порядок первого worker rollout

1. На тестовом контуре проверить новый desktop как producer без consumer, а
   server-worker — как единственный обработчик совместимой очереди.
2. Развернуть `docflow-server` рядом с PostgreSQL/MinIO, но не допускать
   production consumption до начала окна обслуживания.
3. Объявить окно обслуживания и запретить новые входы.
4. Закрыть все Wails-приложения и проверить отсутствие старых процессов/DB
   sessions. Простая замена общего executable без этого шага недостаточна.
5. Дождаться завершения событий, взятых прежними desktop-worker, и проверить
   отсутствие активных claims либо дождаться их stale timeout.
6. Опубликовать общий desktop executable без встроенного worker.
7. Запустить `docflow-server`, дождаться readiness и обработки контрольного
   события.
8. Выполнить smoke test desktop и открыть систему пользователям.
9. Наблюдать очередь, terminal failures и отсутствие процессов предыдущей
   desktop-версии.

### Порядок rollout каждого API-сценария

1. Развернуть additive migration.
2. Развернуть server endpoint, ещё не используемый клиентами.
3. Выполнить contract/smoke tests.
4. Опубликовать централизованный desktop с feature flag `direct|api`, оставив
   production значение `direct` до проверки.
5. В согласованное окно закрыть запущенные процессы и централизованно включить
   `api`.
6. Выполнить production smoke и сравнить latency, errors, audit и результаты
   запросов.
7. При успехе открыть систему всем пользователям; при ошибке вернуть общий flag
   в `direct` и перезапустить desktop.
8. После rollback window удалить direct path.

### Rollback

- server image откатывается на предыдущую совместимую версию;
- desktop feature flag временно возвращает ещё существующий direct path;
- schema не откатывается автоматически;
- destructive migrations не выполняются до завершения rollback window;
- rollback outbox требует окна обслуживания: закрыть desktop, остановить
  сервер, вернуть предыдущий общий executable со встроенным worker и проверить
  совместимость event payload;
- credentials не возвращаются на рабочие места после окончательного сетевого
  закрытия без отдельного решения об аварийном режиме.

## 12. Эксплуатация

Необходимо подготовить инструкции:

- установка сервиса;
- проверка конфигурации;
- старт/стоп/restart;
- проверка readiness и версии;
- обновление image/binary;
- применение миграций;
- просмотр метрик и correlation по request ID;
- terminal outbox failure и requeue;
- PostgreSQL/MinIO/Seq outage;
- TLS certificate renewal;
- session revocation;
- backup/restore и проверка согласованности;
- rollback релиза;
- сбор диагностического пакета без secrets.

Для production должны быть определены владельцы:

- приложения;
- PostgreSQL;
- MinIO;
- TLS/DNS;
- backup и тестового restore;
- реакции на alerts;
- обновления desktop-парка.

## 13. Оценка трудоёмкости

Оценка для одного разработчика при доступной инфраструктуре и без полноценного
offline-режима:

| Этап | Оценка |
|---|---:|
| Baseline и архитектурные решения | 3–5 рабочих дней |
| Выделение worker | 1–3 недели |
| Production deployment/observability | 1–2 недели |
| System API и desktop API client | 1–2 недели |
| Request identity и authentication | 2–4 недели |
| Business API без вложений | 6–12 недель |
| Вложения | 2–4 недели |
| Закрытие прямого доступа и hardening | 1–3 недели |
| Rollout, документация и recovery tests | 2–4 недели, частично параллельно |

Полный переход оценивается примерно в 3–6 месяцев одного разработчика. Это не
календарное обязательство: после этапов 0 и 1 оценку необходимо пересчитать по
фактическим метрикам, числу endpoints и сложности permission rules.

Первый полезный production-инкремент — server-side worker — достижим отдельно и
не требует ожидания полного API.

## 14. Основные риски и меры

| Риск | Последствие | Мера |
|---|---|---|
| Смешивание identity параллельных запросов | Доступ от имени другого пользователя | Request-scoped immutable principal, race/integration tests |
| Несовместимые outbox consumers | Terminal failures или неверный эффект | Версия payload, consumer-first rollout |
| Старая desktop-версия против новой схемы | Ошибки или повреждение данных | Expand/contract, compatibility endpoint, minimum client version |
| Потерянный HTTP response после commit | Повторная команда | Idempotency key и сохранённый результат |
| API становится единой точкой отказа | Остановка рабочих мест | Restart policy, health, backup, мониторинг; HA только после измерений |
| Проксирование файлов перегружает сервис | Высокая latency/RAM/network | Streaming, limits; presigned URL после измерений |
| Секреты остаются в локальных config, памяти старых процессов или резервных копиях | Обход API | Закрытие процессов, удаление config, ротация credentials и firewall после миграции |
| Несогласованный rollback | Несовместимость binary/schema | Additive migrations и отдельное окно contract cleanup |
| Seq/логи раскрывают данные | Утечка credentials/PII | Structured redaction и logging tests |
| Несколько server replicas запускают scheduler | Двойные задания | Один replica сначала; позже advisory lock/leader election |

## 15. Решения, которые нужно принять до соответствующих этапов

До этапа 1:

- Docker Compose или systemd;
- серверная ОС и доступные ресурсы;
- где хранится production deployment manifest;
- способ доставки server secrets.

До этапа 3:

- DNS и CA/TLS;
- reverse proxy или Go TLS termination;
- формат/versioning API contract;
- требуемое окно совместимости desktop/server.

До этапа 4:

- idle и absolute session lifetime;
- политика повторного входа после рестарта desktop;
- требования к журналированию IP/device metadata;
- необходимость одновременных сессий одного пользователя.

До этапа 6:

- максимальный размер файлов и конкурентность;
- streaming proxy или необходимость presigned URL;
- требования антивирусной проверки/checksum.

До этапа 7:

- подтверждение публикации общего executable и завершения всех процессов,
  запущенных до публикации;
- firewall change window;
- план ротации старых credentials;
- аварийный порядок действий при недоступности API.

## 16. Общий Definition of Done

Переход считается завершённым, когда одновременно выполнено следующее:

- `docflow-server` воспроизводимо собирается, разворачивается и обновляется;
- outbox и регламентные задачи не зависят от запущенных desktop;
- все защищённые server operations используют request-scoped identity;
- desktop содержит только server URL и не содержит DB/MinIO secrets;
- PostgreSQL и MinIO недоступны из пользовательской сети;
- все доменные permissions проверяются сервером;
- команды с неопределённым результатом безопасно повторяются;
- вложения передаются потоково или через ограниченные presigned URL;
- health, metrics, alerts и operator procedures проверены;
- поддерживаемые версии desktop/server/schema формально определены;
- release gate, integration, end-to-end, security, load и реальный restore test
  пройдены;
- transitional direct paths и старые credentials удалены после rollback window;
- desktop binary не содержит outbox consumer;
- `README.md`, технический справочник, инструкции и release notes отражают
  фактическую production-архитектуру.
