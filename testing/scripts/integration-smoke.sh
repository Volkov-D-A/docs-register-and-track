#!/usr/bin/env bash
# Real PostgreSQL, S3, Samba and built administrative API.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.."
ROOT=$PWD
# Preserve the root development environment after moving Compose files.
compose_env=/dev/null
if [[ -f "$ROOT/.env" ]]; then compose_env="$ROOT/.env"; fi
# Keep smoke ports separate from ordinary integration tests.
export DOCFLOW_TEST_POSTGRES_PORT=${DOCFLOW_TEST_POSTGRES_PORT:-55433} DOCFLOW_TEST_S3_PORT=${DOCFLOW_TEST_S3_PORT:-58334}
export DOCFLOW_TEST_SERVER_PORT=${DOCFLOW_TEST_SERVER_PORT:-58480}
compose=(docker compose --env-file "$compose_env" -p docflow-smoke -f testing/compose/integration.yaml -f testing/compose/backup-smoke.yaml --profile smoke)
stage=$(mktemp -d /tmp/docflow-smoke.XXXXXXXX)
evidence="$ROOT/build/transition-evidence"
mkdir -p "$evidence" "$stage/bin" "$stage/share" "$stage/reports" "$stage/remote"
chmod 755 "$stage/remote"
export DOCFLOW_SMOKE_KEY_FILE="$stage/settings-key" DOCFLOW_SMOKE_ARCHIVE_DIR="$stage/remote"
python3 -c 'import base64; print(base64.b64encode(bytes(range(32))).decode())' > "$DOCFLOW_SMOKE_KEY_FILE"
chmod 644 "$DOCFLOW_SMOKE_KEY_FILE"
docker compose --env-file "$compose_env" -p docflow-backup-test -f testing/compose/backup-test.yaml up -d --build
cleanup() {
  result=$?
  "${compose[@]}" logs --no-color > "$evidence/containers.log" 2>&1 || true
  "${compose[@]}" down -v --remove-orphans || true
  docker compose --env-file "$compose_env" -p docflow-backup-test -f testing/compose/backup-test.yaml down -v --remove-orphans || true
  rm -rf -- "$stage"
  exit "$result"
}
trap cleanup EXIT
export DOCFLOW_INTEGRATION_DSN="postgres://docflow_integration:docflow_integration@127.0.0.1:$DOCFLOW_TEST_POSTGRES_PORT/docflow_test_outbox?sslmode=disable"
export DOCFLOW_INTEGRATION_SERVER_URL=http://127.0.0.1:$DOCFLOW_TEST_SERVER_PORT
export GOCACHE=${GOCACHE:-/tmp/go-build-cache}
"${compose[@]}" up -d --build --wait
DOCFLOW_INTEGRATION_SMB_HOST=127.0.0.1 DOCFLOW_INTEGRATION_SMB_PORT=5445 DOCFLOW_INTEGRATION_SMB_SHARE=backups DOCFLOW_INTEGRATION_SMB_USER=docflow DOCFLOW_INTEGRATION_SMB_PASSWORD=integration-password go test ./internal/backup/smb -run Integration -count=1 -v | tee "$evidence/smb-lock.log"
# Fresh server has applied the ordinary embedded migrations and created its bucket.
go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/api-seed.log"
"${compose[@]}" restart seaweedfs
"${compose[@]}" run --rm s3-ready
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/restart.log"
# Recreate the container too: metadata must not depend on its writable layer.
"${compose[@]}" up -d --no-deps --force-recreate seaweedfs
"${compose[@]}" run --rm s3-ready
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/recreate.log"
"${compose[@]}" exec -T seaweedfs sh -c 'ls -d /data/*/' > "$evidence/storage-directories.txt"

go test ./internal/server -run '^TestBuiltServerBackupIntegration$' -count=1 -v | tee "$evidence/server-backup.log"
go test ./internal/server -run '^TestBuiltAdminRestoreIntegration$' -count=1 -v | tee "$evidence/admin-replacement.log"
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/admin-restored-api.log"
for format in v3 v2; do
  "${compose[@]}" down -v --remove-orphans
  "${compose[@]}" up -d --no-build --wait
  DOCFLOW_SMOKE_RESET="$format" go test ./internal/server -run '^TestBuiltAdminRestoreIntegration$' -count=1 -v | tee "$evidence/admin-reset-$format.log"
  DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/admin-reset-$format-api.log"
done
for command in recovery restore; do
  if "${compose[@]}" exec -T docflow-server docflow-server "$command" > "$evidence/removed-$command.log" 2>&1; then
    echo "Removed command unexpectedly succeeded: $command" >&2; exit 1
  fi
  if ! grep -q 'unknown command' "$evidence/removed-$command.log"; then
    cat "$evidence/removed-$command.log" >&2; exit 1
  fi
done
printf 'PASS: ordinary admin API, replacement, reset recovery v2/v3, restored attachments.\n' | tee "$evidence/result.txt"
