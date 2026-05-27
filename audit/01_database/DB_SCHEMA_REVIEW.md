# DB Schema Review

Дата аудита: 2026-05-27

## Сильные Стороны

- Центральная модель `documents` плюс detail-таблицы по видам документов хорошо соответствует домену.
- Основные enum-like значения ограничены `CHECK`.
- Бизнес-уникальность регистрационного номера частично выражена unique index: `(kind, registration_number, EXTRACT(YEAR FROM registration_date))`.
- Для справочников и join tables есть первичные/уникальные ключи.

## Issues

### ISSUE-007: destructive down migrations

Severity: critical
Пункты: B.01.027, B.02.039, B.06.075
Место: `internal/database/migrations/*.down.sql`

Все `down`-миграции удаляют таблицы через `DROP TABLE IF EXISTS`. При подтвержденной policy, где rollback migrations доступен из UI пользователю с `admin`, это production data-loss path.

Рекомендация: с учетом `DECISION-003` production UI rollback сохраняется; каждую dangerous down migration нужно документировать как destructive, проверять на data-loss impact и запускать только при наличии свежего PostgreSQL+MinIO backup.

### ISSUE-008: cascade conflicts with lifetime journal retention

Severity: major
Пункты: B.02.030, B.02.039
Место: `document_journal`, `admin_audit_log`

`document_journal.document_id` и `document_journal.user_id` используют `ON DELETE CASCADE`; `admin_audit_log.user_id` тоже использует `ON DELETE CASCADE`. Это конфликтует с подтвержденным правилом, что `document_journal` и `admin_audit_log` хранятся весь жизненный цикл проекта.

Локальная проверка:

- delete пользователя с audit-записью удалил связанную строку `admin_audit_log`: `admin_audit_remaining=0`;
- delete документа с journal-записью удалил связанную строку `document_journal`: `document_journal_remaining=0`.

Application scan: штатный repository/service delete для корневых `users` и `documents` не найден; пользователи управляются через `is_active`, документы физически не удаляются через общий document CRUD. Поэтому текущая вероятность ниже, чем при наличии UI-delete, но DB-level invariant все равно не защищен.

Рекомендация: см. `DECISION-005` — физическое удаление users/documents не является штатной application-операцией, а FK журналов нужно заменить на retention-safe strategy (`RESTRICT`/эквивалент).

### ISSUE-006: missing idempotency schema

Severity: major
Пункты: B.02.032, B.04.060, B.05.068
Место: `documents`, document command handlers

Подтвержденное бизнес-правило требует `idempotency_key` для регистрации документа. В схеме нет колонки/таблицы idempotency keys и unique constraint.

Рекомендация: по `DECISION-004`, так как старых документов нет, добавить `idempotency_key` прямо в `documents` и возвращать существующий документ при повторе `(created_by, kind, idempotency_key)`.

Проект миграции `008_registration_idempotency.up.sql`:

```sql
ALTER TABLE documents
    ADD COLUMN idempotency_key UUID NOT NULL;

CREATE UNIQUE INDEX idx_documents_created_by_kind_idempotency_key
    ON documents (created_by, kind, idempotency_key);
```

Проект rollback `008_registration_idempotency.down.sql`:

```sql
DROP INDEX IF EXISTS idx_documents_created_by_kind_idempotency_key;

ALTER TABLE documents
    DROP COLUMN IF EXISTS idempotency_key;
```

Примечание: миграция рассчитана на отсутствие legacy documents. Если ее нужно прогнать на локальной test DB с synthetic rows, test contour нужно пересоздать или временно заполнить ключи перед `NOT NULL`; для production-path backfill не требуется.

## Notes

- `updated_at` обновляется кодом, не DB trigger. Это допустимо, но требует проверки на C/E, чтобы все update paths выставляли timestamp.
- Soft delete как общая стратегия отсутствует. Для users/nomenclature/orders используются `is_active`/`cancelled_at`.
- JSON/JSONB не используется.
