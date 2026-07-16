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
      MOUNT_POINT|SMB_SERVER|SMB_SHARE|SMB_CREDENTIALS_FILE|SMB_VERS|SMB_SEC|POSTGRES_CONTAINER|POSTGRES_DB|POSTGRES_USER|POSTGRES_PASSWORD|MINIO_ENDPOINT|MINIO_ROOT_USER|MINIO_ROOT_PASSWORD|MINIO_BUCKET) export "$key=$value" ;;
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
