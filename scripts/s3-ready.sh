#!/bin/sh
# Disposable startup probe: never depends on the application's bucket.
set -eu
umask 077
[ -z "${S3_ACCESS_KEY_ID_FILE:-}" ] || S3_ACCESS_KEY_ID=$(cat "$S3_ACCESS_KEY_ID_FILE")
[ -z "${S3_SECRET_ACCESS_KEY_FILE:-}" ] || S3_SECRET_ACCESS_KEY=$(cat "$S3_SECRET_ACCESS_KEY_FILE")
export MC_CONFIG_DIR=/tmp/mc-probe
mkdir -p "$MC_CONFIG_DIR"
probe="docflow-probe-$(date +%s)-$$"
cleanup() {
  mc rb --force "probe/$probe" >/dev/null 2>&1 || true
  rm -rf "$MC_CONFIG_DIR"
}
trap cleanup EXIT
attempt=0
until mc alias set probe "$S3_ENDPOINT" "$S3_ACCESS_KEY_ID" "$S3_SECRET_ACCESS_KEY" --api S3v4 --path on >/dev/null 2>&1 &&
      mc ls probe >/dev/null 2>&1 &&
      mc mb --ignore-existing "probe/$probe" >/dev/null 2>&1 &&
      printf 'docflow-ready' | mc pipe "probe/$probe/ready" >/dev/null 2>&1 &&
      [ "$(mc cat "probe/$probe/ready" 2>/dev/null)" = docflow-ready ]; do
  attempt=$((attempt + 1))
  [ "$attempt" -lt 60 ] || { echo 'S3 readiness failed' >&2; exit 1; }
  sleep 2
done
mc rm "probe/$probe/ready" >/dev/null
mc rb "probe/$probe" >/dev/null
