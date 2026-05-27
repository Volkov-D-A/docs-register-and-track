# Decisions

Дата аудита: 2026-05-27

## DECISION-001

Дата: 2026-05-27
Контекст: Пункт A.04.018 требует проверки отсутствия мертвого кода. В рамках этапа A уже выполнен high-level scan: очевидных orphan-модулей не найдено, fixed document type CRUD признан намеренно отключенным поведением. Глубокая dead-code проверка требует статанализа и выходит за цель этапа A, который фиксирует базовый контекст проекта.
Решение: Завершить A.04.018 в рамках этапа A как `not_applicable` для глубокой проверки и перенести dedicated dead-code/static-analysis review на последующие этапы статанализа/code quality.
Причина: Этап A не должен превращаться в глубокий аудит backend/frontend/code quality; перенос сохраняет границы этапов и не скрывает оставшуюся работу.
Альтернативы: Выполнить полный dead-code audit прямо в этапе A; оставить пункт `needs_info`.
Последствия: Этап A можно закрыть. На этапах E/F нужно отдельно проверить dead code, unused exports, generated mocks, неиспользуемые UI-компоненты и устаревшие CRUD-пути.
Какие этапы затрагивает: E, F

## DECISION-002

Дата: 2026-05-27
Контекст: В этапе B пункты B.04.051-B.04.052 требуют `EXPLAIN` и `EXPLAIN ANALYZE`, а B.04.064-B.05.065 требуют актуальной статистики и production-like данных. В текущем проходе доступна только статическая проверка кода, миграций и scripts.
Решение: Зафиксировать статический database audit сейчас, а live plan analysis, index usage stats и restore validation выполнить отдельной подзадачей этапа B после подготовки production-like PostgreSQL dataset.
Причина: Без representative data планы выполнения и статистика будут недостоверны.
Альтернативы: Пытаться делать выводы о планах только по SQL-коду; поднять синтетическую БД без согласованного dataset.
Последствия: Этап B начат и основные schema/transaction/ops риски зафиксированы. 2026-05-27 выполнен локальный representative `EXPLAIN ANALYZE` на synthetic dataset; финальный release gate все равно должен повторить планы на утвержденном production-like dataset и выполнить test restore.
Какие этапы затрагивает: B, F, H

## DECISION-003

Дата: 2026-05-27
Контекст: Runtime rollback path подтвержден: `SettingsService.RollbackMigration()` вызывает `db.RollbackMigration(database.DefaultMigrationsPath)`, а frontend `SettingsPage` предоставляет кнопку отката для пользователя с `admin`. Локальная проверка показала, что rollback миграции 007 переводит `schema_migrations` с `7,false` на `6,false` и удаляет `admin_audit_log`.
Решение: Полный механизм управления миграциями через интерфейс приложения сохраняется в production: статус, применение миграций вперед и rollback последней миграции остаются доступны пользователю с системным правом `admin`.
Причина: Для целевой эксплуатации требуется иметь полный runtime-инструмент управления миграциями из приложения. Риск destructive `down`-миграций принимается как управляемый операционный риск.
Альтернативы: Запретить production rollback через UI/runtime path и выполнять возврат назад только через backup/restore; оставить rollback только в dev/test. Эти варианты отклонены как не соответствующие требованию полного механизма управления миграциями.
Последствия: На этапах C/E/H нужно не удалять `RollbackMigration`, а усилить guardrails: явное destructive warning, подтверждение перед rollback, обязательная рекомендация/проверка свежего backup PostgreSQL+MinIO перед rollback, запись в `admin_audit_log`, документация rollback-runbook и release-gate test restore. Down migrations остаются потенциально destructive и должны проверяться отдельно.
Какие этапы затрагивает: B, C, E, H

## DECISION-004

Дата: 2026-05-27
Контекст: Подтверждено бизнес-правило strict no-gaps numbering и idempotent registration. Текущий `NomenclatureRepository.GetNextNumber` делает autocommit `UPDATE nomenclature SET next_number = next_number + 1`, после чего document repositories создают `documents` и detail rows в отдельных transactions. Локальный failure test подтвердил gap: после выданного номера insert документа упал, `next_number=2`, `docs=0`.
Решение: Целевая модель регистрации документов: `idempotency_key` обязателен для create-команд всех 4 видов документов. Так как старых документов нет, ключ хранится прямо в `documents.idempotency_key UUID NOT NULL` с unique constraint/index на `(created_by, kind, idempotency_key)`. Проверка idempotency key, выдача номера и создание `documents`, detail rows и дочерних записей выполняются в одной DB transaction. Повторный request с тем же `(created_by, kind, idempotency_key)` возвращает уже созданный документ и не инкрементирует `next_number`.
Причина: Только единая transaction boundary и unique idempotency contract защищают одновременно от gap после ошибки, double submit и race condition.
Альтернативы: Оставить только unique registration number; сделать frontend-only protection от double click; использовать sequence. Эти варианты не выполняют strict no-gaps invariant.
Последствия: Нужна миграция `008_registration_idempotency`: добавить `documents.idempotency_key UUID NOT NULL` и unique index/constraint `(created_by, kind, idempotency_key)`. Также нужны изменения request DTO/models/frontend payloads, перенос numbering logic из service-level `GetNextNumber` в repository transaction, тесты failure/concurrent/retry cases.
Какие этапы затрагивает: B, C, D, F

## DECISION-005

Дата: 2026-05-27
Контекст: Подтверждена lifetime retention policy для `document_journal` и `admin_audit_log`. В application code не найден штатный repository/service delete для корневых `documents` и `users`, пользователи управляются через `is_active`. Но schema-level FK сейчас использует `ON DELETE CASCADE`: локальная проверка подтвердила, что delete пользователя удаляет связанную запись `admin_audit_log`, а delete документа удаляет связанную запись `document_journal`.
Решение: Целевая retention strategy: `users` и корневые `documents` не удаляются физически на уровне приложения; для пользователей используется deactivate/reactivate, для документов физический delete не является штатной операцией. FK журналов должны защищать retention: заменить cascade на `RESTRICT`/эквивалентный retention-safe вариант. Если в будущем потребуется физическое удаление, оно должно идти через отдельную архивную/юридическую процедуру, а журналы должны хранить immutable snapshots.
Причина: Lifetime-журналы не должны исчезать из-за случайного delete или служебной операции. Текущее отсутствие UI-delete снижает вероятность, но не является DB-level защитой.
Альтернативы: Оставить `ON DELETE CASCADE` и полагаться на отсутствие application delete; заменить на `SET NULL` с денормализованными именами. Первый вариант слабее для retention; второй потребует nullable FK и snapshot-полей, что можно рассмотреть при появлении legal delete requirements.
Последствия: Нужна миграция изменения FK для `document_journal` и `admin_audit_log`, проверка деактивации пользователей, документных операций, rollback migrations и отчетов/журналов.
Какие этапы затрагивает: B, C, H

## DECISION-006

Дата: 2026-05-27
Контекст: `restore_smb_tar.sh` восстанавливает PostgreSQL, затем MinIO, но сейчас продолжает workflow после любого ненулевого `pg_restore`, считая это обычно некритичными предупреждениями. При этом rollback миграций через UI/runtime остается доступен в production по `DECISION-003`, поэтому backup/restore является обязательной страховкой, но не заменой механизма миграций.
Решение: Restore workflow должен быть fail-fast для неизвестных и fatal ошибок PostgreSQL restore. Ненулевой `pg_restore` нельзя blanket-классифицировать как warning: допустимые warning-классы должны быть явно перечислены, иначе выполнение останавливается до MinIO mirror. Целевой restore включает preflight archive/container checks, `pg_restore` с режимом остановки на ошибках, restore report, post-restore DB smoke validation и проверку согласованности PostgreSQL+MinIO.
Причина: Неполный restore БД опаснее явной остановки: MinIO может быть синхронизирован поверх некорректной БД, после чего оператор получит ложное ощущение успешного восстановления.
Альтернативы: Оставить текущую эвристику "любой nonzero = warning"; всегда восстанавливать MinIO независимо от результата БД. Эти варианты отклонены, потому что скрывают критические ошибки восстановления.
Последствия: Нужно доработать `restore_smb_tar.sh` и runbook: проверять наличие `database.dump` и `minio_files/`, останавливать workflow при fatal/unknown `pg_restore`, логировать restore report, выполнять smoke queries (`schema_migrations`, ключевые таблицы, базовая целостность), отдельно проверять controlled failure на поврежденном или несовместимом dump.
Какие этапы затрагивает: B, E, H
