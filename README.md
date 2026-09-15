# Docflow

Wails desktop application for registering and tracking documents. The app uses:

- Go backend with Wails bindings;
- React + Ant Design frontend;
- PostgreSQL for relational data;
- SeaweedFS for attachments;
- Seq for technical logs.

`docflow-server` owns PostgreSQL, SeaweedFS, transactional outbox consumption,
database migrations, authentication and all business operations. The desktop
uses the versioned HTTP API and does not create database or object-storage
connections.

The repository keeps application code and a compact maintained documentation set. Production readiness is determined by the release gate plus environment-specific smoke and recovery checks.

## Local Development

Prerequisites:

- Go version required by `go.mod`;
- Node.js and npm compatible with `frontend/package-lock.json`;
- Wails CLI v2;
- Docker Compose for local PostgreSQL, SeaweedFS, Seq, `docflow-server` and Caddy;
- Linux WebKit dependencies required by Wails on the target developer OS.

Start local infrastructure:

```bash
cp docs/examples/.envExample .env
cp docs/examples/config.example.json config/config.json
# Prepare secret files and set their paths in .env (see below).
make storage-up
```

`docker-compose.yaml`, `docs/examples/.envExample` and `docs/examples/config.example.json` are local development examples only. Infrastructure credentials belong in the server environment; desktop `config.json` contains only the server connection settings and no PostgreSQL, SeaweedFS or Seq credentials/endpoints. Authenticated desktop technical logs are sent in bounded batches to `POST /api/v1/telemetry/logs`; `docflow-server` adds the session identity and forwards them through its logging pipeline to Seq.

The root Compose file builds `docflow-service-local:dev` from local sources via
`make storage-up` or `make dev-server`. The production example pulls
`hehelf/docflow-service:<DOCFLOW_SERVER_IMAGE_TAG>` from Docker Hub (default: `latest`); optionally pin a published tag in the production `.env`.
Both wait for PostgreSQL and the S3 probe and include backup storage.
Keep settings and usernames (`POSTGRES_USER`, `S3_ACCESS_KEY_ID`,
`SEQ_ADMIN_USERNAME`) in `.env` beside Compose. Before starting, create four
secret files outside Git, for example in `/etc/docflow/secrets`. Each file
contains only its value, without `NAME=` or quotes:

| Variable in `.env` | File contents |
| --- | --- |
| `POSTGRES_PASSWORD_FILE_PATH` | PostgreSQL password |
| `SEQ_ADMIN_PASSWORD_FILE_PATH` | Initial Seq administrator password |
| `S3_SECRET_KEY_FILE_PATH` | S3 secret access key |
| `DOCFLOW_SETTINGS_KEY_PATH` | Settings encryption key |

Create the encryption key as described in [backup setup](docs/instructions.md#серверное-резервирование-и-восстановление).
Use a protected host directory (mode `0700`). The PostgreSQL password, S3 secret
and settings key must be readable by server UID 65532 (owner 65532, mode `0400`).
The supplied Seq image starts as root; its password file can be owned by root
with mode `0400`. Compose file secrets preserve host file permissions.
Set the four absolute file paths in `.env`; Compose mounts only the required
secrets into each service. Keep existing passwords and the settings key when
migrating an existing deployment. Changing these files does not change an
existing PostgreSQL role password or Seq account password; Seq uses this password
only for initialization ([Seq configuration](https://docs.datalust.co/docs/environment-variables)).
The server image must include support for `POSTGRES_PASSWORD_FILE`; build and
publish the updated server and set its version before deploying this Compose.
The only environment difference is Caddy: dev exposes HTTP on
`DOCFLOW_SERVER_PORT` (default 8080); production exposes HTTPS on port 443 using
`./caddy/Caddyfile` and persistent certificate storage. Provide that production
Caddyfile on the host and create the external `docflow-caddy-data` volume before
starting production. Preserve existing certificate storage when upgrading.
Copy [seaweedfs-entrypoint.sh](scripts/seaweedfs-entrypoint.sh) and
[seq-entrypoint.sh](scripts/seq-entrypoint.sh) to `scripts/` beside the deployed
Compose file. Both environments mount these startup scripts read-only and run
them with `/bin/sh`; preserve LF line endings. When running the production
example directly from this repository, use `--project-directory .` from the
repository root so that `scripts/` and `caddy/` resolve correctly.
On a genuinely empty database the server applies its embedded bootstrap
migrations itself. For an existing outdated database it remains alive in
maintenance mode; the outbox worker starts only after an administrator applies
the migrations from the desktop UI. Start and inspect it with:

```bash
make storage-up
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

For an empty database the desktop checks `GET /api/v1/auth/setup-required` and
creates the first administrator through the one-time
`POST /api/v1/auth/setup` command. The desktop never connects to PostgreSQL for
bootstrap.

Password changes also go through the service:
`POST /api/v1/auth/change-password` requires the current bearer session, while
`POST /api/v1/auth/change-required-password` accepts the credentials that were
rejected by login due to the password-change requirement. A successful password
change, an administrator password reset, or user deactivation revokes every
server session for that user. After changing their own password, the desktop
returns the user to login.

An administrator does not choose a replacement password: reset generates a
random temporary password, returns it once for copying, and marks it for a
mandatory change at the user's next login.

This development baseline expects a fresh database: the migration history was
compacted and ends with migration 11 (`server_sessions`). No direct-login
fallback is enabled in the production composition root.

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
make dev-client
```

The local `docflow-server` is started by `make storage-up` as part of the
Compose stack; a second standalone server process is not required. Fill the
PostgreSQL, SeaweedFS, Seq and outbox values before starting the stack.
The same database and SeaweedFS credentials are used both to initialize the local
containers and to connect `docflow-server`; no duplicate service credentials
are required. The example contains placeholders and is not a production
secret-delivery mechanism. The server reads its configuration exclusively from
environment variables. Desktop builds neither write transactional events
directly nor run an outbox consumer; `docflow-server` owns both production and
delivery.

Build and publish an immutable server image to Docker Hub after authenticating
locally:

```bash
docker login
make docker-server-push
```

The Docker Hub repository is fixed as `hehelf/docflow-service`; Makefile and
Compose read `DOCFLOW_SERVER_VERSION` from `.env`. Before building, the target
checks that this version matches the generated release asset and Wails product
version. Images receive immutable tags `<release>.<commit-count>-<full-SHA>`;
the same image is then published as `latest`. Production Compose uses `latest`
by default; set `DOCFLOW_SERVER_IMAGE_TAG` to a full tag to pin a build.
On the server, run `docker compose up -d --pull always docflow-server` to update.
A running container does not update itself when an image is published.
The desktop still needs to match the deployed server build. The Makefile never accepts or stores a Docker Hub password/token. The
runtime image contains only the static server binary. Pass `.env` with
`--env-file` or the equivalent orchestrator mechanism.
PostgreSQL and SeaweedFS passwords must be supplied as runtime secrets; production
secret delivery must be verified on the target host.

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

The desktop configuration contains no infrastructure credentials. Do not commit production secrets.

## Release And Operations

Maintained project documentation:

- [Technical reference](docs/tech_docs.md)
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
documentation links, Go unit and PostgreSQL integration tests,
Go vet/vulnerability scan, clean frontend dependency installation, frontend
lint/test/build and critical npm vulnerabilities. Docker with Compose and
`POSTGRES_VERSION` are required for the disposable integration database. DB
performance checks, target-OS smoke and backup restore remain separate checks
described in the technical reference.

Test Compose files, helper scripts and fixture images are grouped in
[`testing/`](testing/README.md); existing Make targets remain unchanged.

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
`internal/server/database/migrations`. Admin users inspect, apply and roll back them in
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

- create a fresh PostgreSQL+SeaweedFS backup before migration rollback;
- enter the current administrator password for every schema-changing command;
- never run an older binary against a newer DB schema;
- stop application use if migration status is dirty;
- perform rollback only through the documented confirmation flow.

The backend blocks newer/dirty schema states for login and migration operations. Recovery procedures and target-contour smoke remain release evidence tasks.

## Backup And Restore

Configure direct SMB access and the schedule in Settings → Backups.
See [server backup and recovery](docs/instructions.md#серверное-резервирование-и-восстановление) for Compose
secrets, persistent staging, and restoration through the ordinary admin panel, including after a complete target reset.

Release requires a successful manual test restore of PostgreSQL and SeaweedFS from an actual backup archive or production-like backup set.

## Diagnostics

Operator-facing startup behavior, logging and recovery constraints are described in the [technical reference](docs/tech_docs.md).

SeaweedFS configuration, runtime secrets, dev reset and S3 operations are
covered in the [storage operations guide](docs/instructions.md#эксплуатация-seaweedfs).
Run `make storage-smoke-test` to verify the current server build, restart
persistence and PostgreSQL/S3 restore in disposable volumes.

## Internal build identity

The public release stays three-component (for example `1.0.7`). Desktop and server
also embed the Git commit count and full revision. Connection requires matching
release and source identity; old binaries without this contract are rejected.
Update or roll back client and server together. Internal versions and revisions
are displayed only in system statistics. The server's technical `build-info`
command prints its embedded identity without loading service configuration.

`make build-linux`, `make build-windows` and Docker publication
require committed sources and full Git history. Rebuilding one commit gives the
same identity on all platforms. Do not build distributed binaries using plain
`go build`: without injected metadata they cannot pass compatibility checks.

For local development, run `make dev-server` (or `make storage-up`) to build the
current sources and start/update the whole Compose stack, then `make dev-client`
to launch Wails. No commit, Docker Hub publication or manual image tag in `.env`
is needed. Existing secrets and service configuration are still required.
After edits, rebuild the server and restart/rebuild the client from the same
source state. The client command does not restart containers.

`make storage-down` stops the whole stack and preserves data. `make storage-reset`
deletes all named Compose volumes (including PostgreSQL, SeaweedFS, Seq and backup
staging) and starts the stack again. All storage commands use the root Compose
file. If sources change during compilation, the server update is cancelled.
Use the Make targets for rebuilding: plain `docker compose build` lacks the
source metadata supplied by the wrapper. The production Compose example remains
configured to pull published images from Docker Hub.

For local packaged desktop builds, prefix the build command with
`DOCFLOW_LOCAL_BUILD=1`. Local builds compare a normalized source fingerprint in
addition to the commit; rebuild both sides after edits. Wails uses the compiler
wrapper to recalculate identity on each Go rebuild. Frontend hot reload does not
recompile the embedded Go identity; restart/rebuild both sides before checking a
changed frontend snapshot. Publication always requires clean sources.

The fingerprint includes tracked and new nonignored files, with text CRLF
normalized to LF. Generated Wails bindings, temporary Windows resources, build
output, local configuration and secrets are excluded centrally in
`tools/buildmeta/main.go`. Git-ignored source files cannot participate: add build
inputs to Git. Even documentation commits change the strict source identity.
Docker receives validated metadata through `BUILD_IDENTITY` and `VERSION` build
arguments because `.git` is excluded from its context. For an exported source
archive, retain the identity generated from that exact clean Git checkout and
supply these arguments when building; metadata must not be guessed.

## Make command guide

`make` shows the main commands without starting a build:

| Purpose | Commands |
| --- | --- |
| Development | `dev-client`, `dev-server` |
| Whole local Compose stack | `storage-up`, `storage-down`, `storage-reset` |
| Production artifacts | `build-linux`, `build-windows`, `docker-server-push` |
| Main checks | `release-gate` |

The individual gate steps remain available through `build/make/checks.mk`;
`make help-checks` lists them and the additional diagnostic commands. They are
useful for focused checks without rerunning the whole gate. `storage-smoke-test`
checks the real storage/API/backup lifecycle, while `db-performance-check`
measures database performance; neither is part of `release-gate`.
`integration-db-up` / `integration-db-down` support manual integration debugging.
`release-assets` regenerates release metadata; `clean` removes local binaries only.

The redundant `dev`, `dev-local`, `build-server`, `build-server-dev` and
`docker-server-build` targets were removed. Server development uses Compose;
production server compilation is part of `docker-server-push`. Prerequisite
checks and the compiler wrapper use internal targets prefixed with `_`.
