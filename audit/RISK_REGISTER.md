# Risk Register

Дата аудита: 2026-05-07
Этап: A. Базовый контекст проекта

## RISK-001

Категория: Database
Описание: Автоматическая нумерация документов (`nomenclature.next_number`) выполняется до транзакции создания документа. Локальная проверка подтвердила: после autocommit-инкремента и ошибочного insert документа счетчик стал `next_number=2`, а документ с выданным номером не создан. Также регистрация документов должна быть идемпотентной по backend `idempotency_key`, чтобы повторный submit не создавал дубль и не расходовал следующий номер.
Вероятность: high
Влияние: high
Митигирующее действие: По `DECISION-004` добавить `documents.idempotency_key UUID NOT NULL`, unique `(created_by, kind, idempotency_key)` и перенести проверку idempotency key, выдачу номера и создание документа в одну DB transaction.
Ответственный этап: C

## RISK-002

Категория: Storage
Описание: Вложения распределены между PostgreSQL metadata и MinIO object storage. Компенсация при ошибке есть при upload metadata failure, но полной cross-resource transaction нет.
Вероятность: medium
Влияние: high
Митигирующее действие: Recovery policy подтверждена: при рассинхронизации восстанавливаются одновременно PostgreSQL и MinIO из согласованного backup-набора. На этапах B/C проверить все сценарии upload/delete/bulk delete/download, а на B/H проверить согласованность backup/restore процедуры.
Ответственный этап: B, C

## RISK-003

Категория: Configuration
Описание: Production network/security policy предполагает закрытый защищенный LAN-контур без передачи данных по открытым каналам; production guide базируется на утвержденной документации вне проекта, ссылку на нее нельзя добавить в audit context. Repo examples остаются local/dev и используют local endpoints, disabled SSL и слабые placeholder passwords, которые допустимы только в закрытом контуре разработки.
Вероятность: medium
Влияние: high
Митигирующее действие: Production разворачивается вручную через `config/config.json` по утвержденной документации. На этапах B/E/H вручную сверить production configuration guide с утвержденной документацией, проверить LAN-границы закрытого контура, runtime defaults и секреты перед release; не использовать local/dev example values как production defaults.
Ответственный этап: E

## RISK-004

Категория: Operations
Описание: Backup/restore PostgreSQL+MinIO реализован shell scripts с SMB mount и Docker commands. Абсолютный путь к `.env` подтвержден как допустимый для cron, но production cron/deployment path и обращение с SMB credentials должны быть проверены; production restore не подтвержден.
Вероятность: medium
Влияние: high
Митигирующее действие: Backup policy подтверждена: RPO 1 день, RTO 1-2 дня, retention 15 дней, проверка архивов через ручной test restore PostgreSQL+MinIO перед релизом, offsite copy на другой сервер, Seq не бэкапится. На этапе B/H выполнить отдельный backup/restore review, проверить production cron path к `.env`, handling SMB credentials и то, что скрипты реально выполняют подтвержденную процедуру.
Ответственный этап: B, H

## RISK-005

Категория: Database
Описание: UI позволяет пользователю с системным правом `admin` запускать и откатывать миграции. Откат production migrations может быть разрушительным, если down migration теряет данные.
Вероятность: medium
Влияние: high
Митигирующее действие: По `DECISION-003` полный UI/runtime migration management сохраняется, включая rollback для `admin`; нужны destructive warning/confirmation, свежий backup PostgreSQL+MinIO перед rollback, audit entry и rollback-runbook.
Ответственный этап: B, H

## RISK-006

Категория: Access Control
Описание: Access model сочетает системные права, document permissions, подразделения, поручения и ознакомления. Ошибка в scope может дать слишком широкий или слишком узкий доступ.
Вероятность: medium
Влияние: high
Митигирующее действие: На этапах C/D покрыть access matrix тестами и UI smoke-сценариями.
Ответственный этап: C, D, F

## RISK-007

Категория: Performance
Описание: Frontend build предупреждает о большом основном чанке; фактические показатели нужно сверить с performance budget на ожидаемом объеме данных.
Вероятность: medium
Влияние: medium
Митигирующее действие: Performance budget и baseline объема подтверждены вручную: до 1000 документов в год, вложения пропорционально документам, до 20 пользователей, средний файл около 3 MB, storage около 1 TB, storage warning 80%, critical 90%. На этапе F измерить фактический старт, списки, отчеты, память и размер сборки на production-like данных.
Ответственный этап: F

## RISK-008

Категория: Build
Описание: Go package `main` зависит от существования `frontend/dist` из-за embedded assets; тесты/сборка root package падают до frontend build.
Вероятность: medium
Влияние: medium
Митигирующее действие: Зафиксировать build order в CI/release script: frontend build before Go/Wails build/test.
Ответственный этап: E, H

## RISK-009

Категория: Database
Описание: `document_journal` и `admin_audit_log` должны храниться весь жизненный цикл проекта и не удаляться на уровне приложения. Локальная проверка показала, что текущие `ON DELETE CASCADE` FK удаляют journal/audit строки при delete user/document. При росте объема данных lifetime retention также может повлиять на размер PostgreSQL, индексы, скорость фильтрации и длительность backup/restore; текущий baseline — до 1000 документов в год, до 20 пользователей и storage около 1 TB, поэтому storage thresholds 80%/90% в ближайшие годы математически маловероятны.
Вероятность: medium
Влияние: medium
Митигирующее действие: По `DECISION-005` не вводить application-level physical delete для users/documents и заменить cascade FK журналов на retention-safe strategy; на этапах F/H оценить рост БД, backup window и необходимость архивирования без удаления из production history.
Ответственный этап: B, F, H

## RISK-010

Категория: Database/Migrations
Описание: Destructive down migrations доступны через runtime rollback path для пользователя с `admin`, и по принятому требованию полный механизм управления миграциями сохраняется в production. Локальная проверка application migrator подтвердила: rollback version 7 -> 6 удаляет `admin_audit_log`. Ошибка эксплуатации может удалить production tables/data.
Вероятность: medium
Влияние: high
Митигирующее действие: Сохранить rollback, но усилить guardrails: destructive warning/confirmation, свежий backup PostgreSQL+MinIO перед rollback, audit entry, rollback-runbook, review каждой новой down migration на data-loss impact.
Ответственный этап: B, H

## RISK-011

Категория: Database/Performance
Описание: Representative EXPLAIN ANALYZE на 1000 documents показал быстрые планы, но частые списки документов, поручений и ознакомлений используют сложные filters/access EXISTS, search и OFFSET pagination; возможна деградация при росте данных или отличии production dataset от synthetic baseline.
Вероятность: medium
Влияние: medium
Митигирующее действие: Собрать representative data, выполнить EXPLAIN/EXPLAIN ANALYZE, добавить точечные composite/partial/trigram indexes, повторно проверить write latency.
Ответственный этап: B, F

## RISK-012

Категория: Database/Restore
Описание: Restore script продолжает работу после любого ненулевого `pg_restore`, поэтому может быть принят неполный restore.
Вероятность: medium
Влияние: high
Митигирующее действие: По `DECISION-006` сделать restore fail-fast для fatal/unknown `pg_restore` errors, не запускать MinIO mirror до успешной DB validation, формировать restore report и выполнять mandatory smoke validation после test restore PostgreSQL+MinIO.
Ответственный этап: B, H
