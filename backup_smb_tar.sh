#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/scripts/smb_backup_lib.sh"
load_backup_config

# ==========================================
# Настройки путей
# ==========================================
TMP_DIR="$(mktemp -d /tmp/backup_stage_XXXXXXXXXX)"
chmod 700 "$TMP_DIR"
ARCHIVE_NAME="backup_$(date +%Y%m%d_%H%M%S_%N).tar.gz"
FINAL_ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"
FINAL_MANIFEST_PATH="$FINAL_ARCHIVE_PATH.manifest"
PARTIAL_ARCHIVE_PATH=""
PARTIAL_MANIFEST_PATH=""
ARCHIVE_PUBLISHED=0
MANIFEST_PUBLISHED=0
MOUNTED=0

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Запуск резервного копирования на SMB..."

cleanup() {
  local exit_code=$?

  if [ -n "$PARTIAL_ARCHIVE_PATH" ]; then
    rm -f -- "$PARTIAL_ARCHIVE_PATH"
  fi
  if [ -n "$PARTIAL_MANIFEST_PATH" ]; then
    rm -f -- "$PARTIAL_MANIFEST_PATH"
  fi
  if [ "$exit_code" -ne 0 ] && [ "$ARCHIVE_PUBLISHED" -eq 1 ] && [ "$MANIFEST_PUBLISHED" -eq 0 ]; then
    rm -f -- "$FINAL_ARCHIVE_PATH" "$FINAL_MANIFEST_PATH"
  fi

  if [ "$MOUNTED" -eq 1 ]; then
    echo "Отключение сетевой папки..."
    unmount_smb
  fi

  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi

  exit "$exit_code"
}

trap cleanup EXIT

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
mount_smb
MOUNTED=1

# Temporary files must be created on the same mounted filesystem as the final
# artifacts. Otherwise mv can degrade into a non-atomic copy followed by delete.
PARTIAL_ARCHIVE_PATH="$(mktemp "$MOUNT_POINT/.${ARCHIVE_NAME}.partial.XXXXXXXX")"
PARTIAL_MANIFEST_PATH="$(mktemp "$MOUNT_POINT/.${ARCHIVE_NAME}.manifest.partial.XXXXXXXX")"
chmod 600 "$PARTIAL_ARCHIVE_PATH" "$PARTIAL_MANIFEST_PATH"

# ==========================================
# БЛОК СБОРА ДАННЫХ
# ==========================================
mkdir -p "$TMP_DIR/minio_files"

echo "[2/5] Дамп базы данных PostgreSQL..."
docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB" > "$TMP_DIR/database.dump"

echo "[3/5] Синхронизация файлов из MinIO..."
docker run --rm --network host --entrypoint sh -v "$TMP_DIR/minio_files:/files" minio/mc \
  -c "mc alias set myminio $MINIO_ENDPOINT $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD > /dev/null && mc mirror myminio/$MINIO_BUCKET /files > /dev/null"

# ==========================================
# БЛОК АРХИВАЦИИ
# ==========================================
echo "[4/5] Создание и проверка tar.gz архива на сетевом диске..."
cd "$TMP_DIR"
# dd with conv=fsync calls fsync on the output descriptor before closing it.
# pipefail preserves a tar failure even if dd successfully flushes partial data.
tar -czf - database.dump minio_files/ |
  dd of="$PARTIAL_ARCHIVE_PATH" conv=fsync status=none
tar -tzf "$PARTIAL_ARCHIVE_PATH" > /dev/null

ARCHIVE_SIZE="$(stat -c '%s' -- "$PARTIAL_ARCHIVE_PATH")"
ARCHIVE_SHA256="$(sha256sum -- "$PARTIAL_ARCHIVE_PATH")"
ARCHIVE_SHA256="${ARCHIVE_SHA256%% *}"
CREATED_AT="$(date -Is)"

{
  printf 'format_version=1\n'
  printf 'archive=%s\n' "$ARCHIVE_NAME"
  printf 'created_at=%s\n' "$CREATED_AT"
  printf 'size_bytes=%s\n' "$ARCHIVE_SIZE"
  printf 'sha256=%s\n' "$ARCHIVE_SHA256"
  printf 'postgres_database=%s\n' "$POSTGRES_DB"
  printf 'minio_bucket=%s\n' "$MINIO_BUCKET"
} | dd of="$PARTIAL_MANIFEST_PATH" conv=fsync status=none

# The manifest is the commit marker and therefore must be published last.
# GNU sync -f flushes the containing SMB filesystem after each rename; final
# durability still depends on the SMB server honoring flush requests.
mv -- "$PARTIAL_ARCHIVE_PATH" "$FINAL_ARCHIVE_PATH"
PARTIAL_ARCHIVE_PATH=""
ARCHIVE_PUBLISHED=1
sync -f "$MOUNT_POINT"

mv -- "$PARTIAL_MANIFEST_PATH" "$FINAL_MANIFEST_PATH"
PARTIAL_MANIFEST_PATH=""
sync -f "$MOUNT_POINT"
MANIFEST_PUBLISHED=1

# Очистка старых архивов (старше 15 дней) на SMB-сервере
echo "Удаление архивов старше 15 дней..."
find "$MOUNT_POINT" -maxdepth 1 -name "backup_*.tar.gz" -type f -mtime +15 -print0 |
  while IFS= read -r -d '' expired_archive; do
    rm -f -- "$expired_archive" "$expired_archive.manifest"
  done
find "$MOUNT_POINT" -maxdepth 1 -name '.backup_*.partial.*' -type f -mtime +1 -delete

# ==========================================
# ОЧИСТКА ВРЕМЕННЫХ ФАЙЛОВ
# ==========================================
echo "[5/5] Очистка локальных временных файлов..."
cd /

echo "✅ Резервное копирование успешно завершено! Файл: $ARCHIVE_NAME"
