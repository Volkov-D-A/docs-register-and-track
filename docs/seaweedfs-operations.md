# Эксплуатация SeaweedFS

Docflow использует SeaweedFS **4.46** и AWS SDK for Go v2. Проверку готовности,
экспорт и импорт выполняет `docflow-server storage`; отдельный S3 CLI не нужен.
[Релиз SeaweedFS](https://github.com/seaweedfs/seaweedfs/releases/tag/4.46)
и [описание mini](https://github.com/seaweedfs/seaweedfs/wiki/Quick-Start-with-weed-mini).

`weed mini -dir=/data -s3.port=8333` хранит данные и метаданные в одном постоянном
томе `seaweedfs_data`. Внутренний endpoint — `http://seaweedfs:8333`. Порты Master,
Filer, Volume, WebDAV, Admin и S3 не публикуются в пользовательскую сеть.
Desktop работает с HTTP API Docflow. Одноузловой пример не обеспечивает HA.

В проекте три Compose-файла: `docker-compose.yaml` для dev с локальной сборкой,
`docker-compose.prod.example.yaml` для production и `docker-compose.integration.yaml`
для тестов. В последнем профиль `smoke` дополнительно включает сервер;
обычные интеграционные тесты запускают только PostgreSQL, SeaweedFS и S3 probe.

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

Основной dev-стек всегда собирает сервер из текущих исходников.
Production-пример использует опубликованный образ. Запуск dev:

```bash
make storage-up
```

## Переход dev на пустые данные

Перенос старых объектов, metadata и outbox не выполняется. Нельзя подключать
старую PostgreSQL к пустому SeaweedFS. Перед сбросом остановите работу клиентов,
сохраните нужные внешние данные и проверьте Compose project.

1. Обновите код и `.env` по новому примеру.
2. Просмотрите точный список томов: `bash scripts/reset-dev-storage.sh`.
3. Для намеренного удаления dev-данных выполните `make storage-reset`.

Скрипт выбирает только тома текущего Compose project с labels `pgdata`,
`seaweedfs_data`, затем собирает текущий сервер из основного Compose и запускает чистый стек. Seq и Caddy volumes
сохраняются. Пустая БД получает обычные embedded migrations. Скрипт сброса
**не выполняется** интеграционными тестами и не запускался при переходе.

Для отката сохраните текущие изменения, используйте отдельный checkout предыдущей
версии и прежний `.env` с `MINIO_*`. Остановите текущий стек; удалите только его
PostgreSQL и object-storage volumes по проверенным labels, сохранив Seq/Caddy.
Запустите предыдущий Compose с пустыми PostgreSQL и MinIO. Возврат данных между
двумя вариантами и восстановление архивов v2 старым кодом не поддерживаются.

## Backup и restore

Основная процедура — [серверное резервирование и recovery](server-backup-operations.md).
SMB-подключение выполняется напрямую сервером; настройки и расписание находятся
в панели. Compose secrets передают ключ настроек, постоянный volume хранит staging.
Новый формат v3 включает checksum каждого объекта; legacy v2 читается автономной
командой с исходным manifest. MinIO SDK и CLI больше не используются.

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
Не запускайте несколько экземпляров одной цели одновременно. Smoke включает профиль `smoke` в интеграционном Compose и задаёт отдельные порты
через `DOCFLOW_TEST_POSTGRES_PORT`/`DOCFLOW_TEST_S3_PORT`. Его отчёты сохраняются в
`build/transition-evidence/`; это локальные артефакты, исключённые из Git.
CIFS mount в smoke заменён локальной директорией: SMB транспорт и устойчивость
сетевой записи требуют отдельной эксплуатационной проверки. Desktop GUI на
Linux и Windows проверяется отдельно от API smoke.
