#!/usr/bin/env bash
set -euo pipefail

# Use clients from the disposable database container and keep archives on the host.
tool=${0##*/}
container=${DOCFLOW_INTEGRATION_PG_CONTAINER:?integration PostgreSQL container required}
if [[ "$tool" == pg_restore ]]; then
    archive=${!#}
    set -- "${@:1:$#-1}"
    exec < "$archive"
fi

exec docker exec -i -e PGUSER -e PGDATABASE "$container" sh -c '
    export PGHOST=127.0.0.1 PGPORT=5432 PGSSLMODE=disable PGCONNECT_TIMEOUT=10
    export PGPASSWORD="$POSTGRES_PASSWORD"
    exec "$@"
' sh "$tool" "$@"
