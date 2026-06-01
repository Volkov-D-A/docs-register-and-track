# Release Checklist

Дата: 2026-05-28
Статус текущего candidate: not_ready

Этот checklist нельзя считать разрешением на релиз, пока открыты critical blockers.

## 1. Stop Gates

- [x] `ISSUE-007` закрыт: rollback guardrails implemented and tested for destructive `down` migrations.
- [x] `ISSUE-032` закрыт: Go toolchain `1.26.3+`, `golang.org/x/net v0.53.0+`, `govulncheck ./...` clean.
- [x] `ISSUE-010` закрыт: restore script fail-fast, report and smoke validation implemented.
- [x] `ISSUE-050` закрыт: maintained root README and release/diagnostics runbooks added.
- [ ] Нет open `critical` issues in `audit/REVIEW_LOG.md`.
- [ ] Release candidate worktree clean: `git status --short` has no output.

## 2. Version And Changelog

- [ ] Target version selected.
- [ ] `docs/releases.yaml` and embedded/current release assets updated.
- [ ] About UI, binary metadata and installer metadata show the same version.
- [ ] Release notes include audit/remediation changes and accepted known issues.

## 3. Clean Environment

- [ ] Build performed from clean clone or fresh checkout.
- [ ] Required tools verified: Go, Wails, Node/npm, Docker/Compose where needed, PostgreSQL client tools, MinIO client, `govulncheck`.
- [ ] No hidden local `frontend/dist` or generated file dependency outside documented build steps.
- [ ] Required secrets are supplied through approved release path and are not committed.

## 4. Automated Gate

- [ ] `npm ci` in `frontend`.
- [ ] `npm run build` in `frontend`.
- [ ] `go test ./...`.
- [ ] Disposable PostgreSQL integration tests for document registration/idempotency/concurrency/retention FK.
- [ ] `go vet ./...`.
- [ ] `govulncheck ./...`.
- [ ] `npm audit --audit-level=critical`.
- [ ] `make release-gate`.
- [ ] License inventory generated for Go and npm dependencies.
- [x] Frontend lint/static analysis gate introduced and passes.

## 5. Database And Migrations

- [ ] Fresh DB creation documented and tested.
- [ ] Forward migrations from previous release tested.
- [ ] Dirty migration state runbook tested on disposable DB.
- [ ] Newer-schema/downgrade guard tested.
- [ ] Rollback action requires explicit warning, entity/version, backup reminder/check and admin audit entry.
- [ ] Backup exists before migration/rollback smoke.

## 6. Backup And Restore

- [ ] Backup cron path and `.env` path match production documentation.
- [ ] SMB credentials handling approved; no password leakage in process list/logs.
- [ ] Test restore PostgreSQL+MinIO completed from actual backup archive.
- [ ] Restore stops before MinIO mirror on fatal/unknown `pg_restore` failure.
- [ ] Restore report stored with release evidence.
- [ ] RPO/RTO/retention confirmed: RPO 1 day, RTO 1-2 days, retention 15 days.

## 7. Install And Runtime Smoke

- [ ] Linux artifact smoke.
- [ ] Windows artifact/installer smoke if target release includes Windows.
- [ ] Launch from target install path, path with spaces/Cyrillic and shortcut/default cwd.
- [ ] Missing/invalid config produces actionable diagnostics.
- [ ] DB/MinIO/Seq unavailable scenarios produce actionable diagnostics.
- [ ] Minimal smoke test from `SMOKE_TEST.md` completed.

## 8. UX And Safety Acceptance

- [ ] Frontend uses structured error codes for critical errors.
- [ ] Critical submit actions have local submitting/confirmLoading guards.
- [ ] Dirty document/settings forms warn before closing.
- [ ] Destructive confirmations name entity and consequence.
- [ ] Terminology rules from `TERMS_GLOSSARY.md` applied or accepted as known issue.

## 9. Final Evidence

- [ ] `audit/08_docs_release/KNOWN_ISSUES.md` updated.
- [ ] Open major issues have owner, mitigation and explicit release acceptance/postpone decision.
- [ ] Performance baseline recorded: startup, login/dashboard, list/search, save, statistics, memory.
- [ ] Artifacts checksums and version metadata recorded.
- [ ] Release tag created from clean worktree.
