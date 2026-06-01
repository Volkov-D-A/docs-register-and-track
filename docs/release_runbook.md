# Release Build Runbook

Дата обновления: 2026-06-01

## Purpose

This runbook defines the maintained release procedure for a non-author operator. It does not override open critical blockers in `audit/08_docs_release/KNOWN_ISSUES.md`.

## Stop Gates

Do not publish a production artifact until all of these are true:

- no open critical issues in `audit/REVIEW_LOG.md`;
- `govulncheck ./...` reports no reachable vulnerabilities;
- release is built from a clean worktree;
- target version is selected and matches About UI, binary metadata, installer metadata and release notes;
- backup/restore and migration smoke evidence is attached to the release.

## Required Tools

- Go toolchain required by `go.mod`;
- Wails CLI v2;
- Node.js and npm compatible with `frontend/package-lock.json`;
- PostgreSQL client tools for smoke and restore validation;
- Docker/Compose for local or disposable service contours;
- MinIO client if backup/restore smoke is executed manually;
- `govulncheck`.

## Clean Checkout

Use a clean clone or a fresh checkout of the candidate commit:

```bash
git status --short
```

Expected output: empty.

If output is not empty, stop and either commit/tag approved changes or remove non-release changes before building.

## Secrets

Production secrets must not be committed. The build reads `ENCRYPTION_KEY` through the approved release environment or `.env` and passes it into Go ldflags. Build targets fail before Wails build if the key is missing.

Before building:

- confirm approved secret delivery path;
- confirm `.env` permissions are limited to the release operator;
- confirm `ENCRYPTION_KEY` is present;
- confirm `config.json` and CIFS credentials file permissions follow `docs/secret_policy.md`;
- record key rotation owner and procedure.

## Build Gate

Run the maintained release gate:

```bash
make release-gate
```

The gate verifies `ENCRYPTION_KEY`, generated release asset freshness, Go tests, Go vet, `govulncheck`, `npm ci`, frontend lint/build, `npm audit --audit-level=critical`, npm GPL-family license check and dependency inventory generation.

Expanded manual steps are listed below for troubleshooting.

Run frontend install and build:

```bash
cd frontend
npm ci
npm run build
cd ..
```

Generate release assets:

```bash
make release-assets
```

`docs/releases.yaml` is the release version source. This target updates both `internal/releaseassets/current_release.yaml` for About/release notes and `wails.json` `info.productVersion` for Wails/binary metadata.

Run Go checks:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
go vet ./...
govulncheck ./...
```

Run npm security gate:

```bash
cd frontend
npm audit --audit-level=critical
cd ..
```

`make release-gate` writes dependency inventories to:

```text
build/release-evidence/go-modules.json
build/release-evidence/npm-dependencies.json
build/release-evidence/LICENSE_REPORT.md
```

These files are local release evidence and are ignored by Git. `LICENSE_REPORT.md` must be archived with release evidence and included with release notices/artifacts as applicable. Frontend lint/static-analysis is part of `make release-gate`.

## Build Artifacts

Linux:

```bash
make build-linux
```

Windows:

```bash
make build-windows
```

After build:

- record artifact names and checksums;
- verify About UI, `wails.json` `info.productVersion`, binary properties and installer metadata use the same release version;
- verify generated `internal/releaseassets/current_release.yaml` freshness;
- verify installer metadata if a Windows installer is produced.

## Fresh Database Smoke

On a disposable database:

1. Create a fresh PostgreSQL DB and MinIO bucket.
2. Point `DOCFLOW_CONFIG_PATH` or executable-relative `config/config.json` to the disposable services.
3. Start the app.
4. Apply embedded migrations through `Settings -> Migrations` or first-run setup, according to the accepted release policy.
5. Confirm migration status is not dirty.
6. Create initial admin if needed.
7. Log in and open dashboard, document lists, settings and files.

## Migration Compatibility Smoke

On disposable DB only:

- current binary against old DB schema should allow migration flow;
- old binary against newer DB schema must be blocked with a clear conflict;
- dirty migration state must block unsafe use and direct operator to recovery.

Rollback smoke must follow `docs/migration_rollback_runbook.md`.

## Backup And Restore Smoke

Follow `docs/backup_restore_runbook.md`.

Release evidence must include:

- successful PostgreSQL+MinIO test restore;
- restore report;
- controlled fatal PostgreSQL restore failure that stops before MinIO mirror;
- confirmation that SMB password is not visible in process arguments.
- confirmation that release evidence and technical logs do not contain passwords, tokens or full encrypted secret material.

## Target OS Install Smoke

For every target OS/artifact:

- install or unpack artifact using the approved install policy in `docs/install_policy.md`;
- on Windows, confirm installer elevation is required for the per-machine install and the installed app runs for an ordinary user without elevation;
- launch from default shortcut/cwd;
- launch from a path with spaces and Cyrillic characters;
- verify missing/invalid config diagnostics;
- verify DB/MinIO/Seq unavailable diagnostics;
- complete the minimal smoke test in `docs/smoke_test.md`.

## Release Evidence

Attach to the release:

- completed checklist from `docs/release_checklist.md`;
- command output summaries;
- artifact checksums;
- target OS smoke result;
- backup/restore smoke result;
- [known issues](known_issues.md) with mitigation/owner/acceptance;
- clean `git status --short` at tag.
