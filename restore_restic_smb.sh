#!/bin/bash

# ==========================================
# Настройки SMB (Сетевая папка)
# ==========================================
SMB_SERVER="192.168.1.100"
SMB_SHARE="shared_backups"
SMB_USER="YourSMBUser"
SMB_PASSWORD="YourSMBPassword"
SMB_DOMAIN="WORKGROUP"
MOUNT_POINT="/mnt/smb_backup"

# ==========================================
# Настройки базы данных и MinIO
# ==========================================
POSTGRES_CONTAINER="postgres_db"
DB_USER="postgres"
DB_PASSWORD="your_db_password"
DB_NAME="doc_registration_db"

MINIO_ENDPOINT="http://localhost:9000"
MINIO_USER="admin_user"
MINIO_PASSWORD="secure_password123"
MINIO_BUCKET="documents"

# ==========================================
# Настройки Restic
# ==========================================
export RESTIC_REPOSITORY="$MOUNT_POINT/doc_registration_repo"
export RESTIC_PASSWORD="YOUR_RESTIC_PASSWORD"

# ==========================================
# Проверка аргументов
# ==========================================
SNAPSHOT_ID=$1

if [ -z "$SNAPSHOT_ID" ]; then
  echo "Ошибка: Не указан ID снимка Restic."
  echo "Использование: sudo ./restore_restic_smb.sh <ID_снимка> (или 'latest')"
  exit 1
fi

TMP_DIR="/tmp/restore_stage_$(date +%s)"

# Останавливаем скрипт при любой ошибке
set -e

echo "Начало процесса восстановления из снимка: $SNAPSHOT_ID"

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
mount -t cifs -o username="$SMB_USER",password="$SMB_PASSWORD",domain="$SMB_DOMAIN" "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT"

# ГАРАНТИЯ ОТКЛЮЧЕНИЯ: сработает при любом завершении скрипта
trap 'echo "Отключение сетевой папки..."; umount "$MOUNT_POINT"' EXIT

# ==========================================
# БЛОК ИЗВЛЕЧЕНИЯ ДАННЫХ
# ==========================================
echo "[2/5] Извлечение файлов из снимка $SNAPSHOT_ID во временную папку $TMP_DIR..."
mkdir -p "$TMP_DIR"
restic restore "$SNAPSHOT_ID" --target "$TMP_DIR"

# ==========================================
# БЛОК ВОССТАНОВЛЕНИЯ (PostgreSQL + MinIO)
# ==========================================
echo "[3/5] Восстановление базы данных PostgreSQL..."
docker exec -i -e PGPASSWORD="$DB_PASSWORD" "$POSTGRES_CONTAINER" pg_restore -U "$DB_USER" -d "$DB_NAME" --clean --if-exists < "$TMP_DIR/database.dump"

echo "[4/5] Восстановление файлов в MinIO..."
docker run --rm --network host --entrypoint sh -v "$TMP_DIR/minio_files:/files" minio/mc \
  -c "mc alias set myminio $MINIO_ENDPOINT $MINIO_USER $MINIO_PASSWORD > /dev/null && mc mirror --overwrite --remove /files myminio/$MINIO_BUCKET > /dev/null"

# ==========================================
# ОЧИСТКА
# ==========================================
echo "[5/5] Очистка локальных временных файлов..."
rm -rf "$TMP_DIR"

echo "✅ Восстановление успешно завершено!"
# Скрипт завершается, срабатывает trap, и папка отмонтируется