#!/usr/bin/env bash
# Shared, deliberately non-executable configuration and CIFS helpers.

load_backup_config() {
  local config_file="${DOCFLOW_BACKUP_ENV_FILE:-/etc/docflow/backup.env}" line key value
  [[ -f "$config_file" ]] || { echo "Ошибка: конфигурация $config_file не найдена." >&2; return 1; }
  [[ ! -L "$config_file" ]] || { echo "Ошибка: конфигурация не должна быть symbolic link." >&2; return 1; }
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    [[ "$line" == *=* ]] || { echo "Ошибка: неверная строка конфигурации." >&2; return 1; }
    key="${line%%=*}"; value="${line#*=}"
    case "$key" in
      MOUNT_POINT|SMB_SERVER|SMB_SHARE|SMB_CREDENTIALS_FILE|SMB_VERS|SMB_SEC|POSTGRES_CONTAINER|POSTGRES_DB|POSTGRES_USER|POSTGRES_PASSWORD|S3_ENDPOINT|S3_ACCESS_KEY_ID|S3_SECRET_ACCESS_KEY|S3_BUCKET|S3_USE_SSL|S3_NETWORK|DOCFLOW_SERVER_CONTAINER) export "$key=$value" ;;
      *) echo "Ошибка: недопустимый ключ $key в $config_file." >&2; return 1 ;;
    esac
  done < "$config_file"
  : "${MOUNT_POINT:?не задан MOUNT_POINT}" "${SMB_SERVER:?не задан SMB_SERVER}" "${SMB_SHARE:?не задан SMB_SHARE}" "${SMB_CREDENTIALS_FILE:?не задан SMB_CREDENTIALS_FILE}"
  [[ -f "$SMB_CREDENTIALS_FILE" && ! -L "$SMB_CREDENTIALS_FILE" ]] || { echo "Ошибка: CIFS credentials file недоступен." >&2; return 1; }
  command -v mount.cifs >/dev/null || { echo "Ошибка: установите пакет cifs-utils (mount.cifs)." >&2; return 1; }
}

mount_smb() {
  mkdir -p "$MOUNT_POINT"
  mount.cifs "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT" -o "credentials=$SMB_CREDENTIALS_FILE,vers=${SMB_VERS:-3.0},sec=${SMB_SEC:-ntlmssp}"
}

unmount_smb() { mountpoint -q "$MOUNT_POINT" && umount "$MOUNT_POINT" || true; }


# Stop the only writer, including its outbox worker, before either snapshot.
quiesce_server() {
  : "${DOCFLOW_SERVER_CONTAINER:?не задан DOCFLOW_SERVER_CONTAINER}"
  SERVER_WAS_RUNNING=0
  if [ "$(docker inspect -f '{{.State.Running}}' "$DOCFLOW_SERVER_CONTAINER")" = true ]; then
    docker stop -t 40 "$DOCFLOW_SERVER_CONTAINER" >/dev/null
    SERVER_WAS_RUNNING=1
  fi
}

resume_server() {
  if [ "${SERVER_WAS_RUNNING:-0}" = 1 ]; then
    docker start "$DOCFLOW_SERVER_CONTAINER" >/dev/null
    SERVER_WAS_RUNNING=0
  fi
}

prepare_s3_client() {
  : "${S3_NETWORK:?не задан S3_NETWORK}" "${S3_ENDPOINT:?не задан S3_ENDPOINT}"
  : "${S3_ACCESS_KEY_ID:?не задан S3_ACCESS_KEY_ID}" "${S3_SECRET_ACCESS_KEY:?не задан S3_SECRET_ACCESS_KEY}" "${S3_BUCKET:?не задан S3_BUCKET}"
  [[ "$S3_BUCKET" =~ ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$ ]] || { echo 'Неверное имя S3 bucket.' >&2; return 1; }
  mkdir -m 700 "$TMP_DIR/s3-secrets"
  printf '%s' "$S3_ACCESS_KEY_ID" > "$TMP_DIR/s3-secrets/access"
  printf '%s' "$S3_SECRET_ACCESS_KEY" > "$TMP_DIR/s3-secrets/secret"
  chmod 600 "$TMP_DIR/s3-secrets/"*
  S3_CLIENT_IMAGE="$(docker inspect -f '{{.Config.Image}}' "$DOCFLOW_SERVER_CONTAINER")"
}

s3_transfer() {
  docker run --rm --user "$(id -u):$(id -g)" --network "$S3_NETWORK" \
    -e S3_ENDPOINT -e S3_BUCKET -e S3_USE_SSL \
    -e S3_ACCESS_KEY_ID_FILE=/run/s3-secrets/access \
    -e S3_SECRET_ACCESS_KEY_FILE=/run/s3-secrets/secret \
    -v "$TMP_DIR/s3-secrets:/run/s3-secrets:ro" -v "$TMP_DIR/objects:/files" \
    "$S3_CLIENT_IMAGE" storage "$@"
}
