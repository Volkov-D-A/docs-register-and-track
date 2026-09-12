# Границы и карта сценариев разделения

Снимок этапа 1 от 8 сентября 2026 года. Последовательность переноса — в
[плане](desktop-server-separation-plan.md), полный перечень экспортов и мест
вызова — в [карте Wails API](desktop-wails-api-baseline.md).

## Проверяемый исходный контракт

В двух регистрациях `internal/app/app.go` по 25 сервисов, 157 методов.
108 методов имеют прямые вызовы в production UI (включая динамические импорты,
переименования и вызовы через объект модуля; тесты UI исключены).
`internal/app/testdata/wails-api.json` фиксирует точные имена. Тест
`TestWailsAPIContract` сравнивает Go reflection, generated JS и TypeScript.
`TestCompositionRootStartupAndShutdownIntegration` сравнивает фактическую
runtime-регистрацию с регистрацией генератора. Он не требует PostgreSQL.

Снимок — верхняя граница на время миграции: при удалении экспорта удалять запись
в JSON вместе с обновлением UI/bindings. Добавление сеттеров и lifecycle-методов
не является допустимым обновлением снимка. Сейчас служебных методов 26;
`SetTheme` — пользовательская операция. Остальные методы без UI-вызовов требуют
разбора по внутренним Go-потребителям, а не автоматического удаления.

`internal/architecture/imports_test.go` автоматически включает все новые пакеты
`internal/desktop/**`, вложенные `internal/server/**` и общие dto, models,
operations, observability, releaseassets. Проверяется полный транзитивный `Deps`
из `go list`, только production-импорты. Старый корневой `internal/server` пока
смешанный и будет включён в окончательное правило на этапе 28; его новые
подпакеты исключений не получают. Новым пакетам запрещён обход через старые
services/app; desktop запрещена серверная инфраструктура, server — desktop,
старый serverclient и Wails, общим пакетам — обе стороны.

Проверки входят в штатный `make go-test` через `./internal/...`, а значит и в
release gate. Отдельной ручной регистрации каждого переносимого пакета нет.

## Владельцы support-пакетов и порядок зависимостей

Получено через `GOCACHE=/tmp/go-build-cache go list -deps -json ./...`.
В колонке зависимостей приведены прямые внутренние production-импорты.

| Текущий пакет | Зависимости | Владелец / действие |
| --- | --- | --- |
| models | нет | Общие модели, оставить |
| dto | models | Общие HTTP/UI-контракты, этапы 2–3 |
| observability | нет | Общие измерения, оставить |
| releaseassets | нет | Общие встроенные сведения о выпуске, оставить |
| coordination | нет | Серверные storage locks, этап 27 |
| security | нет | Серверные пароли и токены; перенести после удаления legacy auth |
| startupdiag | нет | Desktop-диагностика запуска, этап 25 |
| config | нет | Смешанная конфигурация, разделить на этапе 24 |
| database | config, observability | Сервер, контракт статуса вынести первым, перенос этап 26 |
| serverclient | database, dto, models | Desktop; удалить database на этапе 2, перенос этап 25 |
| logger | config, models, serverclient | Общая обработка + desktop-доставка; разделить на этапе 24 |
| background | database, models | Серверный coordinator worker/schema; контракт статуса этап 2, перенос этап 27 |
| app | background, config, database, logger, models, observability, releaseassets, serverclient, services, startupdiag | Desktop composition root; убрать schema-связи, этап 25 |
| services | coordination, database, dto, models, observability, security, serverclient | Разделить по этапам 4–23; общий lifecycle → operations |
| repository | coordination, database, models, security | Сервер, этап 26 |
| storage | config | Сервер, после разделения config, этап 27 |
| outbox | models, observability, repository | Сервер, после repository, этап 27 |
| mocks | models | Тестовые doubles; обновлять вместе с ports на этапе 11 |
| testutil/integrationdb | database | Тестовый support, обновить на этапе 26 |

Сначала независимые контракты и operations; затем desktop auth и простые
адаптеры; затем server ports/effects и политика доступа; затем предметные
сервисы; затем config/logging, composition roots и инфраструктура. Нумерация
документов принадлежит серверу; нормализация имени вложения — отдельному общему
пакету этапа 14. Универсальный пакет helpers не создаётся.

Исходный граф: сервер получает Wails runtime через services; desktop получает
migration/database через serverclient, app и background. Это зависимости
компиляции, а не прямые подключения desktop к PostgreSQL.

## Карта production-сценариев и legacy-тестов

Пути ниже указаны относительно `internal/`. Наличие теста означает найденное
покрытие, а не утверждение о выполнении PostgreSQL-теста на этом этапе.

| Сценарий | Production-путь / имеющееся покрытие | Legacy и работа перед удалением |
| --- | --- | --- |
| Login, bearer /me, logout | server/auth.go; auth_test.go `TestAuthLoginCreatesHashedSessionAndBearerAuthenticatesMe`, `TestAuthMeRejectsMissingBearerToken`; auth_integration_test.go `TestServerAuthSessionLifecycleIntegration` | services/auth_service_test.go Login, Logout, GetCurrentUser, IsAuthenticated: перенести отрицательные login-сценарии и повторный logout |
| Неверный пароль, пятая попытка, сброс счётчика, неактивный пользователь | Серверный login и auth repository | Legacy Login содержит эти подпункты; fakeAuthUsers возвращает фиксированный счётчик. На этапе 6 добавить HTTP-проверки накопления попыток/блокировки, сброса и ошибок хранилища |
| Обязательная/просроченная смена | server/auth_test.go `TestChangeRequiredPasswordUpdatesCredentials` | Legacy Login и ChangeRequiredPassword: запрет полного входа, срок пароля, неверный старый и слабый новый пароль — перенести отрицательные проверки |
| Смена пароля и отзыв всех сессий | auth_test.go `TestChangePasswordUsesBearerSessionAndInvalidatesOldCredentials`; auth_integration_test.go `TestServerPasswordChangeRevokesAllSessionsIntegration` | Legacy ChangePassword: отсутствие сессии, удалённый пользователь, неверный/слабый пароль |
| Профиль | server/profile_test.go `TestProfileAPIUpdatesBearerPrincipalWithAtomicAudit` | Legacy UpdateProfile: отсутствие сессии и ошибка repository; сохранить atomic audit |
| Bootstrap | server/initial_setup_test.go `TestInitialSetupAPIIsServerOwnedAndOneTime` | Legacy NeedsInitialSetup, InitialSetup: существующие пользователи, слабый пароль, ошибка repository. Не менять принятое решение по риску №2 |
| Principal и отзыв прав | server/document_commands_test.go `TestRequestDocumentPrincipalChecksSystemPermission`, statistics_test.go `TestRequestPrincipalChecksSessionPermissions`, users_test.go `TestUserAPIRequiresAdminPermission` | Legacy SessionPrincipalChecks, SystemPermissionChecks, GetCurrentAuditInfo: удаление/деактивация после login, any-permission, ошибка чтения; проверить матрицу на request principal, тестовый setup заменить fake principal |
| Maintenance | server/maintenance_test.go, management_test.go; desktop services/auth_service_test.go `TestAuthServiceSchemaLifecycleGate` | Сохранить доступ администратора к миграциям и блокировку бизнес-операций, не копировать серверную auth в desktop |
| Revision сессии, поздний ответ, 401 | serverclient/session_test.go и errors_test.go | Оставить desktop-проверки при этапах 6–7; тесты UsesRequiredServerSession/UsesServerForPasswordChanges/UsesServerForProfileUpdate уже проверяют HTTP-ветку |
| Собственное замещение | server/profile_test.go `TestProfileAPISelfSubstitutionUsesSessionUserAndAudit`, `TestProfileAPISubstitutionCandidatesReturnOnlyActiveUsers` | services/user_substitution_service_test.go UpdateMySubstitution: тот же отдел без participant, другой отдел, очистка — перенести всю матрицу через HTTP |
| Административное замещение | server/user_administration_test.go `TestUserSubstitutionAPIValidatesAndPersistsSameDepartmentSubstitute`; user_administration_integration_test.go `TestUserAdministrationAPIPersistsAccessAndSubstitutionWithAuditIntegration` | services/user_substitution_service_test.go `TestUserSubstitutionServiceUpdateUserSubstitutionPassesAuditEffectToAtomicStore`; repeated_audit_integration_test.go повторные set/clear — перенести на HTTP/PostgreSQL, затем удалить saveForPrincipal |
| Access summary | server/user_administration_test.go `TestCurrentAccessSummaryUsesBearerPrincipalOnServer` | services/document_kind_service_test.go: clerk, participant без full read, admin-only, admin maintenance, non-admin maintenance. Перенести на HTTP, затем удалить getCurrentAccessSummaryDirect |
| Администрирование прав | server/user_administration_test.go `TestUserAccessAPIReplacesProfileWithAtomicAudit`, `TestUserAccessAPIRejectsUnsupportedActionBeforePersistence`; PostgreSQL-тест выше | services/document_access_admin_service_test.go: профиль, недопустимое действие, отсутствующий пользователь; HTTP-адаптерные проверки сохранить при этапе 8 |

Private `saveForPrincipal` вызывается только тестовыми клиентами и тестом
повторного audit; `getCurrentAccessSummaryDirect` — тестовым клиентом
DocumentKindService. Production уже использует HTTP-обработчики. Этапы 5–6
должны переносить отсутствующие утверждения, затем удалять private-копии и
legacy setup; новый серверный AuthService не требуется.

## Изменение границ после этапа 2

Статус миграций теперь принадлежит dto, ошибка совместимости — models;
расчёт совместимости и embedded migrations остаются в database. Из production
serverclient, background и app удалены прямые импорты database. Desktop пока
сохраняет транзитивную связь через legacy AuthService в services до этапа 6.
Проверка архитектуры уже запрещает возврат database в serverclient/background.
Исходная таблица выше сохранена как снимок этапа 1.

## Актуализация после этапа 17

12.09.2026, код `563a111`; исходные таблицы выше остаются историческим снимком.
Desktop не зависит от database/storage/backup. Сервер пока получает старые
services и serverclient; окончательное закрытие composition root — этап 28.

| Пакет / зависимость | Владелец и действие |
| --- | --- |
| backup → config, database, storage, models, liveevents, backup/smb | Сервер; конфигурация этап 24, импорты database этап 26, перенос в server/backup этап 27 |
| backup/smb → models | Сервер; перенос вместе с backup, прямой SMB остаётся вне desktop |
| liveevents → sync | Серверная шина уведомлений об изменении данных; server/liveevents на этапе 27 |
| storage → config, AWS SDK v2 | Серверный SeaweedFS S3 adapter и helpers для backup/CLI; перенос этап 27 |
| serverclient/events → models | Desktop SSE, сессия и scoped operation status; перенос этап 25 |
| desktop/services/backup → models, serverclient | Пользовательские методы SettingsService, уже отделены |
| services/StatisticsService → refreshRunner | Сервер учитывает фоновые refresh через app.detached; конструктор и ожидание перед backup сохраняются на этапе 23 |
| server/app → backup Snapshot/Replace/Reload | Серверный maintenance, lease, остановка workers и замена DB pool; сохранить на этапах 26–27 |
| testutil/integrations3 → config, storage | Тестовая инфраструктура; обновить с этапами 24/27 |

До переноса backup/** и liveevents/** проверяются как серверные пакеты:
запрещены транзитивные desktop, serverclient, Wails и старые services/app.
Desktop и общие пакеты не могут импортировать backup/liveevents. Проверка
производственного графа автоматически охватывает их новые подпакеты.

Дополнительные сценарии для переноса:

- Backup: согласованный снимок БД/S3, SMB-передача, verify/delete, восстановление
  v2/v3, replacement/reset, recovery-required, смена пула без потери lease.
- Maintenance: ожидание активных запросов и detached refresh, остановка worker
  вместе со startup work, корректное возобновление после восстановления.
- SSE: доставка после сохранения user event, фильтрация admin topics, повторная
  проверка сессии, reconnect и отмена; scoped statusToken доступен во время
  восстановления и не передаётся через Wails events.
- Wails: новые пользовательские backup-методы уже внесены в JSON-снимок;
  служебные методы по-прежнему удаляются при разделении.

Покрытие: server/backup_test.go, restore_status_test.go, events_test.go,
serverclient/events_test.go и backup_test.go, background/lifecycle_test.go,
backup/*_test.go. Реальный storage/backup/restore проверяет
`make storage-smoke-test` с изолированными PostgreSQL/SeaweedFS/SMB;
`make integration-test` остаётся отдельной проверкой SQL и транзакций.
