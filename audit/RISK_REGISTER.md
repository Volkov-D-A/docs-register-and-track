# Risk Register

Дата аудита: 2026-05-07
Этап: A. Базовый контекст проекта

## RISK-001

Категория: Database
Описание: Автоматическая нумерация документов (`nomenclature.next_number`) выполняется до транзакции создания документа. При ошибке после получения номера возможен пропуск номера. Также регистрация документов должна быть идемпотентной по backend `idempotency_key`, чтобы повторный submit не создавал дубль и не расходовал следующий номер.
Вероятность: high
Влияние: high
Митигирующее действие: На этапе B изменить/проверить транзакционную модель регистрации так, чтобы инкремент `next_number` происходил только вместе с успешным созданием документа, и добавить/проверить backend `idempotency_key` для повторных запросов регистрации.
Ответственный этап: B

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
Митигирующее действие: Право запуска/rollback подтверждено только для `admin`; дополнительный confirmation/audit workflow не обязателен. На этапе B проверить все down migrations на сохранность production data.
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
Описание: `document_journal` и `admin_audit_log` должны храниться весь жизненный цикл проекта и не удаляться на уровне приложения. Экспорт и отдельная фильтрация не требуются; изменение возможно только прямым доступом к БД, которого в штатной эксплуатации нет. При росте объема данных это может повлиять на размер PostgreSQL, индексы, скорость фильтрации и длительность backup/restore; текущий baseline — до 1000 документов в год, до 20 пользователей и storage около 1 TB, поэтому storage thresholds 80%/90% в ближайшие годы математически маловероятны.
Вероятность: medium
Влияние: medium
Митигирующее действие: На этапе B проверить отсутствие application-level delete/update операций для журналов, индексы и планы запросов, а на этапах F/H оценить рост БД, backup window и необходимость архивирования без удаления из production history.
Ответственный этап: B, F, H
