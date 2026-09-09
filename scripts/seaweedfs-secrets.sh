#!/bin/sh
set -eu
AWS_ACCESS_KEY_ID=$(cat "$S3_ACCESS_KEY_ID_FILE")
AWS_SECRET_ACCESS_KEY=$(cat "$S3_SECRET_ACCESS_KEY_FILE")
: "${AWS_ACCESS_KEY_ID:?empty S3 access key}" "${AWS_SECRET_ACCESS_KEY:?empty S3 secret key}"
export AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY
exec weed "$@"
