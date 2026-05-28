# DB Backup Restore Review

Дата аудита: 2026-05-27

## Backup

`backup_smb_tar.sh`:

- читает `.env` по абсолютному пути, что подтверждено как допустимое для cron;
- монтирует SMB;
- делает `pg_dump -Fc`;
- зеркалит MinIO bucket через `mc mirror`;
- архивирует `database.dump` и `minio_files/`;
- удаляет архивы старше 15 дней.

Backup policy из этапа A: RPO 1 день, RTO 1-2 дня, retention 15 дней, manual test restore перед релизом, offsite copy на другой сервер, Seq не входит в backup.

## Restore

`restore_smb_tar.sh`:

- распаковывает архив;
- проверяет наличие `database.dump` и `minio_files/`;
- копирует dump в PostgreSQL container;
- запускает `pg_restore --clean --if-exists --exit-on-error`;
- пишет restore report;
- выполняет DB smoke validation через `psql -v ON_ERROR_STOP=1`;
- зеркалит MinIO с `--overwrite --remove` только после успешного PostgreSQL restore и smoke validation.

Целевой restore contract по `DECISION-006`:

- preflight до destructive действий: архив существует, содержит `database.dump` и `minio_files/`, PostgreSQL container доступен, target DB совпадает с ожидаемой;
- PostgreSQL restore выполняется до MinIO restore и останавливает workflow при fatal/unknown ошибке;
- ненулевой `pg_restore` допустим только для явно описанных warning-классов, иначе считается failed restore;
- MinIO mirror запускается только после успешной DB smoke validation;
- restore report фиксирует архив, время, exit codes, ключевые проверки и итоговый статус.

## Production Rollback Policy

По `DECISION-003` полный механизм управления миграциями через UI/runtime path сохраняется в production, включая rollback последней миграции. Поэтому backup/restore workflow становится обязательной страховкой перед любым rollback: перед откатом должен существовать свежий согласованный backup PostgreSQL+MinIO, а процедура эксплуатации должна явно предупреждать, что `down`-миграция может удалить данные.

## Issues

### ISSUE-010: restore continues after any pg_restore nonzero exit

Severity: major
Статус: fixed
Пункты: B.06.075
Место: `restore_smb_tar.sh`

Было: скрипт временно делал `set +e`, сохранял `RESTORE_CODE`, затем при любом ненулевом коде писал, что это обычно некритичные предупреждения, и продолжал восстановление MinIO. Это могло скрыть реальную ошибку восстановления БД.

Исправлено 2026-05-28:

- включен `set -euo pipefail`;
- добавлены preflight-проверки содержимого архива;
- `pg_restore` запускается с `--exit-on-error`;
- PostgreSQL restore и DB smoke validation выполняются до MinIO mirror;
- при ошибке PostgreSQL restore/validation workflow останавливается, а MinIO не восстанавливается;
- restore report пишется в `RESTORE_REPORT_DIR` или `./restore_reports`.

### SMB credentials exposure

Severity: major
Статус: fixed
Пункты: B.06.074
Место: `backup_smb_tar.sh`, `restore_smb_tar.sh`

Было: SMB password передавался в аргументах `mount`. Это уже зафиксировано в A как ops/security risk.

Исправлено: scripts use CIFS `credentials=...`; production can provide `SMB_CREDENTIALS_FILE`, otherwise scripts create a temporary credentials file with `0600` permissions and remove it through cleanup trap. Runbook documents env path, credentials file permissions and process-list release check.

## Findings

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| B.06.074 | issue | major | Backup описан скриптом, но manual test restore должен быть подтвержден перед релизом. |
| B.06.075 | ok | none | Restore script fail-fast: `pg_restore --exit-on-error`, DB smoke validation and restore report are in place; MinIO mirror starts only after successful DB restore/validation. |
| B.06.076 | ok | none | Test seed в миграциях не найден; default settings являются production defaults. |
