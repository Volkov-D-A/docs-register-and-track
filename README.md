# Docflow

Wails desktop application for registering and tracking documents. The app uses:

- Go backend with Wails bindings;
- React + Ant Design frontend;
- PostgreSQL for relational data;
- MinIO for attachments;
- Seq for technical logs.

This repository currently contains both application code and audit/release evidence. The current production candidate is not ready until the critical blockers in `audit/08_docs_release/KNOWN_ISSUES.md` are closed or formally accepted.

## Local Development

Prerequisites:

- Go version required by `go.mod`;
- Node.js and npm compatible with `frontend/package-lock.json`;
- Wails CLI v2;
- Docker Compose for local PostgreSQL, MinIO and Seq;
- Linux WebKit dependencies required by Wails on the target developer OS.

Start local infrastructure:

```bash
cp .envExample .env
cp config.example.json config/config.json
make storage-up
```

`docker-compose.yaml`, `.envExample` and `config.example.json` are local development examples only. Do not use their localhost endpoints, disabled TLS settings or example secrets as production defaults.

Install frontend dependencies and build assets:

```bash
cd frontend
npm ci
npm run build
cd ..
make release-assets
```

Run the app in development mode:

```bash
make dev
```

Run automated checks:

```bash
make release-assets
GOCACHE=/tmp/go-build-cache go test ./...
cd frontend
npm run build
```

## Configuration

The current application loads config from:

```text
config/config.json
```

relative to the current working directory. This is a known production risk tracked as `ISSUE-025`; target install smoke must verify launch from shortcut/default cwd, paths with spaces and Cyrillic characters, and missing/invalid config behavior.

Encrypted config values use `ENC:` and require `ENCRYPTION_KEY` to be supplied at build/runtime according to the approved release process. Do not commit production secrets.

## Release And Operations

Maintained runbooks:

- [Release build runbook](docs/release_runbook.md)
- [Migration rollback runbook](docs/migration_rollback_runbook.md)
- [Backup and restore runbook](docs/backup_restore_runbook.md)
- [Diagnostics runbook](docs/diagnostics_runbook.md)

Audit release artifacts:

- [Known issues](audit/08_docs_release/KNOWN_ISSUES.md)
- [Release checklist](audit/08_docs_release/RELEASE_CHECKLIST.md)
- [Smoke test](audit/08_docs_release/SMOKE_TEST.md)
- [Final summary](audit/FINAL_SUMMARY.md)

Release must be performed from a clean worktree and must not rely on hidden local state except approved secret injection.

Minimum automated gate before producing artifacts:

```bash
make release-gate
```

Equivalent expanded commands:

```bash
cd frontend
npm ci
npm run build
cd ..
make release-assets
GOCACHE=/tmp/go-build-cache go test ./...
go vet ./...
govulncheck ./...
cd frontend
npm audit --audit-level=critical
```

`make release-gate` also generates local dependency inventories under `build/release-evidence/`. That directory is ignored by Git and should be attached to release evidence when needed.

## Production Build

Linux test build:

```bash
make build-linux
```

Windows build:

```bash
make build-windows
```

Before release, verify that About UI, release notes, binary metadata and installer metadata refer to the same target version. Version unification is tracked separately as `ISSUE-023`.

## Database And Migrations

Migrations are embedded in the binary from `internal/database/migrations`. Admin users can inspect and run migrations in `Settings -> Migrations`.

Safety rules:

- create a fresh PostgreSQL+MinIO backup before migration rollback;
- never run an older binary against a newer DB schema;
- stop application use if migration status is dirty;
- perform rollback only through the documented confirmation flow.

The backend blocks newer/dirty schema states for login and migration operations. Recovery procedures and target-contour smoke remain release evidence tasks.

## Backup And Restore

Use:

- `backup_smb_tar.sh`
- `restore_smb_tar.sh`
- [docs/backup_restore_runbook.md](docs/backup_restore_runbook.md)

Release requires a successful manual test restore of PostgreSQL and MinIO from an actual backup archive or production-like backup set.

## Diagnostics

For operator-facing startup and runtime failures, use [docs/diagnostics_runbook.md](docs/diagnostics_runbook.md). Several diagnostics improvements remain tracked as open issues; the runbook records current behavior and expected release checks.
