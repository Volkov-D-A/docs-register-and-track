# Documentation Review

Дата: 2026-05-28
Этап: I. Документация, release checklist и финальная готовность
Итог: issue

## Контекст

Проверены `docs/releases.yaml`, `build/README.md`, backup/restore scripts, audit artifacts, `git status --short` и поиск `TODO`/`FIXME`/temporary markers. Корневого `README.md`, пользовательского changelog, production runbook, release checklist, smoke-test и known-issues документа вне audit artifacts в репозитории не найдено.

## I.01.257 TODO/FIXME/temporary

Статус: issue
Severity: major
Release-риск: restore workflow содержит явно временное ослабление strict error handling.
Доказательство: `restore_smb_tar.sh:63` содержит комментарий `Временно отключаем строгую остановку при ошибках`; это совпадает с открытым `ISSUE-010`.
Что нужно сделать до релиза: заменить временную restore-логику на fail-fast policy, restore report и smoke validation.
Можно ли отложить: нет, если restore должен быть release fallback для rollback/migration/storage failures.
Финальная проверка: controlled restore failure должен остановиться до MinIO mirror; успешный restore должен завершаться smoke validation.
Связанные пункты: B.06.075, E.05.180, I.02.268

## I.01.258 README dev environment

Статус: issue
Severity: major
Release-риск: clean clone нельзя воспроизвести без автора проекта или внешней памяти.
Доказательство: корневой `README.md` отсутствует; `build/README.md` описывает только стандартную структуру Wails build directory.
Что нужно сделать до релиза: добавить корневой README с prerequisites, Docker/dev services, config path, frontend/go commands and first-run steps.
Можно ли отложить: можно только если release выполняет автор проекта вручную; для production handover нельзя.
Финальная проверка: новый участник поднимает dev contour по README на clean clone.
Связанные пункты: I.02.265, I.02.266

## I.01.259 README production build

Статус: issue
Severity: major
Release-риск: production artifact может быть собран не тем toolchain, без нужного `ENCRYPTION_KEY`, с устаревшими assets или неверной версией.
Доказательство: нет production build runbook; `ISSUE-023`, `ISSUE-024`, `ISSUE-032`, `ISSUE-033` остаются open.
Что нужно сделать до релиза: описать deterministic release build, required env, version source, frontend asset generation, Wails/Linux/Windows artifact checks.
Можно ли отложить: нет для production candidate.
Финальная проверка: clean-machine release build по инструкции с проходом всех gates.
Связанные пункты: E.01.154-E.01.159, F.01.181, I.02.269

## I.01.260 README migrations

Статус: issue
Severity: major
Release-риск: оператор может применить/откатить миграции без backup, dirty-state runbook или понимания destructive `down`.
Доказательство: rollback UI/runtime остается production requirement; `ISSUE-007` and `ISSUE-027` open; migration runbook отсутствует.
Что нужно сделать до релиза: описать startup/admin migration policy, backup-before-rollback, dirty migration recovery, downgrade/newer-schema guard.
Можно ли отложить: нет, пока rollback доступен в production.
Финальная проверка: migration/rollback smoke on disposable DB and audit log entry.
Связанные пункты: B.01.027, E.03.164-E.03.168, I.02.267

## I.01.261 README backup/restore

Статус: issue
Severity: major
Release-риск: backup может существовать, но restore не подтвержден; неполный restore может быть принят как успешный.
Доказательство: `backup_smb_tar.sh`, `restore_smb_tar.sh` существуют, но runbook отсутствует; `ISSUE-002`, `ISSUE-010`, `ISSUE-031` open.
Что нужно сделать до релиза: оформить RPO/RTO/retention, cron path, SMB credentials policy, test restore steps, restore report and validation queries.
Можно ли отложить: нет для production release.
Финальная проверка: manual test restore PostgreSQL+MinIO из production backup set.
Связанные пункты: B.06.074-B.06.075, E.05.180, I.02.268

## I.01.262 README diagnostics

Статус: issue
Severity: major
Release-риск: missing/invalid config, DB/MinIO/Seq failures and migration failures завершают процесс или показывают недостаточно actionable diagnostics.
Доказательство: root diagnostics guide отсутствует; `ISSUE-025`, `ISSUE-028`, `ISSUE-044` open.
Что нужно сделать до релиза: добавить диагностику типовых ошибок запуска, config lookup, DB/MinIO/Seq connectivity, migration dirty state, restore failures, log locations.
Можно ли отложить: нет для non-author operation.
Финальная проверка: target OS smoke for missing config, invalid config, unavailable DB/MinIO/Seq.
Связанные пункты: E.02.160-E.02.163, H.03.232-H.03.240

## I.02.263 Changelog

Статус: issue
Severity: major
Release-риск: пользователь About/release notes могут расходиться с фактическим production candidate and remediation.
Доказательство: `docs/releases.yaml` latest version is `1.0.4` from 2026-04-27; текущие audit/remediation changes are 2026-05-27/2026-05-28 and not represented as production candidate release notes.
Что нужно сделать до релиза: зафиксировать target version, update changelog/release notes/current release assets, verify About/binary/installer metadata.
Можно ли отложить: нет для final artifact.
Финальная проверка: About UI, binary metadata, installer DisplayVersion and release notes show same version.
Связанные пункты: E.01.154-E.01.159, I.02.269

## I.02.264 Known issues

Статус: issue
Severity: major
Release-риск: open critical/major issues were spread across audit logs and not packaged as release-facing known issues.
Доказательство: no product-level known issues doc existed before this stage; created `audit/08_docs_release/KNOWN_ISSUES.md` as audit artifact.
Что нужно сделать до релиза: turn accepted postponed issues into release notes/known issues with owner, mitigation and acceptance.
Можно ли отложить: no for release decision; product-facing formatting can be refined after blockers are fixed.
Финальная проверка: known issues match `REVIEW_LOG.md` open critical/major state.
Связанные пункты: I.02.265, I.02.270

## I.02.265 Release checklist executable by non-author

Статус: issue
Severity: major
Release-риск: release can depend on tribal knowledge.
Доказательство: no root release checklist or script exists; audit checklist is created in this stage.
Что нужно сделать до релиза: move/convert audit checklist into maintained release runbook/script and validate on clean machine.
Можно ли отложить: нет для handover.
Финальная проверка: non-author follows checklist and produces verified artifacts.
Связанные пункты: I.02.266-I.02.269

## I.02.266 Clean clone build

Статус: needs_info
Severity: major
Release-риск: clean clone build may fail due missing env/toolchain/generated assets or undocumented order.
Доказательство: current repo has dirty worktree and no clean-clone build instruction; earlier gates passed in existing workspace only.
Что нужно сделать до релиза: run clean clone build with documented prerequisites and no hidden local state except approved secrets.
Можно ли отложить: нет for production candidate.
Финальная проверка: `npm ci`, `npm run build`, `go test ./...`, `go vet ./...`, release build on clean checkout.
Связанные пункты: E.01, F.01, I.02.269

## I.02.267 Clean DB

Статус: issue
Severity: major
Release-риск: fresh install and migration policy are not documented end-to-end.
Доказательство: migrations exist and tests pass, but clean DB creation from operator instruction is not documented; rollback/downgrade/dirty guard remains open.
Что нужно сделать до релиза: document DB creation, migration application, initial admin/organization setup, safe integration/test DB separation.
Можно ли отложить: нет.
Финальная проверка: fresh PostgreSQL database migrates and app reaches login/first setup.
Связанные пункты: B.01, G.01, I.02.268

## I.02.268 Minimal smoke after install

Статус: issue
Severity: major
Release-риск: installed artifact may pass build but fail login/document/file/migration/restore workflows.
Доказательство: no root smoke-test doc existed before this stage; created `audit/08_docs_release/SMOKE_TEST.md`.
Что нужно сделать до релиза: run smoke on target OS/build with disposable production-like DB/MinIO/Seq.
Можно ли отложить: нет.
Финальная проверка: smoke checklist completed and attached to release evidence.
Связанные пункты: G.02-G.04, H.03, I.02.269

## I.02.269 Final build passes all automated checks

Статус: issue
Severity: critical
Release-риск: current automated checks include a reachable vulnerability blocker and missing release gates.
Доказательство: `go test ./...`, `go vet ./...`, `npm run build`, `npm audit --audit-level=critical` passed; `govulncheck ./...` reported reachable vulnerabilities in `go1.26.2` and `golang.org/x/net@v0.52.0`.
Что нужно сделать до релиза: upgrade Go to `1.26.3+`, `golang.org/x/net` to `v0.53.0+`, repeat `govulncheck`, then run full release gate.
Можно ли отложить: no, unless a formal security exception is approved.
Финальная проверка: clean `govulncheck ./...` plus release checklist pass.
Связанные пункты: F.01.181, I.02.265

## I.02.270 No uncommitted changes

Статус: issue
Severity: major
Release-риск: final candidate is not reproducible if audit/remediation changes are only in local workspace.
Доказательство: `git status --short` shows modified audit docs, modified backend/frontend files, and untracked audit directories/migrations/repository files.
Что нужно сделать до релиза: review, test, commit/tag release candidate or intentionally discard non-release artifacts.
Можно ли отложить: no for final production candidate.
Финальная проверка: `git status --short` clean at release tag.
Связанные пункты: I.02.266, I.02.269

