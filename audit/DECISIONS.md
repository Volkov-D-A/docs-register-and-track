# Decisions

Дата аудита: 2026-05-27

## DECISION-001

Дата: 2026-05-27
Контекст: Пункт A.04.018 требует проверки отсутствия мертвого кода. В рамках этапа A уже выполнен high-level scan: очевидных orphan-модулей не найдено, fixed document type CRUD признан намеренно отключенным поведением. Глубокая dead-code проверка требует статанализа и выходит за цель этапа A, который фиксирует базовый контекст проекта.
Решение: Завершить A.04.018 в рамках этапа A как `not_applicable` для глубокой проверки и перенести dedicated dead-code/static-analysis review на последующие этапы статанализа/code quality.
Причина: Этап A не должен превращаться в глубокий аудит backend/frontend/code quality; перенос сохраняет границы этапов и не скрывает оставшуюся работу.
Альтернативы: Выполнить полный dead-code audit прямо в этапе A; оставить пункт `needs_info`.
Последствия: Этап A можно закрыть. На этапах E/F нужно отдельно проверить dead code, unused exports, generated mocks, неиспользуемые UI-компоненты и устаревшие CRUD-пути.
Какие этапы затрагивает: E, F

## DECISION-002

Дата: 2026-05-27
Контекст: В этапе B пункты B.04.051-B.04.052 требуют `EXPLAIN` и `EXPLAIN ANALYZE`, а B.04.064-B.05.065 требуют актуальной статистики и production-like данных. В текущем проходе доступна только статическая проверка кода, миграций и scripts.
Решение: Зафиксировать статический database audit сейчас, а live plan analysis, index usage stats и restore validation выполнить отдельной подзадачей этапа B после подготовки production-like PostgreSQL dataset.
Причина: Без representative data планы выполнения и статистика будут недостоверны.
Альтернативы: Пытаться делать выводы о планах только по SQL-коду; поднять синтетическую БД без согласованного dataset.
Последствия: Этап B начат и основные schema/transaction/ops риски зафиксированы. 2026-05-27 выполнен локальный representative `EXPLAIN ANALYZE` на synthetic dataset; финальный release gate все равно должен повторить планы на утвержденном production-like dataset и выполнить test restore.
Какие этапы затрагивает: B, F, H

## DECISION-003

Дата: 2026-05-27
Контекст: Runtime rollback path подтвержден: `SettingsService.RollbackMigration()` вызывает `db.RollbackMigration(database.DefaultMigrationsPath)`, а frontend `SettingsPage` предоставляет кнопку отката для пользователя с `admin`. Локальная проверка показала, что rollback миграции 007 переводит `schema_migrations` с `7,false` на `6,false` и удаляет `admin_audit_log`.
Решение: Полный механизм управления миграциями через интерфейс приложения сохраняется в production: статус, применение миграций вперед и rollback последней миграции остаются доступны пользователю с системным правом `admin`.
Причина: Для целевой эксплуатации требуется иметь полный runtime-инструмент управления миграциями из приложения. Риск destructive `down`-миграций принимается как управляемый операционный риск.
Альтернативы: Запретить production rollback через UI/runtime path и выполнять возврат назад только через backup/restore; оставить rollback только в dev/test. Эти варианты отклонены как не соответствующие требованию полного механизма управления миграциями.
Последствия: На этапах C/E/H нужно не удалять `RollbackMigration`, а усилить guardrails: явное destructive warning, подтверждение перед rollback, обязательная рекомендация/проверка свежего backup PostgreSQL+MinIO перед rollback, запись в `admin_audit_log`, документация rollback-runbook и release-gate test restore. Down migrations остаются потенциально destructive и должны проверяться отдельно.
Какие этапы затрагивает: B, C, E, H

## DECISION-004

Дата: 2026-05-27
Контекст: Подтверждено бизнес-правило strict no-gaps numbering и idempotent registration. Текущий `NomenclatureRepository.GetNextNumber` делает autocommit `UPDATE nomenclature SET next_number = next_number + 1`, после чего document repositories создают `documents` и detail rows в отдельных transactions. Локальный failure test подтвердил gap: после выданного номера insert документа упал, `next_number=2`, `docs=0`.
Решение: Целевая модель регистрации документов: `idempotency_key` обязателен для create-команд всех 4 видов документов. Так как старых документов нет, ключ хранится прямо в `documents.idempotency_key UUID NOT NULL` с unique constraint/index на `(created_by, kind, idempotency_key)`. Проверка idempotency key, выдача номера и создание `documents`, detail rows и дочерних записей выполняются в одной DB transaction. Повторный request с тем же `(created_by, kind, idempotency_key)` возвращает уже созданный документ и не инкрементирует `next_number`.
Причина: Только единая transaction boundary и unique idempotency contract защищают одновременно от gap после ошибки, double submit и race condition.
Альтернативы: Оставить только unique registration number; сделать frontend-only protection от double click; использовать sequence. Эти варианты не выполняют strict no-gaps invariant.
Последствия: Нужна миграция `008_registration_idempotency`: добавить `documents.idempotency_key UUID NOT NULL` и unique index/constraint `(created_by, kind, idempotency_key)`. Также нужны изменения request DTO/models/frontend payloads, перенос numbering logic из service-level `GetNextNumber` в repository transaction, тесты failure/concurrent/retry cases.
Какие этапы затрагивает: B, C, D, F

## DECISION-005

Дата: 2026-05-27
Контекст: Подтверждена lifetime retention policy для `document_journal` и `admin_audit_log`. В application code не найден штатный repository/service delete для корневых `documents` и `users`, пользователи управляются через `is_active`. Но schema-level FK сейчас использует `ON DELETE CASCADE`: локальная проверка подтвердила, что delete пользователя удаляет связанную запись `admin_audit_log`, а delete документа удаляет связанную запись `document_journal`.
Решение: Целевая retention strategy: `users` и корневые `documents` не удаляются физически на уровне приложения; для пользователей используется deactivate/reactivate, для документов физический delete не является штатной операцией. FK журналов должны защищать retention: заменить cascade на `RESTRICT`/эквивалентный retention-safe вариант. Если в будущем потребуется физическое удаление, оно должно идти через отдельную архивную/юридическую процедуру, а журналы должны хранить immutable snapshots.
Причина: Lifetime-журналы не должны исчезать из-за случайного delete или служебной операции. Текущее отсутствие UI-delete снижает вероятность, но не является DB-level защитой.
Альтернативы: Оставить `ON DELETE CASCADE` и полагаться на отсутствие application delete; заменить на `SET NULL` с денормализованными именами. Первый вариант слабее для retention; второй потребует nullable FK и snapshot-полей, что можно рассмотреть при появлении legal delete requirements.
Последствия: Нужна миграция изменения FK для `document_journal` и `admin_audit_log`, проверка деактивации пользователей, документных операций, rollback migrations и отчетов/журналов.
Какие этапы затрагивает: B, C, H

## DECISION-006

Дата: 2026-05-27
Контекст: `restore_smb_tar.sh` восстанавливает PostgreSQL, затем MinIO, но сейчас продолжает workflow после любого ненулевого `pg_restore`, считая это обычно некритичными предупреждениями. При этом rollback миграций через UI/runtime остается доступен в production по `DECISION-003`, поэтому backup/restore является обязательной страховкой, но не заменой механизма миграций.
Решение: Restore workflow должен быть fail-fast для неизвестных и fatal ошибок PostgreSQL restore. Ненулевой `pg_restore` нельзя blanket-классифицировать как warning: допустимые warning-классы должны быть явно перечислены, иначе выполнение останавливается до MinIO mirror. Целевой restore включает preflight archive/container checks, `pg_restore` с режимом остановки на ошибках, restore report, post-restore DB smoke validation и проверку согласованности PostgreSQL+MinIO.
Причина: Неполный restore БД опаснее явной остановки: MinIO может быть синхронизирован поверх некорректной БД, после чего оператор получит ложное ощущение успешного восстановления.
Альтернативы: Оставить текущую эвристику "любой nonzero = warning"; всегда восстанавливать MinIO независимо от результата БД. Эти варианты отклонены, потому что скрывают критические ошибки восстановления.
Последствия: Нужно доработать `restore_smb_tar.sh` и runbook: проверять наличие `database.dump` и `minio_files/`, останавливать workflow при fatal/unknown `pg_restore`, логировать restore report, выполнять smoke queries (`schema_migrations`, ключевые таблицы, базовая целостность), отдельно проверять controlled failure на поврежденном или несовместимом dump.
Какие этапы затрагивает: B, E, H

## DECISION-007

Дата: 2026-05-27
Контекст: Этап C подтвердил, что Wails `ErrorFormatter` возвращает frontend plain string, а backend смешивает `models.AppError` и `fmt.Errorf`. Для production UI, тестов и будущей idempotency/conflict обработки frontend не должен зависеть от текстов PostgreSQL, storage или Go errors.
Решение: Целевой backend/Wails error contract — structured envelope `{code, message, details?}`. `code` является стабильным domain code (`UNAUTHORIZED`, `FORBIDDEN`, `VALIDATION_ERROR`, `NOT_FOUND`, `CONFLICT`, `IDEMPOTENCY_CONFLICT`, `STORAGE_ERROR`, `INTERNAL_ERROR`), `message` безопасен для пользователя, `details` не содержит секретов/PII и по умолчанию не нужен production UI.
Причина: Stable machine-readable codes нужны для frontend behavior, smoke tests и безопасного показа ошибок пользователю.
Альтернативы: Оставить plain strings; использовать только integer HTTP-like code. Эти варианты недостаточны для desktop Wails contract и доменных конфликтов.
Последствия: Нужно изменить `models.AppError`, Wails `ErrorFormatter`, service boundary mapping и frontend error handling. Existing UI messages нужно перепроверить на D/H.
Какие этапы затрагивает: C, D, F, H

## DECISION-008

Дата: 2026-05-27
Контекст: Этап C выявил широкое использование `context.Background()` в MinIO/file/link/journal/statistics paths. Wails shutdown закрывает DB/logger, но не отменяет активные операции и не координирует long-running requests.
Решение: Целевая lifecycle model — app root context из Wails startup/shutdown, request-level context/timeout для долгих операций и проброс context через service/repository/storage interfaces там, где операция может блокироваться.
Причина: Desktop-приложение должно предсказуемо завершать upload/download/list/query operations при закрытии окна или потере backend resource.
Альтернативы: Оставить `context.Background()` и полагаться на быстрые операции; добавлять timeout только в MinIO client. Эти варианты не закрывают DB/journal/link/statistics paths.
Последствия: Понадобится изменение интерфейсов, tests на cancellation и careful frontend smoke для file operations, statistics и document registration.
Какие этапы затрагивает: C, E, F, G

## DECISION-009

Дата: 2026-05-28
Контекст: Этап D подтвердил, что backend structured error envelope уже введен на C, но frontend по-прежнему использует `err?.message || String(err)` и auth lockout через поиск текста. Это сохраняет зависимость UI от raw strings.
Решение: Целевой frontend error handling — единый adapter/hook для Wails errors. Он читает stable `code`, safe `message`, optional `status/details`, возвращает пользовательский текст и позволяет UI выбирать поведение для validation, forbidden, not_found, conflict/idempotency и internal errors. Raw string остается только fallback для unexpected client-side errors.
Причина: Frontend должен быть устойчив к изменению текста backend errors и не показывать internal DB/storage details как контракт.
Альтернативы: Оставить локальные `message.error(err?.message || String(err))` в каждом компоненте; разбирать русские тексты ошибок в конкретных flows. Эти варианты хрупкие и плохо тестируются.
Последствия: Нужно обновить auth store, page/component catch handlers, form validation mapping и smoke tests для основных error classes.
Какие этапы затрагивает: D, F, H

## DECISION-010

Дата: 2026-05-28
Контекст: Этап E выявил разрыв между About/release notes version and Wails/binary/installer metadata. Также production build зависит от локального `.env` и не имеет единого release gate.
Решение: Целевая release model — один version source для release notes, About UI, Wails product metadata, binary properties and installer DisplayVersion. Release build выполняется через единый script/target, который валидирует required secrets, использует deterministic dependency install, генерирует release assets, запускает tests/build and verifies artifact metadata.
Причина: Production support and update diagnostics need one trustworthy version and reproducible artifact pipeline.
Альтернативы: Оставить version in release notes only; вручную сверять installer/binary metadata. Эти варианты плохо масштабируются и создают риск неверной сборки.
Последствия: Нужно выбрать source of truth, обновить Wails config/build scripts, добавить release checklist metadata check.
Какие этапы затрагивает: E, H

## DECISION-011

Дата: 2026-05-28
Контекст: Runtime config сейчас ищется как `config/config.json` относительно cwd, но production запускается через installer/shortcut/portable binary and config is managed manually.
Решение: Требуется явно выбрать production config placement policy: executable-relative config for managed portable install, OS system config dir, or user config dir. Lookup order and failure UX must be documented and tested on target OS. После remediation lookup order выбран: `DOCFLOW_CONFIG_PATH`, executable-relative config, затем cwd fallback for local development; fatal pre-UI failure diagnostics fixed by `ISSUE-028`.
Причина: Desktop app should start predictably regardless of shortcut working directory and should show actionable diagnostics when config is missing.
Альтернативы: Оставить cwd-relative config and rely on operator discipline. Этот вариант хрупкий для стандартной установки.
Последствия: После выбора политики нужно изменить startup lookup or installer/runbook, добавить smoke tests for missing/invalid/unreadable config.
Какие этапы затрагивает: E, H

## DECISION-012

Дата: 2026-05-28
Контекст: Этап E подтвердил отсутствие downgrade guard and unclear migration timing policy for updates. Runtime rollback remains required by `DECISION-003`.
Решение: Target update policy must include schema compatibility guard: binary must refuse to run unsafe operations if DB schema version is newer than embedded migrations or dirty. Migration execution policy must be explicit: auto-migrate on startup or admin-run before use, with backup/runbook requirements.
Причина: Running older code against newer schema is a high-risk update/downgrade failure mode.
Альтернативы: Полагаться на operator discipline and migration status page. Это недостаточно для production crash/update safety.
Последствия: Нужно добавить compatibility check, dirty-state UX/runbook and update/downgrade smoke tests.
Какие этапы затрагивает: E, H

## DECISION-013

Дата: 2026-05-28
Контекст: Этап F выявил reachable Go vulnerabilities through `govulncheck` and absence of automated security/license gates.
Решение: Production release gate must include dependency vulnerability and license checks: `govulncheck ./...`, `npm audit --audit-level=critical`, Go/npm license inventory, plus existing `go test`, `go vet`, `npm run build`. A release with reachable critical/high dependency vulnerability is blocked unless a documented exception is approved.
Причина: Manual one-off checks do not protect future releases; desktop distribution needs repeatable security and license evidence.
Альтернативы: Проверять зависимости вручную перед релизом. Это оставляет высокий риск пропуска.
Последствия: Нужно добавить scripts/checklist and define vulnerability/license exception process.
Какие этапы затрагивает: F, H

## DECISION-014

Дата: 2026-05-28
Контекст: Frontend TypeScript build passes, but ESLint/Prettier are absent and `@ts-ignore` is used around Wails/generated service boundaries.
Решение: Adopt a minimal frontend static-analysis gate before broad frontend remediation: TypeScript build remains mandatory, ESLint starts with high-signal React/TypeScript rules, and suppressions must use `@ts-expect-error` with a reason when unavoidable.
Причина: A noisy all-at-once lint migration would distract from production blockers, but no lint gate leaves contract and hooks regressions unguarded.
Альтернативы: Add a strict full preset immediately; keep no lint. Strict immediate migration is likely noisy, no lint is too weak.
Последствия: Need incremental lint config and cleanup plan for existing suppressions/`any` usage.
Какие этапы затрагивает: F, H

## DECISION-015

Дата: 2026-05-28
Контекст: Этап G подтвердил strong Go unit coverage and passing PostgreSQL integration tests. After `ISSUE-039`, integration tests are run through `make integration-test` with disposable DB provisioning. After `ISSUE-038`, frontend helper tests and production build asset smoke are release-gated. After `ISSUE-043`, browser/Wails UX safety checklist coverage is release-gated by `make ux-smoke-check`.
Решение: Production release test gate must include Go unit tests, safe disposable PostgreSQL integration tests, frontend type/lint/helper checks, production-build smoke and UX safety checklist validation. Integration tests must refuse unsafe DSNs. Completed target OS browser/Wails UX lifecycle smoke remains release evidence.
Причина: Critical no-gaps/idempotency and retention invariants must run in release gate, while frontend remediation needs UI-level protection.
Альтернативы: Keep manual testing only; rely on `go test ./...`. These miss gated DB tests and all frontend/e2e behavior.
Последствия: Need release test script, safe test DB provisioning, frontend helper/build smoke tooling, maintained UX safety checklist and manual target OS UX smoke evidence.
Какие этапы затрагивает: G, H, I

## DECISION-016

Дата: 2026-05-28
Контекст: Production SLO is documented, and DB synthetic EXPLAIN baseline exists. After `ISSUE-041`, static bundle/binary baseline generation is release-gated; target Wails startup/UI/memory measurements are captured as manual release evidence fields.
Решение: Establish performance baseline as release evidence: startup to login, login/dashboard, list open/search/filter, registration save, statistics/report open, memory after repeated usage, bundle/binary size.
Причина: Without measured baseline, SLO remains aspirational and regressions from remediation cannot be evaluated.
Альтернативы: Use DB EXPLAIN only or subjective manual checks. These do not cover desktop UI/runtime behavior.
Последствия: Keep `make performance-baseline` in release gate and fill Linux/Windows timing fields on production build with production-like data.
Какие этапы затрагивает: G, I

## DECISION-017

Дата: 2026-05-28
Контекст: Этап H выявил mixed terms: `вид документа`/`тип документа`, `дело`/`номенклатура`, multiple meanings of `исполнитель`, and abbreviations without explanation.
Решение: Adopt UX terminology rules in `TERMS_GLOSSARY.md`: `вид документа` for document family, `тип документа` for document type value, `дело` as primary user-facing term for nomenclature item, qualified executor labels by context, and expanded abbreviations/tooltips for `ПОС`, `Рег. №`, `Проср.`.
Причина: Consistent terminology reduces wrong data entry and support ambiguity.
Альтернативы: Keep current mixed terms. This preserves existing habit but leaves confusion.
Последствия: UI text changes need form/list/card smoke; audited frontend terminology was applied in `ISSUE-045`, while future business term changes must update `TERMS_GLOSSARY.md`.
Какие этапы затрагивает: H, I

## DECISION-018

Дата: 2026-05-28
Контекст: H confirmed that destructive actions and error messages need stronger, safer wording.
Решение: User-facing destructive confirmations must name the entity and consequence; user-facing errors must be mapped from structured codes to safe copy with next steps. Raw backend/system text is not acceptable as user microcopy except as a fallback during development.
Причина: Production desktop users need actionable, non-technical messages and clear consequences before destructive actions.
Альтернативы: Continue showing backend messages and generic `Удалить?`. This is faster but unsafe.
Последствия: Error adapter, destructive modals and smoke/e2e tests must be updated together.
Какие этапы затрагивает: H, I

## DECISION-019

Дата: 2026-05-28
Контекст: Этап I проверил документацию, release checklist, smoke-test, known issues and final production candidate readiness. На момент решения были открыты critical issues: destructive rollback guardrails, reachable Go vulnerabilities, missing release-grade root docs/runbooks and dirty worktree. После remediation fixed: rollback guardrails, release-grade root docs/runbooks and dirty-worktree blocker.
Решение: Финальный статус текущего production candidate — `not_ready`.
Причина: Нельзя назвать candidate готовым при unresolved critical issues and missing reproducible release evidence.
Альтернативы: `ready_with_risks` was rejected because it would require no open critical blockers and explicit acceptance for remaining major issues.
Последствия: Следующий проход должен закрыть remaining major blockers, выполнить clean-clone release gate, target OS smoke and manual PostgreSQL+MinIO test restore, затем переоценить candidate.
Какие этапы затрагивает: I
