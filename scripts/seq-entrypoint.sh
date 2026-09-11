#!/bin/sh
set -eu

# Seq reads the initial administrator password from stdin.
test -s /run/secrets/seq_admin_password
exec /bin/seqentry --default-admin-password-stdin "$@" < /run/secrets/seq_admin_password
