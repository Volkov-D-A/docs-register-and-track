#!/bin/bash
# ==========================================
# Загрузка конфигурации из .env
# ==========================================
ENV_FILE="/home/dimas/projects/docs-register-and-track/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo "Ошибка: Файл конфигурации $ENV_FILE не найден!"
    exit 1
fi

set -a
source "$ENV_FILE"
set +a

# ==========================================
# Настройки путей
# ==========================================
TMP_DIR="/tmp/backup_stage_$(date +%Y%m%d_%H%M%S)"
ARCHIVE_NAME="backup_$(date +%Y%m%d_%H%M%S).tar.gz"
FINAL_ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"

set -e
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Запуск резервного копирования на SMB..."

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
mount -t cifs -o username="$SMB_USER",password="$SMB_PASSWORD",domain="$SMB_DOMAIN" "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT"

# ГАРАНТИЯ ОТКЛЮЧЕНИЯ: сработает при любом завершении скрипта
trap 'echo "Отключение сетевой папки..."; umount "$MOUNT_POINT"' EXIT

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

# Очистка старых архивов (старше 7 дней) на SMB-сервере
echo "Удаление архивов старше 7 дней..."
find "$MOUNT_POINT" -name "backup_*.tar.gz" -type f -mtime +7 -exec rm {} \;

# ==========================================
# ОЧИСТКА ВРЕМЕННЫХ ФАЙЛОВ
# ==========================================
echo "[5/5] Очистка локальных временных файлов..."
cd /
rm -rf "$TMP_DIR"

echo "✅ Резервное копирование успешно завершено! Файл: $ARCHIVE_NAME"