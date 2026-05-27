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
- копирует dump в PostgreSQL container;
- запускает `pg_restore --clean --if-exists`;
- зеркалит MinIO с `--overwrite --remove`.

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
Пункты: B.06.075
Место: `restore_smb_tar.sh`

Скрипт временно делает `set +e`, сохраняет `RESTORE_CODE`, затем при любом ненулевом коде пишет, что это обычно некритичные предупреждения, и продолжает восстановление MinIO. Это может скрыть реальную ошибку восстановления БД.

Рекомендация:

- по `DECISION-006` сделать restore fail-fast для fatal/unknown ошибок;
- отдельно классифицировать ожидаемые warnings, не считать любой nonzero безопасным;
- писать restore report;
- выполнять manual test restore и smoke validation до MinIO mirror и после полного восстановления.

### SMB credentials exposure

Severity: major
Пункты: B.06.074
Место: `backup_smb_tar.sh`, `restore_smb_tar.sh`

SMB password передается в аргументах `mount`. Это уже зафиксировано в A как ops/security risk.

Рекомендация: credentials file с ограниченными правами или другой approved secret handling.

## Findings

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| B.06.074 | issue | major | Backup описан скриптом, но manual test restore должен быть подтвержден перед релизом. |
| B.06.075 | issue | major | Restore script не отличает warnings от fatal errors; целевой fail-fast contract зафиксирован в `DECISION-006`. |
| B.06.076 | ok | none | Test seed в миграциях не найден; default settings являются production defaults. |
