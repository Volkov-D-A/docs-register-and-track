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
Описание: Production network/security policy предполагает закрытый защищенный LAN-контур без передачи данных по открытым каналам; production guide базируется на утвержденной документации вне проекта, ссылку на нее нельзя добавить в audit context. Repo examples остаются local/dev и используют local endpoints, disabled SSL и слабые placeholder passwords, которые допустимы только в закрытом контуре разработки.
Вероятность: medium
Влияние: high
Митигирующее действие: Production разворачивается вручную через `config/config.json` по утвержденной документации. На этапах B/E/H вручную сверить production configuration guide с утвержденной документацией, проверить LAN-границы закрытого контура, runtime defaults и секреты перед release; не использовать local/dev example values как production defaults.
Ответственный этап: E

## RISK-004

Категория: Operations
Описание: Backup/restore PostgreSQL+MinIO реализован shell scripts с SMB mount и Docker commands. Абсолютный путь к `.env` подтвержден как допустимый для cron, но production cron/deployment path и обращение с SMB credentials должны быть проверены; production restore не подтвержден.
Вероятность: medium
Влияние: high
Митигирующее действие: Backup policy подтверждена: RPO 1 день, RTO 1-2 дня, retention 15 дней, проверка архивов через ручной test restore PostgreSQL+MinIO перед релизом, offsite copy на другой сервер, Seq не бэкапится. На этапе B/H выполнить отдельный backup/restore review, проверить production cron path к `.env`, handling SMB credentials и то, что скрипты реально выполняют подтвержденную процедуру.
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
Описание: Frontend получает ошибки backend как plain strings. Из-за этого UI может зависеть от текстов Go/PostgreSQL/storage errors и не сможет стабильно обрабатывать validation, forbidden, not found, conflict и idempotency cases.
Вероятность: high
Влияние: high
Митигирующее действие: По `DECISION-007` ввести structured error envelope со стабильными codes и safe user messages.
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
Описание: Technical production logs могут содержать ФИО пользователя (`app_user`) и business identifiers, которые должны быть ограничены доменным audit trail или минимизированы.
Вероятность: medium
Влияние: medium
Митигирующее действие: Ввести PII minimization/redaction policy для Seq; в technical logs использовать user ID/correlation fields, а ФИО хранить только в PostgreSQL audit/journal, где это бизнес-требование.
Ответственный этап: C, F, H

## RISK-017

Категория: Frontend/Error Handling
Описание: Frontend продолжает показывать и местами разбирать raw error strings вместо stable structured error codes, несмотря на backend remediation. UI может неверно обработать forbidden/not_found/conflict/internal cases или показать пользователю небезопасный текст fallback-ошибки.
Вероятность: high
Влияние: medium
Митигирующее действие: По `DECISION-009` внедрить единый frontend error adapter для Wails errors, перевести auth/forms/tables/actions на `code/message`, добавить smoke cases.
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
Описание: Версия приложения и release notes могут разойтись между About UI, binary metadata и installer metadata; production build зависит от локального `.env`/`ENCRYPTION_KEY` и не оформлен как deterministic release gate.
Вероятность: medium
Влияние: high
Митигирующее действие: Ввести единый version source, release build script, required env validation, `npm ci`, generated asset freshness check и artifact metadata verification.
Ответственный этап: E, H

## RISK-021

Категория: Installation/Runtime Config
Описание: Приложение ищет `config/config.json` относительно текущей рабочей директории; запуск из shortcut, standard install path или другого cwd может завершиться до UI.
Вероятность: medium
Влияние: high
Митигирующее действие: Зафиксировать production config location and lookup order; добавить startup diagnostics для missing/invalid config; включить target OS smoke.
Ответственный этап: E, H

## RISK-022

Категория: Update/Migrations
Описание: Нет явного downgrade/schema compatibility guard; older binary может быть запущен против newer DB schema, а dirty migration recovery/runbook не оформлен.
Вероятность: medium
Влияние: high
Митигирующее действие: Блокировать incompatible schema versions, определить auto/explicit migration policy, добавить dirty-state runbook and smoke tests.
Ответственный этап: E, H

## RISK-023

Категория: Filesystem/Secrets
Описание: Backup/restore temp dirs и SMB mount arguments могут раскрывать database dumps, MinIO files или credentials при ошибке/прерывании процесса.
Вероятность: medium
Влияние: medium
Митигирующее действие: Cleanup traps для temp dirs, restrictive permissions, CIFS credentials file, process-list/log secret check.
Ответственный этап: E, H

## RISK-024

Категория: Filesystem/User Data
Описание: Attachment download writes to Downloads using sanitized original filename, but can overwrite existing local files without confirmation.
Вероятность: medium
Влияние: medium
Митигирующее действие: Collision-safe filenames or explicit overwrite confirmation; regression tests for duplicate names.
Ответственный этап: E, H

## RISK-025

Категория: Security/Dependencies
Описание: Current Go release environment (`go1.26.2`) and `golang.org/x/net@v0.52.0` have reachable vulnerabilities reported by `govulncheck`.
Вероятность: high
Влияние: high
Митигирующее действие: Upgrade Go toolchain to `go1.26.3+`, upgrade `x/net` to `v0.53.0+`, repeat `govulncheck`, include it in release gate.
Ответственный этап: F, H

## RISK-026

Категория: Release/Security Gates
Описание: Vulnerability and license checks are manual and not part of a repeatable release gate; future dependency updates can introduce critical advisories or license blockers.
Вероятность: medium
Влияние: high
Митигирующее действие: Add release script/checklist for `govulncheck`, `npm audit`, license reports, `go vet`, `go test`, `npm run build`.
Ответственный этап: F, H

## RISK-027

Категория: Static Analysis/Frontend
Описание: Frontend has no ESLint/Prettier gate and uses `@ts-ignore`/broad `any` around Wails contract boundaries.
Вероятность: medium
Влияние: medium
Митигирующее действие: Add minimal lint gate and gradually remove suppressions while typing Wails/document DTO boundaries.
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
Описание: Destructive confirmations do not always name the affected entity or consequence, increasing operator error risk.
Вероятность: medium
Влияние: high
Митигирующее действие: Strengthen destructive copy and include these paths in smoke/e2e tests.
Ответственный этап: H, I

## RISK-035

Категория: Documentation/Operations
Описание: Release-grade root README/runbooks are missing for dev setup, production build, migrations, backup/restore and diagnostics. Production operation can depend on author memory or external undocumented steps.
Вероятность: high
Влияние: high
Митигирующее действие: Add maintained root docs/runbooks and validate them by clean-clone build, fresh DB setup, target OS install smoke and manual test restore.
Ответственный этап: I

## RISK-036

Категория: Release/Reproducibility
Описание: Current production candidate worktree contains many modified and untracked files, including audit artifacts and product remediation. Without clean commit/tag, release artifact is not reproducible.
Вероятность: high
Влияние: high
Митигирующее действие: Review changes, run full release gate, commit/tag candidate from clean status and store release evidence.
Ответственный этап: I

## RISK-037

Категория: Release/Process
Описание: Release checklist, smoke-test and known issues existed only as audit findings before stage I and are not yet a maintained project release process.
Вероятность: medium
Влияние: high
Митигирующее действие: Promote `audit/08_docs_release` checklist/smoke/known issues into maintained release docs or script; require completed checklist for every release.
Ответственный этап: I
