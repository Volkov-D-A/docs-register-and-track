#!/bin/bash

# ==========================================
# Настройки SMB (Сетевая папка)
# ==========================================
SMB_SERVER="192.168.1.100"       # IP или hostname сервера с шарой
SMB_SHARE="shared_backups"       # Имя расшаренной папки
SMB_USER="YourSMBUser"
SMB_PASSWORD="YourSMBPassword"
SMB_DOMAIN="WORKGROUP"           # Оставьте WORKGROUP или укажите свой домен
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

TMP_DIR="/tmp/backup_stage_$(date +%Y%m%d_%H%M%S)"

# Останавливаем скрипт при любой ошибке
set -e

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Запуск резервного копирования на SMB..."

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/6] Подключение сетевой папки SMB..."
mount -t cifs -o username="$SMB_USER",password="$SMB_PASSWORD",domain="$SMB_DOMAIN" "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT"

# ГАРАНТИЯ ОТКЛЮЧЕНИЯ: Команда trap выполнит umount при любом выходе из скрипта (успешном или при ошибке)
trap 'echo "Отключение сетевой папки..."; umount "$MOUNT_POINT"' EXIT

# Проверяем, инициализирован ли репозиторий Restic в этой папке
if ! restic snapshots >/dev/null 2>&1; then
  echo "Репозиторий Restic не найден в $RESTIC_REPOSITORY. Инициализация..."
  restic init
fi

# ==========================================
# БЛОК СБОРА ДАННЫХ
# ==========================================
mkdir -p "$TMP_DIR/minio_files"

echo "[2/6] Дамп базы данных PostgreSQL..."
docker exec -e PGPASSWORD="$DB_PASSWORD" -t "$POSTGRES_CONTAINER" pg_dump -U "$DB_USER" -Fc "$DB_NAME" > "$TMP_DIR/database.dump"

echo "[3/6] Синхронизация файлов из MinIO..."
docker run --rm --network host --entrypoint sh -v "$TMP_DIR/minio_files:/files" minio/mc \
  -c "mc alias set myminio $MINIO_ENDPOINT $MINIO_USER $MINIO_PASSWORD > /dev/null && mc mirror myminio/$MINIO_BUCKET /files > /dev/null"

# ==========================================
# БЛОК RESTIC
# ==========================================
echo "[4/6] Сохранение снимка в удаленный репозиторий Restic..."
cd "$TMP_DIR"
restic backup . --tag "daily_backup"

echo "[5/6] Удаление старых бэкапов (оставляем последние 7 дней)..."
restic forget --keep-daily 7 --prune

# ==========================================
# ОЧИСТКА
# ==========================================
echo "[6/6] Очистка локальных временных файлов..."
cd /
rm -rf "$TMP_DIR"

echo "✅ Резервное копирование успешно завершено!"
# Здесь скрипт завершается, и автоматически срабатывает trap, отмонтируя папку.