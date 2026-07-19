#!/bin/bash
set -euo pipefail

# ==========================================
# Загрузка конфигурации из .env
# ==========================================
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/scripts/smb_backup_lib.sh"
load_backup_config

# ==========================================
# Проверка аргументов
# ==========================================
ARCHIVE_NAME="${1:-}"
LEGACY_MODE="${2:-}"

if [ "$#" -gt 2 ]; then
  echo "Ошибка: Слишком много параметров."
  exit 1
fi
if [ -z "$ARCHIVE_NAME" ]; then
  echo "Ошибка: Не указано имя файла резервной копии."
  echo "Использование: sudo ./restore_smb_tar.sh backup_20260304_120000_123456789.tar.gz [--allow-legacy-without-manifest]"
  exit 1
fi
if [[ ! "$ARCHIVE_NAME" =~ ^backup_[0-9]{8}_[0-9]{6}(_[0-9]{9})?\.tar\.gz$ ]]; then
  echo "Ошибка: Недопустимое имя резервной копии: $ARCHIVE_NAME"
  exit 1
fi
if [ -n "$LEGACY_MODE" ] && [ "$LEGACY_MODE" != "--allow-legacy-without-manifest" ]; then
  echo "Ошибка: Неизвестный параметр: $LEGACY_MODE"
  exit 1
fi

TMP_DIR="$(mktemp -d /tmp/restore_stage_XXXXXXXXXX)"
chmod 700 "$TMP_DIR"
ARCHIVE_PATH="$MOUNT_POINT/$ARCHIVE_NAME"
MANIFEST_PATH="$ARCHIVE_PATH.manifest"
REPORT_DIR="${RESTORE_REPORT_DIR:-$(pwd)/restore_reports}"
REPORT_FILE="$REPORT_DIR/restore_$(date +%Y%m%d_%H%M%S).log"
MOUNTED=0
BACKUP_VERIFICATION=""
VERIFIED_SHA256=""
VERIFIED_SIZE=""

echo "Начало процесса восстановления из файла: $ARCHIVE_NAME"

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

write_report_header() {
  mkdir -p "$REPORT_DIR"
  chmod 700 "$REPORT_DIR"
  {
    echo "restore_started_at=$(date -Is)"
    echo "archive=$ARCHIVE_NAME"
    echo "backup_verification=$BACKUP_VERIFICATION"
    echo "archive_size_bytes=$VERIFIED_SIZE"
    echo "archive_sha256=$VERIFIED_SHA256"
    echo "postgres_container=$POSTGRES_CONTAINER"
    echo "postgres_db=$POSTGRES_DB"
    echo "minio_bucket=$MINIO_BUCKET"
    echo
  } > "$REPORT_FILE"
  chmod 600 "$REPORT_FILE"
}

verify_manifest() {
  local line key value actual_size actual_checksum checksum_output
  local -A fields=()

  if [[ ! -f "$MANIFEST_PATH" || -L "$MANIFEST_PATH" ]]; then
    echo "Ошибка: Manifest недоступен или является symbolic link: $MANIFEST_PATH"
    return 1
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" == *=* ]] || { echo "Ошибка: Некорректная строка manifest."; return 1; }
    key="${line%%=*}"
    value="${line#*=}"
    case "$key" in
      format_version|archive|created_at|size_bytes|sha256|postgres_database|minio_bucket) ;;
      *) echo "Ошибка: Неизвестное поле manifest: $key"; return 1 ;;
    esac
    [[ ! -v "fields[$key]" ]] || { echo "Ошибка: Поле manifest повторяется: $key"; return 1; }
    fields["$key"]="$value"
  done < "$MANIFEST_PATH"

  [[ "${fields[format_version]:-}" == "1" ]] || { echo "Ошибка: Неподдерживаемая версия manifest."; return 1; }
  [[ "${fields[archive]:-}" == "$ARCHIVE_NAME" ]] || { echo "Ошибка: Имя архива не соответствует manifest."; return 1; }
  [[ "${fields[size_bytes]:-}" =~ ^[0-9]+$ ]] || { echo "Ошибка: Некорректный размер в manifest."; return 1; }
  [[ "${fields[sha256]:-}" =~ ^[0-9a-f]{64}$ ]] || { echo "Ошибка: Некорректная SHA-256 в manifest."; return 1; }
  [[ -n "${fields[created_at]:-}" && -n "${fields[postgres_database]:-}" && -n "${fields[minio_bucket]:-}" ]] || {
    echo "Ошибка: Manifest не содержит обязательных метаданных."
    return 1
  }

  actual_size="$(stat -c '%s' -- "$ARCHIVE_PATH")"
  [[ "$actual_size" == "${fields[size_bytes]}" ]] || { echo "Ошибка: Размер архива не соответствует manifest."; return 1; }
  checksum_output="$(sha256sum -- "$ARCHIVE_PATH")"
  actual_checksum="${checksum_output%% *}"
  [[ "$actual_checksum" == "${fields[sha256]}" ]] || { echo "Ошибка: SHA-256 архива не соответствует manifest."; return 1; }

  BACKUP_VERIFICATION="manifest-v1"
  VERIFIED_SIZE="$actual_size"
  VERIFIED_SHA256="$actual_checksum"
}

verify_backup_artifact() {
  if [ -f "$MANIFEST_PATH" ]; then
    verify_manifest
  elif [ "$LEGACY_MODE" = "--allow-legacy-without-manifest" ]; then
    echo "ВНИМАНИЕ: восстановление legacy backup без checksum и manifest."
    BACKUP_VERIFICATION="legacy-unverified"
    VERIFIED_SIZE="$(stat -c '%s' -- "$ARCHIVE_PATH")"
    VERIFIED_SHA256=""
  else
    echo "Ошибка: Для архива отсутствует manifest: $MANIFEST_PATH"
    echo "Для осознанного восстановления старой копии укажите --allow-legacy-without-manifest."
    return 1
  fi

  tar -tzf "$ARCHIVE_PATH" > /dev/null || {
    echo "Ошибка: Архив поврежден или имеет неверный формат."
    return 1
  }
}

# ==========================================
# БЛОК МОНТИРОВАНИЯ
# ==========================================
mkdir -p "$MOUNT_POINT"

echo "[1/5] Подключение сетевой папки SMB..."
mount_smb
MOUNTED=1

if [[ ! -f "$ARCHIVE_PATH" || -L "$ARCHIVE_PATH" ]]; then
  echo "Ошибка: Файл $ARCHIVE_PATH не найден на сетевом диске."
  exit 1
fi

echo "Проверка manifest, размера, checksum и структуры архива..."
verify_backup_artifact
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
