# Open Questions

Дата аудита: 2026-05-07

## Вопросы, Влияющие На Следующие Этапы

На текущий момент открытых вопросов этапа A не осталось; оставшиеся пункты ниже являются проверками следующих этапов.

## Подтвержденные Ответы

- PostgreSQL version policy: версия PostgreSQL для production определяется утвержденной production-документацией и не фиксируется в репозитории. Значение `POSTGRES_VERSION` в `.envExample` считается local/dev примером, а не production contract.
- Network boundary policy: PostgreSQL, MinIO и Seq доступны только из LAN; доступ из открытых сетей не допускается.
- Attachment recovery policy: при рассинхронизации PostgreSQL attachment metadata и MinIO objects восстанавливаются одновременно PostgreSQL и MinIO из согласованного backup-набора; частичное восстановление только одной стороны не является штатной recovery-процедурой.
- Migration execution policy: для `RunMigrations` и `RollbackMigration` в production достаточно системного права `admin`; дополнительный confirmation/audit workflow не является обязательным требованием.
- Backup test restore policy: test restore PostgreSQL+MinIO выполняется вручную перед релизом.
- Backup archive/offsite policy: проверка backup-архивов выполняется через manual test restore; offsite copy реализован штатно, так как backup-скрипт отправляет копию на другой сервер.
- Production guide policy: production template/guide базируется на утвержденной документации за пределами проекта; ссылку на нее нельзя добавлять в репозиторий или audit context.
- Dev examples security policy: слабые примерные пароли в `.envExample` допустимы для local/dev шаблона, потому что разработка ведется в закрытом контуре и доступ извне к ресурсам невозможен; эти значения не являются production defaults.
- Production config policy: production разворачивается вручную через `config/config.json` по утвержденной production-документации; отдельные repo-конфиги для `dev`, `test`, `staging`, `production` не являются обязательным требованием.
- Expected data volume: ориентировочно не более 1000 документов в год, количество вложений пропорционально документам, не более 20 пользователей, средний размер файла около 3 MB, доступный storage около 1 TB.
- Storage monitoring policy: при доступном storage около 1 TB warning threshold — 80%, critical threshold — 90%; при текущем baseline достижение этих порогов в ближайшие годы считается математически маловероятным.
- Audit/journal access policy: `document_journal` и `admin_audit_log` не удаляются на уровне приложения; экспорт не предполагается; отдельная фильтрация не требуется; изменение возможно только прямым доступом к БД, которого в штатной эксплуатации нет.
- Document type policy: document types в ближайшее время не планируется расширять; они остаются фиксированными в коде, а не переводятся в справочник.
- Public app settings contract: `organization_short_name` для общего layout отдается через `SettingsService.GetOrganizationShortName()`, доступный всем авторизованным пользователям. Сами настройки не считаются секретом для чтения, но изменять их может только пользователь с системным правом `admin`.
- Document registration idempotency policy: регистрация документа должна поддерживать backend `idempotency_key`. Повторный запрос с тем же ключом должен возвращать уже созданный документ и не должен создавать второй документ, повторно брать регистрационный номер или инкрементировать `nomenclature.next_number`.
- Backup/restore script path policy: абсолютный путь к `.env` в `backup_smb_tar.sh` и `restore_smb_tar.sh` допустим и используется из-за особенностей запуска через cron; скрипты не обязаны работать из произвольной рабочей директории, но фактический путь должен совпадать с production cron/deployment path по утвержденной документации.

## Файлы, Обязательные Для Этапа B

Передавать ИИ на этапе B обязательно:

- `audit/00_project_context/PROJECT_CONTEXT.md`
- `audit/00_project_context/BUSINESS_RULES.md`
- `audit/00_project_context/ARCHITECTURE_MAP.md`
- `audit/00_project_context/ASSUMPTIONS.md`
- `audit/00_project_context/OPEN_QUESTIONS.md`
- `audit/RISK_REGISTER.md`
- `internal/database/postgres.go`
- `internal/database/migrations_embed.go`
- `internal/database/migrations/*.up.sql`
- `internal/database/migrations/*.down.sql`
- `internal/repository/*.go`
- `internal/services/*command_handler.go`
- `internal/services/settings.go`
- `internal/services/auth_service.go`
- `internal/services/document_access_service.go`
- `config.example.json`
- `.envExample`
- `docker-compose.yaml`
- `backup_smb_tar.sh`
- `restore_smb_tar.sh`
- утвержденная production-документация по конфигурации и эксплуатации проверяется вручную; ссылку на нее нельзя добавлять в репозиторий или audit context

## PostgreSQL-Риски Для Этапа B

- проверить и обеспечить strict no-gaps атомарность `nomenclature.next_number` относительно создания документов;
- проверить уникальность регистрационных номеров по году и совместимость `EXTRACT(YEAR FROM registration_date)` с индексами/планами;
- проверить транзакции create/update документов и дочерних таблиц;
- проверить внешние ключи и `ON DELETE` для потери истории;
- проверить индексы для фильтров списков, access scope, поручений, ознакомлений и журнала;
- проверить миграции `down` на сохранность production data;
- проверить initial setup и миграции при пустой/частично созданной БД;
- проверить backup/restore PostgreSQL вместе с MinIO.
