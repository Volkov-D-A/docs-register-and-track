# Regression Map

Дата аудита: 2026-05-27

## CHANGE-001

Что изменено: Планируется перенос выдачи `nomenclature.next_number` внутрь транзакции регистрации документа и добавление `documents.idempotency_key UUID NOT NULL` с unique `(created_by, kind, idempotency_key)`.
Затронутые файлы: `internal/database/migrations/008_registration_idempotency.*.sql`, `internal/repository/nomenclature_repo.go`, document command handlers/repositories, migrations, DTO/request models.
Затронутые пункты плана: B.02.032, B.04.060, B.05.066, B.05.067, B.05.068, C.02, C.03.
Что перепроверить: no-gaps numbering after failed create; repeated submit with same idempotency key returns existing document; concurrent duplicate idempotency key rolls back second numbering increment; duplicate registration number conflict; concurrent registration for same nomenclature with different keys; frontend repeated click behavior.
Что не нужно перепроверять: unrelated reference CRUD and dashboard counters except if DTO changes affect them.

## CHANGE-002

Что изменено: По `DECISION-005` планируется пересмотр FK/cascade strategy для `document_journal` и `admin_audit_log`; target — retention-safe FK без cascade удаления history/audit.
Затронутые файлы: migrations, journal/admin audit repositories, user/document delete paths.
Затронутые пункты плана: B.02.030, B.02.039, C.04, H.03.
Что перепроверить: user deactivation/reactivation; отсутствие штатного document/user physical delete; direct delete блокируется или сохраняет журналы согласно выбранной FK strategy; journal retention; admin audit retention; backup/restore.
Что не нужно перепроверять: document form rendering.

## CHANGE-003

Что изменено: Возможное добавление индексов под document lists/access/acknowledgments/assignments и удаление дублирующего `idx_users_login`.
Затронутые файлы: migrations.
Затронутые пункты плана: B.04.053-B.04.064, F.06.
Что перепроверить: EXPLAIN before/after; insert/update latency for document registration and assignments; login/auth lookup; migration time on production-like data; отсутствие structural duplicate indexes.
Что не нужно перепроверять: UI copy.

## CHANGE-004

Что изменено: Полный production runtime/UI механизм управления миграциями сохраняется, включая rollback; планируется усилить guardrails вокруг destructive rollback.
Затронутые файлы: `internal/services/settings.go`, `internal/database/postgres.go`, `frontend/src/pages/SettingsPage.tsx`, release/ops docs.
Затронутые пункты плана: B.01.027, B.06.074, B.06.075, E.04, H.03.
Что перепроверить: admin settings migration UI; destructive rollback warning/confirmation; first-run migrations; forward migration from old version; rollback audit entry; backup restore; admin audit entries for migration operations.
Что не нужно перепроверять: document list filters unless migration UI shared state changes.

## CHANGE-005

Что изменено: Планируется hardening `restore_smb_tar.sh` по `DECISION-006`: fail-fast PostgreSQL restore, restore report, preflight/smoke checks, запрет MinIO mirror до успешной DB validation.
Затронутые файлы: `restore_smb_tar.sh`, `backup_smb_tar.sh`, release/ops docs.
Затронутые пункты плана: B.06.074, B.06.075, E.01, H.03.
Что перепроверить: успешный restore PostgreSQL+MinIO из валидного архива; corrupted/incompatible dump останавливает workflow до MinIO; отсутствующий `minio_files/` или `database.dump` блокирует restore; cleanup временных файлов; отчет восстановления; RPO/RTO/retention.
Что не нужно перепроверять: UI migration management, кроме сценария fresh backup перед rollback.
