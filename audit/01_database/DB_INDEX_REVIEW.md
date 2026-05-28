# DB Index Review

Дата аудита: 2026-05-27

## Existing Useful Indexes

- `documents(kind)`, `documents(nomenclature_id)`, `documents(registration_date)`, `documents(created_at)`.
- unique `documents(kind, registration_number, EXTRACT(YEAR FROM registration_date))`.
- FK/access indexes on `assignments(document_id)`, `assignments(executor_id)`, `assignment_co_executors(user_id)`.
- document child indexes: correspondents, resolutions, incoming/outgoing/citizen/order details.
- journal indexes: `document_journal(document_id)`, `document_journal(created_at DESC)`, `admin_audit_log(created_at DESC)`.

Найденный structural duplicate исправлен 2026-05-28: `users.login` имеет `UNIQUE`, который создает `users_login_key`, а отдельный `idx_users_login` на тот же столбец удаляется forward-миграцией `010_drop_duplicate_users_login_index`. Для lookup по login остается unique index.

## Representative Plan Result

Локально был загружен synthetic dataset: 1000 documents за 2026 год, 20 users, 500 assignments, 200 acknowledgments, 333 attachment metadata rows, 1000 document journal rows, 500 admin audit rows. После `ANALYZE` сняты representative `EXPLAIN ANALYZE`.

| Query | План/наблюдение | Execution |
| --- | --- | --- |
| Incoming list by date/order | `idx_documents_kind` + seq scan по 250 incoming details + top-N sort | ~0.28 ms |
| Incoming restricted access | backward `idx_documents_created_at`, nested access EXISTS, hashed acknowledgment subplan | ~2.89 ms |
| Assignment list by executor/status | seq/hash over 500 assignments, top-N sort | ~0.70 ms |
| Pending acknowledgments | seq scan 400 `acknowledgment_users`, PK lookups | ~0.24 ms |
| Journal by document | `idx_document_journal_doc_id` | ~0.08 ms |
| Admin audit list | `idx_admin_audit_log_created_at` | ~0.05 ms |
| ILIKE document search | `idx_documents_kind`, filter over 250 docs, top-N sort | ~0.21 ms |

Вывод: при подтвержденном baseline до 1000 документов/год текущие планы быстрые и не требуют срочного добавления индексов. Кандидаты ниже остаются conditional: добавлять только если final production-like dataset, реальные планы или рост данных покажут деградацию.

## Index Stats Snapshot

`pg_stat_user_indexes` снят на локальной synthetic DB после representative checks. Это не production usage profile, но подтверждает, что stats path работает и что для финального release gate нужно повторить замер после длительного сценария.

Top observed index scans:

- `users_pkey`: 14 scans;
- `idx_nomenclature_kind_code`: 6 scans;
- `documents_pkey`: 5 scans;
- `nomenclature_pkey`: 3 scans;
- `idx_users_login` и `users_login_key`: по 2 scans.

## Conditional Index Candidates

Добавлять только после подтверждения через `EXPLAIN` на production-like data:

```sql
CREATE INDEX idx_documents_kind_created_at_desc
ON documents (kind, created_at DESC);

CREATE INDEX idx_documents_kind_nomenclature_created_at_desc
ON documents (kind, nomenclature_id, created_at DESC);

CREATE INDEX idx_assignments_executor_status_deadline
ON assignments (executor_id, status, deadline);

CREATE INDEX idx_assignments_document_executor
ON assignments (document_id, executor_id);

CREATE INDEX idx_acknowledgments_document_created_at
ON acknowledgments (document_id, created_at DESC);

CREATE INDEX idx_acknowledgment_users_user_confirmed
ON acknowledgment_users (user_id, confirmed_at);

CREATE INDEX idx_admin_audit_log_user_created_at
ON admin_audit_log (user_id, created_at DESC);
```

Для `ILIKE '%term%'` и `LOWER(...) LIKE`:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

И только для реально частых search fields после замера:

```sql
CREATE INDEX ... USING gin (content gin_trgm_ops);
```

## Findings

| Пункт | Статус | Severity | Доказательство | Рекомендация |
| --- | --- | --- | --- | --- |
| B.04.053 | issue | minor | Representative plans быстрые, но access/search paths частично используют seq/hash scans. | Не добавлять индексы без evidence на финальном dataset. |
| B.04.054 | ok | none | FK joins на representative data быстрые; journal/admin audit используют индексы. | Повторить при росте данных. |
| B.04.055 | ok | none | Sort operations top-N и быстрые на baseline. | Composite sort indexes условны. |
| B.04.056 | issue | minor | OFFSET pagination используется для списков. | Baseline допустим; при росте перейти на keyset. |
| B.04.057 | issue | minor | Search uses `ILIKE '%term%'` and `LOWER(...) LIKE`. | При росте добавить trigram/FTS. |
| B.04.058 | ok | none | Текущие планы не требуют новых составных индексов для SLO. | Вернуться при росте. |
| B.04.061 | ok | none | Добавлена миграция `010_drop_duplicate_users_login_index`: `DROP INDEX IF EXISTS idx_users_login`; unique index `users_login_key` остается. | На production-like DB после миграций проверить `pg_indexes`, что на `users(login)` остался unique constraint index. |
| B.04.062 | ok | none | `pg_stat_user_indexes` снят на synthetic DB; top scans ожидаемо на PK/login/nomenclature indexes. | Повторить после длительной нагрузки или финального production-like сценария. |
