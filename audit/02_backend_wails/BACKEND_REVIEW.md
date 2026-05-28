# Backend Review

Дата аудита: 2026-05-27
Этап: C. Go backend, Wails bridge, DTO, ошибки, ресурсы

## Общий Вывод

Backend имеет понятную слоистую структуру: Wails bindings в `main.go` публикуют сервисы из `internal/services`, сервисы используют DTO/mappers и репозитории, репозитории работают с PostgreSQL через `database/sql`. Основные SQL-запросы параметризованы; динамическая сборка SQL в списках строится из backend-controlled fragments и параметров `$N`, прямой подстановки пользовательских значений в SQL не найдено.

Главные production-риски этапа C:

- Wails `ErrorFormatter` превращает все ошибки в строку, поэтому frontend не получает стабильный структурированный error contract;
- часть ошибок backend/repository возвращается как `fmt.Errorf` с внутренним контекстом, английскими сообщениями и wrapped DB errors;
- регистрация документов все еще не реализует backend `idempotency_key`, а `GetNextNumber` вызывается до создания документа;
- публичный `DocumentRegistrationService.Register/Update` принимает `any` и нормализует payload через `json.Marshal`/`json.Unmarshal`, лишние поля silently ignored;
- долгие операции используют `context.Background()` и не отменяются при закрытии окна;
- production-логи добавляют `app_user` с ФИО во все события и могут содержать названия файлов, номера документов и имена пользователей.

## Critical/Major Issues

| Issue | Severity | Кратко | Ответственный этап |
| --- | --- | --- | --- |
| ISSUE-012 | major | Нет структурированного Wails/backend error contract со стабильными кодами. | C/D |
| ISSUE-013 | major | Регистрация документов не имеет backend `idempotency_key`; номер выдается до transaction create. | C/D/F |
| ISSUE-014 | major | Document registration Wails API принимает `any`, неизвестные поля не отклоняются. | C/D |
| ISSUE-015 | major | Операции MinIO/журналов/связей используют `context.Background()` и не отменяются при shutdown. | C/E/F |
| ISSUE-016 | major | Production-логи содержат ФИО/имена пользователей и business identifiers без явной minimization policy. | C/F/H |

## Контрольные Пункты C

| Код | Статус | Severity | Место | Доказательство | Вывод / рекомендация |
| --- | --- | --- | --- | --- | --- |
| C.01.077 | ok | none | repositories | Динамические list-запросы собирают SQL из фиксированных fragments; пользовательские значения идут через args/`$N`. | Явной unsafe SQL-конкатенации пользовательских значений не найдено. |
| C.01.078 | ok | none | repositories | `Query`, `QueryRow`, `Exec` используют positional parameters; dynamic filters добавляют args. | Параметризация в целом соблюдена. |
| C.01.079 | issue | major | services/repositories | Многие repo errors возвращаются как `fmt.Errorf("failed to ...: %w", err)`, а Wails отдает `err.Error()`. | Ввести boundary mapping: DB/internal errors -> structured safe app errors + internal log. |
| C.01.080 | issue | major | `models.AppError`, `main.go` | `AppError` содержит только числовой HTTP-like `Code`; `ErrorFormatter` возвращает строку. | Нужны стабильные machine-readable codes: `VALIDATION_ERROR`, `FORBIDDEN`, `NOT_FOUND`, etc. |
| C.01.081 | issue | major | Wails bindings | В UI может уйти `failed to create document root`, `pq`/DB context или английские file/storage ошибки. | Пользователю отдавать безопасное сообщение, подробности только в лог. |
| C.01.082 | issue | minor | logger/ErrorFormatter | Ошибки логируются, но нет централизованной redaction policy; часть internal errors может содержать значения. | Добавить sanitization/redaction guidelines и не логировать секреты/payloads. |
| C.02.083 | issue | major | Wails Bind services | Контракты выводятся из Go method signatures; отдельного registry/schema для публичных methods нет. | Зафиксировать публичный API contract в `WAILS_CONTRACTS_REVIEW.md` и стабилизировать ошибки. |
| C.02.084 | issue | major | command handlers | Валидация есть частично: UUID/date/permission проверяются, но часть required strings/pages проверяется неодинаково. | Упростить: отдельные validate methods для каждого request с единым `AppError`. |
| C.02.085 | issue | major | `DocumentRegistrationService.Register/Update` | Payload `any` декодируется через `json.Unmarshal`; unknown fields silently ignored. | Декодировать strict-mode или использовать typed public methods/DTO. |
| C.03.086 | issue | major | document command requests | DTO регистрации не содержат `idempotencyKey`; публичный `any` contract нестабилен. | Добавить обязательный `idempotencyKey` и typed request DTO. |
| C.03.087 | ok | none | dto/models | DTO в основном строковые ID и не раскрывают DB-only UUID fields; mappers отделяют models от DTO. | Существенной утечки DB schema в DTO не найдено. |
| C.03.088 | issue | minor | DTO dates | Response использует `time.Time`, command requests часто принимают date strings `YYYY-MM-DD`, местами RFC3339. | Зафиксировать единый date/datetime contract для TS. |
| C.03.089 | issue | minor | DTO nullable | Nullable строки представлены `*string`, но request layer использует пустую строку как null. | Зафиксировать empty-string/null policy и привести команды к ней. |
| C.03.090 | issue | major | frontend/backend errors | Backend не возвращает structured error codes; frontend вынужден ориентироваться на строки. | После `DECISION-007` frontend должен читать code/message, не PostgreSQL text. |
| C.04.091 | issue | major | validation errors | Часть validation returns `models.NewBadRequest`, часть `fmt.Errorf`. | Единый validation error contract. |
| C.04.092 | ok | none | access/auth services | Unauthorized/forbidden в основном идут через `models.ErrUnauthorized`/`ErrForbidden`. | Права предсказуемее остальных ошибок; сохранить codes. |
| C.04.093 | ok | none | not found paths | Missing document/assignment/user target/acknowledgment row/nomenclature/department paths now return structured `NOT_FOUND`/404 instead of validation, silent nil or plain not-found errors. | Keep nullable read responses explicit; do not introduce new plain `fmt.Errorf("... not found")` production paths. |
| C.05.094 | issue | major | services/storage | MinIO, journal, link, statistics используют `context.Background()`. | Пробросить Wails/request context и timeout/cancel. |
| C.05.095 | ok | none | logger Seq writer | `Close` больше не ждет через sleep; flush синхронизирован через `sync.WaitGroup`, repeated close idempotent. | Покрыто `TestSeqAsyncWriterCloseFlushesBufferedLogs`; общий context/shutdown lifecycle остается в C.05.094/C.05.097. |
| C.05.096 | issue | major | DB/MinIO/files/processes | DB закрывается; MinIO операций с cancel нет; `exec.Command(...).Start()` не отслеживает процесс. | Для долгих операций использовать context, timeout и явную lifecycle policy. |
| C.05.097 | issue | major | `main.go` OnShutdown | `OnShutdown` закрывает DB/logger, но не отменяет операции; `closeLogger` вызывается и defer, и shutdown. | Ввести app context, единый shutdown coordinator, idempotent close. |
| C.06.098 | ok | none | slog | `slog` поддерживает уровни; Wails logger настроен на ERROR, app logs через slog. | Уровни доступны. |
| C.06.099 | issue | major | logger/audit | `app_user` добавляет ФИО ко всем логам; audit details содержат имена, файлы, номера. | Нужна PII/logging minimization policy и redaction для production logs. |
| C.06.100 | issue | minor | services | Critical startup errors логируются; business critical actions в основном идут в journal/admin audit, но не всегда в slog с correlation/context. | Добавить request/action context для registration, migration, file ops. |

## Что Передать На D/F/E

- Frontend должен перейти на structured error `{code,message,details?}` и перестать разбирать строки.
- Формы регистрации должны отправлять обязательный `idempotencyKey`.
- Повторный submit и double-click нужно проверять после backend idempotency.
- Frontend date payloads нужно сверить с единым `YYYY-MM-DD` / RFC3339 contract.
- E/F должны проверить shutdown/cancel, rollback guardrails, logging redaction и long-running file operations.

## Closure

Статус этапа C: completed with open remediation issues.

Все обязательные артефакты C созданы: `BACKEND_REVIEW.md`, `WAILS_CONTRACTS_REVIEW.md`, `DTO_REVIEW.md`, `ERROR_HANDLING_REVIEW.md`, `LOGGING_REVIEW.md`, `RESOURCE_LIFECYCLE_REVIEW.md`.
