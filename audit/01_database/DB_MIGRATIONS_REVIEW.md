# DB Migrations Review

Дата аудита: 2026-05-27

## Инвентаризация

| Migration | Назначение | Статус |
| --- | --- | --- |
| 001_core_users | departments, users, system permissions | ok |
| 002_nomenclature | nomenclature, document permissions, organizations, resolution executors | ok |
| 003_documents | documents, detail tables, correspondents, resolutions, links | ok |
| 004_assignments | assignments, co-executors | ok |
| 005_attachments_and_settings | attachments, system settings, acknowledgments | issue |
| 006_journal | document_journal | issue |
| 007_admin_audit_log | admin_audit_log | issue |

## Findings

### B.01.024

Статус: ok
Severity: none
Место: migrations
Доказательство: локальный Docker PostgreSQL test contour был сброшен через `docker compose --env-file .envExample down -v`, затем поднят заново. Все `internal/database/migrations/*.up.sql` применились последовательно через `psql -v ON_ERROR_STOP=1`. После применения получено 25 public tables, 74 indexes и 5 строк `system_settings`. Отдельно проверен application migrator/golang-migrate через embedded migrations: `schema_migrations=7,false`, `GetMigrationStatus`: `version=7 dirty=false total=7 uptodate=true`.
Проблема: не обнаружена для SQL-воспроизводимости схемы с нуля.
Рекомендация: повторить application migrator check в CI/release gate.

### B.01.027

Статус: issue
Severity: critical
Место: `*.down.sql`
Доказательство: down migrations используют `DROP TABLE IF EXISTS`; runtime rollback через application migrator откатил `schema_migrations` с `7,false` на `6,false`, после чего `to_regclass('public.admin_audit_log')` вернул NULL.
Проблема: rollback последней миграции удаляет production history/data.
Рекомендация: так как по `DECISION-003` full production UI/runtime rollback сохраняется, каждый rollback должен считаться destructive operation и требовать явного подтверждения, свежего backup PostgreSQL+MinIO, audit entry и rollback-runbook. Down migrations для новых версий должны проходить отдельный review на data-loss impact.
Decision: см. `DECISION-003` — полный механизм управления миграциями через UI/runtime path сохраняется, включая rollback.

### 005 seed idempotency

Статус: issue
Severity: minor
Место: `005_attachments_and_settings.up.sql`
Доказательство: default settings вставляются обычным `INSERT`, без `ON CONFLICT`.
Проблема: при ручном частичном replay после dirty migration вставка может упасть на primary key.
Рекомендация: использовать `INSERT ... ON CONFLICT (key) DO NOTHING/UPDATE`, если ожидается manual replay.

## Проверки, Которые Еще Нужны

- Повторить application migrator check в CI/release gate.
- Последовательный rollback на тестовой БД с явным подтверждением ожидаемой потери данных.
- Проверка dirty migration recovery.
- Schema dump после migration up.
