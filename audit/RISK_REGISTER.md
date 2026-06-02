# Risk Register

Дата аудита: 2026-05-07
Этап: A. Базовый контекст проекта

## RISK-001

Категория: Database
Описание: Автоматическая нумерация документов (`nomenclature.next_number`) выполняется до транзакции создания документа. Локальная проверка подтвердила: после autocommit-инкремента и ошибочного insert документа счетчик стал `next_number=2`, а документ с выданным номером не создан. Также регистрация документов должна быть идемпотентной по backend `idempotency_key`, чтобы повторный submit не создавал дубль и не расходовал следующий номер.
Вероятность: high
Влияние: high
Митигирующее действие: По `DECISION-004` добавить `documents.idempotency_key UUID NOT NULL`, unique `(created_by, kind, idempotency_key)` и перенести проверку idempotency key, выдачу номера и создание документа в одну DB transaction.
Ответственный этап: C

## RISK-002

Категория: Storage
Описание: Вложения распределены между PostgreSQL metadata и MinIO object storage. Компенсация при ошибке есть при upload metadata failure, но полной cross-resource transaction нет.
Вероятность: medium
Влияние: high
Митигирующее действие: Recovery policy подтверждена: при рассинхронизации восстанавливаются одновременно PostgreSQL и MinIO из согласованного backup-набора. На этапах B/C проверить все сценарии upload/delete/bulk delete/download, а на B/H проверить согласованность backup/restore процедуры.
Ответственный этап: B, C

## RISK-003

Категория: Configuration
Описание: Production network/security policy предполагает закрытый защищенный LAN-контур без передачи данных по открытым каналам; production guide базируется на утвержденной документации вне проекта, ссылку на нее нельзя добавить в audit context. Repo examples are explicitly marked local/dev after `ISSUE-001` remediation and use local endpoints, disabled SSL and weak placeholder passwords that are allowed only in the closed development contour.
Вероятность: medium
Влияние: high
Митигирующее действие: Production разворачивается вручную через `config/config.json` по утвержденной документации. Repo examples are marked local/dev only. На этапах B/E/H вручную сверить production configuration guide с утвержденной документацией, проверить LAN-границы закрытого контура, runtime defaults и секреты перед release; не использовать local/dev example values как production defaults.
Ответственный этап: E

## RISK-004

Категория: Operations
Описание: Backup/restore PostgreSQL+MinIO реализован shell scripts с SMB mount и Docker commands. Абсолютный путь к `.env` подтвержден как допустимый для cron и описан в `docs/backup_restore_runbook.md`; scripts support CIFS credentials file instead of password in mount args. Production restore still needs target-contour confirmation.
Вероятность: medium
Влияние: high
Митигирующее действие: Backup policy подтверждена: RPO 1 день, RTO 1-2 дня, retention 15 дней, проверка архивов через ручной test restore PostgreSQL+MinIO перед релизом, offsite copy на другой сервер, Seq не бэкапится. Runbook фиксирует production cron path к `.env`, `SMB_CREDENTIALS_FILE`, права файлов и release-gate проверки; перед релизом выполнить manual test restore.
Ответственный этап: B, H

## RISK-005

Категория: Database
Описание: UI позволяет пользователю с системным правом `admin` запускать и откатывать миграции. Откат production migrations может быть разрушительным, если down migration теряет данные.
Вероятность: medium
Влияние: high
Митигирующее действие: По `DECISION-003` полный UI/runtime migration management сохраняется, включая rollback для `admin`; нужны destructive warning/confirmation, свежий backup PostgreSQL+MinIO перед rollback, audit entry и rollback-runbook.
Ответственный этап: B, H

## RISK-006

Категория: Access Control
Описание: Access model сочетает системные права, document permissions, подразделения, поручения и ознакомления. Ошибка в scope может дать слишком широкий или слишком узкий доступ.
Вероятность: medium
Влияние: high
Митигирующее действие: На этапах C/D покрыть access matrix тестами и UI smoke-сценариями.
Ответственный этап: C, D, F

## RISK-007

Категория: Performance
Описание: Frontend build предупреждает о большом основном чанке; фактические показатели нужно сверить с performance budget на ожидаемом объеме данных.
Вероятность: medium
Влияние: medium
Митигирующее действие: Performance budget и baseline объема подтверждены вручную: до 1000 документов в год, вложения пропорционально документам, до 20 пользователей, средний файл около 3 MB, storage около 1 TB, storage warning 80%, critical 90%. На этапе F измерить фактический старт, списки, отчеты, память и размер сборки на production-like данных.
Ответственный этап: F

## RISK-008

Категория: Build
Описание: Go package `main` зависит от существования `frontend/dist` из-за embedded assets; тесты/сборка root package падают до frontend build.
Вероятность: medium
Влияние: medium
Митигирующее действие: Зафиксировать build order в CI/release script: frontend build before Go/Wails build/test.
Ответственный этап: E, H

## RISK-009

Категория: Database
Описание: `document_journal` и `admin_audit_log` должны храниться весь жизненный цикл проекта и не удаляться на уровне приложения. Локальная проверка показала, что текущие `ON DELETE CASCADE` FK удаляют journal/audit строки при delete user/document. При росте объема данных lifetime retention также может повлиять на размер PostgreSQL, индексы, скорость фильтрации и длительность backup/restore; текущий baseline — до 1000 документов в год, до 20 пользователей и storage около 1 TB, поэтому storage thresholds 80%/90% в ближайшие годы математически маловероятны.
Вероятность: medium
Влияние: medium
Митигирующее действие: По `DECISION-005` не вводить application-level physical delete для users/documents и заменить cascade FK журналов на retention-safe strategy; на этапах F/H оценить рост БД, backup window и необходимость архивирования без удаления из production history.
Ответственный этап: B, F, H

## RISK-010

Категория: Database/Migrations
Описание: Destructive down migrations доступны через runtime rollback path для пользователя с `admin`, и по принятому требованию полный механизм управления миграциями сохраняется в production. Локальная проверка application migrator подтвердила: rollback version 7 -> 6 удаляет `admin_audit_log`. Ошибка эксплуатации может удалить production tables/data.
Вероятность: medium
Влияние: high
Митигирующее действие: Сохранить rollback, но усилить guardrails: destructive warning/confirmation, свежий backup PostgreSQL+MinIO перед rollback, audit entry, rollback-runbook, review каждой новой down migration на data-loss impact.
Ответственный этап: B, H

## RISK-011

Категория: Database/Performance
Описание: Representative EXPLAIN ANALYZE на 1000 documents показал быстрые планы, но частые списки документов, поручений и ознакомлений используют сложные filters/access EXISTS, search и OFFSET pagination; возможна деградация при росте данных или отличии production dataset от synthetic baseline.
Вероятность: medium
Влияние: medium
Митигирующее действие: Собрать representative data, выполнить EXPLAIN/EXPLAIN ANALYZE, добавить точечные composite/partial/trigram indexes, повторно проверить write latency.
Ответственный этап: B, F

## RISK-012

Категория: Database/Restore
Описание: Restore script продолжает работу после любого ненулевого `pg_restore`, поэтому может быть принят неполный restore.
Вероятность: medium
Влияние: high
Митигирующее действие: По `DECISION-006` сделать restore fail-fast для fatal/unknown `pg_restore` errors, не запускать MinIO mirror до успешной DB validation, формировать restore report и выполнять mandatory smoke validation после test restore PostgreSQL+MinIO.
Ответственный этап: B, H

## RISK-013

Категория: Backend/Wails
Описание: Backend/Wails и frontend после remediation используют structured error envelope and frontend adapter for stable codes. Residual risk remains in end-to-end coverage: login/forbidden/validation/not_found/conflict/internal scenarios still need release smoke/e2e evidence.
Вероятность: medium
Влияние: medium
Митигирующее действие: Structured backend envelope and frontend `formatAppError` adapter implemented; cover the main error families in `ISSUE-038`/`ISSUE-043` frontend/e2e smoke work.
Ответственный этап: C, D

## RISK-014

Категория: Backend/Documents
Описание: Backend registration commands не имеют `idempotencyKey`, а Wails `Register/Update` принимает `any` и silently ignores unknown fields. При double submit или typo в payload возможны дубли/расход номера/непредсказуемая валидация.
Вероятность: high
Влияние: high
Митигирующее действие: По `DECISION-004` добавить обязательный `idempotencyKey`; public command DTO сделать typed или strict, unknown fields отклонять.
Ответственный этап: C, D, F

## RISK-015

Категория: Backend/Resource Lifecycle
Описание: Долгие операции используют `context.Background()` и не отменяются при закрытии окна; shutdown закрывает DB/logger без coordination active requests.
Вероятность: medium
Влияние: high
Митигирующее действие: По `DECISION-008` пробросить app/request context и timeout/cancel policy для MinIO/DB-heavy operations.
Ответственный этап: C, E, F, G

## RISK-016

Категория: Logging/Privacy
Описание: Technical production logs ранее могли содержать ФИО пользователя (`app_user`) и business identifiers, которые должны быть ограничены доменным audit trail или минимизированы.
Вероятность: low
Влияние: medium
Митигирующее действие: Выполнено: technical logs используют `app_user_id` вместо ФИО, Wails binding error logs не пишут полный текст ошибки; ФИО и бизнес-детали остаются в PostgreSQL audit/journal.
Ответственный этап: C, F, H

## RISK-017

Категория: Frontend/Error Handling
Описание: Frontend error adapter внедрен после audit, но остаётся риск непроверенных end-to-end сценариев: forbidden/not_found/conflict/internal cases могут регрессировать без UI/e2e smoke.
Вероятность: medium
Влияние: medium
Митигирующее действие: `formatAppError`/`normalizeAppError` внедрены для Wails errors; добавить smoke cases в рамках `ISSUE-038`/`ISSUE-043`.
Ответственный этап: D, F, H

## RISK-018

Категория: Frontend/UX
Описание: Критичные frontend actions не везде имеют локальный submitting/loading guard, а длинные формы в модалках можно закрыть без dirty confirmation.
Вероятность: medium
Влияние: medium
Митигирующее действие: Для create/update/delete/rollback/upload/assignment actions добавить локальный submit lock; для document/settings forms добавить unsaved changes confirmation.
Ответственный этап: D, F

## RISK-019

Категория: Frontend/Maintainability
Описание: Крупные страницы settings/statistics/document view смешивают UI, data loading, mutations и modal lifecycle, что повышает риск регрессий при дальнейших исправлениях.
Вероятность: medium
Влияние: medium
Митигирующее действие: Декомпозировать эти зоны постепенно при функциональных изменениях; использовать hooks/subcomponents и smoke checklist.
Ответственный этап: D, F

## RISK-020

Категория: Build/Release
Описание: Production build now has a deterministic release gate with required `ENCRYPTION_KEY`, generated asset freshness check and `npm ci`. Secret delivery, permissions and rotation are defined in `docs/secret_policy.md`. Remaining risk is target OS artifact metadata and evidence execution.
Вероятность: low
Влияние: high
Митигирующее действие: Run `make release-gate` from clean checkout, build with approved secret injection, follow `docs/secret_policy.md`, then verify binary properties/installer metadata and secret exposure checks on target OS.
Ответственный этап: E, H

## RISK-021

Категория: Installation/Runtime Config
Описание: Runtime config lookup supports `DOCFLOW_CONFIG_PATH`, executable-relative `config/config.json` and cwd fallback for local development. Remaining risk is missing/invalid config diagnostics and target OS smoke.
Вероятность: low
Влияние: high
Митигирующее действие: Use `DOCFLOW_CONFIG_PATH` or executable-relative config for production; verify shortcut/default cwd/path with spaces/Cyrillic and missing/invalid config in target OS smoke.
Ответственный этап: E, H

## RISK-022

Категория: Update/Migrations
Описание: Downgrade/schema compatibility guard добавлен после remediation `ISSUE-027`: newer/dirty DB schema blocks login and migration operations through structured `CONFLICT`, and migration UI distinguishes `schemaTooNew`. Residual risk remains for target-contour smoke and dirty migration recovery/runbook.
Вероятность: medium
Влияние: high
Митигирующее действие: Выполнено: incompatible newer/dirty schema versions блокируются backend guard. Далее определить auto/explicit migration policy, добавить dirty-state runbook and target OS smoke tests.
Ответственный этап: E, H

## RISK-023

Категория: Filesystem/Secrets
Описание: Backup/restore temp dirs and SMB credential exposure were reduced through cleanup traps, `chmod 700` temp dirs and CIFS credentials file support. Residual risk remains until target-contour interruption/process-list smoke is executed.
Вероятность: medium
Влияние: medium
Митигирующее действие: Cleanup traps для temp dirs, restrictive permissions, CIFS credentials file, process-list/log secret check in release gate.
Ответственный этап: E, H

## RISK-024

Категория: Filesystem/User Data
Описание: Attachment download overwrite risk was reduced: downloads now use collision-safe names with `O_EXCL` and suffixes like `file (1).ext`.
Вероятность: medium
Влияние: low
Митигирующее действие: Regression test `TestWriteDownloadFileWithoutOverwrite`; manual Download/OpenFile/OpenFolder smoke on target OS.
Ответственный этап: E, H

## RISK-025

Категория: Security/Dependencies
Описание: Go reachable vulnerability blocker fixed after `ISSUE-032`: `go.mod` requires `go1.26.3`, `golang.org/x/net@v0.53.0`, and `govulncheck` reports 0 reachable vulnerabilities. Residual risk remains if release is built with a different toolchain or without the security gate.
Вероятность: low
Влияние: high
Митигирующее действие: Keep Go toolchain at `go1.26.3+`, keep `x/net` at `v0.53.0+`, run `govulncheck` in release gate from clean checkout.
Ответственный этап: F, H

## RISK-026

Категория: Release/Security Gates
Описание: Repeatable security/dependency release gate added after `ISSUE-033` and license policy completed after `ISSUE-037`: `make release-gate` runs `govulncheck`, `npm audit`, `go vet`, `go test`, `npm run lint`, `npm run build`, npm/Go license report generation, unknown-license blocking and GPL/LGPL/AGPL-family blocking.
Вероятность: low
Влияние: high
Митигирующее действие: Run `make release-gate` from clean checkout; archive `build/release-evidence/LICENSE_REPORT.md` with release evidence and notices.
Ответственный этап: F, H

## RISK-027

Категория: Static Analysis/Frontend
Описание: Frontend ESLint gate added after `ISSUE-034`; `npm run lint` and `make release-gate` pass with warnings. Residual risk remains from existing warnings and broad `any` around Wails contract boundaries.
Вероятность: low
Влияние: medium
Митигирующее действие: Keep `npm run lint` in release gate; reduce warnings gradually during frontend remediation; replace broad `any` with typed contracts where practical.
Ответственный этап: F, H

## RISK-028

Категория: Testing/Frontend-E2E
Описание: Frontend forms, error states, empty states, navigation and full user lifecycle are not covered by automated frontend/e2e tests.
Вероятность: high
Влияние: high
Митигирующее действие: Add frontend component tests and production-build e2e smoke for critical workflows before release.
Ответственный этап: G, H

## RISK-029

Категория: Testing/Integration Data Safety
Описание: Critical PostgreSQL integration tests are gated by an external DSN and reset `public` schema. They can be skipped in release gate or damage data if pointed at a non-disposable DB.
Вероятность: medium
Влияние: high
Митигирующее действие: Provision disposable DB automatically; add DSN/database-name safety guard; include integration tests in release pipeline.
Ответственный этап: G, H

## RISK-030

Категория: Performance
Описание: Production SLO exists, but startup/list/save/statistics/memory metrics are not measured on target Wails build and OS.
Вероятность: medium
Влияние: high
Митигирующее действие: Establish performance baseline on target OS/build with production-like data and repeat after major DB/backend/frontend changes.
Ответственный этап: G, H

## RISK-031

Категория: Long Running
Описание: Long desktop sessions, repeated modals, repeated file operations, network outages and app shutdown during operations are not tested; cancellation lifecycle remains open.
Вероятность: medium
Влияние: high
Митигирующее действие: Add long-running smoke and automated cancellation tests after context propagation remediation.
Ответственный этап: G, H

## RISK-032

Категория: UX/Error Handling
Описание: Raw backend/system errors can appear in user-facing UI without actionable next steps.
Вероятность: high
Влияние: medium
Митигирующее действие: Map structured error codes to safe UX copy and keep technical details in logs/audit.
Ответственный этап: H, I

## RISK-033

Категория: UX/Terminology
Описание: Inconsistent document terminology (`вид`/`тип`, `дело`/`номенклатура`, different meanings of `исполнитель`) can cause wrong data entry or support confusion.
Вероятность: medium
Влияние: medium
Митигирующее действие: Adopt `TERMS_GLOSSARY.md` UX terminology and review all visible labels/messages.
Ответственный этап: H, I

## RISK-034

Категория: UX/Safety
Описание: Destructive confirmations now name the affected entity and consequence after `ISSUE-046`; remaining residual risk is lack of automated e2e/smoke coverage for these paths.
Вероятность: low
Влияние: high
Митигирующее действие: Keep strengthened destructive copy; include file delete, assignment delete, reference delete, migration rollback and bulk file delete in smoke/e2e tests under `ISSUE-043`.
Ответственный этап: H, I

## RISK-035

Категория: Documentation/Operations
Описание: Release-grade root README/runbooks for dev setup, production build, migrations, backup/restore and diagnostics were added after remediation `ISSUE-050`. Residual risk remains until non-author clean-clone execution, target OS smoke and manual restore evidence are completed.
Вероятность: medium
Влияние: medium
Митигирующее действие: Maintain `README.md`, `docs/release_runbook.md`, `docs/diagnostics_runbook.md`, backup/restore and rollback runbooks; validate them by clean-clone build, fresh DB setup, target OS install smoke and manual test restore.
Ответственный этап: I

## RISK-036

Категория: Release/Reproducibility
Описание: Audit/remediation changes were committed in `1592e9b`, and `git status --short` is clean after the commit. Residual reproducibility risk remains until the final release tag is created from a clean checkout and release evidence is attached.
Вероятность: medium
Влияние: high
Митигирующее действие: Build from clean committed/tagged checkout, run full release gate and store release evidence/checksums for the final production delivery.
Ответственный этап: I

## RISK-037

Категория: Release/Process
Описание: Release checklist, smoke-test and known issues are maintained project docs. Remaining risk is failure to execute and attach them from a clean checkout for each release.
Вероятность: low
Влияние: high
Митигирующее действие: Complete `docs/release_checklist.md` and `docs/smoke_test.md` from clean checkout; attach completed evidence and keep `docs/known_issues.md` synchronized at release tag.
Ответственный этап: I
