# Backup and Restore Runbook

Дата обновления: 2026-05-28

## Назначение

`backup_smb_tar.sh` создает согласованный архив PostgreSQL dump и MinIO files на SMB share.
`restore_smb_tar.sh` восстанавливает PostgreSQL, проверяет БД и только затем зеркалит MinIO.

Seq logs не входят в backup.

## Production Contract

- RPO: 1 день.
- RTO: 1-2 дня.
- Retention: 15 дней на SMB share.
- Offsite copy: отдельный сервер по утвержденной внешней production-документации.
- Перед релизом обязателен manual test restore PostgreSQL+MinIO на целевом контуре или production-like контуре.

## Environment File

Скрипты читают:

```text
/home/dimas/projects/docs-register-and-track/.env
```

Этот абсолютный путь является частью cron/deployment contract. Если production path меняется, нужно менять путь в обоих скриптах и в cron job одновременно.

Файл `.env` должен:

- храниться вне Git;
- иметь права `600`;
- быть доступен только пользователю, который запускает backup/restore;
- содержать актуальные `POSTGRES_*`, `MINIO_*`, `SMB_*`, `MOUNT_POINT`.

## SMB Credentials

Предпочтительный способ для production:

```text
SMB_CREDENTIALS_FILE=/etc/docflow/smb-backup.credentials
```

Файл credentials должен иметь права `600` и содержать:

```text
username=<smb-user>
password=<smb-password>
domain=<smb-domain>
```

Если `SMB_CREDENTIALS_FILE` не задан, скрипты создают временный credentials file в `/tmp`, используют его для `mount -t cifs -o credentials=...` и удаляют через `cleanup` trap. Пароль не передается в аргументах `mount`.

## Backup

Запуск:

```bash
sudo ./backup_smb_tar.sh
```

Скрипт:

1. Монтирует SMB share.
2. Делает `pg_dump -Fc`.
3. Зеркалит MinIO bucket в temp dir.
4. Создает `backup_YYYYMMDD_HHMMSS.tar.gz` на SMB share.
5. Удаляет backup archives старше 15 дней.
6. Удаляет temp dir и временный credentials file при любом завершении.

## Restore

Запуск:

```bash
sudo ./restore_smb_tar.sh backup_YYYYMMDD_HHMMSS.tar.gz
```

Скрипт:

1. Монтирует SMB share.
2. Проверяет наличие архива, `database.dump` и `minio_files/`.
3. Запускает `pg_restore --clean --if-exists --exit-on-error`.
4. Выполняет DB smoke validation через `psql -v ON_ERROR_STOP=1`.
5. Только после успешной БД зеркалит MinIO.
6. Пишет restore report в `RESTORE_REPORT_DIR` или `./restore_reports`.
7. Удаляет temp dir и временный credentials file при любом завершении.

Если PostgreSQL restore или validation завершается ошибкой, MinIO restore не запускается.

## Release-Gate Checks

- `bash -n backup_smb_tar.sh`
- `bash -n restore_smb_tar.sh`
- проверить права `.env` и `SMB_CREDENTIALS_FILE`;
- проверить, что `ps` во время mount не показывает SMB password в аргументах;
- прервать backup/restore на каждом этапе и убедиться, что temp dirs удалены;
- выполнить успешный test restore PostgreSQL+MinIO;
- выполнить controlled failure restore с поврежденным/несовместимым dump и убедиться, что MinIO не был изменен.
