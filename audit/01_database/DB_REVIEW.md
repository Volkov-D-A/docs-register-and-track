# DB Review

Дата аудита: 2026-05-27
Этап: B. PostgreSQL, схема, миграции, индексы, транзакции

## Общий Вывод

Статический аудит PostgreSQL-схемы и SQL-кода выполнен по миграциям `001`-`007`, репозиториям `internal/repository/*`, database wrapper и backup/restore scripts.

Схема в целом соответствует предметной области: есть общая таблица `documents`, detail-таблицы по видам документов, номенклатура, пользователи, права, поручения, ознакомления, вложения, связи, журнал документа и административный аудит.

Главные production-риски этапа B:

- strict no-gaps нумерация и `idempotency_key` регистрации документов пока не обеспечены на уровне БД/транзакции;
- `down`-миграции удаляют production-таблицы и данные, при этом rollback доступен из UI пользователю с `admin`;
- `document_journal` и `admin_audit_log` каскадно удаляются при удалении документов/пользователей, что конфликтует с подтвержденной lifetime retention policy;
- representative `EXPLAIN ANALYZE` на локальной БД с 1000 документами показал быстрые планы, но индексные решения нужно перепроверить на фактическом production dataset перед релизом;
- restore script продолжает выполнение при любом ненулевом `pg_restore`, что может скрыть реальную ошибку восстановления.

## Critical/Major Issues

| Issue | Severity | Кратко | Ответственный этап |
| --- | --- | --- | --- |
| ISSUE-006 | major | `nomenclature.next_number` инкрементируется отдельно от транзакции создания документа; нет backend/DB `idempotency_key`. | B/C |
| ISSUE-007 | critical | `down`-миграции удаляют production tables/data; UI rollback разрешен для `admin`. | B/H |
| ISSUE-008 | major | Lifetime journals используют `ON DELETE CASCADE` на `documents`/`users`. | B/C |
| ISSUE-009 | minor | Representative plans быстрые на 1000 docs, но access/list/search queries требуют повторной проверки на фактическом dataset и при росте данных. | B/F |
| ISSUE-010 | major | Restore script игнорирует ненулевой `pg_restore` и продолжает MinIO restore. | B/H |

## Что Передать На Этап C

- Требование перенести получение/инкремент регистрационного номера в одну транзакцию с созданием документа.
- Требование добавить `idempotency_key` в request/schema/unique constraint для регистрации документов.
- Решение по журналам: запретить физическое удаление связанных строк или заменить cascade на retention-safe FK strategy.
- Список SQL-ошибок/constraint errors, которые backend должен маппить в пользовательские ошибки.

## Принятые Решения Для Remediation

- `DECISION-003`: полный production UI/runtime механизм миграций сохраняется, включая rollback; destructive rollback risk принят как управляемый операционный риск с обязательными guardrails.
- `DECISION-004`: регистрация документов должна иметь обязательный `documents.idempotency_key UUID NOT NULL` с unique `(created_by, kind, idempotency_key)`; проверка ключа, выдача номера и создание документа выполняются в одной DB transaction.
- `DECISION-005`: users/documents не удаляются физически на уровне приложения; FK журналов должны быть retention-safe, без cascade удаления history/audit.
- `DECISION-006`: restore workflow должен быть fail-fast для fatal/unknown `pg_restore` errors; MinIO restore запускается только после успешной DB validation.

## Контрольные Пункты B

| Код | Статус | Severity | Место | Доказательство | Вывод / рекомендация |
| --- | --- | --- | --- | --- | --- |
| B.01.024 | ok | none | migrations | Чистый Docker PostgreSQL был поднят через `.envExample`; прямой `psql` прогон `*.up.sql` прошел с `ON_ERROR_STOP=1`; application migrator/golang-migrate через embedded migrations тоже прошел: `schema_migrations=7,false`, `total=7`, `uptodate=true`. | Повторить в CI/release gate. |
| B.01.025 | ok | none | migrations | Файлы имеют тематическое назначение: users, nomenclature, documents, assignments, attachments/settings, journal, admin audit. | Назначение понятно. |
| B.01.026 | ok | none | migrations | Нумерация `001`-`007`; FK идут после базовых таблиц. | Порядок логичный. |
| B.01.027 | issue | critical | down migrations | Runtime rollback через application migrator успешно откатил version 7 -> 6 и удалил `admin_audit_log` (`to_regclass` вернул NULL). | По `DECISION-003` full UI/runtime rollback сохраняется; нужны guardrails, rollback-runbook и backup/test-restore discipline. |
| B.02.028 | ok | none | schema | Таблицы соответствуют сущностям из `BUSINESS_RULES.md`. | Схема отражает домен. |
| B.02.029 | ok | none | schema | Все доменные таблицы имеют PK; join table `department_nomenclature`, `assignment_co_executors` имеют composite PK. | PK корректны. |
| B.02.030 | issue | major | FK | `document_journal`/`admin_audit_log` используют cascade на retention data; локально подтверждено удаление журналов при delete user/document. | По `DECISION-005` заменить cascade на retention-safe FK strategy. |
| B.02.031 | ok | none | schema | Ключевые поля документов, прав, вложений, поручений `NOT NULL`. | В целом корректно. |
| B.02.032 | issue | major | schema | Нет unique/idempotency key для регистрации документа; уникальность номера есть только по `(kind, registration_number, year)`. | По `DECISION-004` добавить `documents.idempotency_key UUID NOT NULL` и unique `(created_by, kind, idempotency_key)`. |
| B.02.033 | ok | none | schema | CHECK для kinds, permissions, actions, document_type, appeal_type, link_type. | Справочные значения ограничены. |
| B.02.034 | ok | none | schema | UUID, DATE, TIMESTAMPTZ, VARCHAR/TEXT используются разумно. | Явно избыточных типов не найдено. |
| B.02.035 | not_applicable | none | schema | Денежных колонок нет. | Не применимо. |
| B.02.036 | ok | none | schema | Timestamps хранятся как `TIMESTAMP WITH TIME ZONE`, даты документов как `DATE`. | Стратегия согласована. |
| B.02.037 | not_applicable | none | schema | Soft delete почти не используется; есть `is_active` для users/nomenclature/orders. | Уникальность soft delete не ломает. |
| B.02.038 | issue | minor | schema | `updated_at` задается, но нет DB triggers; обновление зависит от кода. | На C/E проверить единообразие writes. |
| B.02.039 | issue | major | FK cascade | Cascade удаляет дочерние документы, вложения, журналы; для journal/audit это нарушает lifetime retention. | Для history/audit данных использовать retention-safe FK strategy. |
| B.02.040 | ok | none | schema | Nullable поля соответствуют optional-сценариям: deadline, report, details, cancelled_at. | В целом корректно. |
| B.02.041 | not_applicable | none | schema | JSON/JSONB не используется. | Не применимо. |
| B.02.042 | ok | none | schema | Enum-like logic через CHECK и кодовые значения. | Для расширения нужны миграции, но текущая policy фиксированная. |
| B.02.043 | ok | none | schema | Имена таблиц/индексов в целом единообразны. | Существенной проблемы нет. |
| B.02.044 | ok | none | schema | Дублирующих таблиц не найдено. | Нет. |
| B.02.045 | transferred | none | schema | Очевидных устаревших таблиц/колонок на уровне DB review не найдено; финальная семантическая сверка требует backend/UI context. | Передано на C/D/E. |
| B.03.046 | ok | none | SQL | SQL inventory составлен в `DB_QUERY_PLANS.md`. | Достаточно для B. |
| B.03.047 | ok | none | SQL | Запросы сгруппированы по сценариям. | См. `DB_QUERY_PLANS.md`. |
| B.03.048 | ok | none | SQL | Частые reads: document lists, assignments, acknowledgments, access checks. | См. `DB_INDEX_REVIEW.md`. |
| B.03.049 | ok | none | SQL | Representative `EXPLAIN ANALYZE` снят на dataset: 1000 docs, 500 assignments, 200 acknowledgments, 1000 journal rows, 500 admin audit rows. | Повторить на финальном production-like dataset. |
| B.03.050 | ok | none | SQL | Частые writes: document create/update, assignments, acknowledgments, attachments, settings. | См. `DB_TRANSACTIONS_REVIEW.md`. |
| B.04.051 | ok | none | EXPLAIN | Representative plans собраны для document list/access, assignments, acknowledgments, journal, admin audit, search. | См. `DB_QUERY_PLANS.md`. |
| B.04.052 | ok | none | EXPLAIN ANALYZE | На 1000 docs планы выполняются примерно за 0.05-2.9 ms. | Повторить на финальном dataset. |
| B.04.053 | issue | minor | indexes | Текущие планы быстрые, но access/search paths частично используют seq/hash scans; при подтвержденном объеме это допустимо. | Не добавлять индексы без роста данных или ухудшения plan evidence. |
| B.04.054 | ok | none | indexes | FK joins на representative data быстрые; journal/admin audit используют индексы. | Повторить при росте данных. |
| B.04.055 | ok | none | indexes | Sort operations top-N и быстрые на baseline. | Composite sort indexes условны. |
| B.04.056 | issue | minor | pagination | Используется OFFSET pagination. | При baseline 1000/year допустимо; при росте перейти на keyset. |
| B.04.057 | issue | minor | search | `ILIKE '%term%'`/`LOWER(...) LIKE` без trigram indexes. | При росте данных добавить pg_trgm или FTS. |
| B.04.058 | ok | none | indexes | На текущем representative dataset составные индексы не являются обязательными для SLO. | Вернуться при росте данных. |
| B.04.059 | issue | minor | indexes | Частичные индексы есть только для pending order acknowledgment people. | Рассмотреть partial indexes для active/pending assignments/acknowledgments. |
| B.04.060 | issue | major | unique indexes | Нет unique для idempotency key. | Добавить unique index `(created_by, kind, idempotency_key)`. |
| B.04.061 | issue | minor | indexes | Structural duplicate: `users_login_key` и `idx_users_login` оба индексируют `users(login)`. | Удалить отдельный `idx_users_login`, оставив unique constraint index. |
| B.04.062 | ok | none | indexes | `pg_stat_user_indexes` снят на synthetic DB; top scans ожидаемо на PK/login/nomenclature indexes. | Повторить после длительной нагрузки или финального production-like сценария. |
| B.04.063 | ok | none | writes | Индексов умеренно много; baseline небольшой. | Новые индексы добавлять точечно. |
| B.04.064 | ok | none | stats | Выполнен `ANALYZE`; `pg_stat_user_tables` показал ожидаемые live tuples по ключевым таблицам. | Повторить после финальной загрузки test data. |
| B.05.065 | ok | none | test data | Загружен synthetic dataset: 1000 docs, 20 users, 500 assignments, 200 acknowledgments, 333 attachments metadata. | Согласовать, достаточно ли synthetic dataset для release gate. |
| B.05.066 | issue | major | transactions | Document create tx не включает `GetNextNumber`; локальный failure test подтвердил gap после ошибки insert. Attachment metadata/object не cross-transaction. | По `DECISION-004` перенести проверку `documents.idempotency_key`, numbering и document create в одну transaction. |
| B.05.067 | issue | major | isolation | Default isolation плюс отдельный `UPDATE next_number` не обеспечивает no-gaps при post-number failure. | Номер выдавать внутри tx с row lock/update. |
| B.05.068 | issue | major | concurrency | Повторный submit без idempotency key может создать дубль/потратить номер. | Добавить unique idempotency key. |
| B.05.069 | transferred | none | locks | Долгие ожидания требуют нагрузочного сценария после изменения registration transaction. | Передано на C/F: concurrent registration/assignment test. |
| B.05.070 | transferred | none | errors | Constraint errors есть, но пользовательский mapping ошибок является backend concern. | Передано на C. |
| B.05.071 | transferred | none | DB disconnect | `Ping` warning есть, но runtime recovery требует backend/resource lifecycle теста. | Передано на C/E. |
| B.05.072 | issue | minor | pool | `sql.DB` pool не настроен явно. | Для desktop задать conservative pool settings. |
| B.05.073 | issue | minor | contexts | Многие repository queries без context timeout. | На C добавить context/cancel для долгих queries. |
| B.06.074 | issue | major | backup | Backup script есть, но production test restore должен быть ручно подтвержден перед релизом. | Проверить процедуру. |
| B.06.075 | issue | major | restore | `pg_restore` nonzero code игнорируется и restore продолжается. | По `DECISION-006` сделать fail-fast для fatal/unknown errors, restore report и smoke validation до MinIO mirror. |
| B.06.076 | ok | none | seed/test | Seed production-данных в миграциях нет; default settings являются системными defaults. | Явных test данных нет. |

## Closure

Статус этапа B: completed with open remediation issues.

Все обязательные артефакты B созданы: `DB_REVIEW.md`, `DB_SCHEMA_REVIEW.md`, `DB_MIGRATIONS_REVIEW.md`, `DB_INDEX_REVIEW.md`, `DB_QUERY_PLANS.md`, `DB_TRANSACTIONS_REVIEW.md`, `DB_BACKUP_RESTORE_REVIEW.md`, `REVIEW_LOG.md`, `DECISIONS.md`, `RISK_REGISTER.md`, `REGRESSION_MAP.md`.

Открытые critical/major проблемы:

- `ISSUE-006` / major: strict no-gaps registration и backend `idempotency_key` не реализованы; target schema: `documents.idempotency_key UUID NOT NULL`, unique `(created_by, kind, idempotency_key)`.
- `ISSUE-007` / critical: destructive `down` migrations доступны через production UI/runtime rollback для `admin`; rollback остается, нужны guardrails по `DECISION-003`.
- `ISSUE-008` / major: `document_journal` и `admin_audit_log` имеют cascade FK, конфликтующий с lifetime retention.
- `ISSUE-010` / major: restore продолжает workflow после любого nonzero `pg_restore`; нужен fail-fast contract по `DECISION-006`.

Transferred checks:

- `B.02.045` -> C/D/E: сверка устаревших колонок с backend/UI semantics.
- `B.05.069` -> C/F: lock wait/concurrent registration после изменения transaction model.
- `B.05.070` -> C: mapping unique/FK/constraint errors в пользовательские ошибки.
- `B.05.071` -> C/E: DB disconnect/reconnect и resource lifecycle.

Изменения для `PROJECT_CONTEXT.md`:

- Зафиксировать, что этап B завершен на локальном test contour с synthetic dataset: 1000 documents, 20 users, 500 assignments, 200 acknowledgments, 333 attachment metadata rows.
- Добавить принятые DB decisions: full production migration UI/runtime сохраняется; registration idempotency хранится в `documents.idempotency_key`; journal/audit retention требует retention-safe FK; restore должен быть fail-fast.
- Уточнить release gates: test restore PostgreSQL+MinIO, final EXPLAIN на production-like dataset, rollback guardrails validation.

Изменения для `BUSINESS_RULES.md`:

- Явно добавить правило: регистрация документов должна быть идемпотентной по backend `idempotency_key`; повтор `(created_by, kind, idempotency_key)` возвращает существующий документ и не расходует номер.
- Явно добавить правило strict no-gaps: регистрационный номер выделяется только в transaction успешного создания документа.
- Уточнить lifetime retention: `document_journal` и `admin_audit_log` не удаляются штатными application paths.

Backend-аудит пункты, на которые влияет B:

- C.02 registration command handlers: transaction boundary, idempotency, no-gaps numbering.
- C.03 repository layer: shared transaction handling, constraint errors, context/cancel.
- C.04 journal/audit behavior and retention-safe deletes.
- C.05 resource lifecycle: DB disconnect/reconnect, pool settings, timeout policy.
- E/H release/ops: migration rollback guardrails, backup/restore runbook, test restore.

Регрессионные проверки после DB changes:

- Fresh migrate up/down including `008_registration_idempotency`.
- Failed create after numbering does not advance `nomenclature.next_number`.
- Repeated registration with same `idempotency_key` returns the same document.
- Concurrent duplicate `idempotency_key` does not create duplicate document and does not burn a number.
- Journal/admin audit rows survive or block unsafe delete according to retention-safe FK strategy.
- Runtime migration UI still supports status/up/rollback for `admin` with guardrails.
- Restore with corrupted dump stops before MinIO mirror; valid PostgreSQL+MinIO restore passes smoke validation.
