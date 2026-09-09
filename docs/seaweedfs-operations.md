# Эксплуатация SeaweedFS

Docflow использует SeaweedFS **4.46**, `minio-go/v7` **v7.3.0** и S3 CLI
`minio/mc:RELEASE.2025-08-13T08-35-41Z`. SDK и CLI являются совместимыми
S3-клиентами. [Релиз SeaweedFS](https://github.com/seaweedfs/seaweedfs/releases/tag/4.46)
и [описание mini](https://github.com/seaweedfs/seaweedfs/wiki/Quick-Start-with-weed-mini).

`weed mini -dir=/data -s3.port=8333` хранит данные и метаданные в одном постоянном
томе `seaweedfs_data`. Внутренний endpoint — `http://seaweedfs:8333`. Порты Master,
Filer, Volume, WebDAV, Admin и S3 не публикуются в пользовательскую сеть.
Desktop работает с HTTP API Docflow. Одноузловой пример не обеспечивает HA.

## Конфигурация и первый запуск

Скопируйте [.envExample](../.envExample) в новый `.env` и задайте `S3_ENDPOINT`,
`S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_BUCKET` и при необходимости
`S3_USE_SSL`. Схема `http://` или `https://` имеет приоритет над SSL-флагом.
Переменные `MINIO_*` больше не читаются. SDK использует Static V4,
регион `us-east-1` и path-style URL.

SeaweedFS получает credentials через `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`.
Не задавайте одновременно альтернативные identities-файлы или `S3_BUCKET`
самому SeaweedFS: bucket приложения создаёт сервер. `s3-ready` проверяет S3v4,
создание временного bucket, запись, чтение и удаление до запуска приложения.
Read-only проверка приложения никогда не создаёт bucket.

Для production используйте [пример Compose](../docker-compose.prod.example.yaml):
`S3_ACCESS_KEY_FILE_PATH` и `S3_SECRET_KEY_FILE_PATH` указывают на внешние файлы
runtime secrets. Они монтируются в `/run/secrets`; сервер читает их через
`S3_ACCESS_KEY_ID_FILE`/`S3_SECRET_ACCESS_KEY_FILE`. Не задавайте одновременно
секрет и его `_FILE`. Учтите доступ на чтение для UID 65532 сервера: локальные
Compose file secrets сохраняют права исходного файла. Храните их вне Git,
в закрытом каталоге; настройте владельца/ACL для runtime пользователя.

Проверка текущих исходников вместо опубликованного образа:

```bash
docker compose -f docker-compose.yaml -f docker-compose.local-build.yaml up -d --build
```

## Переход dev на пустые данные

Перенос старых объектов, metadata и outbox не выполняется. Нельзя подключать
старую PostgreSQL к пустому SeaweedFS. Перед сбросом остановите работу клиентов,
сохраните нужные внешние данные и проверьте Compose project.

1. Обновите код и `.env` по новому примеру.
2. Просмотрите точный список томов: `bash scripts/reset-dev-storage.sh`.
3. Для намеренного удаления dev-данных выполните `make storage-reset`.

Скрипт выбирает только тома текущего Compose project с labels `pgdata`,
`minio_data`, `seaweedfs_data`, затем собирает текущий сервер через local-build override и запускает чистый стек. Seq и Caddy volumes
сохраняются. Пустая БД получает обычные embedded migrations. Скрипт сброса
**не выполняется** интеграционными тестами и не запускался при переходе.

Для отката сохраните текущие изменения, используйте отдельный checkout предыдущей
версии и прежний `.env` с `MINIO_*`. Остановите текущий стек; удалите только его
PostgreSQL и object-storage volumes по проверенным labels, сохранив Seq/Caddy.
Запустите предыдущий Compose с пустыми PostgreSQL и MinIO. Возврат данных между
двумя вариантами и восстановление архивов v2 старым кодом не поддерживаются.

## Backup и restore

Настройте [backup.env.example](../backup.env.example), в том числе
`DOCFLOW_SERVER_CONTAINER`, `POSTGRES_CONTAINER` и `S3_NETWORK` реального Compose
project. S3 endpoint должен разрешаться внутри этой сети; `localhost` контейнера
не указывает на SeaweedFS. На хосте нужны Docker Compose V2, bash, Python 3,
GNU tar/coreutils и cifs-utils. `pg_dump`/`pg_restore` запускаются в PostgreSQL
контейнере, `mc` — в закреплённом образе.

```bash
sudo bash backup_smb_tar.sh
sudo bash restore_smb_tar.sh backup_20260909_120000_123456789.tar.gz
```

Backup останавливает сервер и outbox, делает PostgreSQL dump и S3 mirror,
публикует архив и manifest v2 атомарно, затем возобновляет ранее работавший сервер.
При внешних писателях остановите их тоже. Формат: `database.dump`, `objects/`,
manifest с `s3_bucket`, размером и SHA-256. Секреты записываются в JSON с правами
0600 во временном каталоге 0700, удаляемом при выходе.

Restore предназначен для пустого целевого стека. Он проверяет manifest/checksum,
останавливает сервер, восстанавливает и проверяет PostgreSQL, затем выполняет
S3 mirror. При любой ошибке restore ранее работавший сервер остаётся остановленным;
изучите отчёт, исправьте причину и повторите восстановление согласованного набора.
После успеха сервер возобновляется, только если работал до restore. Отчёты по
умолчанию — `/var/log/docflow/restore_reports`, вне репозитория. Формат v1 и
архивы без manifest отклоняются.

## Воспроизводимые проверки

- `make integration-test` — реальные PostgreSQL и S3; UUID buckets, multipart
  65 MiB, checksum, отмена, 1005 объектов, HTTP lifecycle, compensation,
  reconciliation, outbox retry и координация статистики.
- `make storage-smoke-test` — отдельный project `docflow-smoke`, сборка сервера,
  пустые тома, HTTP upload/download/delete, рестарт и пересоздание контейнера SeaweedFS, backup/restore в
  новые пустые тома, сверка содержимого и отсутствие missing/orphan.
  Повреждённый PostgreSQL dump должен завершить restore до создания S3 bucket.
- `make go-test go-vet frontend-lint frontend-test frontend-build docs-links-check`.

Оба Compose test project удаляют только свои тестовые тома через EXIT trap.
Порты integration — 55432/58333, smoke — 55433/58334/58080, только loopback.
Не запускайте несколько экземпляров одной цели одновременно. Smoke требует
Compose с поддержкой `!override` (2.24.4+). Его отчёты сохраняются в
`build/transition-evidence/`; это локальные артефакты, исключённые из Git.
CIFS mount в smoke заменён локальной директорией: SMB транспорт и устойчивость
сетевой записи требуют отдельной эксплуатационной проверки. Desktop GUI на
Linux и Windows проверяется отдельно от API smoke.
