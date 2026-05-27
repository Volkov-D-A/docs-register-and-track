# Review Log

Дата аудита: 2026-05-07
Этап: A. Базовый контекст проекта

## ISSUE-001

Категория: Configuration
Пункт плана: A.05.021, A.05.022, A.05.023
Severity: major
Статус: open
Место: `config.example.json`, `.envExample`, `docker-compose.yaml`
Проблема: В репозитории есть только local/dev-oriented configuration examples: `localhost`, `sslmode: disable`, MinIO `useSSL: false`, Seq по `http://localhost`, слабые примерные пароли в `.envExample`. Пользователь подтвердил, что weak example passwords допустимы для закрытого контура разработки, но production template/guide базируется на утвержденной документации за пределами проекта, ссылку на нее нельзя добавить в audit context.
Почему важно: Текущие defaults безопасны только как local/dev шаблон в закрытом контуре разработки. Пользователь подтвердил, что production разворачивается вручную через `config/config.json` по утвержденной документации, поэтому production-конфигурацию, границы контура, секреты и runtime defaults нужно сверять вручную.
Рекомендация: На этапах B/E/H сверить production config/ops с утвержденной документацией, явно пометить текущие examples как local-only и не использовать их как production defaults.
Проверка после исправления: Проверить production config/ops по утвержденной документации и отсутствие слабых defaults в фактическом production path.
Связанные пункты: B.06, E.01, E.02, H.02

## ISSUE-002

Категория: Configuration
Пункт плана: A.05.021, A.05.022
Severity: major
Статус: open
Место: `backup_smb_tar.sh`, `restore_smb_tar.sh`
Проблема: Скрипты backup/restore читают `.env` по абсолютному пути `/home/dimas/projects/docs-register-and-track/.env` и передают SMB password в аргументах `mount`. Пользователь подтвердил, что абсолютный путь к `.env` нужен из-за особенностей запуска через cron. Backup policy подтверждена отдельно: RPO 1 день, RTO 1-2 дня, retention 15 дней, проверка архивов через ручной test restore PostgreSQL+MinIO перед релизом, offsite copy на другой сервер, Seq не бэкапится.
Почему важно: Абсолютный env path допустим для cron, но должен совпадать с production cron/deployment path по утвержденной документации. Секреты могут попадать в process list/shell history/операционные логи в зависимости от окружения из-за передачи SMB password в аргументах `mount`.
Рекомендация: Документировать production cron запуск и путь к `.env`, сверить его с утвержденной документацией, рассмотреть credentials file для CIFS и минимизацию exposure секретов.
Проверка после исправления: Запустить manual test restore PostgreSQL+MinIO перед релизом из production cron/deployment path и сверить соблюдение RPO/RTO/retention.
Связанные пункты: B.06, E.01, H.03

## ISSUE-003

Категория: Architecture
Пункт плана: A.03.013
Severity: minor
Статус: open
Место: `frontend/src/store/useAuthStore.ts`, `internal/services/dashboard_service.go`
Проблема: UX-классификация профиля пользователя дублируется во frontend `resolveUserProfile` и backend `determineDashboardProfile`.
Почему важно: `admin`/`clerk`/`executor`/`mixed` больше не являются бизнес-ролями и не должны становиться источником прав. При изменении permission model frontend и backend могут разойтись, что приведет к неверной навигации, release-note behavior или dashboard profile.
Рекомендация: Сделать backend access summary единственным источником profile classification или добавить тест/контракт на синхронизацию.
Проверка после исправления: Проверить одинаковые профили для admin/clerk/executor/mixed cases.
Связанные пункты: C.01, D.03

## ISSUE-004

Категория: Build/Performance
Пункт плана: A.01.007
Severity: minor
Статус: open
Место: `frontend` production build
Проблема: `npm run build` проходит, но Vite предупреждает о большом основном чанке `index-HhgWsiDR.js` около 3000.72 kB, gzip около 872.68 kB.
Почему важно: Для desktop Wails это не обязательно blocker, но влияет на старт frontend и будущий performance budget.
Рекомендация: На frontend performance stage решить, нужен ли code splitting или увеличить budget осознанно.
Проверка после исправления: Повторить `npm run build` и сравнить chunk report.
Связанные пункты: D.07, F.06

## ISSUE-005

Категория: Frontend/Configuration
Пункт плана: A.01.003, A.05.021
Severity: minor
Статус: open
Место: `frontend/src/components/MainLayout.tsx`
Проблема: Название организации в сайдбаре захардкожено как "УСЗН Озерск".
Почему важно: Для production-инсталляции название должно соответствовать системной настройке `organization_short_name`; иначе UI может показывать неверный бренд/организацию после первичной настройки.
Рекомендация: Брать название сайдбара из `GetPublicAppSettings()`, доступного всем авторизованным пользователям; чтение настроек не считается секретным, но изменение настроек должно оставаться только для пользователя с системным правом `admin`. Использовать fallback на название продукта.
Проверка после исправления: Изменить `organization_short_name` и убедиться, что сайдбар обновляет название после входа/перезапуска.
Связанные пункты: D.01, D.03, H.01

## ISSUE-006

Категория: Database
Пункт плана: A.02.009
Severity: major
Статус: open
Место: `internal/repository/nomenclature_repo.go`, document command handlers
Проблема: Бизнес-правило требует строгую нумерацию без пропусков, но `GetNextNumber` увеличивает `nomenclature.next_number` до транзакции создания документа. Пользователь также подтвердил, что регистрация должна быть идемпотентной по backend `idempotency_key`.
Почему важно: Если после получения номера регистрация завершится ошибкой, номер будет потерян. Если повторный submit создаст второй документ, появится дубль и может быть израсходован следующий регистрационный номер. Это нарушает подтвержденные бизнес-инварианты.
Рекомендация: На этапе B включить получение/инкремент номера в ту же транзакцию, что и создание документа и дочерних записей, или применить эквивалентный механизм без пропусков. Добавить backend `idempotency_key`: повторный запрос с тем же ключом должен возвращать уже созданный документ без повторного создания и инкремента номера.
Проверка после исправления: Смоделировать ошибку после получения номера и убедиться, что следующий успешный документ получает тот же номер, без пропуска. Повторить запрос регистрации с тем же `idempotency_key` и убедиться, что возвращается тот же документ без дубля.
Связанные пункты: B.03, B.05, C.02
