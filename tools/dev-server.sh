#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
export DOCFLOW_LOCAL_BUILD=1
export GOCACHE="${GOCACHE:-/tmp/go-build-cache}"

identity=$(build/bin/docflow-go identity)
version=$(go run ./cmd/docflow-server version)

export DOCFLOW_LOCAL_VERSION="$version"
export DOCFLOW_LOCAL_IDENTITY="$identity"
docker compose -f docker-compose.yaml build docflow-server

# Do not replace the running server if the source changed during the build.
if [[ "$identity" != "$(build/bin/docflow-go identity)" ]]; then
  echo "Исходники изменились во время сборки. Повторите make dev-server." >&2
  exit 1
fi

# Compose recreates the server when the image changes and starts dependencies
# on first use. The root Compose file uses only the local server image.
docker compose -f docker-compose.yaml up -d --no-build --wait
