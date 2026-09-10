#!/usr/bin/env bash
# Real PostgreSQL/S3 and built API; only the SMB mount is replaced by a local directory.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
ROOT=$PWD
# Keep smoke ports separate from ordinary integration tests.
export DOCFLOW_TEST_POSTGRES_PORT=55433 DOCFLOW_TEST_S3_PORT=58334
export DOCFLOW_TEST_SERVER_PORT=${DOCFLOW_TEST_SERVER_PORT:-58480}
compose=(docker compose -p docflow-smoke -f docker-compose.integration.yaml -f docker-compose.backup-smoke.yaml --profile smoke)
stage=$(mktemp -d /tmp/docflow-smoke.XXXXXXXX)
evidence="$ROOT/build/transition-evidence"
mkdir -p "$evidence" "$stage/bin" "$stage/share" "$stage/reports" "$stage/remote"
chmod 755 "$stage/remote"
export DOCFLOW_SMOKE_KEY_FILE="$stage/settings-key" DOCFLOW_SMOKE_ARCHIVE_DIR="$stage/remote"
python3 -c 'import base64; print(base64.b64encode(bytes(range(32))).decode())' > "$DOCFLOW_SMOKE_KEY_FILE"
chmod 644 "$DOCFLOW_SMOKE_KEY_FILE"
docker compose -p docflow-backup-test -f docker-compose.backup-test.yaml up -d --build
cleanup() {
  result=$?
  docker rm -f docflow-smoke-recovery >/dev/null 2>&1 || true
  "${compose[@]}" logs --no-color > "$evidence/containers.log" 2>&1 || true
  "${compose[@]}" down -v --remove-orphans || true
  docker compose -p docflow-backup-test -f docker-compose.backup-test.yaml down -v --remove-orphans || true
  rm -rf -- "$stage"
  exit "$result"
}
trap cleanup EXIT
# Simulate only CIFS mount operations, not Docker, pg_dump, pg_restore, S3 transfers or tar.
for command in mount.cifs umount; do
  printf '#!/bin/sh\nexit 0\n' > "$stage/bin/$command"
done
printf '#!/bin/sh\nexit 1\n' > "$stage/bin/mountpoint"
chmod +x "$stage/bin/"*
touch "$stage/credentials"
cat > "$stage/backup.env" <<CONFIG
MOUNT_POINT=$stage/share
SMB_SERVER=local-fixture
SMB_SHARE=backup
SMB_CREDENTIALS_FILE=$stage/credentials
POSTGRES_CONTAINER=docflow-smoke-postgres-1
POSTGRES_DB=docflow_test_outbox
POSTGRES_USER=docflow_integration
POSTGRES_PASSWORD=docflow_integration
S3_ENDPOINT=http://seaweedfs:8333
S3_ACCESS_KEY_ID=docflow_integration
S3_SECRET_ACCESS_KEY=docflow_integration_secret
S3_BUCKET=docflow-test-smoke
S3_NETWORK=docflow-smoke_default
DOCFLOW_SERVER_CONTAINER=docflow-smoke-docflow-server-1
CONFIG
chmod 600 "$stage/backup.env" "$stage/credentials"
export DOCFLOW_BACKUP_ENV_FILE="$stage/backup.env" RESTORE_REPORT_DIR="$stage/reports"
export DOCFLOW_INTEGRATION_DSN='postgres://docflow_integration:docflow_integration@127.0.0.1:55433/docflow_test_outbox?sslmode=disable'
export DOCFLOW_INTEGRATION_SERVER_URL=http://127.0.0.1:$DOCFLOW_TEST_SERVER_PORT
export GOCACHE=${GOCACHE:-/tmp/go-build-cache}
"${compose[@]}" up -d --build --wait
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

PATH="$stage/bin:$PATH" bash ./backup_smb_tar.sh | tee "$evidence/backup.log"
archive=$(basename "$stage/share/"*.tar.gz)
cp "$stage/share/$archive.manifest" "$evidence/backup.manifest"
# The restore target has empty PostgreSQL AND S3 volumes; preserve neither.
"${compose[@]}" down -v --remove-orphans
"${compose[@]}" up -d --wait postgres seaweedfs s3-ready
"${compose[@]}" create --no-build docflow-server
# A valid v2 archive containing an invalid pg_dump must fail before S3 mirror.
mkdir "$stage/bad"
tar -xzf "$stage/share/$archive" -C "$stage/bad"
printf 'invalid PostgreSQL dump' > "$stage/bad/database.dump"
bad_archive=backup_20000101_000000_000000000.tar.gz
tar -czf "$stage/share/$bad_archive" -C "$stage/bad" database.dump objects
python3 - "$stage/share/$archive.manifest" "$stage/share/$bad_archive" <<'PYMANIFEST'
import hashlib, pathlib, sys
source, archive = map(pathlib.Path, sys.argv[1:])
fields = dict(line.split('=', 1) for line in source.read_text().splitlines())
fields.update(archive=archive.name, size_bytes=str(archive.stat().st_size), sha256=hashlib.sha256(archive.read_bytes()).hexdigest())
pathlib.Path(str(archive)+'.manifest').write_text(''.join(k+'='+v+'\n' for k,v in fields.items()))
PYMANIFEST
if PATH="$stage/bin:$PATH" bash ./restore_smb_tar.sh "$bad_archive" > "$evidence/restore-rejected.log" 2>&1; then
  echo 'Invalid PostgreSQL dump unexpectedly restored' >&2; exit 1
fi
chmod 755 "$stage/share"
chmod 644 "$stage/share/"*.tar.gz "$stage/share/"*.manifest
if "${compose[@]}" run --rm --no-deps -v "$stage/share:/restore:ro" docflow-server restore "/restore/$bad_archive" > "$evidence/server-restore-rejected.log" 2>&1; then
  echo 'Server accepted invalid PostgreSQL dump' >&2; exit 1
fi
if "${compose[@]}" run --rm --no-deps docflow-server storage bucket-check >/dev/null 2>&1; then
  echo 'S3 bucket was created after failed PostgreSQL restore' >&2; exit 1
fi
PATH="$stage/bin:$PATH" bash ./restore_smb_tar.sh "$archive" | tee "$evidence/restore.log"
cp "$stage/reports/"*.log "$evidence/"
"${compose[@]}" up -d --no-build --wait
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/restored-api.log"
# Verify that the autonomous command also reads the original v2 format.
"${compose[@]}" down -v --remove-orphans
"${compose[@]}" up -d --wait postgres seaweedfs s3-ready
"${compose[@]}" run --rm --no-deps -v "$stage/share:/restore:ro" docflow-server restore "/restore/$archive" | tee "$evidence/server-v2-restore.log"
"${compose[@]}" up -d --no-build --wait
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/server-v2-restored-api.log"
go test ./internal/server -run '^TestBuiltServerBackupIntegration$' -count=1 -v | tee "$evidence/server-backup.log"
"${compose[@]}" down -v --remove-orphans
# Recovery UI can authenticate and list SMB copies while PostgreSQL/S3 are absent.
"${compose[@]}" run --rm -d --no-deps --name docflow-smoke-recovery -p 127.0.0.1:58481:8081 \
  -e DOCFLOW_RECOVERY_SECRET_FILE=/run/secrets/settings_key -e DOCFLOW_RECOVERY_LISTEN_ADDRESS=:8081 docflow-server recovery
DOCFLOW_RECOVERY_TEST_URL=http://127.0.0.1:58481 go test ./internal/server -run '^TestBuiltRecoveryPanelWithoutDatabaseIntegration$' -count=1 -v | tee "$evidence/recovery-panel.log"
docker stop docflow-smoke-recovery >/dev/null
"${compose[@]}" up -d --wait postgres seaweedfs s3-ready
"${compose[@]}" run --rm --no-deps -v "$stage/remote:/restore:ro" docflow-server restore "/restore/$(basename "$stage/remote/"*.tar.gz)" | tee "$evidence/server-restore.log"
"${compose[@]}" up -d --no-build --wait
DOCFLOW_SMOKE_VERIFY=1 go test ./internal/server -run '^TestBuiltServerAttachmentsIntegration$' -count=1 -v | tee "$evidence/server-restored-api.log"
printf 'PASS: built server, restart, legacy v2 and server v3 backup/restore; direct SMB verified.\n' | tee "$evidence/result.txt"
