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
# Проверка аргументов
# ==========================================
ARCHIVE_NAME="${1:-}"

if [ -z "$ARCHIVE_NAME" ]; then
  echo "Ошибка: Не указано имя файла резервной копии."
  echo "Использование: sudo ./restore_smb_tar.sh backup_20260304_120000.tar.gz"
  exit 1
fi

TMP_DIR="$(mktemp -d /tmp/restore_stage_XXXXXXXXXX)"
chmod 700 "$TMP_DIR"
ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"
REPORT_DIR="${RESTORE_REPORT_DIR:-$(pwd)/restore_reports}"
REPORT_FILE="$REPORT_DIR/restore_$(date +%Y%m%d_%H%M%S).log"
SMB_CREDENTIALS_TMP=""
MOUNT_OPTIONS=""
MOUNTED=0

echo "Начало процесса восстановления из файла: $ARCHIVE_NAME"

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

write_report_header() {
  mkdir -p "$REPORT_DIR"
  chmod 700 "$REPORT_DIR"
  {
    echo "restore_started_at=$(date -Is)"
    echo "archive=$ARCHIVE_NAME"
    echo "postgres_container=$POSTGRES_CONTAINER"
    echo "postgres_db=$POSTGRES_DB"
    echo "minio_bucket=$MINIO_BUCKET"
    echo
  } > "$REPORT_FILE"
  chmod 600 "$REPORT_FILE"
}

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
set_mount_options
mount -t cifs -o "$MOUNT_OPTIONS" "//$SMB_SERVER/$SMB_SHARE" "$MOUNT_POINT"
MOUNTED=1

if [ ! -f "$ARCHIVE_PATH" ]; then
  echo "Ошибка: Файл $ARCHIVE_PATH не найден на сетевом диске."
  exit 1
fi

write_report_header

# ==========================================
# БЛОК ИЗВЛЕЧЕНИЯ ДАННЫХ
# ==========================================
echo "[2/5] Распаковка архива во временную директорию $TMP_DIR..."
mkdir -p "$TMP_DIR"
tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

if [ ! -f "$TMP_DIR/database.dump" ]; then
  echo "Ошибка: в архиве отсутствует database.dump."
  exit 1
fi

if [ ! -d "$TMP_DIR/minio_files" ]; then
  echo "Ошибка: в архиве отсутствует каталог minio_files/."
  exit 1
fi

# ==========================================
# БЛОК ВОССТАНОВЛЕНИЯ (PostgreSQL)
# ==========================================
echo "[3/5] Восстановление базы данных PostgreSQL..."

# Копируем файл внутрь контейнера во временную папку (это решает проблему с потоком)
docker cp "$TMP_DIR/database.dump" "$POSTGRES_CONTAINER":/tmp/database_restore.dump

restore_cleanup() {
  docker exec "$POSTGRES_CONTAINER" rm -f /tmp/database_restore.dump > /dev/null 2>&1 || true
}
trap 'restore_cleanup; cleanup' EXIT

echo "pg_restore_started_at=$(date -Is)" >> "$REPORT_FILE"
if ! docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" \
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists --exit-on-error /tmp/database_restore.dump \
  >> "$REPORT_FILE" 2>&1; then
  echo "Ошибка: pg_restore завершился неуспешно. MinIO не будет восстановлен. Отчет: $REPORT_FILE"
  exit 1
fi
echo "pg_restore_finished_at=$(date -Is)" >> "$REPORT_FILE"

restore_cleanup
trap cleanup EXIT

echo "Проверка восстановленной БД..." | tee -a "$REPORT_FILE"
docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 \
  -c "SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1;" \
  -c "SELECT COUNT(*) AS documents_count FROM documents;" \
  >> "$REPORT_FILE" 2>&1

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
echo "restore_finished_at=$(date -Is)" >> "$REPORT_FILE"

echo "✅ Восстановление успешно завершено! Отчет: $REPORT_FILE"
