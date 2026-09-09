#!/usr/bin/env bash
# Explicitly invoked destructive dev reset. Never run as part of verification.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
project=$(docker compose config --format json | python3 -c 'import json,sys; print(json.load(sys.stdin)["name"])')
volumes=()
for kind in pgdata minio_data seaweedfs_data; do
  listed=$(docker volume ls -q --filter "label=com.docker.compose.project=$project" --filter "label=com.docker.compose.volume=$kind")
  while IFS= read -r volume; do
    [ -z "$volume" ] || volumes+=("$volume")
  done <<< "$listed"
done
printf 'Dev project: %s\nData volumes to discard: %s\nSeq and Caddy volumes are preserved.\n' "$project" "${volumes[*]:-(none)}"
if [ "${1:-}" != --execute ]; then
  echo 'Review above, then invoke make storage-reset to execute.'
  exit 0
fi
docker compose down --remove-orphans
if [ "${#volumes[@]}" -gt 0 ]; then docker volume rm "${volumes[@]}"; fi
docker compose up -d --build
