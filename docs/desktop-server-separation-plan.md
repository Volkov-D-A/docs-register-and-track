# План разделения desktop- и серверного кода

Дата: 7 сентября 2026 года. Статус: выполняется; этапы 1–12 завершены 8 сентября, этапы 13–17 — 9 сентября 2026 года.

Основание: аудит кода после разделения вложений в коммите `76e90f7` и
[риск №3 из ревью перехода на сервер](server-transition-code-review.md).

## Цель и ограничения

Desktop отвечает за Wails API, локальные пользовательские файлы, интерфейс ОС и
HTTP-клиент. Сервер отвечает за аутентификацию запросов, права, бизнес-операции,
repository, БД, object storage и transactional outbox. Общие пакеты содержат
контракты и действительно общие функции без зависимости от одной из сторон.

Переход выполняется небольшими законченными этапами. После каждого этапа проект
собирается, переведённые сценарии проверены, а следующий этап можно выполнять
отдельным изменением. Каждый пункт с флажком ниже — отдельная единица работы;
объединять несколько пунктов в один большой рефакторинг не требуется.

**Обратную совместимость специально не поддерживаем.**

- При изменении типа, конструктора или пакета сразу переводим всех его
  потребителей, тесты и генерацию bindings в том же этапе.
- Не оставляем старые конструкторы, forwarding-обёртки, type aliases, re-export,
  deprecated-методы, двойную регистрацию Wails и feature flags для старого пути.
- Старый код удаляем после переноса актуальных сценариев. Тестовая потребность
  не является основанием сохранять production-реализацию legacy-пути.
- Уже существующие смешанные сервисы могут оставаться до своего этапа. Это
  очередь миграции, а не поддержка двух версий переведённого сервиса.
- Не поддерживаем старые пути Go-импортов, generated JS/TS-импортов и удалённые
  Wails-методы. Frontend и desktop backend обновляются вместе.
- Не вводим поддержку сочетаний старого desktop и нового сервера специально для
  этого перехода. Если контракт необходимо изменить, обе стороны обновляются
  согласованно. Существующие механизмы проверки версии отдельно не переделываем.
- HTTP endpoints, JSON и бизнес-правила не меняем без необходимости: перенос
  кода сам по себе не требует нового протокола, DTO-дубликатов или миграций БД.

Остаются один репозиторий и один `go.mod`. Выделение микросервисов, нескольких
Go-модулей, переделка схемы БД и исправление остальных рисков ревью не входят в
план. Принятое решение по риску №2 не меняется.

## Подтверждённое исходное состояние

- Вложения уже имеют независимые desktop- и server-типы, но оба находятся в
  `internal/services`. Это завершённое разделение объектов, а не пакетов.
- `DocumentQueryService` и `DocumentQueryEngine` также уже являются разными
  типами; им не нужно повторное разделение реализации.
- Поручения, ознакомления, регистрация документов, статистика, настройки,
  ссылки, журналы и ряд других сервисов совмещают HTTP-клиент и серверные данные.
- `AuthService` содержит прежнюю аутентификацию через БД, хотя production-сервер
  использует собственные HTTP-обработчики. Новый дублирующий server-auth тип не нужен.
- `UserService` и `DocumentAccessAdminService` имеют остаточные зависимости;
  прежние операции замещений и вычисления access summary используются тестами.
- В текущих bindings обнаружено 26 служебных методов: сеттеры зависимостей и
  `Startup`. Пользовательский `SetTheme` в это число не включён. Есть и другие
  лишние операции, например внутренний `AdminAuditLogService.LogAction`.
- Граф `go list -deps` показывает Wails runtime в зависимостях сервера через
  общий пакет services. Desktop зависит от database через services, lifecycle
  и `serverclient.MigrationClient`, использующий `database.MigrationStatus`.
  Repository и storage в граф desktop сейчас не входят.
- Общий logger содержит desktop-доставку логов через serverclient. Разделение
  одних сервисов не устранит все связи между сторонами.

Это описание графа компиляции; оно не означает, что desktop сейчас открывает
соединение с PostgreSQL или что сервер требует установленного GUI.

## Целевая структура

```text
main.go                         # точка входа Wails остаётся на месте
cmd/docflow-server/
frontend/

internal/
  desktop/
    app/                        # Wails composition root, startup/shutdown
    services/                   # только операции UI
    serverclient/               # HTTP, сессия, отмена запросов
    config/                     # локальная конфигурация
    logging/                    # Wails adapter, отправка логов через HTTP
  server/
    ...                         # текущий HTTP API и composition root
    services/                   # серверные бизнес-операции и document access
    ports/                      # интерфейсы repository/storage/principal
    effects/                    # построение journal/audit/user-event outbox
    repository/
    database/
    storage/
    config/                     # загрузка серверного окружения
    logging/                    # серверная настройка логирования
  dto/                          # общие DTO и HTTP-контракты
  models/                       # общие модели без инфраструктуры
  operations/                   # общий operation lifecycle и измерение операций
  observability/
  ...                           # другие пакеты с определённым владельцем
```

Небольшие support-пакеты создаём только под реальные зависимости. Не переносим
весь прежний `helpers.go` в новый универсальный `common` или `utils`. Например,
нумерация документов принадлежит серверу, а нормализация имени вложения нужна
обеим сторонам. Для оставшихся пакетов фиксируем владельца в этапе 1.

Имя Go-пакета desktop-сервисов можно оставить `services`, если это естественно
сохраняет Wails namespace. Это не требование совместимости: при изменении
namespace сразу обновляем frontend и bindings, без старого фасада.

## Правила проверки каждого этапа

1. Проверить все изменённые потребители и сборку обоих приложений. При переносе
   пакетов проверить также инструменты, mocks, тесты, Makefile и Dockerfile.
2. Запустить прицельные Go-тесты и `go vet` затронутых пакетов. Для сессии,
   lifecycle, startup и общего изменяемого состояния выполнить `go test -race`.
3. При изменении Wails API или пакета публикуемых типов выполнить
   `make wails-bindings`, проверить diff, frontend lint/build и относящиеся к
   сценарию frontend-тесты. Generated-файлы не редактировать вручную.
4. При изменении SQL, repository, транзакций или поведения БД выполнить
   `make integration-test`. Для переноса важных серверных сценариев использовать
   существующие HTTP/PostgreSQL integration tests; не заменять их mock-тестом.
5. При переносе только пакетов сохранять тесты поведения; не добавлять тесты,
   повторяющие расположение каждого файла. Границы импортов проверять отдельно.
6. Проверить удаление старых точек входа и отсутствие новых слоёв совместимости.
   Обновить ссылки документации через `make docs-links-check`.

После успешной проверки расширять или повторять её только при новых изменениях,
сбоях или оставшихся сомнениях. Полный release gate не нужен на каждом шаге.

Этап помечается завершённым только после выполнения его критериев. Недоступные
проверки записываются как незавершённые проверки с причиной, а не как успешные.
После восстановления окружения выполняется накопленная проверка применимых этапов.

## Последовательность этапов

### Подготовка независимых контрактов

- [x] **1. Зафиксировать границы и карту сценариев.**
  Составить перечень Wails-методов с реальными вызовами UI, отдельно отметить
  внутренние Go-вызовы. Зафиксировать точный текущий набор bindings тестом,
  совпадение двух регистраций и запрет расширения служебного API. Составить
  карту production-сценариев и старых тестов для auth, замещений и прав.
  По `go list` определить владельцев support-пакетов и порядок их зависимостей.
  Подготовить проверку импортов для новых целевых пакетов; включать её для
  каждого переносимого пакета, не дожидаясь завершения всего перехода.
  Результат: проверяемый исходный список; при каждом следующем этапе разрешённый
  список сокращается. Текущие лишние методы не объявляются целевым контрактом.

  Выполнено 08.09.2026: [границы и карта сценариев](desktop-server-boundaries.md),
  [перечень 157 Wails-методов и вызовов UI/Go](desktop-wails-api-baseline.md).
  Точный снимок JS/TS/Go проверяет `TestWailsAPIContract`; существующий
  composition-root тест проверяет совпадение runtime и generator Bind.
  Транзитивные границы новых пакетов проверяются автоматически через
  `internal/architecture`, включённый в штатный `make go-test`.
  Commit: пока не создан, изменения находятся в рабочем дереве.

  Проверки:

  - `GOCACHE=/tmp/go-build-cache go list -deps -json ./...` — успешно,
    владельцы и зависимости зафиксированы в карте.
  - `GOCACHE=/tmp/go-build-cache go test ./internal/app ./internal/architecture`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/app` — успешно.
  - `GOCACHE=/tmp/go-build-cache go vet ./internal/app ./internal/architecture`
    — успешно.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — общий проход остаётся незавершённым из-за четырёх
    ранее известных битых ссылок, перечисленных в конце плана; ошибок в новых
    документах нет.

  Production-код и Wails API не менялись; генерация bindings, frontend-проверки
  и PostgreSQL integration suite для этого подготовительного этапа не требуются.

- [x] **2. Отвязать контракты миграций от database.**
  Перенести статус миграций и необходимые потребителям данные совместимости в
  независимые DTO/контракты. Обновить HTTP-клиент, сервер, desktop/background
  lifecycle и тесты. Доступ к миграциям остаётся серверным.
  Результат: для чтения статуса serverclient и lifecycle не импортируют database;
  JSON, maintenance-поведение и обработка ошибок сохранены, старых aliases нет.

  Реализация 08.09.2026: `dto.MigrationStatus` и
  `models.MigrationCompatibilityError` вынесены без aliases. Расчёт совместимости
  остался в database. Lifecycle принимает функцию чтения статуса; путь к
  embedded migrations задаёт серверный composition root. Обновлены HTTP-клиент,
  настройки, сервер и тесты. В generated Wails API тип заменён на
  `dto.MigrationStatus`, поля JSON и имена методов сохранены.
  Архитектурный тест запрещает транзитивный database для serverclient/background.
  Commit: пока не создан, изменения находятся в рабочем дереве.

  Проверки:

  - `GOCACHE=/tmp/go-build-cache go test ./internal/database ./internal/dto ./internal/models ./internal/serverclient ./internal/background ./internal/app ./internal/services ./internal/server ./internal/architecture`
    — успешно; включает JSON-контракт, совместимость, maintenance и bindings.
  - `GOCACHE=/tmp/go-build-cache go vet ./internal/database ./internal/dto ./internal/models ./internal/serverclient ./internal/background ./internal/app ./internal/services ./internal/server ./internal/architecture`
    — успешно.
  - `make wails-bindings` — успешно; diff generated JS/TS проверен.
  - `make frontend-lint frontend-build` — успешно.
  - Из `frontend`: `npm run test:components -- test/components/adminTabs.test.tsx`
    — успешно, 3 теста.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/app ./internal/background ./internal/serverclient ./internal/server -run 'Lifecycle|Maintenance|Migration|Session'`
    — успешно.
  - Первый `make integration-test` — выполнен вне песочницы с доступом к Docker,
    выявил прежние ошибки: `TestAttachmentDeletionSagaIntegration` (tombstone visible)
    и `TestAdminOperationsAPIPersistsRequeueIntegration` (nil lifecycle).
    Изолированные контейнеры и тестовый том удалены штатной целью.
    Оба сбоя воспроизведены на исходном коммите `0f9fd0e` из отдельной копии:
    `go test ./internal/repository ./internal/server -run 'TestAttachmentDeletionSagaIntegration|TestAdminOperationsAPIPersistsRequeueIntegration' -count=1 -p=1`
    с `DOCFLOW_INTEGRATION_DSN` из Makefile и `GOCACHE=/tmp/go-build-cache`.
    Повторная прицельная проверка также выявила nil lifecycle в
    `TestServerAuthSessionLifecycleIntegration`.
  - `make docs-links-check` — те же четыре ранее известные битые ссылки,
    новых ошибок нет.

  Отдельная PostgreSQL-проверка затронутых сценариев прошла:
  `go test ./internal/background ./internal/server -run 'TestLifecycleProcessesOutboxAfterMigrationIntegration|TestDesktopSessionInvalidationIntegration|TestServerProcessesOutboxWithoutWailsIntegration' -count=1 -p=1 -v`
  с теми же DSN/cache. Проверены применение миграции и запуск outbox worker,
  серверный startup/shutdown, истечение/отзыв сессии, деактивация и сброс пароля.
  Временный стек остановлен через `make integration-db-down` с удалением тома.

  Исправление выявленных тестов 08.09.2026:

  - `TestAttachmentDeletionSagaIntegration` приведён к контракту repository:
    отсутствующее/удаляемое вложение возвращается как `(nil, nil)`. Проверяется
    доступность до удаления и отсутствие в поиске по ID и списке документа
    после установки отметки. SQL и production-поведение не изменялись.
  - Общий setup `newIntegrationManagementAPI` создаёт настоящий schema lifecycle,
    вызывает reconcile и проверяет готовность БД. На него переведены HTTP-тесты,
    ранее создававшие App без lifecycle; типизированный nil больше не попадает
    в интерфейс. Cleanup останавливает lifecycle до закрытия тестовой БД.
  - `GOCACHE=/tmp/go-build-cache go test ./internal/server ./internal/repository ./internal/app ./internal/architecture`
    и `GOCACHE=/tmp/go-build-cache go vet ./internal/server ./internal/repository`
    — успешно.

  Повторный полный `make integration-test` после исправлений — успешно:
  все PostgreSQL-пакеты, включая repository и server, прошли. Compose удалил
  тестовые контейнеры, сеть и том. Этап 2 завершён; следующий — этап 3.



- [x] **3. Выделить общие контракты команд документов.**
  Отделить передаваемые запросы регистрации, обновления, административного
  черновика и переопределения номера от серверных обработчиков. Использовать
  существующие dto/models, где они уже подходят; не заводить вторые копии.
  Результат: HTTP-клиент и будущие desktop-адаптеры не требуют server services
  ради типов запросов; нормализация не подменяет серверную валидацию и права.

  Реализация 08.09.2026: 13 типов команд регистрации/обновления четырёх видов
  документов, вложенных реквизитов, административного черновика и override
  перенесены в `dto/document_commands.go`. Общая нормализация JSON находится
  в dto; права, бизнес-валидация и преобразование override остаются в серверных
  обработчиках. Старые определения удалены без aliases. Поля, типы, JSON tags
  и порядок полей всех 13 структур сверены с исходной версией и сохранены.
  HTTP-клиент принимает типизированный `dto.AdminDraftCreateRequest`;
  обновлены потребители, frontend административного черновика и generated bindings.
  Commit: пока не создан, изменения находятся в рабочем дереве.

  Проверки:

  - `GOCACHE=/tmp/go-build-cache go test ./internal/dto ./internal/services ./internal/serverclient ./internal/server ./internal/app ./internal/architecture`
    — успешно. Новые тесты dto покрывают typed/JSON-преобразование четырёх видов,
    вложенные реквизиты, неизвестные поля и неверные типы; существующие тесты
    серверных обработчиков продолжают проверять права и бизнес-валидацию.
  - `GOCACHE=/tmp/go-build-cache go vet ./internal/dto ./internal/services ./internal/serverclient ./internal/server ./internal/app ./internal/architecture`
    — успешно.
  - `make wails-bindings` — успешно, generated diff проверен;
    namespace двух административных типов изменён с services на dto.
  - `make frontend-lint frontend-build` — успешно.
  - Из `frontend`: `npm run test:components -- test/components/adminTabs.test.tsx test/components/documentRegistration.test.tsx`
    — успешно, 6 тестов.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make integration-test` — успешно: полный PostgreSQL-прогон прошёл,
    тестовые контейнеры, сеть и том удалены штатной целью.

  Проверка и исправление прежних ссылок документации отложены по указанию
  пользователя; новых ссылок на этом этапе не добавлено.


- [x] **4. Вынести общий operation lifecycle и метрики.**
  Перенести нужные обеим сторонам lifecycle и функции измерения в operations.
  Перевести вызовы напрямую, удалить прежние определения.
  Результат: независимый пакет с тестами timeout, cancellation, shutdown и race;
  он не импортирует services, Wails, serverclient или серверную инфраструктуру.

  Выполнено 08.09.2026: `operations.Lifecycle`, `NewLifecycle`, `Measure` и
  `MeasureError` заменили определения в services. Все потребители используют
  новый пакет напрямую. Обёртка `serviceOperationContext` удалена: метод
  lifecycle уже поддерживает nil. Старых типов, функций и aliases не оставлено.
  Перенесены прежние lifecycle-тесты и добавлены timeout, deadline shutdown,
  повторный release, конкурентные start/release/shutdown и проверка метрик.
  Namespace типа в generated bindings теперь `operations.Lifecycle`.
  Служебные сеттеры пока остаются до этапов переноса соответствующих сервисов.
  Commit: пока не создан, изменения находятся в рабочем дереве.

  Проверки:

  - `GOCACHE=/tmp/go-build-cache go test ./internal/operations ./internal/services ./internal/app ./internal/server ./internal/architecture`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/operations ./internal/services ./internal/app ./internal/server -run 'Lifecycle|Attachment|Operation|Measure|Statistics|Session'`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go vet ./internal/operations ./internal/services ./internal/app ./internal/server ./internal/architecture`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go test ./internal/architecture -count=1`
    — успешно; проверен актуальный production-граф нового пакета без кэша теста.
  - `make wails-bindings` — успешно, generated diff проверен.
  - `make frontend-lint frontend-build` — успешно.
  - `make frontend-test`: utility-тесты прошли, 7 component-тестов превысили
    стандартный timeout 5 секунд при параллельном запуске проверок.
    Повторный полный component-прогон из `frontend`:
    `npm run test:components -- --maxWorkers=1` — 35 тестов в 12 файлах прошли.
    Таймауты и assertions не изменялись.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make integration-test` — полный PostgreSQL-прогон прошёл,
    тестовые контейнеры, сеть и том удалены штатной целью.

  Работы с прежними ссылками документации отложены по указанию пользователя;
  новых ссылок на этом этапе не добавлено.


### Очистка legacy и первые desktop-пакеты

- [x] **5. Перевести тесты замещений и access summary на production-путь.**
  Перенести актуальные проверки старых `saveForPrincipal` и
  `getCurrentAccessSummaryDirect` на действующие серверные обработчики.
  Проверить собственные/административные операции, замещения и audit/outbox.
  Удалить старые private-реализации и ненужную связь `desktopAuth` в document
  access. Результат: тесты проверяют код, используемый сервером, а не его копию.

  Реализация 08.09.2026: private `saveForPrincipal` и
  `getCurrentAccessSummaryDirect`, их тестовые HTTP-имитации и зависимость
  `DocumentAccessService.desktopAuth` удалены. Вместе с private-путями удалены
  неиспользуемые repository/auth/access-поля адаптеров замещений и метаданных
  видов документов; конструкторы и desktop composition root обновлены напрямую.
  Старый тест кандидатов сохранён в `user_service_test.go`.

  Матрица HTTP-тестов проверяет собственные и административные замещения:
  тот же/другой отдел, неактивный или отсутствующий заместитель, отсутствие
  participant-флага и подразделения, самозамещение, даты, очистка и права.
  Проверяются principal из сессии/URL, actor и атомарно передаваемый audit.
  Повторные set/clear вынесены из legacy service-теста в серверный
  PostgreSQL-тест для обоих режимов: состояние БД, уникальные outbox-ключи и
  доставка всех четырёх событий каждого режима в audit log.
  Проверки повторных блокировок login пока остаются до этапа 6.

  Access summary проверяется через `api.Handler()` для clerk, participant без
  полного чтения и admin без document permissions. Выявленное различие legacy
  maintenance-теста зафиксировано по production-поведению: общий HTTP gate
  возвращает 503 для `/api/v1/access/current` независимо от роли, а отдельный
  маршрут статуса миграций остаётся доступен. Порядок middleware и HTTP-контракт
  не менялись; старое ожидание сокращённого summary не перенесено как требование.
  Commit: пока не создан, изменения находятся в рабочем дереве.

  Проверки:

  - `GOCACHE=/tmp/go-build-cache go test ./internal/server ./internal/services ./internal/app ./internal/serverclient`
    — успешно; включает контракт Wails API и обе регистрации.
  - `GOCACHE=/tmp/go-build-cache go vet ./internal/server ./internal/services ./internal/app ./internal/serverclient`
    — успешно.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/app ./internal/services ./internal/server -run 'Substitution|CurrentAccessSummary|DocumentAccess|CompositionRoot'`
    — успешно.
  - `make integration-test`: первый прогон выявил ошибку нового тестового
    SQL-фильтра (`action` вместо фактического `Action` в audit JSON).
    Запрос исправлен; повторный полный прогон прошёл. Compose удалил
    тестовые контейнеры, сеть и том.
  - `GOCACHE=/tmp/go-build-cache go test ./internal/architecture -count=1`
    — успешно.

  Wails-методы и типы их аргументов/результатов не изменялись; повторная
  генерация bindings и frontend-проверки не требуются. Работы со ссылками
  документации отложены по указанию пользователя.


- [x] **6. Удалить legacy-аутентификацию.**
  Сопоставить старые auth-тесты с текущими HTTP-тестами: пароль, блокировка,
  обязательная смена, профиль, bootstrap, права, maintenance, завершение сессии.
  Перенести недостающее покрытие и перевести setup других service-тестов на
  fake principal без собственной реализации login. Затем удалить DB/repository
  fallback. Результат: одна production-реализация серверной аутентификации;
  прежние правила и принятое решение по bootstrap сохранены.

  Реализация этапа 6:
  - Из `AuthService` удалены DB/repository fallback, локальная проверка паролей,
    блокировка, bootstrap через БД и сеттеры хранилищ. Без HTTP-клиента адаптер
    возвращает ошибку конфигурации. Остальные зависимости передаются через
    существующие сеттеры до этапа 7.
  - Setup service-тестов использует `testPrincipal` без login и хеширования
    паролей; зависимости от конкретного AuthService заменены узкими интерфейсами.
  - Старые auth-сценарии сопоставлены с серверными проверками: пароль,
    пятая неудачная попытка, неактивная/заблокированная запись, сброс счётчика,
    обязательная смена и срок пароля — `auth_scenarios_test.go`; профиль —
    `profile_test.go`; bootstrap — `initial_setup_test.go`; maintenance —
    `maintenance_test.go`; отзыв/завершение сессий — auth/session integration
    и `serverclient/session_test.go`. Недостающие отрицательные сценарии добавлены.
  - Повторные блокировки проверяются через HTTP и PostgreSQL, включая две
    отдельные записи аудита. Используется существующий серверный аудит;
    механизм transactional outbox для аутентификации не менялся.
  - Bindings сгенерированы заново; удалены только два сеттера хранилищ,
    обновлён снимок Wails API. Пользовательские методы сохранены.

  Проверки этапа 6 (8 сентября 2026 года): `make go-test go-vet`,
  `make integration-test`, `make wails-bindings`, `make frontend-lint frontend-build`,
  `npm run test:utils` и `npm run test:components -- --maxWorkers=1`
  (35 тестов, 12 файлов) — успешно. Также прошли
  `GOCACHE=/tmp/go-build-cache go test ./internal/architecture -count=1` и
  `GOCACHE=/tmp/go-build-cache go test -race ./internal/services ./internal/serverclient`.
  В первом интеграционном прогоне исправлена фикстура блокировок: добавлен
  отдельный активный администратор для соблюдения ограничения БД.
  Общая документация и проверка её ссылок остаются отложенными.

- [x] **7. Сделать узкий desktop AuthService.**
  Передать HTTP-клиенты, lifecycle и metrics конструктору; отделить внутренний
  доступ к principal от публикуемых UI-методов. Перенести адаптер в
  `internal/desktop/services`, обновить обе регистрации Wails.
  Результат: нет auth-сеттеров и служебных permission/audit-методов в bindings;
  тесты revision сессии, запоздавших ответов, logout и `401` проходят.

  Реализация этапа 7:
  - `AuthService` перенесён в `internal/desktop/services`. HTTP auth/setup clients,
    lifecycle операций и metrics передаются конструктору; сеттеров нет.
  - Внутренние permission/audit-методы выделены в отдельный `Principal`, который
    не регистрируется в Wails. Проверка maintenance передаётся его конструктору;
    SettingsService получает узкий интерфейс principal.
  - Состояние и revision сессии принадлежат HTTP-клиенту. Локальная копия UUID
    удалена. Все сетевые операции адаптера участвуют в lifecycle и отменяются
    при shutdown, сохраняя прежние таймауты.
  - Обе регистрации Wails используют новый тип. Проверка composition root
    сравнивает полные Go-типы, включая пакет. Сгенерированы bindings и сокращён
    снимок API: осталось 10 пользовательских методов AuthService. Имя Wails
    namespace сохранено, frontend-импорты не меняются.
  - Тесты адаптера перенесены; добавлены HTTP-сценарии медленного logout,
    запоздавшего login, инвалидации principal после `401` и отмены при shutdown.
    Проверки запоздавших ответов и revision в serverclient сохранены.

  Проверки этапа 7 (8 сентября 2026 года): `make go-test go-vet`,
  `make integration-test`, `make wails-bindings`, `make frontend-lint frontend-build`,
  `npm run test:utils`, `npm run test:components -- --maxWorkers=1` — успешно.
  Прошли также архитектурные проверки с `-count=1` и
  `go test -race ./internal/desktop/services ./internal/serverclient`.
  Тесты локального HTTP-сервера выполнялись вне песочницы из-за запрета сокетов;
  тестовый PostgreSQL и его том удалены после интеграционного прогона.
  Общая документация по-прежнему отложена.

- [x] **8. Очистить адаптеры пользователей и прав.**
  Перевести `UserService`, `UserSubstitutionService`, `DocumentAccessAdminService`
  и `DocumentKindService` в desktop/services. Удалить неиспользуемые поля
  repository/auth/access, заменить сеттеры обязательными конструкторами.
  Результат: только HTTP-операции; серверные проверки не копируются в desktop.

  Реализация этапа 8:
  - Четыре адаптера перенесены в `internal/desktop/services`; неиспользуемые
    repository/auth/access-поля удалены. Клиенты обязательны в сигнатурах
    конструкторов. UserService требует также ExecutorClient, без необязательного
    type assertion и скрытого отключения списка исполнителей.
  - Обе регистрации Wails обновлены, bindings сгенерированы заново. Из снимка
    API удалены четыре SetServerClient; пользовательские методы и namespace
    сохранены.
  - Старые тестовые клиенты с собственной авторизацией, генерацией паролей
    и проверкой прав удалены. Desktop-тесты проверяют 13 HTTP-операций:
    передачу аргументов без локальных правил, результатов (включая временный
    пароль), ошибок клиента и сохранение таймаута.
  - Недостающие серверные HTTP-сценарии перенесены: запрет управления без admin,
    некорректные/отсутствующие цели, чтение профиля прав, деактивация сессии,
    ошибка списка исполнителей и конфликт последнего администратора.
    Удалён оставшийся невостребованный mapper этого конфликта из services;
    существующая серверная реализация проверяется через HTTP.

  Проверки этапа 8 (8 сентября 2026 года): `make go-test go-vet`,
  `make wails-bindings`, `make frontend-lint frontend-build` — успешно.
  Архитектурная проверка выполнена с `-count=1`; новые desktop-пакеты
  не зависят от services, repository или БД. Серверные HTTP-тесты и проверка
  Wails-контракта входят в общий Go-прогон.
  Интеграционный прогон с PostgreSQL не повторялся: SQL, репозитории и
  производственное поведение БД не менялись. Общая документация отложена.

- [x] **9. Перенести простые справочные адаптеры.**
  Перевести `DepartmentService`, `NomenclatureService`, `ReferenceService` в
  desktop/services; закрыть настройку зависимостей. Разобрать методы справочника
  типов документов: UI-операции оставить, неиспользуемые запреты CRUD удалить.
  Результат: поведение справочников сохранено, лишние Wails-методы удалены.

  Реализация 08.09.2026:
  - DepartmentService, NomenclatureService и ReferenceService вместе с тестами
    перенесены в `internal/desktop/services`. HTTP-клиенты передаются через
    конструкторы; три SetServerClient удалены.
  - Из ReferenceService удалены неиспользуемые CreateDocumentType,
    UpdateDocumentType и DeleteDocumentType. Чтение встроенного списка типов
    и его проверка аутентификации сохранены; тест использует узкий stub вместо
    старого principal с repository mock и дополнен проверкой отказа доступа.
  - Обе регистрации Wails обновлены, bindings сгенерированы штатной целью.
    Проверенный diff и снимок API содержат ровно шесть удалённых методов;
    остальные сигнатуры и namespace сохранены.
  Commit: общий коммит этапов 9–12
  `refactor: separate desktop adapters and server access policy`.

  Проверки этапа 9:
  - `GOCACHE=/tmp/go-build-cache go test ./internal/desktop/services ./internal/services ./internal/app ./internal/architecture`
    — успешно вне песочницы; первый запуск блокировал sandbox-запрет локального
    порта httptest. Проверены адаптеры, обе регистрации, Wails API и границы импортов.
  - `GOCACHE=/tmp/go-build-cache go vet ./internal/desktop/services ./internal/services ./internal/app ./internal/architecture`
    — успешно.
  - `make wails-bindings`, `make frontend-lint frontend-build` — успешно.
  - Из `frontend`: `npm run test:components -- test/components/adminTabs.test.tsx test/components/documentRegistration.test.tsx`
    — успешно, 6 тестов.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — общий проход не завершён: четыре описанные ниже
    битые ссылки и две ссылки из server-transition-code-review.md на удалённые
    ранее `internal/services/auth_service.go` и `internal/services/user_service.go`.
    Новые ссылки в этом этапе не добавлялись; общая документация остаётся
    отложенной до этапа 29.

  SQL, repository, поведение БД, сессии и concurrency не менялись;
  PostgreSQL integration и race для этого этапа не требуются. Нативный GUI
  вручную не проверялся.

- [x] **10. Перенести локальные desktop-сервисы.**
  Перевести Theme, ReleaseNote и System. Заменить `SystemService.Startup` на
  callback конструктора; сохранить пользовательские пути Linux/Windows.
  Результат: локальные настройки, заметки выпуска и системные операции работают,
  `Startup` отсутствует в bindings; `SetTheme` остаётся пользовательской операцией.

  Реализация 08.09.2026:
  - ThemeService, ReleaseNoteService и SystemService вместе с тестами перенесены
    в `internal/desktop/services`; обе регистрации и зависимости composition root
    обновлены. Локальные пути по-прежнему определяются через os.UserConfigDir;
    тесты путей задают XDG_CONFIG_HOME и AppData для Linux/Windows.
  - Единственный конструктор NewSystemService принимает HTTP-клиент и версию,
    возвращает сервис и callback запуска. Публичный Startup и прежние
    конструкторы удалены. Чтение/запись контекста защищены mutex.
  - Вместо теста присваивания поля проверяются наследование контекста Wails,
    отмена, таймаут 15 секунд, освобождение контекста после запроса и конкурентный
    запуск/чтение. Проверки совместимости, готовности, темы и release state сохранены.
  - Bindings сгенерированы штатной целью: изменение этого этапа — удаление
    SystemService.Startup; пользовательский SetTheme и остальные методы сохранены.
  Commit: общий коммит этапов 9–12
  `refactor: separate desktop adapters and server access policy`.

  Проверки этапа 10 (с `GOCACHE=/tmp/go-build-cache` для Go):
  - `go test ./internal/desktop/services ./internal/app ./internal/services ./internal/architecture -run 'SystemService|Theme|Release|EmbeddedCurrentRelease|Wails|CompositionRoot|ProductionImport|BoundaryPolicy'`
    — успешно; в старом services подходящих тестов больше нет.
  - `go vet ./internal/desktop/services ./internal/app ./internal/services ./internal/architecture`
    — успешно.
  - `go test -race ./internal/desktop/services ./internal/app -run 'SystemService|Theme|Release|EmbeddedCurrentRelease|CompositionRoot'`
    — успешно.
  - `make wails-bindings`, `make frontend-lint frontend-build` — успешно.
  - Из `frontend`: `npm run test:components -- test/components/systemBootstrapGate.test.tsx test/components/sessionApp.test.tsx`
    — успешно, 4 теста.
  - `CGO_ENABLED=0 go build -o /tmp/docflow-server-separation ./cmd/docflow-server`,
    `go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .` и
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/docflow-desktop-separation.exe .`
    — успешно.
  - `git diff --check` — успешно.

  PostgreSQL integration не запускалась: SQL, repository и поведение БД
  не менялись. Нативный GUI Linux/Windows вручную не проверялся. Ссылки
  документации не менялись; шесть ошибок общего прохода из этапа 9 остаются.

### Подготовка серверных сервисов

- [x] **11. Отделить серверные интерфейсы и эффекты.**
  Перенести repository/storage/principal-контракты в server/ports, построители
  outbox-событий — в server/effects. Общие UI/HTTP-типы туда не переносить.
  Обновить consumers и mocks напрямую. Разместить серверные support-функции по
  их владельцу, если они нужны следующим переносам.
  Результат: новые server-пакеты не импортируют прежний internal/services;
  семантика транзакционных эффектов и ключей дедупликации сохранена.

  Реализация 08.09.2026:
  - В `internal/server/ports` перенесены 53 интерфейса: repository/storage,
    транзакционные расширения хранилищ, серверные principal-контракты и настройки
    вложений. Потребители и тестовые doubles используют новые типы напрямую;
    aliases и forwarding-обёрток нет. Старый interfaces.go удалён.
  - Desktop settingsPrincipal, HTTP DocumentCommandClient и интерфейсы handlers
    остались у потребителей. DTO и модели не переносились. Дополнительные
    support-пакеты для следующего переноса DocumentAccessService не понадобились.
  - Три построителя journal/audit/user-event вынесены в `internal/server/effects`
    без изменения тел функций. Тесты проверяют тип события, ключ и payload;
    существующий PostgreSQL-тест коллизии дедупликации перенесён в effects.
  - Generated mocks не зависят от прежнего services и не требуют изменения
    сигнатур; добавлены compile-time проверки соответствия всех 14 mocks новым
    ports. Границы новых пакетов проверяются штатным архитектурным тестом.
  - Bindings сгенерированы: в двух оставшихся SetSubstitutionStore тип параметра
    изменён с services.UserSubstitutionStore на ports.UserSubstitutionStore.
    Удаление самих сеттеров остаётся в этапах разделения соответствующих сервисов.
  Commit: общий коммит этапов 9–12
  `refactor: separate desktop adapters and server access policy`.

  Проверки этапа 11:
  - `make go-test go-vet` — успешно вне песочницы. Первый запуск блокировал
    запрет локального порта httptest; полный повторный прогон завершился успешно.
  - `GOCACHE=/tmp/go-build-cache go test ./internal/architecture -count=1`
    — успешно; новые server-пакеты не импортируют services/desktop/Wails.
  - `make wails-bindings`, `make frontend-lint frontend-build` — успешно.
  - Из `frontend`: `npm run test:components -- test/components/assignmentSeriesControls.test.tsx test/components/assignmentCompletionModal.test.tsx`
    — успешно, 3 теста.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make integration-test` — успешно 08.09.2026 после запуска Docker Desktop,
    вне песочницы с доступом к Docker daemon. Полный PostgreSQL-прогон включает
    repository, серверные HTTP-сценарии и перенесённый в server/effects тест
    коллизии дедупликации. Контейнер, тестовый том и сеть удалены штатной целью.
    Первые попытки до запуска Docker останавливались на проверке окружения;
    после запуска sandbox по-прежнему не предоставлял доступ к daemon.
  - `make docs-links-check` — те же шесть прежних битых ссылок, что в этапе 9.
    `git diff --check` — успешно.

  SQL, алгоритмы транзакций и concurrency не менялись. Race повторно не запускался;
  PostgreSQL-проверка перенесённых серверных сценариев unit-тестами не заменяется.

- [x] **12. Перенести DocumentAccessService.**
  Перевести серверную политику доступа и необходимые ей helpers в server/services.
  Использовать request principal; убрать зависимость от concrete desktop auth.
  Результат: проверки домена, номенклатуры, участия, замещений и действий
  выполняются сервером; новые desktop-сервисы не импортируют эту реализацию.

  Реализация 08.09.2026:
  - DocumentAccessService перенесён в `internal/server/services`, все потребители
    и серверные фабрики используют новый тип напрямую. Зависимости — модели
    и server/ports; identity предоставляет request principal через интерфейс.
    Старый production-файл удалён, aliases и обёрток нет.
  - Dashboard использует опубликованный серверный метод чтения субъектов
    замещения. Отбор получателей уведомлений по явному разрешению перенесён
    в метод политики доступа: поручения и ознакомления больше не читают её
    приватный accessRepo. Общие для этих серверных сценариев операции с ID
    размещены рядом в user_ids.go.
  - Перенесены 15 групп тестов политики доступа: домен, номенклатуры, участие,
    замещения, пакетное чтение, действия, связи и журнал. Principal в этих тестах
    — узкий stub; отзыв сессии и деактивация проверяются серверными HTTP-тестами.
    Общие fixtures оставшихся command/query-тестов сохранены отдельно.
  - Тесты команд передают изменяемый тестовый document store при создании;
    тесты поручений и связей настраивают зависимости через конструктор.
    Прямые изменения приватных полей DocumentAccessService из старого пакета
    устранены, production-сеттеры ради тестов не добавлялись.
  Commit: общий коммит этапов 9–12
  `refactor: separate desktop adapters and server access policy`.

  Проверки этапа 12:
  - `GOCACHE=/tmp/go-build-cache go test ./internal/services ./internal/server/services ./internal/server ./internal/architecture -run 'DocumentAccess|ProductionImport' -count=1`
    — успешно; политика доступа и транзитивные границы нового пакета проверены.
  - `make go-test go-vet` — успешно.
  - `make integration-test` — успешно вне песочницы с доступом к Docker daemon;
    полный PostgreSQL-прогон проверяет серверные HTTP-сценарии и repository.
    Контейнер, тестовый том и сеть удалены штатной целью. Запуск в sandbox
    остановился на недоступности daemon до создания стека.
  - `make wails-bindings` — успешно, дополнительных изменений bindings
    относительно этапа 11 нет.
  - `make frontend-lint frontend-build` — успешно.
  - Из `frontend`: `npm run test:components -- test/components/accessVisibility.test.tsx test/components/assignmentCompletionModal.test.tsx`
    — успешно, 5 тестов.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — те же шесть прежних битых ссылок, что в этапе 9;
    новых ошибок нет. `git diff --check` — успешно.

  Сессии, lifecycle и concurrency не менялись; race повторно не запускался.
  Нативный GUI вручную не проверялся.

- [x] **13. Разделить SettingsService.**
  Оставить управление настройками и миграциями в desktop/services, а чтение
  repository и серверную интерпретацию настроек — в server/services.
  Передать зависимости конструктором, не публиковать внутренние getters без UI.
  Результат: нет выбора settingsStore/settingsClient в одном объекте; default
  limits/types, maintenance и миграционные сценарии сохранены.

  Реализация 09.09.2026:
  - Desktop SettingsService перенесён в `internal/desktop/services`: HTTP-клиенты,
    principal и узкий callback lifecycle передаются конструктором. Сохранены
    семь UI-методов, проверки миграций без schema-readiness и сериализация
    apply/rollback. Внутренние setters и три getter-метода без UI удалены.
  - Независимый SettingsService в `internal/server/services` читает SettingsStore
    и интерпретирует размер, типы файлов и разрешение файлов при завершении
    поручения. Фабрика вложений и её тесты используют серверный тип напрямую.
    Лимиты в HTTP-валидации используют серверную константу.
  - Перенесены тесты desktop-операций и серверных defaults; добавлены проверки
    отсутствующих HTTP-клиентов и граничных размеров файлов. Удалены смешанный
    сервис, ConfigureSchemaLifecycle и неиспользуемые legacy migration helpers
    и fixtures. Старых aliases и forwarding-обёрток нет.
  - Обе Wails-регистрации переведены, bindings сгенерированы штатной командой;
    точный API-снимок сокращён на пять методов. Namespace UI сохранён.
  Commit: общий коммит этапов 13–15
  `refactor: separate settings attachments and document queries`.

  Проверки этапа 13:
  - `make go-test go-vet` — успешно вне песочницы; sandbox запрещал создание
    локального HTTP listener в существующем auth-тесте.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/desktop/services ./internal/app -run 'Settings|Migration|Lifecycle|Maintenance'`
    — успешно.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/desktop/services ./internal/server/services -run Settings`
    и `GOCACHE=/tmp/go-build-cache go vet ./internal/desktop/services ./internal/server/services`
    — успешно после добавления граничных тестов.
  - `make integration-test` — успешно вне песочницы с доступом к Docker;
    PostgreSQL-сценарии настроек, audit outbox, миграций и вложений прошли.
    Контейнер, тестовый том и сеть удалены штатной целью.
  - `make wails-bindings` — успешно; diff JS/TS проверен.
  - `make frontend-lint frontend-build` — успешно.
  - `npm --prefix frontend run test:components -- test/components/adminTabs.test.tsx test/components/assignmentCompletionModal.test.tsx`
    — успешно, 4 теста.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — те же шесть прежних битых ссылок, что в этапе 12;
    новых ошибок нет. `git diff --check` — успешно.

  HTTP-контракты, SQL и бизнес-правила не изменены. Нативный GUI вручную
  не проверялся.

### Перенос и разделение предметных сервисов

В этапах 14–23 одновременно переводятся desktop-регистрация, HTTP-фабрика,
bindings, тесты и импорты. Для каждого сервиса старый тип/файл удаляется в его
этапе. Оставшиеся сервисы временно живут на прежнем месте, без новых обёрток.

- [x] **14. Перенести уже разделённые вложения.**
  Desktop AttachmentService — в desktop/services, ServerAttachmentService — в
  server/services. Общие правила имени разместить независимо от обеих сторон.
  Результат: ровно десять UI-методов; streaming, закрытие файлов/body,
  compensation, permissions и PostgreSQL lifecycle подтверждены тестами.

  Реализация 09.09.2026:
  - AttachmentService и его тесты перенесены в `internal/desktop/services`,
    ServerAttachmentService и серверные тесты — в `internal/server/services`.
    Обновлены runtime/generator Wails-регистрации, HTTP-фабрика и проверка
    соответствия attachmentAPI. Старые файлы удалены без aliases и обёрток.
  - Общие правила имени файла и их тесты перенесены в независимый пакет
    `internal/attachmentname`; пакет включён в проверку общих транзитивных
    зависимостей. Проверка typed nil остаётся локальной в конструкторах.
  - Сохранены тесты streaming, закрытия выбранного файла и HTTP body, ошибок
    скачивания, compensation, прав, повторной проверки поручения после upload,
    outbox-удаления и reconciliation. Серверные fixtures используют узкий
    principal без desktop auth. Точный контракт проверяет десять UI-методов.
  - Обновлены ссылки на реализации и тесты в серверном ревью.
  Commit: общий коммит этапов 13–15
  `refactor: separate settings attachments and document queries`.

  Проверки этапа 14:
  - `make integration-test` — успешно вне песочницы с доступом к Docker:
    полный PostgreSQL-прогон, включая HTTP lifecycle вложений и outbox.
    Первый запуск выявил зависимость перенесённого теста от старого fixture;
    после исправления повторный прогон прошёл. Контейнер, том и сеть удалены.
  - `make go-test` и `make go-vet` — успешно; полный Go-прогон выполнен вне
    песочницы для локальных HTTP listeners. При переносе исправлены зависимости
    тестов от старых fixtures и относительный путь к generated bindings.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/desktop/services ./internal/server/services ./internal/app -run 'Attachment|CompositionRoot'`
    — успешно.
  - `make wails-bindings` — успешно; дополнительных изменений bindings
    относительно этапа 13 нет.
  - `make frontend-lint frontend-build` — успешно.
  - `npm --prefix frontend run test:components -- test/components/assignmentCompletionModal.test.tsx`
    — успешно, 1 тест.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — те же шесть прежних битых ссылок; новых ошибок нет.
    `git diff --check` — успешно.

  HTTP-контракты, SQL и поведение операций не менялись. Нативный GUI вручную
  не проверялся.

- [x] **15. Перенести уже разделённые запросы документов.**
  Desktop DocumentQueryService — в desktop/services, Engine, registry и query
  handlers — на сервер. Удалить desktop-сеттеры.
  Результат: карточки, фильтры и ограничение области чтения сохранены.

  Реализация 09.09.2026:
  - DocumentQueryService перенесён в `internal/desktop/services`; HTTP-клиент
    и metrics передаются конструктором. В Wails опубликованы только GetByID
    и GetList, оба setters удалены. Runtime/generator регистрации и точный
    снимок API обновлены, bindings сгенерированы штатной командой.
  - DocumentQueryEngine, DocumentKindQueryRegistry и четыре query handlers
    перенесены в `internal/server/services` вместе с тестами. HTTP-фабрика
    использует новые типы и передаёт metrics при создании engine.
    Старые файлы удалены, aliases и forwarding-обёрток нет.
  - Сохранены проверки карточек, фильтров, пагинации и доступа. Дополнительно
    проверены передача метрик через конструктор, ошибки HTTP-клиента и замена
    переданной неограниченной AccessScope серверной ограниченной областью.
  Commit: общий коммит этапов 13–15
  `refactor: separate settings attachments and document queries`.

  Проверки этапа 15:
  - `make integration-test` — успешно вне песочницы с доступом к Docker;
    полный PostgreSQL-прогон включает списки и карточки с серверным доступом,
    фильтрацию и пагинацию. Контейнер, тестовый том и сеть удалены штатной целью.
  - `make go-test go-vet` — успешно вне песочницы с поддержкой локальных
    HTTP listeners; включает Wails-контракт и транзитивные границы импортов.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/desktop/services ./internal/server/services -run 'DocumentQuery|QueryHandler|QueryRegistry'`
    — успешно.
  - `make wails-bindings` — успешно; diff JS/TS удаляет только два setters
    DocumentQueryService относительно этапа 14.
  - `make frontend-lint frontend-build` — успешно.
  - `npm --prefix frontend run test:components -- test/components/documentRegistration.test.tsx test/components/accessVisibility.test.tsx`
    — успешно, 7 тестов.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — те же шесть прежних битых ссылок; новых ошибок нет.
    `git diff --check` — успешно.

  HTTP-контракты, SQL и бизнес-правила не менялись. Нативный GUI вручную
  не проверялся.

- [x] **16. Разделить регистрацию документов.**
  Desktop DocumentRegistrationService оставляет HTTP-вызовы; сервер получает
  независимый исполнитель и command handlers. Перевести внутренние типы и
  helpers, не создавая второго набора DTO.
  Результат: регистрация/изменение всех видов документов, номера, admin draft,
  override, права и атомарные эффекты проверены на серверном пути.

  Реализация 09.09.2026:
  - DocumentRegistrationService перенесён в `internal/desktop/services`;
    HTTP-клиент, lifecycle и metrics передаются конструктору. В Wails остаются
    Register, Update и CreateAdminDraft; два служебных setters удалены.
  - Независимый DocumentCommandEngine, registry, четыре command handlers,
    правила admin override, проверки количества листов и тесты перенесены в
    `internal/server/services`. HTTP-фабрика использует новый исполнитель.
    Удалена неиспользуемая зависимость handlers от JournalService; запись
    эффектов остаётся через атомарные repository/outbox операции.
  - Общие DTO сохранены. Старые реализации удалены без aliases и обёрток.
    Сохранены серверные тесты прав и команд; добавлены проверки HTTP-адаптера,
    метрик и отмены при shutdown, а также HTTP/PostgreSQL-сценарий для четырёх
    видов: регистрация, идемпотентность, изменение, admin draft с литерой,
    запрет регистрации без прав и отсутствие дублирования journal effects.
  Commit: `refactor: separate document registration desktop and server`.

  Проверки этапа 16:
  - `make go-test go-vet` — успешно вне песочницы для локальных HTTP listeners.
    После добавления тестов повторены прицельные race/vet затронутых пакетов.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/desktop/services ./internal/server/services -run 'DocumentRegistration|DocumentCommand|CommandHandler|AdminNumber|PageCounts'`
    — успешно.
  - `make wails-bindings` — успешно; diff JS/TS удаляет только два setters.
  - `make frontend-lint frontend-build` — успешно.
  - `npm --prefix frontend run test:components -- test/components/documentRegistration.test.tsx test/components/accessVisibility.test.tsx`
    — успешно, 7 тестов.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make integration-test` — успешно вне песочницы, включая новый HTTP-тест
    четырёх видов документов. При подготовке теста исправлены обязательные поля
    fixture и учтены существующие форматы JSON journal effects. Финальный полный
    PostgreSQL-прогон прошёл; контейнер, тестовый том и сеть удалены штатной целью.
  - `make docs-links-check` — те же шесть прежних битых ссылок; новых ошибок нет.
    `git diff --check` — успешно.

  HTTP-контракты, SQL и бизнес-правила не менялись. Нативный GUI вручную
  не проверялся.

- [x] **17. Разделить UserEventService.**
  Отделить UI-чтение/подтверждение событий от серверных операций, зависимостей
  repository и principal. Результат: формирование событий остаётся серверным,
  UI получает только свои события; потребители не требуют legacy-сервиса.

  Реализация 09.09.2026:
  - HTTP-адаптер UserEventService перенесён в `internal/desktop/services`,
    клиент передаётся конструктору. Серверное чтение и подтверждение событий
    перенесены в `internal/server/services` с request principal и repository.
    Runtime/generator регистрации и HTTP-фабрика обновлены.
  - Удалены смешанный сервис, неиспользуемый прямой метод create и остаточные
    зависимости AssignmentService/AcknowledgmentService от UserEventService.
    Формирование событий остаётся в серверных transactional outbox effects;
    тесты поручений включают существующую настройку emitUserEvents напрямую.
  - Сохранены проверки серверного чтения и подтверждения. Добавлены тесты
    изоляции отметок по получателю, обязательного principal, HTTP-адаптера,
    передачи фильтра/ID, ошибок клиента и отсутствующего клиента.
  Commit: `refactor: separate user events desktop and server`.

  Проверки этапа 17:
  - `make go-test go-vet` — успешно вне песочницы для локальных HTTP listeners.
  - `GOCACHE=/tmp/go-build-cache go test -race ./internal/desktop/services ./internal/server/services -run UserEvent`
    — успешно; после финальных изменений повторены серверные race/vet.
  - `make integration-test` — успешно вне песочницы; включает HTTP/PostgreSQL
    сценарий ограничения событий получателем и сценарии transactional outbox.
    Изолированный контейнер, том и сеть удалены штатной целью.
  - `make wails-bindings` — успешно, generated JS/TS diff пустой: пять методов
    и namespace сохранены. Wails-контракт и архитектурные тесты проходят.
  - `make frontend-lint frontend-build` — успешно.
  - `npm --prefix frontend run test:components -- test/components/accessVisibility.test.tsx`
    — успешно, 4 теста.
  - `CGO_ENABLED=0 GOCACHE=/tmp/go-build-cache go build -o /tmp/docflow-server-separation ./cmd/docflow-server`
    и `GOCACHE=/tmp/go-build-cache go build -tags webkit2_41 -o /tmp/docflow-desktop-separation .`
    — успешно.
  - `make docs-links-check` — те же шесть прежних битых ссылок; новых ошибок нет.
    `git diff --check` — успешно.

  HTTP-контракты, SQL и бизнес-правила не менялись. Нативный GUI вручную
  не проверялся.

- [ ] **18. Разделить AssignmentService.**
  Перенести все HTTP-операции в desktop, серверу оставить поручения и серии,
  замещения, переходы статусов и outbox. Настройки событий передать при создании.
  Результат: create/update, исполнение, возврат, завершение итерации и создание
  следующей, история серии и вложения проходят существующие сценарии.

- [ ] **19. Разделить AcknowledgmentService.**
  Выделить desktop-адаптер и серверную реализацию, передать substitutions/events
  конструктору. Результат: назначения, ознакомление, замещения и события
  сохранены; серверный сервис не ссылается на desktop UserEventService.

- [ ] **20. Разделить LinkService и AdministrativeOrderService.**
  Перенести HTTP-адаптеры и серверные реализации. Результат: права на оба
  документа, связи, последствия отмены приказа и ознакомление по приказу
  проверены; локального доступа к документам у desktop нет.

- [ ] **21. Разделить журналы.**
  Перевести JournalService и AdminAuditLogService. Удалить `LogAction` из UI API;
  внутренние записи и транзакционный audit остаются на сервере.
  Результат: чтение журналов защищено прежними правами, запись не доступна через
  произвольный Wails-вызов; setters отсутствуют.

- [ ] **22. Разделить DashboardService и OutboxAdminService.**
  Создать независимые реализации в целевых пакетах.
  Результат: dashboard учитывает права, административные операции outbox
  защищены сервером; UI получает те же результаты и ошибки.

- [ ] **23. Разделить StatisticsService.**
  Desktop оставляет HTTP-запросы; отчёты, конкурентные запросы, storage refresh
  и system diagnostics принадлежат серверу. Передать metrics/diagnostics
  конструктору. Результат: фильтры, права на отчёты, timeout, refresh/lease и
  метрики проверены; desktop не содержит storage и серверной диагностики.

### Завершение структуры каталогов

- [ ] **24. Разделить конфигурацию и доставку логов.**
  Отделить desktop config и серверные env-настройки, Wails/HTTP logging и
  серверное логирование. Общую обработку записей оставить независимой;
  узкие интерфейсы определить у потребителя.
  Результат: серверное логирование не импортирует desktop HTTP-клиент,
  desktop-конфигурация не требует серверной инфраструктуры.

- [ ] **25. Перенести HTTP-клиент и desktop composition root.**
  Перевести serverclient и app в desktop, обновить main, генератор Wails,
  lifecycle и tooling. Проверить оставшиеся background/schema зависимости.
  Результат: корневой Wails main собирает только desktop-граф; серверный
  production-код не импортирует desktop. HTTP-тесты могут использовать клиент.

- [ ] **26. Перенести database и repository.**
  Перенести database вместе с embedded migrations, затем repository в server.
  Одновременно обновить импорты, пути в tests/tools и конфигурацию генераторов.
  Результат: схемы и SQL не изменены, миграции доступны серверной сборке,
  PostgreSQL integration suite проходит.

- [ ] **27. Перенести storage и оставшуюся серверную инфраструктуру.**
  Перевести object storage, серверные workers/координаторы по карте этапа 1;
  общий код не переносить на сервер только ради очистки дерева.
  Результат: streaming/outbox/reconciliation сохранены, Dockerfile и команды
  запуска используют актуальные пути, прежние каталоги удалены.

- [ ] **28. Закрыть границы архитектурными проверками.**
  Удалить остатки internal/services и исключения исходного списка Wails.
  Проверять полный граф production-импортов: desktop не зависит от server,
  database/repository/storage; server не зависит от desktop и Wails; общие
  пакеты не зависят от обеих сторон. Проверять runtime и generator Bind,
  точные списки UI-методов, отсутствие lifecycle/setters и серверных типов.
  Результат: проверки добавлены в штатные цели/CI и предотвращают возврат смешения.

- [ ] **29. Итоговая проверка и обновление архитектурной документации.**
  Выполнить полный набор Go-тестов/vet, race для затронутых concurrency-пакетов,
  PostgreSQL integration suite, frontend lint/test/build и повторную генерацию
  bindings. Проверить server build с `CGO_ENABLED=0`, desktop Linux/Windows и
  нативные сценарии выбора/скачивания/открытия файлов, login/logout, maintenance.
  Обновить README, tech_docs, ревью и команды, удалить устаревшие описания.
  Результат: все критерии ниже подтверждены, ограничения явно записаны.

## Как управлять размером этапов

Этапы выполняются сверху вниз; следующий этап использует уже перенесённые
контракты и зависимости. Если обнаружена незапланированная общая зависимость,
сначала выносится минимальный самостоятельный подготовительный этап с переводом
всех её потребителей. Серверный пакет не должен импортировать старый services
ради обхода цикла, а desktop-пакет — тянуть его транзитивно.

Парные этапы 20–22 и перенос инфраструктуры 26 можно разделить на изменения по
одному сервису/пакету. Большой сервис, например поручения, можно предварительно
разгрузить переносом чистых серверных helpers. Само переключение его реализации
и потребителей остаётся атомарным: не оставляем половину публичных методов в
старом фасаде, а половину в новом.

В каждом завершённом пункте записываются commit, выполненные команды и результат.
Отдельный журнал совместимости и срок поддержки legacy API не ведутся.

## Критерии завершения всего перехода

- В Wails публикуется только согласованный UI API; callbacks и конфигурация
  зависимостей не являются публичными методами сервисов.
- Нет объектов, выбирающих между HTTP и repository по наличию клиента.
- Серверные бизнес-операции используют request principal, desktop — HTTP API.
- Ни один новый или перенесённый пакет не возвращает зависимость через старый
  фасад, alias или универсальный пакет со смешанным содержимым.
- Границы импортов выполняются транзитивно для production-графов обоих приложений.
- Старые реализации и тесты, обслуживающие только их, удалены; актуальное
  покрытие относится к production-коду.
- Права, session invalidation, maintenance, streaming, transactional effects
  и пользовательские сценарии сохранены и проверены.
- Нет дополнительных механизмов обратной совместимости для этого перехода.

## Известные ограничения окружения на момент составления

В предыдущей реализации Docker Desktop был недоступен в WSL, поэтому
PostgreSQL-интеграция не выполнялась. Перед этапами с этой проверкой необходимо
обеспечить штатный integration-стек; подмена unit-тестами не закрывает проверку.
Нативный GUI также не проверялся вручную.

Обновление 08.09.2026 (этап 2): Docker доступен вне песочницы; PostgreSQL-стек
запущен и удалён штатными целями. Выявленные прежние сбои тестов исправлены;
повторный полный PostgreSQL-прогон прошёл. Подробности — в записи этапа 2.

`make docs-links-check` ранее выявлял четыре существующие битые ссылки:
две на отсутствующий `docs/bugs.md` и две на
`docs/https-internal-ca-setup.md`. Эти дефекты не относятся к разделению;
новые ссылки плана должны быть корректными независимо от них.
