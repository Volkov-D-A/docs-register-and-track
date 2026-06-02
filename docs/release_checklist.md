# Release Checklist

Дата обновления: 2026-06-02
Статус: maintained release checklist

This checklist must be completed from a clean checkout for every production candidate. It is not a release approval while critical blockers remain open in `audit/REVIEW_LOG.md`.

## Stop Gates

- [ ] No open critical issues in `audit/REVIEW_LOG.md`.
- [ ] Release candidate worktree is clean: `git status --short` has no output.
- [ ] Target version is selected in `docs/releases.yaml`.
- [ ] `internal/releaseassets/current_release.yaml` and `wails.json` metadata are fresh.
- [ ] `docs/known_issues.md` matches accepted open issues and owners.

## Clean Environment

- [ ] Build performed from clean clone or fresh checkout.
- [ ] Required tools verified: Go, Wails CLI v2, Node/npm, PostgreSQL client tools, MinIO client if needed, Docker/Compose where needed, `govulncheck`.
- [ ] Required secrets are supplied through approved release path and are not committed.
- [ ] `ENCRYPTION_KEY` is present for production build targets.
- [ ] `.env`, `config.json` and CIFS credentials permissions follow `docs/secret_policy.md`.

## Automated Gate

- [ ] `make release-gate` completed.
- [ ] Go tests completed.
- [ ] Disposable PostgreSQL integration tests completed through `make integration-test`.
- [ ] Go vet completed.
- [ ] `govulncheck` completed with no reachable vulnerabilities.
- [ ] `npm ci`, frontend lint, frontend tests, frontend build, production build smoke, UX safety smoke checklist validation, long-running smoke checklist validation, DB performance evidence checklist validation and `npm audit --audit-level=critical` completed.
- [ ] `build/release-evidence/PERFORMANCE_BASELINE.md` generated and target OS manual timings filled or explicitly accepted.
- [ ] License report and dependency inventories generated under `build/release-evidence/`.

## Build Artifacts

- [ ] Linux artifact built if included in release.
- [ ] Windows artifact/installer built if included in release.
- [ ] Artifact names and checksums recorded.
- [ ] About UI, binary metadata, installer metadata and release notes show the same version.

## Database And Migrations

- [ ] Fresh DB migration smoke completed on disposable PostgreSQL/MinIO.
- [ ] Forward migration from previous release tested where applicable.
- [ ] Dirty migration state runbook tested on disposable DB.
- [ ] Newer-schema/downgrade guard tested.
- [ ] Rollback smoke follows `docs/migration_rollback_runbook.md`.
- [ ] `make db-performance-check` completed.
- [ ] Production-like DB query plans from `docs/db_performance_evidence.md` attached or explicitly accepted.

## Backup And Restore

- [ ] Backup cron path and `.env` path match production documentation.
- [ ] SMB credentials handling approved; no password leakage in process list/logs.
- [ ] Release evidence and technical logs checked for passwords, tokens and full encrypted secret material.
- [ ] Manual PostgreSQL+MinIO test restore completed from actual backup archive.
- [ ] Restore report stored with release evidence.
- [ ] RPO/RTO/retention confirmed: RPO 1 day, RTO 1-2 days, retention 15 days.

## Target OS Smoke

- [ ] Linux artifact smoke completed if Linux is included.
- [ ] Windows artifact/installer smoke completed if Windows is included.
- [ ] Windows installer elevation and ordinary-user app launch verified according to `docs/install_policy.md`.
- [ ] Launch from target install path, shortcut/default cwd and path with spaces/Cyrillic verified.
- [ ] Missing/invalid config diagnostics verified.
- [ ] DB/MinIO/Seq unavailable diagnostics verified.
- [ ] Minimal smoke test in `docs/smoke_test.md` completed.

## UX And Safety Acceptance

- [ ] `make ux-smoke-check` completed.
- [ ] Dedicated UX safety smoke in `docs/ux_safety_smoke.md` completed on target OS artifact or accepted in `docs/known_issues.md`.
- [ ] Structured error behavior smoke completed for validation/forbidden/not found/conflict/internal cases.
- [ ] Dirty forms, destructive confirmations, repeat submit guards, empty states and terminology smoke completed.

## Long Running Acceptance

- [ ] `make long-running-smoke-check` completed.
- [ ] Dedicated long-running smoke in `docs/long_running_smoke.md` completed on target OS artifact or accepted in `docs/known_issues.md`.
- [ ] Memory checkpoints, shutdown cancellation and DB/MinIO outage recovery evidence attached.

## Final Evidence

- [ ] Completed checklist attached to release evidence.
- [ ] Command output summaries attached.
- [ ] Target OS smoke result attached.
- [ ] Long-running smoke result attached.
- [ ] Backup/restore smoke result attached.
- [ ] Performance baseline attached or accepted in `docs/known_issues.md`.
- [ ] DB performance evidence attached or accepted in `docs/known_issues.md`.
- [ ] Clean `git status --short` at tag attached.
