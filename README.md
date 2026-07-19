# Docflow

Wails desktop application for registering and tracking documents. The app uses:

- Go backend with Wails bindings;
- React + Ant Design frontend;
- PostgreSQL for relational data;
- MinIO for attachments;
- Seq for technical logs.

The repository keeps application code and a compact maintained documentation set. Review findings and their current status are tracked in [`docs/bugs.md`](docs/bugs.md); production readiness is determined by the release gate plus environment-specific smoke and recovery checks.

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

The application loads config in this order:

```text
DOCFLOW_CONFIG_PATH
<executable directory>/config/config.json
<current working directory>/config/config.json
```

The current working directory fallback is for local development. Production installs should use `DOCFLOW_CONFIG_PATH` or place the approved config next to the executable under `config/config.json`. Target install smoke must still verify launch from shortcut/default cwd, paths with spaces and Cyrillic characters, and missing/invalid config behavior.

Encrypted config values use `ENC:` and require `ENCRYPTION_KEY` to be supplied at build/runtime according to the approved release process. Do not commit production secrets.

## Release And Operations

Maintained project documentation:

- [Technical reference](docs/tech_docs.md)
- [Review findings and fixes](docs/bugs.md)
- [Setup and backup/restore instructions](docs/instructions.md)
- [Release notes source](docs/releases.yaml)
- [Optimization notes](docs/opti.md)

Release must be performed from a clean worktree and must not rely on hidden local state except approved secret injection.

Minimum automated gate before producing artifacts:

```bash
make release-gate
```

The gate checks the release environment and generated release asset, Go tests/vet/vulnerability scan, clean frontend dependency installation, frontend lint/test/build and critical npm vulnerabilities. Integration tests, DB performance checks, target-OS smoke and backup restore are separate checks described in the technical reference.

## Production Build

Linux test build:

```bash
make build-linux
```

Windows build:

```bash
make build-windows
```

Before release, verify that About UI, release notes, binary metadata and installer metadata refer to the same target version.

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
- [setup and backup/restore instructions](docs/instructions.md)

Release requires a successful manual test restore of PostgreSQL and MinIO from an actual backup archive or production-like backup set.

## Diagnostics

Operator-facing startup behavior, logging and recovery constraints are described in the [technical reference](docs/tech_docs.md). Remaining diagnostics and security debt are tracked in [review findings](docs/bugs.md).
