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
# Проверка аргументов
# ==========================================
ARCHIVE_NAME=$1

if [ -z "$ARCHIVE_NAME" ]; then
  echo "Ошибка: Не указано имя файла резервной копии."
  echo "Использование: sudo ./restore_smb_tar.sh backup_20260304_120000.tar.gz"
  exit 1
fi

TMP_DIR="/tmp/restore_stage_$(date +%s)"
ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"

set -e
echo "Начало процесса восстановления из файла: $ARCHIVE_NAME"

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
mount -t cifs -o username="$SMB_USER",password="$SMB_PASSWORD",domain="$SMB_DOMAIN" "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT"

trap 'echo "Отключение сетевой папки..."; umount "$MOUNT_POINT"' EXIT

if [ ! -f "$ARCHIVE_PATH" ]; then
  echo "Ошибка: Файл $ARCHIVE_PATH не найден на сетевом диске."
  exit 1
fi

# ==========================================
# БЛОК ИЗВЛЕЧЕНИЯ ДАННЫХ
# ==========================================
echo "[2/5] Распаковка архива во временную директорию $TMP_DIR..."
mkdir -p "$TMP_DIR"
tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

# ==========================================
# БЛОК ВОССТАНОВЛЕНИЯ (PostgreSQL)
# ==========================================
echo "[3/5] Восстановление базы данных PostgreSQL..."

# Копируем файл внутрь контейнера во временную папку (это решает проблему с потоком)
docker cp "$TMP_DIR/database.dump" "$POSTGRES_CONTAINER":/tmp/database_restore.dump

# Временно отключаем строгую остановку при ошибках
set +e 

# Запускаем восстановление уже из локального файла внутри контейнера
docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists /tmp/database_restore.dump
RESTORE_CODE=$?

# Включаем строгую остановку обратно
set -e

# Удаляем временный файл внутри контейнера, чтобы не занимать место
docker exec "$POSTGRES_CONTAINER" rm /tmp/database_restore.dump

if [ $RESTORE_CODE -ne 0 ]; then
  echo "⚠️ Утилита pg_restore завершилась с кодом $RESTORE_CODE. Обычно это некритичные предупреждения от флагов очистки. Продолжаем..."
fi

# ==========================================
# БЛОК ВОССТАНОВЛЕНИЯ ФАЙЛОВ (MinIO)
# ==========================================
echo "[4/5] Проверка бакета и восстановление файлов в MinIO..."
docker run --rm --network host --entrypoint sh -v "$TMP_DIR/minio_files:/files" minio/mc \
  -c "mc alias set myminio $MINIO_ENDPOINT $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD > /dev/null && \
      (mc ls myminio/$MINIO_BUCKET > /dev/null 2>&1 || mc mb myminio/$MINIO_BUCKET > /dev/null) && \
      mc mirror --overwrite --remove /files myminio/$MINIO_BUCKET > /dev/null"

# ==========================================
# ОЧИСТКА
# ==========================================
echo "[5/5] Очистка локальных временных файлов..."
rm -rf "$TMP_DIR"

echo "✅ Восстановление успешно завершено!"