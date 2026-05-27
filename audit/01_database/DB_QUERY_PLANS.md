# DB Query Plans

Дата аудита: 2026-05-27

## Live Проверка

Выполнен локальный live-прогон на чистом Docker PostgreSQL test contour:

- все `*.up.sql` миграции применены с `ON_ERROR_STOP=1`;
- загружен synthetic dataset: 1000 documents, 20 users, 500 assignments, 200 acknowledgments, 333 attachment metadata rows, 1000 document journal rows, 500 admin audit rows;
- выполнен `ANALYZE`;
- сняты representative `EXPLAIN ANALYZE` для ключевых query families.

## Query Inventory By Scenario

### Регистрация и просмотр документов

- `IncomingDocumentRepository.GetList/GetByID/Create/Update`
- `OutgoingDocumentRepository.GetList/GetByID/Create/Update`
- `CitizenAppealRepository.GetList/GetByID/Create/Update`
- `AdministrativeOrderRepository.GetList/GetByID/Create/Update`

Частые patterns:

- `documents d JOIN <detail> ON detail.document_id = d.id`
- `WHERE d.kind = ...`
- filters by `nomenclature_id`, dates, org, document type
- access `EXISTS` through assignments and acknowledgments
- `ORDER BY d.created_at DESC LIMIT/OFFSET`

### Поручения

- `AssignmentRepository.GetList/GetByID/GetCountByStatus/HasDocumentAccess/GetAccessibleDocumentIDs`

Частые patterns:

- `WHERE a.executor_id = ? OR EXISTS assignment_co_executors`
- `WHERE a.status`, `deadline`, overdue expression
- `ORDER BY a.created_at DESC LIMIT/OFFSET`

### Ознакомления

- `AcknowledgmentRepository.GetPendingForUser/GetByDocumentID/HasDocumentAccess/GetAccessibleDocumentIDs`

Частые patterns:

- `acknowledgment_users.user_id`
- `confirmed_at IS NULL`
- join to `acknowledgments(document_id)`

### Журналы

- `JournalRepository` by document ordered by created_at.
- `AdminAuditLogRepository` count/list ordered by created_at.

### Справочники и пользователи

- users by login/id/all, permissions by user, department nomenclatures.
- organizations/resolution executors exact and `ILIKE` search.

## EXPLAIN ANALYZE Summary

| Query | Key plan | Execution time | Вывод |
| --- | --- | --- | --- |
| Incoming list by date/order | `idx_documents_kind`, hash join to details, top-N sort | ~0.28 ms | OK на baseline. |
| Incoming restricted access | backward `idx_documents_created_at`, nested `EXISTS`, hashed acknowledgment subplan | ~2.89 ms | Самый тяжелый из снятых, но в пределах SLO. |
| Assignment list executor/status | seq/hash over 500 assignments, top-N sort | ~0.70 ms | OK на baseline; индексировать при росте. |
| Pending acknowledgments | seq scan 400 `acknowledgment_users`, PK lookups | ~0.24 ms | OK на baseline. |
| Journal by document | `idx_document_journal_doc_id` | ~0.08 ms | OK. |
| Admin audit list | `idx_admin_audit_log_created_at` | ~0.05 ms | OK. |
| ILIKE document search | `idx_documents_kind`, filter over 250 docs, top-N sort | ~0.21 ms | OK на baseline; trigram нужен только при росте. |

## Required EXPLAIN Set For Final Release

1. Списки каждого вида документа без фильтров и с типовыми фильтрами.
2. Списки документов с restricted participant access.
3. Assignment list for executor/status/deadline/overdue.
4. Pending acknowledgments for user.
5. Journal by document.
6. Admin audit list with pagination.
7. Search filters with `ILIKE`.
8. Registration unique conflict path for duplicate registration number.

## Findings

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| B.03.046 | ok | none | SQL-запросы инвентаризованы на уровне repository methods. |
| B.03.047 | ok | none | Запросы сгруппированы по сценариям. |
| B.03.048 | ok | none | Частые reads выделены. |
| B.03.049 | ok | none | Representative heavy reads проверены; restricted access list самый тяжелый из снятых, ~2.89 ms. |
| B.04.051 | ok | none | Representative `EXPLAIN` собран. |
| B.04.052 | ok | none | Representative `EXPLAIN ANALYZE` собран. |
