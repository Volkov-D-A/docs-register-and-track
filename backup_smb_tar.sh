#!/bin/bash
set -euo pipefail
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
TMP_DIR="$(mktemp -d /tmp/backup_stage_XXXXXXXXXX)"
chmod 700 "$TMP_DIR"
ARCHIVE_NAME="backup_$(date +%Y%m%d_%H%M%S).tar.gz"
FINAL_ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"
SMB_CREDENTIALS_TMP=""
MOUNT_OPTIONS=""
MOUNTED=0

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Запуск резервного копирования на SMB..."

cleanup() {
  local exit_code=$?

  if [ "$MOUNTED" -eq 1 ]; then
    echo "Отключение сетевой папки..."
    umount "$MOUNT_POINT" || true
  fi

  if [ -n "$SMB_CREDENTIALS_TMP" ] && [ -f "$SMB_CREDENTIALS_TMP" ]; then
    rm -f "$SMB_CREDENTIALS_TMP"
  fi

  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi

  exit "$exit_code"
}

trap cleanup EXIT

set_mount_options() {
  local credentials_file="${SMB_CREDENTIALS_FILE:-}"

  if [ -n "$credentials_file" ]; then
    if [ ! -f "$credentials_file" ]; then
      echo "Ошибка: CIFS credentials file $credentials_file не найден."
      exit 1
    fi
    MOUNT_OPTIONS="credentials=$credentials_file,domain=$SMB_DOMAIN"
    return
  fi

  SMB_CREDENTIALS_TMP="$(mktemp /tmp/docflow_cifs_credentials_XXXXXXXXXX)"
  chmod 600 "$SMB_CREDENTIALS_TMP"
  {
    printf 'username=%s\n' "$SMB_USER"
    printf 'password=%s\n' "$SMB_PASSWORD"
    printf 'domain=%s\n' "$SMB_DOMAIN"
  } > "$SMB_CREDENTIALS_TMP"

  MOUNT_OPTIONS="credentials=$SMB_CREDENTIALS_TMP"
}

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
set_mount_options
mount -t cifs -o "$MOUNT_OPTIONS" "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT"
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
