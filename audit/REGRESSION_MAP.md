# Regression Map

Дата аудита: 2026-05-27

## CHANGE-001

Что изменено: Выполнен перенос выдачи `nomenclature.next_number` внутрь транзакции регистрации документа и добавлен `documents.idempotency_key UUID NOT NULL` с unique `(created_by, kind, idempotency_key)`.
Затронутые файлы: `internal/database/migrations/008_registration_idempotency.*.sql`, `internal/repository/document_registration.go`, document command handlers/repositories, request models, frontend registration pages.
Затронутые пункты плана: B.02.032, B.04.060, B.05.066, B.05.067, B.05.068, C.02, C.03.
Что перепроверить: no-gaps numbering after failed create — covered by `TestDocumentRegistrationIdempotencyIntegration`; repeated submit with same idempotency key returns existing document — covered by `TestDocumentRegistrationIdempotencyIntegration`; concurrent duplicate idempotency key returns one document and one numbering increment — covered by `TestDocumentRegistrationConcurrencyIntegration`; concurrent registration for same nomenclature with different keys produces contiguous unique numbers — covered by `TestDocumentRegistrationConcurrencyIntegration`; duplicate registration number conflict; frontend repeated click behavior.
Что не нужно перепроверять: unrelated reference CRUD and dashboard counters except if DTO changes affect them.

## CHANGE-002

Что изменено: По `DECISION-005` выполнен пересмотр FK/cascade strategy для `document_journal` и `admin_audit_log`; добавлена миграция `009_retention_safe_journal_fks` с retention-safe `ON DELETE RESTRICT`.
Затронутые файлы: `internal/database/migrations/009_retention_safe_journal_fks.*.sql`, `internal/repository/document_registration_integration_test.go`.
Затронутые пункты плана: B.02.030, B.02.039, C.04, H.03.
Что перепроверить: direct delete блокируется и журналы сохраняются — covered by `TestJournalRetentionFKIntegration`; user deactivation/reactivation; отсутствие штатного document/user physical delete; journal retention; admin audit retention; backup/restore.
Что не нужно перепроверять: document form rendering.

## CHANGE-003

Что изменено: Добавлена миграция `010_drop_duplicate_users_login_index`: `up` удаляет дублирующий `idx_users_login`, `down` восстанавливает его при rollback. Conditional indexes for document lists/access/acknowledgments/assignments остаются отложенными до evidence на production-like dataset.
Затронутые файлы: `internal/database/migrations/010_drop_duplicate_users_login_index.*.sql`.
Затронутые пункты плана: B.04.053-B.04.064, F.06.
Что перепроверить: login/auth lookup; `pg_indexes` после миграции; отсутствие structural duplicate indexes; rollback миграции `010` на disposable DB. EXPLAIN before/after for document list indexes остается актуальным только для будущих conditional indexes.
Что не нужно перепроверять: UI copy.

## CHANGE-004

Что изменено: Полный production runtime/UI механизм управления миграциями сохраняется, включая rollback; планируется усилить guardrails вокруг destructive rollback.
Затронутые файлы: `internal/services/settings.go`, `internal/database/postgres.go`, `frontend/src/pages/SettingsPage.tsx`, release/ops docs.
Затронутые пункты плана: B.01.027, B.06.074, B.06.075, E.04, H.03.
Что перепроверить: admin settings migration UI; destructive rollback warning/confirmation; first-run migrations; forward migration from old version; rollback audit entry; backup restore; admin audit entries for migration operations.
Что не нужно перепроверять: document list filters unless migration UI shared state changes.

## CHANGE-005

Что изменено: Выполнен hardening `restore_smb_tar.sh` по `DECISION-006`: fail-fast PostgreSQL restore через `pg_restore --exit-on-error`, restore report, preflight archive content checks, DB smoke validation, запрет MinIO mirror до успешной DB validation.
Затронутые файлы: `restore_smb_tar.sh`, `audit/REVIEW_LOG.md`, `audit/01_database/DB_BACKUP_RESTORE_REVIEW.md`, `audit/08_docs_release/KNOWN_ISSUES.md`.
Затронутые пункты плана: B.06.074, B.06.075, E.01, H.03.
Что перепроверить: успешный restore PostgreSQL+MinIO из валидного архива; corrupted/incompatible dump останавливает workflow до MinIO; отсутствующий `minio_files/` или `database.dump` блокирует restore; cleanup временных файлов; отчет восстановления; RPO/RTO/retention. Статически проверено: `bash -n restore_smb_tar.sh`.
Что не нужно перепроверять: UI migration management, кроме сценария fresh backup перед rollback.

## CHANGE-006

Что изменено: Выполнен переход backend/Wails errors на structured envelope по `DECISION-007`, and frontend now consumes it through `formatAppError`/`normalizeAppError`.
Затронутые файлы: `internal/models/errors.go`, `main.go`, service boundary helpers, `frontend/src/utils/appError.ts`, frontend Wails error handling.
Затронутые пункты плана: C.01.079-C.01.081, C.03.090, C.04.091-C.04.093, D.05, D.06.
Что перепроверить: login failure; unauthorized session; forbidden action; validation error; not found; duplicate/idempotency conflict; internal DB/storage failure; frontend notifications/forms/empty states; tests no longer compare raw strings.
Что не нужно перепроверять: SQL query plans, except if repository error mapping changes queries.

## CHANGE-007

Что изменено: Выполнен строгий DTO decode для document registration/update и обязательный `idempotencyKey` в create-командах.
Затронутые файлы: `internal/services/document_kind_command_handler.go`, `*_command_handler.go`, request models, frontend registration forms/API calls, migrations/repositories по `CHANGE-001`.
Затронутые пункты плана: C.02.083-C.02.085, C.03.086-C.03.089, D.05, F.02.
Что перепроверить: все 4 формы регистрации; update forms; payload с unknown field отклоняется; missing idempotencyKey отклоняется; date/null serialization; repeated submit returns existing document.
Что не нужно перепроверять: dashboard charts unless document DTO response shape changes.

## CHANGE-008

Что изменено: Планируется проброс app/request context и timeout/cancel policy по `DECISION-008`.
Затронутые файлы: Wails startup/shutdown, `internal/services/attachment.go`, `link_service.go`, `journal_service.go`, `statistics_service.go`, repository/storage interfaces using context.
Затронутые пункты плана: C.05.094-C.05.097, E.05, F.06, G.07.
Что перепроверить: upload/download/delete attachment; download to disk; open file/folder; link graph; journal writes after document actions; storage statistics; app shutdown during long MinIO operation; DB disconnect/reconnect behavior.
Что не нужно перепроверять: static document form validation unless method signatures change.

## CHANGE-009

Что изменено: Планируется переход frontend на единый Wails error adapter по `DECISION-009`.
Затронутые файлы: `frontend/src/store/useAuthStore.ts`, frontend pages/components catch handlers, form validation helpers.
Затронутые пункты плана: D.02.108, D.05.134, D.06.142, F.04, H.03.
Что перепроверить: login failure; locked account; forbidden action; validation error; missing document; duplicate/idempotency conflict; internal DB/storage failure; toast/inline error copy; absence of raw PostgreSQL/storage text in UI.
Что не нужно перепроверять: SQL query plans and migrations.

## CHANGE-010

Что изменено: Планируется добавление frontend submitting guards и dirty confirmation для критичных форм/actions.
Затронутые файлы: document pages/modals, `DocumentKindPage`, settings/actions, assignments/files where applicable.
Затронутые пункты плана: D.02.110, D.02.113, D.04.123-D.04.130.
Что перепроверить: double-click submit; closing dirty registration/edit modal; settings CRUD; migration rollback/apply buttons; file upload/delete; assignments create/update/return/complete.
Что не нужно перепроверять: backend no-gaps transaction except repeated registration smoke.

## CHANGE-011

Что изменено: Планируется унификация version source and release build gate по `DECISION-010`.
Затронутые файлы: `docs/releases.yaml`, `internal/releaseassets/current_release.yaml`, Wails config/build metadata, `Makefile`/release script, installer metadata.
Затронутые пункты плана: E.01.154-E.01.159, H.01-H.03.
Что перепроверить: clean-machine build; generated release assets freshness; About version; binary properties; installer DisplayVersion; Linux/Windows artifact names; frontend/go tests in release gate.
Что не нужно перепроверять: document access matrix unless generated Wails bindings change.

## CHANGE-012

Что изменено: Планируется изменение production config lookup/startup diagnostics по `DECISION-011`.
Затронутые файлы: `internal/config/config.go`, `main.go`, installer/runbook/config docs.
Затронутые пункты плана: E.02.160-E.02.163, E.04.169, E.04.174.
Что перепроверить: launch from shortcut; launch from different cwd; path with spaces/Cyrillic; missing config; invalid JSON config; unreadable config; encrypted config with missing/wrong key; startup diagnostics content.
Что не нужно перепроверять: SQL query plans.

## CHANGE-013

Что изменено: Планируется добавление schema compatibility/downgrade/dirty guard по `DECISION-012`.
Затронутые файлы: `internal/database/postgres.go`, startup/migration services, Settings migration UI, release/ops docs.
Затронутые пункты плана: E.03.164-E.03.168, H.03.
Что перепроверить: current binary on old DB; old binary on newer DB; dirty migration state; failed migration UX; rollback guardrails; backup-before-migration checklist.
Что не нужно перепроверять: attachment filename behavior.

## CHANGE-014

Что изменено: Выполнен hardening backup/restore temp cleanup, SMB credential handling and attachment download collision safety: temp dirs создаются через `mktemp -d`, ограничиваются `chmod 700`, удаляются общим cleanup trap при любом выходе; SMB mount uses CIFS credentials file support instead of password in mount args; `DownloadToDisk` uses `O_EXCL` and collision suffixes. Broader secret handling remains planned.
Затронутые файлы: `backup_smb_tar.sh`, `restore_smb_tar.sh`, `.envExample`, `docs/backup_restore_runbook.md`, `internal/services/attachment.go`, `internal/services/attachment_test.go`, `audit/REVIEW_LOG.md`, `audit/04_build_install_update/FILESYSTEM_REVIEW.md`, `audit/08_docs_release/KNOWN_ISSUES.md`, `audit/08_docs_release/DOCUMENTATION_REVIEW.md`.
Затронутые пункты плана: E.04.170-E.04.173, E.05.180, H.03.
Что перепроверить: backup interruption cleanup; restore interruption cleanup; temp dir permissions; duplicate attachment filename download; OpenFile/OpenFolder path validation; process list/log secret exposure. Duplicate filename helper covered by `TestWriteDownloadFileWithoutOverwrite`.
Что не нужно перепроверять: Wails installer metadata unless packaging scripts change.

## CHANGE-015

Что изменено: Закрыт `ISSUE-032`: `go.mod` now requires `go 1.26.3`; `golang.org/x/net` upgraded to `v0.53.0` with compatible `x/crypto v0.50.0` and `x/text v0.36.0`.
Затронутые файлы: `go.mod`, `go.sum`, release/security audit docs.
Затронутые пункты плана: F.01.181, H.02.
Что перепроверить: `govulncheck ./...`; `go test ./...`; `go vet ./...`; integration registration tests; MinIO upload/download; Seq logging; Wails build on Linux/Windows.
Что не нужно перепроверять: frontend visual layout unless Wails/generated bindings change.

## CHANGE-016

Что изменено: Закрыт `ISSUE-033`: добавлены Makefile targets `release-gate`, `security-gate`, `govulncheck`, `npm-audit`, `npm-license-check`, `license-inventory`; release docs point to `make release-gate`.
Затронутые файлы: `Makefile`, `.gitignore`, `README.md`, `docs/release_runbook.md`, audit release/security docs.
Затронутые пункты плана: F.01.181-F.02.189, H.02.
Что перепроверить: release gate succeeds on clean machine; fails on known vulnerable dependency; npm audit; npm GPL-family license check; dependency inventories; go vet; npm lint/build. Full unknown-license policy remains under `ISSUE-037`.
Что не нужно перепроверять: database migration semantics unless Go module updates affect runtime behavior.

## CHANGE-017

Что изменено: Закрыт `ISSUE-034`: добавлены frontend ESLint flat config, `npm run lint`, dev dependencies and release-gate integration. Existing lint warnings are accepted as technical debt for the initial gate.
Затронутые файлы: `frontend/eslint.config.js`, `frontend/package.json`, `frontend/package-lock.json`, `Makefile`, frontend pages/components/hooks identified by lint warnings.
Затронутые пункты плана: F.02.188-F.02.189, D.01, D.05.
Что перепроверить: `npm run lint`; `make release-gate`; all document list/register/edit flows if future lint warning fixes touch behavior; assignment and acknowledgment flows; dashboard/statistics; `npm run build`.
Что не нужно перепроверять: backup/restore scripts.

## CHANGE-018

Что изменено: Планируется добавление release test gate with safe DB integration tests and frontend/e2e smoke.
Затронутые файлы: test scripts, CI/release scripts, frontend test config, e2e config, integration test helpers.
Затронутые пункты плана: G.01.190-G.02.203, H/I release gates.
Что перепроверить: `go test ./...`; disposable PostgreSQL integration tests; frontend component tests; production-build e2e smoke; test data isolation; failure when unsafe DSN is used.
Что не нужно перепроверять: license inventory unless release script also changes license tooling.

## CHANGE-019

Что изменено: Планируется добавление performance baseline and long-running smoke suite.
Затронутые файлы: performance scripts/docs, smoke checklist, possibly e2e tooling.
Затронутые пункты плана: G.03.204-G.04.220.
Что перепроверить: startup <=5s; lists/search <=2s; statistics <=5s; registration save latency; memory <=512 MB under normal work; repeated modals/views/files; app close during long operation; DB/MinIO outage handling.
Что не нужно перепроверять: pure SQL migrations unless performance scripts provision data.

## CHANGE-020

Что изменено: Планируется remediation for context cancellation/shutdown lifecycle and tests around long operations.
Затронутые файлы: Wails startup/shutdown, file/storage/statistics/link/journal services, repository/storage interfaces, frontend progress/cancel UI.
Затронутые пункты плана: C.05, E.05, G.04.
Что перепроверить: upload/download/delete; statistics; link graph; journal writes; app shutdown during operation; cancellation messages; no goroutine/resource leaks.
Что не нужно перепроверять: document form static labels unless UI copy changes.

## CHANGE-021

Что изменено: Планируется унификация UX терминологии по `DECISION-017`.
Затронутые файлы: frontend forms/pages/components, document configs, glossary, maybe backend validation messages.
Затронутые пункты плана: H.02.225-H.02.231, H.04.241-H.04.247, H.05.248-H.05.256.
Что перепроверить: all document registration/edit forms; document view card; settings/reference tabs; assignments filters; statistics filters; no semantic change in DTO payloads.
Что не нужно перепроверять: DB migrations unless backend validation messages are changed.

## CHANGE-022

Что изменено: Планируется безопасная UX-карта ошибок and stronger destructive confirmations по `DECISION-018`.
Затронутые файлы: frontend error adapter/catch handlers, Modal/Popconfirm destructive actions, Settings migration/storage actions, file/assignment/reference deletes.
Затронутые пункты плана: H.03.232-H.03.240, G.04.218-G.04.220.
Что перепроверить: login/locked account; validation errors; forbidden; not found; conflict/idempotency; DB/MinIO failures; file delete; assignment delete; reference delete; migration rollback; bulk file delete.
Что не нужно перепроверять: SQL query plans.

## CHANGE-023

Что изменено: Планируется улучшение empty states, placeholders, tooltips and style consistency.
Затронутые файлы: frontend list/table/chart/file components, filters, visible text constants.
Затронутые пункты плана: H.02.228-H.02.231, H.03.238, H.04.241-H.04.245.
Что перепроверить: empty document lists; empty assignments; empty statistics charts; files tab with/without upload permission; keyboard/tooltip accessibility; mobile/narrow layouts for longer labels.
Что не нужно перепроверять: backend transaction invariants.

## CHANGE-024

Что изменено: Созданы финальные audit release artifacts for stage I: documentation review, release checklist, smoke test, known issues, final production candidate review and final summary.
Затронутые файлы: `audit/08_docs_release/*`, `audit/FINAL_SUMMARY.md`, `audit/REVIEW_LOG.md`, `audit/RISK_REGISTER.md`, `audit/REGRESSION_MAP.md`, `audit/DECISIONS.md`, `audit/00_project_context/PROJECT_CONTEXT.md`.
Затронутые пункты плана: I.01.257-I.02.270.
Что перепроверить: consistency between `KNOWN_ISSUES.md`, `REVIEW_LOG.md`, `RISK_REGISTER.md` and final decision; no release checklist item claims readiness while critical issues remain open; generated docs are promoted into maintained project docs before release.
Что не нужно перепроверять: runtime document registration behavior unless release docs/scripts change build/test inputs.

## CHANGE-025

Что изменено: Закрыт `ISSUE-050`: добавлены maintained root `README.md`, `docs/release_runbook.md` and `docs/diagnostics_runbook.md`; root README links release, rollback, backup/restore and diagnostics runbooks. Clean release execution remains covered by `ISSUE-052`/`ISSUE-053`.
Затронутые файлы: `README.md`, `docs/release_runbook.md`, `docs/diagnostics_runbook.md`, `docs/backup_restore_runbook.md`, `docs/migration_rollback_runbook.md`, audit release artifacts.
Затронутые пункты плана: I.01.258-I.02.270.
Что перепроверить: clean clone build; non-author checklist execution; fresh DB setup; migration/rollback runbook; backup/restore smoke; target OS install; version metadata; clean git status at release tag.
Что не нужно перепроверять: low-level SQL plans unless release scripts provision performance dataset.

## CHANGE-026

Что изменено: Закрыт `ISSUE-007`: runtime rollback миграций теперь требует backend-enforced подтверждений свежего PostgreSQL+MinIO backup, backup reference, acknowledgment риска потери данных and control phrase; frontend показывает отдельную форму подтверждения; rollback action writes admin audit entries; добавлен maintained rollback runbook.
Затронутые файлы: `internal/models/settings.go`, `internal/services/settings.go`, `internal/services/settings_test.go`, `frontend/src/pages/SettingsPage.tsx`, `frontend/wailsjs/go/services/SettingsService.*`, `frontend/wailsjs/go/models.ts`, `docs/migration_rollback_runbook.md`, `audit/*`.
Затронутые пункты плана: B.01.027, B.02.039, B.06.075, E.03.168, H.03.
Что перепроверить: migration settings tab; rollback without each required confirmation; rollback with valid request on disposable DB; audit log entries; dirty schema handling; backup/restore runbook path.
Что не нужно перепроверять: document registration idempotency and retention FK behavior unless migration sequence changes.

## CHANGE-027

Что изменено: Закрыт `ISSUE-027`: добавлен schema compatibility guard для embedded migrations. `GetMigrationStatus` больше не считает DB schema version выше embedded migrations актуальной; `RunMigrations`, `RollbackMigration` and login блокируют newer/dirty schema через structured `CONFLICT`; migration UI показывает состояние `Схема новее приложения` and disables unsafe actions.
Затронутые файлы: `internal/database/postgres.go`, `internal/database/postgres_test.go`, `internal/services/auth_service.go`, `internal/services/settings.go`, `internal/services/settings_test.go`, `frontend/src/pages/SettingsPage.tsx`, `frontend/wailsjs/go/models.ts`, `audit/REVIEW_LOG.md`, `audit/08_docs_release/KNOWN_ISSUES.md`.
Затронутые пункты плана: B.01.027, E.03.164-E.03.166, H.03.
Что перепроверить: old binary against newer DB schema; current binary against old DB schema; dirty migration state; initial setup on empty DB; admin migration status tab; login error copy for incompatible schema; apply/rollback button disabled states.
Что не нужно перепроверять: document registration numbering/idempotency and attachment storage behavior.
