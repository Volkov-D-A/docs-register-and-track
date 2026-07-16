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
ARCHIVE_NAME="backup_$(date +%Y%m%d_%H%M%S).tar.gz"
FINAL_ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"
MOUNTED=0

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Запуск резервного копирования на SMB..."

cleanup() {
  local exit_code=$?

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
echo "[4/5] Создание tar.gz архива напрямую на сетевом диске..."
cd "$TMP_DIR"
tar -czf "$FINAL_ARCHIVE_PATH" database.dump minio_files/

# Очистка старых архивов (старше 15 дней) на SMB-сервере
echo "Удаление архивов старше 15 дней..."
find "$MOUNT_POINT" -name "backup_*.tar.gz" -type f -mtime +15 -exec rm {} \;

# ==========================================
# ОЧИСТКА ВРЕМЕННЫХ ФАЙЛОВ
# ==========================================
echo "[5/5] Очистка локальных временных файлов..."
cd /

echo "✅ Резервное копирование успешно завершено! Файл: $ARCHIVE_NAME"
