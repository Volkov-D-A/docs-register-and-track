#!/usr/bin/env bash
# Exercise the production Compose topology with isolated test secrets and ports.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.."
root=$PWD
compose_env=/dev/null
if [[ -f "$root/.env" ]]; then compose_env="$root/.env"; fi
stage=$(mktemp -d /tmp/docflow-prod-smoke.XXXXXXXX)
suffix=${stage##*.}
project="docflow-prod-smoke-${suffix,,}"
evidence="$root/build/transition-evidence/prod-compose-smoke"
mkdir -p "$evidence"

export POSTGRES_VERSION=${POSTGRES_VERSION:-18.3}
export SEQ_VERSION=${SEQ_VERSION:-2025.2.16020-x64}
export CADDY_VERSION=${CADDY_VERSION:-2.11.4-alpine}
export SEAWEEDFS_VERSION=${SEAWEEDFS_VERSION:-4.46}
export POSTGRES_USER=docflow_integration POSTGRES_DB=docflow_test_prod_smoke
export S3_ACCESS_KEY_ID=docflow_integration S3_BUCKET=docflow-test-prod-smoke
export SEQ_ENABLED=false
export POSTGRES_PASSWORD_FILE_PATH="$stage/postgres-password"
export SEQ_ADMIN_PASSWORD_FILE_PATH="$stage/seq-password"
export S3_SECRET_KEY_FILE_PATH="$stage/s3-secret"
export DOCFLOW_SETTINGS_KEY_PATH="$stage/settings-key"
export DOCFLOW_CADDYFILE_PATH="$stage/Caddyfile"
export DOCFLOW_HTTPS_BIND=127.0.0.1:58443
export DOCFLOW_POSTGRES_CONTAINER_NAME="${project}-postgres"
export DOCFLOW_SEQ_CONTAINER_NAME="${project}-seq"
export DOCFLOW_SERVER_CONTAINER_NAME="${project}-server"
export DOCFLOW_CADDY_CONTAINER_NAME="${project}-caddy"
export DOCFLOW_CADDY_DATA_VOLUME="${project}-caddy-data"
export DOCFLOW_SMOKE_VERSION
DOCFLOW_SMOKE_VERSION=$(GOCACHE="${GOCACHE:-/tmp/go-build-cache}" go run ./cmd/docflow-server version)
export DOCFLOW_SMOKE_BUILD_IDENTITY
DOCFLOW_SMOKE_BUILD_IDENTITY=$(DOCFLOW_LOCAL_BUILD=1 GOCACHE="${GOCACHE:-/tmp/go-build-cache}" go run ./tools/buildmeta identity)

printf '%s\n' docflow_integration > "$POSTGRES_PASSWORD_FILE_PATH"
printf '%s\n' integration-seq-password > "$SEQ_ADMIN_PASSWORD_FILE_PATH"
printf '%s\n' docflow_integration_secret > "$S3_SECRET_KEY_FILE_PATH"
python3 -c 'import base64; print(base64.b64encode(bytes(range(32))).decode())' > "$DOCFLOW_SETTINGS_KEY_PATH"
cat > "$DOCFLOW_CADDYFILE_PATH" <<'CADDY'
https://localhost {
    tls internal
    reverse_proxy docflow-server:8080
}
CADDY
chmod 644 "$stage"/*

compose=(docker compose --project-directory "$root" --env-file "$compose_env" -p "$project" -f docs/examples/docker-compose.prod.example.yaml -f testing/compose/prod-smoke.override.yaml)
cleanup() {
  result=$?
  "${compose[@]}" logs --no-color > "$evidence/containers.log" 2>&1 || true
  "${compose[@]}" down -v --remove-orphans || true
  docker volume rm "$DOCFLOW_CADDY_DATA_VOLUME" >/dev/null 2>&1 || true
  rm -rf -- "$stage"
  exit "$result"
}
trap cleanup EXIT

docker volume create "$DOCFLOW_CADDY_DATA_VOLUME" >/dev/null
"${compose[@]}" config -q
"${compose[@]}" up -d --build --wait
export DOCFLOW_INTEGRATION_DSN="postgres://docflow_integration:docflow_integration@127.0.0.1:${DOCFLOW_SMOKE_POSTGRES_PORT:-55433}/docflow_test_prod_smoke?sslmode=disable"
export DOCFLOW_INTEGRATION_SERVER_URL="http://127.0.0.1:${DOCFLOW_SMOKE_SERVER_PORT:-58480}"
export GOCACHE=${GOCACHE:-/tmp/go-build-cache}

curl --fail --silent --show-error --insecure --resolve localhost:58443:127.0.0.1 \
  https://localhost:58443/api/v1/system/status > "$evidence/caddy-status.json"
go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v > "$evidence/seed.log"
"${compose[@]}" restart seaweedfs
"${compose[@]}" up -d --no-deps --wait --wait-timeout 120 seaweedfs
"${compose[@]}" up -d --no-deps --force-recreate seaweedfs
"${compose[@]}" up -d --no-deps --wait --wait-timeout 120 seaweedfs
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v > "$evidence/storage-recreate.log"
"${compose[@]}" restart docflow-server
"${compose[@]}" up -d --no-deps --wait --wait-timeout 120 docflow-server
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v > "$evidence/server-restart.log"
printf 'PASS: production Compose starts and preserves attachments across storage and server restarts.\n'
