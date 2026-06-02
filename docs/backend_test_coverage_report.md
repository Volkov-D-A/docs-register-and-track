# Отчёт: Анализ покрытия бекенда тестами

Дата: 2026-06-02  
Метод анализа: `go test -coverprofile`, ручной анализ соответствия тестов бизнес-правилам и кодовой базе

---

## 1. Общие метрики покрытия

### 1.1 Покрытие по пакетам (go test -cover)

| Пакет | Покрытие | Оценка |
|-------|----------|--------|
| `internal/security` | **95.5%** | ✅ Отлично |
| `internal/config` | **71.1%** | ⚠️ Хорошо |
| `internal/startupdiag` | **69.0%** | ⚠️ Хорошо |
| `internal/database` | **55.2%** | ⚠️ Удовлетворительно |
| `internal/dto` | **52.8%** | ⚠️ Удовлетворительно |
| `internal/services` | **51.5%** | ⚠️ Удовлетворительно |
| `internal/logger` | **50.0%** | ⚠️ Удовлетворительно |
| `internal/repository` | **43.5%** | ❌ Низкое |
| `internal/storage` | **0.0%** | ❌ Отсутствует |
| `internal/models` | **0.0%** | ℹ️ Data-only |
| `internal/mocks` | **0.0%** | ℹ️ Генерированный код |
| **Общее (total statements)** | **38.9%** | ❌ Низкое |

> **Примечание**: общий показатель 38.9% включает mocks и models без логики. Эффективное покрытие production-кода — **~47–52%** (если исключить mocks/models).

### 1.2 Объём кода

| Категория | Строк кода |
|-----------|-----------|
| Production Go-код (без mocks) | 15 990 |
| Тестовый Go-код | 9 261 |
| **Соотношение тест/код** | **0.58** |
| Тестовых функций (Test*) | **~170** |

---

## 2. Покрытие по слоям архитектуры

### 2.1 Слой Services — 51.5%

**Файлы с тестами (21 из 30+ файлов):**

| Сервис | Тесты | Бизнес-сценарии | Комментарий |
|--------|-------|-----------------|-------------|
| `auth_service` | 10 тестов, 588 строк | Login, Logout, ChangePassword, InitialSetup, Permissions | ✅ Хорошее покрытие ядра аутентификации |
| `assignment_service` | 6 тестов, 749 строк | CRUD, co-executors, status transitions | ✅ Хорошее покрытие |
| `link_service` | 4 теста, 553 строки | Link/Unlink, graph, access filtering | ✅ Хорошее покрытие, включая фильтрацию по доступу |
| `attachment` | 7 тестов, 409 строк | Upload, Download, Delete, path traversal, bulk delete | ✅ Хорошее покрытие |
| `settings` | 12 тестов, 485 строк | GetAll, Update, Migrations, Rollback | ✅ Хорошее покрытие, включая rollback guardrails |
| `acknowledgment_service` | 7 тестов, 298 строк | CRUD, mark viewed/confirmed | ✅ Хорошее покрытие |
| `nomenclature_service` | 5 тестов, 258 строк | CRUD, uniqueness | ✅ Достаточно |
| `department_service` | 4 теста, 212 строк | CRUD | ✅ Достаточно |
| `reference_service` | 9 тестов, 244 строки | Org/executor CRUD, find-or-create | ✅ Хорошее покрытие |
| `user_service` | 5 тестов, 273 строки | GetAll, Create, Update | ✅ Достаточно |
| `helpers` | 4 теста, 376 строк | Вспомогательные функции | ✅ Достаточно |
| `document_kind_command_handler` | 3 теста, 103 строки | Routing register/update | ✅ Базовый routing |
| `document_kind_service` | 1 тест, 98 строк | List kinds | ⚠️ Минимально |
| `dashboard_service` | 1 тест, 128 строк | GetDashboard | ⚠️ Минимально |
| `journal_service` | 2 теста, 139 строк | GetEntries, LogAction | ⚠️ Минимально |
| `admin_audit_log_service` | 2 теста, 62 строки | GetAll, Create | ⚠️ Минимально |
| `document_access_admin_service` | 2 теста, 85 строк | GetProfile, ReplaceProfile | ⚠️ Минимально |
| `release_note_service` | 2 теста, 80 строк | GetNotes, MarkViewed | ⚠️ Минимально |
| `operation_lifecycle` | 2 теста, 46 строк | Context, Shutdown | ⚠️ Минимально |
| `theme_service` | 2 теста, 44 строки | SetTheme, resolveAppTheme | ⚠️ Минимально |

**Файлы БЕЗ тестов (полное отсутствие):**

| Файл | Строк | Критичность | Комментарий |
|------|-------|-------------|-------------|
| `statistics_service.go` | 418 | 🔴 Высокая | Нет ни одного теста. 23 функции с 0% покрытием |
| `citizen_appeal_command_handler.go` | 428 | 🔴 Высокая | Регистрация/обновление обращений граждан — без тестов |
| `incoming_letter_command_handler.go` | 299 | 🔴 Высокая | Регистрация/обновление входящих — без тестов |
| `outgoing_letter_command_handler.go` | 202 | 🔴 Высокая | Регистрация/обновление исходящих — без тестов |
| `administrative_order_command_handler.go` | 252 | 🔴 Высокая | Регистрация/обновление приказов — без тестов |
| `document_access_service.go` | 760 | 🔴 Высокая | Ядро авторизации документов — без unit-тестов |
| `administrative_order_service.go` | 78 | 🟡 Средняя | Wails-facing API для приказов — без тестов |
| `document_query_service.go` | 78 | 🟡 Средняя | Общий query handler — без тестов |
| `incoming_letter_query_handler.go` | 48 | 🟡 Средняя | Query handler — без тестов |
| `outgoing_letter_query_handler.go` | 64 | 🟡 Средняя | Query handler — без тестов |
| `citizen_appeal_query_handler.go` | 48 | 🟡 Средняя | Query handler — без тестов |
| `administrative_order_query_handler.go` | 46 | 🟡 Средняя | Query handler — без тестов |
| `document_kind_query_handler.go` | 45 | 🟡 Средняя | Query handler — без тестов |
| `system_service.go` | 25 | 🟢 Низкая | Startup wrapper |
| `interfaces.go` | 240 | ℹ️ | Интерфейсы, тестирование не требуется |
| `access_policy.go` | 20 | 🟢 Низкая | Константы |

### 2.2 Слой Repository — 43.5%

**С тестами:**

| Репозиторий | Тестов | Строк теста | Комментарий |
|-------------|--------|-------------|-------------|
| `incoming_doc_repo` | 5 | 307 | ✅ CRUD через sqlmock |
| `outgoing_doc_repo` | 5 | 274 | ✅ CRUD через sqlmock |
| `user_repo` | 5 | 260 | ✅ CRUD через sqlmock |
| `acknowledgment_repo` | 8 | 256 | ✅ CRUD через sqlmock |
| `assignment_repo` | 5 | 222 | ✅ CRUD через sqlmock |
| `nomenclature_repo` | 7 | 222 | ✅ CRUD через sqlmock |
| `reference_repo` | 9 | 201 | ✅ Org/executor find-or-create через sqlmock |
| `department_repository` | 6 | 197 | ✅ CRUD через sqlmock |
| `attachment_repo` | 4 | 137 | ⚠️ Базовый CRUD |
| `link_repo` | 4 | 123 | ⚠️ Базовый CRUD |
| `journal_repo` | 3 | 114 | ⚠️ Create/GetByDocumentID |
| `settings` | 3 | 89 | ⚠️ GetAll/Get/Update |
| `administrative_order_repo` | 1 | 71 | ⚠️ Минимально |
| `dashboard_repo` | 1 | 35 | ⚠️ Минимально |
| `document_registration (integration)` | 5 | 490 | ✅ Идемпотентность, конкурентность, constraints |

**Без тестов:**

| Файл | Строк | Критичность |
|------|-------|-------------|
| `citizen_appeal_repo.go` | 479 | 🔴 Высокая |
| `statistics_repo.go` | 419 | 🔴 Высокая |
| `document_access_repo.go` | 154 | 🟡 Средняя |
| `document_repo.go` | 99 | 🟡 Средняя |
| `document_registration.go` | 123 | ⚠️ Покрыт интеграционно |
| `admin_audit_log_repo.go` | 66 | 🟡 Средняя |

### 2.3 Слой Storage — 0%

`internal/storage/minio.go` (134 строки) — **полное отсутствие тестов**.

Все 6 функций MinIO (Upload, Download, Delete, GetStorageInfo, NewMinioService, formatSize) имеют 0% покрытия. Тестирование затруднено зависимостью от внешнего MinIO-сервера, но вспомогательные функции (например, `formatSize`) могли бы быть покрыты unit-тестами.

### 2.4 Слой DTO — 52.8%

`internal/dto/mapper.go` (980 строк) — 14 тестов, но 44 функции с 0% покрытием. Покрыты основные mapper-функции для incoming/outgoing документов, но **не покрыты mappers для citizen_appeal и administrative_order**.

### 2.5 Прочие пакеты

| Пакет | Покрытие | Комментарий |
|-------|----------|-------------|
| `config` | 71.1% | ✅ Загрузка конфига, шифрование/дешифрование |
| `security/password` | 95.5% | ✅ Отличное покрытие хеширования и валидации |
| `database/postgres` | 55.2% | ⚠️ Миграции, connection, compatibility check |
| `logger` | 50.0% | ⚠️ Seq writer, основной logger |
| `startupdiag` | 69.0% | ⚠️ Диагностика запуска |
| `releaseassets` | 100% (no stmts) | ℹ️ Только генерация |

---

## 3. Соответствие тестов бизнес-правилам

### 3.1 ✅ Хорошо покрытые бизнес-правила

| Правило из tech_docs.md | Где протестировано | Оценка |
|-------------------------|-------------------|--------|
| Аутентификация: 5 неверных попыток → деактивация | `auth_service_test.go: wrong_password_deactivates_user_on_fifth_attempt` | ✅ |
| Аутентификация: admin audit при блокировке | `auth_service_test.go: wrong_password_deactivates_user_on_fifth_attempt` | ✅ |
| Аутентификация: inactive user → ErrUserLocked | `auth_service_test.go: inactive_after_bruteforce_shows_locked_message` | ✅ |
| Аутентификация: успешный вход сбрасывает счётчик | `auth_service_test.go: success_resets_previous_failed_attempts` | ✅ |
| First-run setup: admin user creation | `auth_service_test.go: TestAuthService_InitialSetup` | ✅ |
| Идемпотентная регистрация (idempotency_key) | `document_registration_integration_test.go` | ✅ |
| Нумерация без пропусков (concurrent) | `document_registration_integration_test.go: TestDocumentRegistrationConcurrencyIntegration` | ✅ |
| next_number не инкрементируется при идемпотентном повторе | `document_registration_integration_test.go` | ✅ |
| Journal FK retention (нельзя удалить документ с journal) | `document_registration_integration_test.go: TestJournalRetentionFKIntegration` | ✅ |
| Admin audit FK retention | `document_registration_integration_test.go: TestJournalRetentionFKIntegration` | ✅ |
| DB constraints: unique reg number per kind+year | `document_registration_integration_test.go: TestDatabaseConstraintsIntegration` | ✅ |
| Dirty migration → блокировка | `document_registration_integration_test.go: TestDatabaseConstraintsIntegration` | ✅ |
| Attachment: max file size | `attachment_test.go: file_too_large` | ✅ |
| Attachment: allowed extensions | `attachment_test.go: forbidden_extension` | ✅ |
| Attachment: download без перезаписи | `attachment_test.go: TestWriteDownloadFileWithoutOverwrite` | ✅ |
| Attachment: path traversal protection | `attachment_test.go: TestAttachmentService_ValidatePathInDownloads` | ✅ |
| Link: нельзя связать с собой | `link_service_test.go: запрещено_связывать_с_собой` | ✅ |
| Link graph: фильтрация по read access | `link_service_test.go: скрывает_связи_с_недоступным_документом`, `скрывает_доступные_узлы_за_недоступным_мостом` | ✅ |
| Rollback: guardrails (backup, data-loss ack, control phrase) | `settings_test.go: TestValidateRollbackMigrationRequest` | ✅ |
| Schema too new → structured error | `settings_test.go: TestMigrationCompatibilityAppError` | ✅ |
| Settings: idempotent save (skip audit when unchanged) | `settings_test.go: skips_update_and_audit_when_value_did_not_change` | ✅ |
| Settings: human readable audit labels | `settings_test.go: uses_human_readable_label_for_known_setting` | ✅ |
| Пароль: сложность | `security/password_test.go` | ✅ |
| Assignments: CRUD + access checks | `assignment_service_test.go` | ✅ |
| Acknowledgments: CRUD + mark | `acknowledgment_service_test.go` | ✅ |
| Nomenclature: uniqueness validation | `nomenclature_service_test.go` | ✅ |
| Reference: find-or-create organization | `reference_service_test.go` | ✅ |
| Permission model: system + document permissions | `auth_service_test.go: TestAuthService_SystemPermissionChecks` | ✅ |
| Integration DSN safety check | `document_registration_integration_test.go: TestValidateIntegrationDSNRequiresSafeDatabaseName` | ✅ |

### 3.2 ❌ Не покрытые бизнес-правила

| Правило из tech_docs.md | Ожидаемое расположение | Критичность |
|-------------------------|----------------------|-------------|
| **Регистрация входящего письма** — валидация полей, idempotency, journal write | `incoming_letter_command_handler` (нет тестов) | 🔴 Критическая |
| **Регистрация исходящего письма** — валидация, find-or-create org, journal | `outgoing_letter_command_handler` (нет тестов) | 🔴 Критическая |
| **Регистрация обращения граждан** — валидация appeal type, required fields, correspondents, resolutions | `citizen_appeal_command_handler` (нет тестов) | 🔴 Критическая |
| **Регистрация приказа** — валидация isActive ↔ cancelledAt, acknowledgment people | `administrative_order_command_handler` (нет тестов) | 🔴 Критическая |
| **Приказ: isActive ↔ cancelled_at** бизнес-правило | `validateOrderActivity` не покрыта тестами | 🔴 Критическая |
| **Типы документов**: валидация фиксированного набора (Письмо, Договор, Акт...) | Есть в коде handlers, но не тестируется | 🟡 Средняя |
| **Document access scope** — participant model через подразделение/номенклатуру/поручение/ознакомление | `document_access_service.go` (760 строк, 0% unit тестов) | 🔴 Критическая |
| **Статистика** — permission checks, date range validation, grouping, complete monthly series | `statistics_service.go` (418 строк, 0% покрытие) | 🟡 Средняя |
| **Ролевая модель**: admin/references/stats_* system permissions per context | Частично покрыта через settings/attachment tests, но нет выделенных тестов | 🟡 Средняя |
| **Обращения**: типы `предложение/заявление/жалоба` — `normalizeAppealType` | Не тестируется | 🟡 Средняя |
| **Nomenclature modes**: `index_and_number/number_only/manual_only` — влияние на нумерацию | Покрыто в integration, но не в unit-тестах сервиса | 🟡 Средняя |
| **Orders**: связи `order_amends` и `order_cancels` — только между приказами | Не тестируется | 🟡 Средняя |
| **MinIO**: upload/download/delete consistency | `storage/minio.go` — 0% | 🟡 Средняя |
| **Document query handlers**: авторизация + пагинация + фильтрация | 4 query handler файла — 0% | 🟡 Средняя |
| **Bulk delete attachments**: full flow with MinIO + metadata | Частично (permission check), но MinIO взаимодействие не покрыто | 🟡 Средняя |

### 3.3 ⚠️ Частично покрытые бизнес-правила

| Правило | Что покрыто | Что не покрыто |
|---------|------------|----------------|
| Журналирование (document_journal) | Create/Read через mock | Не проверяется, что journal записывается при каждом action |
| Admin audit log | Create/GetAll | Не проверяется полнота записи при всех аудируемых действиях |
| DocumentAccessService | Тестируется косвенно через link/attachment tests | Нет выделенных unit-тестов для всех 20+ методов |
| DTO mapping | 14 тестов для incoming/outgoing | Нет тестов для citizen_appeal/administrative_order mapping |
| Error contract | App errors используются в тестах | Не тестируется `ErrorFormatter` для Wails bridge |

---

## 4. Анализ неактуальных / устаревших тестов

### 4.1 Потенциально устаревшие паттерны

| Файл | Наблюдение | Рекомендация |
|------|-----------|--------------|
| `link_service_test.go` | Тест «executor не может создавать связи» использует mock-based role mapping. Если permission model изменилась на per-action permissions, тест может не отражать актуальную логику | Проверить, что роль `executor` в тестах действительно не имеет `link` permission в production |
| `link_service_test.go` | Тест «admin не может читать граф связей» — admin по умолчанию не имеет document read. Это ожидаемое поведение, но может сбивать разработчиков | Добавить комментарий к тесту о том, что admin — системная роль без document read |
| `document_access_test_helper_test.go` | 0 тестовых функций, 48 строк — содержит только helper `newRoleMappedDocumentAccessStore`. Файл вспомогательный, не содержит тестов | ℹ️ Нормально, используется другими тестами |
| `attachment_test.go: TestAttachmentService_ValidatePathInDownloads` | Использует Windows-пути (`C:\Windows\System32`), но проект целевой под Linux и Windows | ℹ️ Валидно — проверяет path traversal для обеих ОС |

### 4.2 Тесты, соответствующие актуальной кодовой базе

Все обнаруженные тестовые сценарии корреспондируют с существующим production-кодом. **Явно устаревших или «мёртвых» тестов не обнаружено.**

### 4.3 Тесты, требующие обновления при изменениях

| Тест | Риск устаревания |
|------|-----------------|
| `setupSettingsService` → использует `sqlmock.New()` | При изменении DB-слоя может потребовать обновления |
| `document_registration_integration_test.go` — миграции применяются через `filepath.Glob("../../internal/database/migrations/*.up.sql")` | Зависимость от относительного пути — хрупкая при реструктуризации |
| Все тесты с `newRoleMappedDocumentAccessStore` | При изменении permission model нужно обновлять store-builder |

---

## 5. Интеграционные тесты

### 5.1 Имеющиеся

- `document_registration_integration_test.go` — 5 тестов, 490 строк:
  - Идемпотентность регистрации
  - Конкурентная нумерация (8 воркеров)
  - Journal/audit FK retention
  - DB constraints
  - DSN safety validation
- Запуск: `DOCFLOW_INTEGRATION_DSN=... go test ./internal/repository/...`
- Release gate: `make integration-test` (disposable DB)

### 5.2 Отсутствующие

- Нет интеграционных тестов для **incoming/citizen_appeal/administrative_order** repositories
- Нет интеграционных тестов для **document_access_repo** (критичная для безопасности логика)
- Нет end-to-end тестов с MinIO (upload → metadata → download → delete)

---

## 6. Сводная карта рисков

```
  Критичность
  │
  │  🔴 document_access_service (760 строк, 0% unit)
  │  🔴 Все 4 command handlers (1181 строка суммарно, 0%)
  │  🔴 citizen_appeal_repo (479 строк, 0%)
  │
  │  🟡 statistics_service (418 строк, 0%)
  │  🟡 statistics_repo (419 строк, 0%)
  │  🟡 document_access_repo (154 строки, 0%)
  │  🟡 Query handlers (4 файла, 251 строка суммарно, 0%)
  │  🟡 storage/minio (134 строки, 0%)
  │
  │  ⚠️ DTO mappers для citizen_appeal + admin_order
  │  ⚠️ admin_audit_log_repo (66 строк, 0%)
  │
  │  ✅ auth_service, assignment_service, link_service
  │  ✅ attachment_service, settings, security/password
  │  ✅ Integration tests (idempotency, concurrency, constraints)
  └─────────────────────────────────────────────────
```

---

## 7. Рекомендации по приоритетам

### Приоритет 1 — Критические пробелы (бизнес-критичный код без тестов)

1. **Command handlers для всех 4 видов документов** (incoming, outgoing, citizen_appeal, administrative_order):
   - Валидация полей, idempotency key, date parsing, correspondents, resolutions
   - Правило `isActive ↔ cancelledAt` для приказов
   - Типы обращений `предложение/заявление/жалоба`
   - Типы документов из фиксированного набора
   - Journal write при CREATE/UPDATE

2. **DocumentAccessService** — unit-тесты для:
   - `RequireDomainRead`, `RequireCreate`, `ResolveReadScope`
   - Participant access через подразделение/номенклатуру
   - Participant access через поручение (assignment)
   - Participant access через ознакомление (acknowledgment)
   - `ResolveReadableDocuments` — bulk access resolution
   - `RequireLink`, `RequireViewJournal`, `RequireDocumentAction`

### Приоритет 2 — Существенные пробелы

3. **StatisticsService** — permission checks, date validation, grouping logic, complete monthly series
4. **Citizen appeal repository** — CRUD через sqlmock
5. **DTO mappers** — citizen_appeal и administrative_order mapping
6. **Document access repo** — HasPermission, HasSystemPermission, GetUserAccessProfile

### Приоритет 3 — Желательные улучшения

7. **Query handlers** — авторизация read-операций
8. **Storage/MinIO** — unit-тесты для `formatSize`, integration smoke для upload/download
9. **Admin audit log repo** — Create/GetAll через sqlmock
10. **Document repo** — GetByID, GetByIDs

---

## 8. Заключение

Проект имеет **зрелую тестовую инфраструктуру** — используются testify, go-sqlmock, mockery-generated mocks, интеграционные тесты с disposable database и release gate (`make release-gate`). Тестовая стратегия многоуровневая: unit + integration + release evidence.

**Сильные стороны:**
- Отличное покрытие аутентификации, brute-force protection и first-run setup
- Сильные интеграционные тесты идемпотентности и конкурентности нумерации
- Хорошее покрытие attachment security (path traversal, file size, extensions)
- Rollback guardrails полностью протестированы
- Link graph фильтрация по read access покрыта включая «bridge node» сценарий

**Основной пробел:**
- **Полное отсутствие unit-тестов для write-операций по всем 4 видам документов** — самый критичный функционал системы (регистрация, валидация, нумерация) тестируется только на уровне repository integration tests, но не на уровне service logic. Валидация полей, permission checks и journal writes в command handlers не покрыты.
- **DocumentAccessService** (760 строк) — ядро авторизации — не имеет собственных unit-тестов, хотя косвенно задействуется через тесты link/attachment сервисов.

**Итоговая оценка: удовлетворительно** — ядро инфраструктуры безопасности (auth, crypto, access) покрыто хорошо, но основные бизнес-операции (регистрация документов) нуждаются в значительном расширении тестового покрытия.
