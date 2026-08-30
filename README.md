# Docflow

Wails desktop application for registering and tracking documents. The app uses:

- Go backend with Wails bindings;
- React + Ant Design frontend;
- PostgreSQL for relational data;
- MinIO for attachments;
- Seq for technical logs.

`docflow-server` owns transactional outbox consumption, database migrations and
desktop authentication. Desktop access to PostgreSQL and MinIO remains only for
business scenarios that have not yet moved to HTTP API.

The repository keeps application code and a compact maintained documentation set. Review findings and their current status are tracked in [`docs/bugs.md`](docs/bugs.md); production readiness is determined by the release gate plus environment-specific smoke and recovery checks.

## Local Development

Prerequisites:

- Go version required by `go.mod`;
- Node.js and npm compatible with `frontend/package-lock.json`;
- Wails CLI v2;
- Docker Compose for local PostgreSQL, MinIO, Seq, `docflow-server` and Caddy;
- Linux WebKit dependencies required by Wails on the target developer OS.

Start local infrastructure:

```bash
cp .envExample .env
cp config.example.json config/config.json
make storage-up
```

`docker-compose.yaml`, `.envExample` and `config.example.json` are local development examples only. Do not use their localhost endpoints, disabled TLS settings or example secrets as production defaults.

Set the immutable `DOCFLOW_SERVER_VERSION` in `.env` next to the versions of
PostgreSQL, MinIO, Seq and Caddy. Compose always pulls
`hehelf/docflow-service:<version>` from Docker Hub and waits for PostgreSQL
readiness before starting it; it never builds the server image on the target
host. On a genuinely empty database the server applies its embedded bootstrap
migrations itself. For an existing outdated database it remains alive in
maintenance mode; the outbox worker starts only after an administrator applies
the migrations from the desktop UI. Start and inspect it with:

```bash
docker compose pull docflow-server
docker compose up -d
docker compose ps
docker compose logs -f docflow-server
```

The container health check uses `GET /health/live`, so maintenance is not
treated as a process crash. Operational readiness is available separately at
`GET /health/ready` and returns HTTP 503 until the schema and dependencies are
ready.

Authentication uses `POST /api/v1/auth/login`, opaque bearer sessions and
`GET /api/v1/auth/me`. Only a SHA-256 token hash is stored in PostgreSQL; the
raw token remains in desktop process memory and is revoked by
`POST /api/v1/auth/logout`. Session lifetime is configured with
`DOCFLOW_AUTH_SESSION_TTL_HOURS` (12 hours by default).

For an existing schema, deploy the new server first and apply migration 14
(`server_sessions`) through the current migration UI. Only then deploy the
desktop build that requires server login. No direct-login fallback is enabled
in the production composition root.

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

Build and inspect the standalone outbox service:

```bash
cp .envExample .env
make build-server
make run-server
```

Fill the PostgreSQL, MinIO, Seq and outbox values before starting the service.
The same database and MinIO credentials are used both to initialize the local
containers and to connect `docflow-server`; no duplicate service credentials
are required. The example contains placeholders and is not a production
secret-delivery mechanism. The server reads its configuration exclusively from
environment variables. Desktop builds continue to enqueue transactional events
but never run an outbox consumer; `docflow-server` is required to deliver them.

Build the server container locally:

```bash
make docker-server-build
```

The resulting tag is `docflow-server:<productVersion>` by default. Publish an
immutable version tag to Docker Hub after authenticating locally:

```bash
docker login
make docker-server-push
```

The Docker Hub repository is fixed as `hehelf/docflow-service`; Makefile and
Compose read `DOCFLOW_SERVER_VERSION` from `.env`. Keep it synchronized with the
product version when publishing a release. The Makefile never accepts or stores
a Docker Hub password/token. The runtime image contains only the static server
binary. Pass `.env` with `--env-file` or the equivalent orchestrator mechanism.
Provide `ENCRYPTION_KEY` when a secret uses the `ENC:` format. Production secret
delivery must be verified on the target host.

Run automated checks:

```bash
make release-assets
GOCACHE=/tmp/go-build-cache go test ./...
cd frontend
npm run build
```

## Desktop Configuration

The desktop application loads config in this order:

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
- [Server service implementation plan](docs/server-service-implementation-plan.md)
- [Review findings and fixes](docs/bugs.md)
- [Setup and backup/restore instructions](docs/instructions.md)
- [Release notes source](docs/releases.yaml)

Historical one-time analysis documents are kept separately:

- [Archived performance audit plan](docs/archive/performance-audit-plan.md)

Release must be performed from a clean worktree and must not rely on hidden local state except approved secret injection.

Minimum automated gate before producing artifacts:

```bash
make release-gate
```

The gate checks the release environment, generated release asset, internal
documentation links, Wails bindings, Go unit and PostgreSQL integration tests,
Go vet/vulnerability scan, clean frontend dependency installation, frontend
lint/test/build and critical npm vulnerabilities. Docker with Compose and
`POSTGRES_VERSION` are required for the disposable integration database. DB
performance checks, target-OS smoke and backup restore remain separate checks
described in the technical reference.

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

Migrations are embedded in `docflow-server` from
`internal/database/migrations`. Admin users inspect, apply and roll back them in
`Settings -> Migrations`, but the desktop process never executes migration SQL:
it calls the server management API. Apply/rollback temporarily stops the
server worker, takes a PostgreSQL advisory lease, changes the schema, records an
administrative audit event and then reconciles the worker lifecycle without a
container restart.

The desktop configuration must contain the service endpoint:

```json
"server": {
  "url": "https://docflow.example.internal"
}
```

Compose includes Caddy and currently publishes a temporary HTTP endpoint on
`${DOCFLOW_SERVER_PORT:-8080}`. `docflow-server:8080` itself is available only
inside the Compose network. To use that endpoint from another workstation,
configure the desktop explicitly:

```json
"server": {
  "url": "http://docflow-server-address:8080",
  "allowInsecureHttp": true
}
```

The opt-in is deliberately disabled by default because administrator
credentials are sent for apply/rollback. Temporary HTTP is suitable only for a
trusted isolated network. After a certificate is installed, configure Caddy on
443, change the URL to `https://...` and remove `allowInsecureHttp` (or set it to
false). Credentials are never put in JSON request bodies, logs or
configuration.

Safety rules:

- create a fresh PostgreSQL+MinIO backup before migration rollback;
- enter the current administrator password for every schema-changing command;
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
