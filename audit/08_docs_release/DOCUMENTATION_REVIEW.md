# Documentation Review

Дата: 2026-05-28
Этап: I. Документация, release checklist и финальная готовность
Итог: issue

## Контекст

Проверены `docs/releases.yaml`, `build/README.md`, backup/restore scripts, maintained docs, audit artifacts, `git status --short` и поиск `TODO`/`FIXME`/temporary markers. После remediation `ISSUE-050` добавлены root `README.md`, `docs/release_runbook.md` and `docs/diagnostics_runbook.md`; release checklist, smoke-test and known-issues remain audit artifacts until `ISSUE-053`.

## I.01.257 TODO/FIXME/temporary

Статус: ok
Severity: none
Release-риск: residual release evidence only; restore workflow no longer has the temporary strict-error bypass noted during audit.
Доказательство: `ISSUE-010` fixed; `restore_smb_tar.sh` uses fail-fast PostgreSQL restore/validation before MinIO mirror and `docs/backup_restore_runbook.md` documents release-gate restore checks.
Что нужно сделать до релиза: выполнить controlled restore failure and successful restore as release evidence.
Можно ли отложить: evidence cannot be skipped for production release.
Финальная проверка: controlled restore failure должен остановиться до MinIO mirror; успешный restore должен завершаться smoke validation.
Связанные пункты: B.06.075, E.05.180, I.02.268

## I.01.258 README dev environment

Статус: ok
Severity: none
Release-риск: residual clean-clone execution evidence remains under `ISSUE-052`/`ISSUE-053`.
Доказательство: root `README.md` added with prerequisites, Docker/dev services, config path, frontend/go commands and first-run steps.
Что нужно сделать до релиза: выполнить clean-clone walkthrough and attach evidence.
Можно ли отложить: можно только если release выполняет автор проекта вручную; для production handover нельзя.
Финальная проверка: новый участник поднимает dev contour по README на clean clone.
Связанные пункты: I.02.265, I.02.266

## I.01.259 README production build

Статус: ok
Severity: none
Release-риск: underlying build/version/security blockers remain tracked separately.
Доказательство: `docs/release_runbook.md` added with deterministic build gate, required env/secret notes, release asset generation and artifact checks. `ISSUE-032` and `ISSUE-033` fixed after remediation; `ISSUE-023`, `ISSUE-024` remain open as implementation/process blockers.
Что нужно сделать до релиза: execute the runbook from clean checkout after underlying blockers are closed.
Можно ли отложить: нет для production candidate.
Финальная проверка: clean-machine release build по инструкции с проходом всех gates.
Связанные пункты: E.01.154-E.01.159, F.01.181, I.02.269

## I.01.260 README migrations

Статус: ok
Severity: none
Release-риск: target-contour migration smoke remains release evidence.
Доказательство: root `README.md`, `docs/release_runbook.md`, `docs/diagnostics_runbook.md` and `docs/migration_rollback_runbook.md` now describe migration policy, backup-before-rollback, dirty state response and newer-schema guard.
Что нужно сделать до релиза: execute migration/rollback/downgrade smoke on disposable DB and attach evidence.
Можно ли отложить: нет, пока rollback доступен в production.
Финальная проверка: migration/rollback smoke on disposable DB and audit log entry.
Связанные пункты: B.01.027, E.03.164-E.03.168, I.02.267

## I.01.261 README backup/restore

Статус: ok
Severity: none
Release-риск: manual restore evidence still required.
Доказательство: `docs/backup_restore_runbook.md` documents RPO/RTO/retention, cron path, SMB credentials policy, test restore steps, restore report and validation queries; root README and release runbook link it.
Что нужно сделать до релиза: perform manual test restore PostgreSQL+MinIO and attach restore report.
Можно ли отложить: нет для production release.
Финальная проверка: manual test restore PostgreSQL+MinIO из production backup set.
Связанные пункты: B.06.074-B.06.075, E.05.180, I.02.268

## I.01.262 README diagnostics

Статус: ok
Severity: none
Release-риск: startup diagnostics UX implementation remains tracked separately.
Доказательство: `docs/diagnostics_runbook.md` added with config lookup, DB/MinIO/Seq connectivity, migration dirty/newer-schema, restore failure and release diagnostics smoke sections. `ISSUE-025`, `ISSUE-028`, `ISSUE-044` remain open as product UX/runtime blockers.
Что нужно сделать до релиза: execute target OS diagnostics smoke and attach evidence.
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

Статус: ok
Severity: none
Release-риск: clean checkout release evidence remains under `ISSUE-052`/`ISSUE-053`, but reachable vulnerability blocker and maintained release gate are fixed.
Доказательство: after remediation `go.mod` requires `go 1.26.3`, `golang.org/x/net@v0.53.0`; `make release-gate` runs `govulncheck`, `go test`, `go vet`, `npm run build`, `npm audit`, npm license check and dependency inventory generation.
Что нужно сделать до релиза: run `make release-gate` from clean checkout and attach evidence.
Можно ли отложить: clean release evidence cannot be skipped for production candidate.
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
