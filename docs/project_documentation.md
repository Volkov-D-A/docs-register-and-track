# Документация проекта: Docflow

## 1. Назначение проекта

**Docflow** — настольная система регистрации и отслеживания документов на базе **Wails**. Приложение предназначено для ведения регистров документов, работы с поручениями, ознакомлениями, вложениями, номенклатурой дел и административными справочниками.

Ключевая особенность текущей версии: backend является источником истины для авторизации. Ограничения доступа не должны обеспечиваться только скрытием экранов во frontend.

## 2. Технологический стек

### Backend

- Go 1.24
- Wails v2
- PostgreSQL
- `github.com/lib/pq`
- миграции БД через `github.com/golang-migrate/migrate/v4`
- `github.com/google/uuid`
- `golang.org/x/crypto`
- `stretchr/testify`, `go-sqlmock`

### Frontend

- React 18
- TypeScript
- Vite
- Ant Design
- Zustand
- Day.js
- `@xyflow/react`

## 3. Архитектура

Проект построен как desktop-приложение с Go backend и React frontend.

Основные директории:

- `internal/database` — подключение к БД и миграции
- `internal/repository` — слой доступа к данным
- `internal/services` — бизнес-логика и авторизация
- `internal/models` — доменные модели
- `internal/dto` — DTO и маппинг
- `frontend/src` — UI и client-side state
- `docs` — проектная документация
- `main.go` — сборка зависимостей и binding сервисов в Wails

### Основные сервисы

- `AuthService` — логин, logout, текущий пользователь и его системные права
- `DocumentAccessService` — централизованная проверка доступа к document-domain по матрице прав, корневому документу и связанным сущностям доступа
- `DocumentKindService` — системные метаданные видов документов и доступных пользователю сценариев регистрации
- `UserService` — пользователи и пароли
- `DepartmentService` — подразделения
- `NomenclatureService` — дела номенклатуры
- `ReferenceService` — справочники типов документов, организаций и исполнителей резолюции
- `DocumentQueryService` — общий read-only API документов по `kindCode`
- `DocumentRegistrationService` — общий command API регистрации и обновления документов по `kindCode`
- `AttachmentService` — вложения и файловые операции
- `LinkService` — связи документов
- `AssignmentService` — поручения
- `AcknowledgmentService` — ознакомления
- `DashboardService` — дашборд по роли
- `SettingsService` — системные настройки и миграции
- `AdminAuditLogService` — административный аудит

### Document-domain модель

Текущая document-архитектура построена вокруг общей корневой сущности документа и системных видов документов:

- системные виды документов пока захардкожены в коде: `incoming_letter`, `outgoing_letter`
- backend-реестр видов документов хранится в `internal/models/document_kind.go`
- frontend использует тот же системный словарь через `frontend/src/constants/documentKinds.ts` и `DocumentKindService`
- `documents` — общий корень документа с полями `id`, `kind`, `nomenclature_id`, `registration_number`, `registration_date`, `document_type_id`, `content`, `pages_count`, `created_by`, `created_at`, `updated_at`
- `incoming_document_details` — типоспецифичные поля входящего документа
- `outgoing_document_details` — типоспецифичные поля исходящего документа

Связанные сущности больше не используют полиморфную пару `document_id + document_type` в БД:

- `assignments`
- `acknowledgments`
- `attachments`
- `document_journal`
- `document_links`

Все они ссылаются на единый `documents(id)`, а тип документа определяется через `documents.kind`.

На уровне DTO и UI связанные сущности тоже переведены на `documentKind`:

- `assignments.documentKind`
- `acknowledgments.documentKind`
- `document_links.sourceKind / targetKind`
- `graph.nodes.kindCode`

Это дает:

- нормальную ссылочную целостность на уровне БД
- более простой сервисный код без дублирующего `switch` по типу документа
- единый путь расширения document-domain для будущих типов документов

## 4. Модель авторизации

### Общие правила

- UI-ограничения не считаются механизмом безопасности
- document-domain и admin-domain проверяются на backend
- `admin` не является суперпользователем document-domain
- role-switching и `activeRole` полностью удалены
- права больше не рассчитываются через document-роли `clerk/executor`
- document-domain живет на прямой матрице пользовательских прав
- встроенные каналы доступа к конкретным документам не настраиваются отдельно в матрице:
  - номенклатура подразделения
  - поручения
  - ознакомления
- отдельный флаг пользователя `isDocumentParticipant` определяет участие в документообороте
- централизованные проверки доступа к document-domain вынесены в `DocumentAccessService`
- document read/write flow проходит через `DocumentQueryService` и `DocumentRegistrationService`
- legacy-значения `incoming/outgoing` нормализуются в системные `incoming_letter/outgoing_letter` через `NormalizeDocumentKind(...)`
- сервисы вложений, связей, поручений, ознакомлений и журнала не должны реализовывать собственные разрозненные policy-проверки, если можно переиспользовать `DocumentAccessService`

## 5. Права доступа

### Системные права

Системные права не связаны напрямую с document-domain и управляют только системными разделами:

- `admin` — пользователи, подразделения, настройки, миграции, административный аудит
- `references` — раздел `Справочники`
- `stats_incoming` — статистика по входящим письмам
- `stats_outgoing` — статистика по исходящим письмам
- `stats_assignments` — статистика по поручениям
- `stats_system` — системная статистика

Важно:

- `admin` не дает доступа к `Справочникам`
- `admin` не дает доступа к статистике
- document-domain права назначаются независимо от системных прав

### Участие в документообороте

У пользователя есть отдельный признак `isDocumentParticipant`.

Он означает:

- пользователь может быть исполнителем поручений
- пользователь может быть участником ознакомления
- пользователь может видеть document-разделы интерфейса
- пользователь может получать доступ к конкретным документам по встроенным каналам

Если пользователь не является участником документооборота:

- он не попадает в списки выбора исполнителей и ознакомляемых
- он не получает встроенный доступ к документам по каналам
- при отсутствии явных document-rights document-разделы ему не показываются

### Матрица document-domain

Матрица document-domain хранит только действия по виду документа.

Поддерживаемые действия:

- `read` — глобальный просмотр всех документов данного вида
- `create` — регистрация документов данного вида
- `update` — редактирование документов данного вида
- `delete` — удаление документов данного вида
- `assign` — управление поручениями в карточке документа
- `acknowledge` — управление ознакомлениями в карточке документа
- `upload` — загрузка и удаление файлов
- `link` — управление связями документов
- `view_journal` — просмотр журнала действий

### Встроенные каналы доступа

Отдельная настройка visibility channels удалена. Доступ к конкретным документам без глобального `read` обеспечивается встроенными правилами:

- номенклатура подразделения пользователя
- поручение пользователю
- ознакомление пользователя

Это означает:

- `read` дает полный просмотр реестра документов данного вида
- без `read` пользователь все равно может видеть конкретные документы, попавшие к нему по встроенным каналам
- для этого пользователь должен быть `isDocumentParticipant = true`

### Практическая семантика прав

- `read` — видеть весь реестр вида документа
- `create` — регистрировать документы
- `update` — редактировать доступные документы
- `upload` — загружать и удалять файлы
- просмотр и скачивание файлов не требуют `upload`, а зависят только от доступности самого документа
- `assign` — управлять поручениями в карточке документа
- `acknowledge` — управлять ознакомлениями в карточке документа
- `link` — управлять связями документов
- `view_journal` — видеть журнал документа

### Карточка документа

Для пользователя с доступом к документу вкладки карточки разделены так:

- `Информация` — доступна всегда, если карточка открылась
- `Файлы` — доступны всегда, если документ доступен пользователю
- `Поручения` — только при `assign`
- `Ознакомление` — только при `acknowledge`
- `Связи` — только при `link`
- `Журнал` — только при `view_journal`

Таким образом:

- простой пользователь работает со своими поручениями и ознакомлениями через отдельные экраны
- управляющие вкладки в карточке документа не открываются только по `read`

## 6. Реализованные изменения по безопасности доступа

На текущем этапе уже внедрено:

- отказ от `activeRole` в backend и frontend
- удаление document-ролей из модели пользователя
- вынос `admin` из document-domain
- прямые пользовательские matrix permissions по виду документа
- отдельные системные права
- отдельный флаг `isDocumentParticipant`
- удаление настраиваемых visibility-каналов из матрицы
- backend-проверки доступа для документов, вложений, поручений, ознакомлений и связей
- централизация document-domain access policy через `DocumentAccessService`
- защита `JournalService` проверкой доступа к документу перед выдачей журнала
- общий `DocumentQueryService` для карточек и списков документов
- общий `DocumentRegistrationService` для регистрации и обновления документов по `kindCode`
- тесты для matrix-only сценариев

## 7. Frontend-архитектура document-domain

Frontend больше не строится вокруг двух отдельных приложений "входящие" и "исходящие". Сейчас используется общий document-kind каркас:

- `DocumentKindPage` — общий orchestration-контейнер страницы вида документа
- `DocumentListPage` / `DocumentListTable` — общий список документов
- `DocumentViewModal` — общая карточка документа с kind-specific detail-блоком
- `useDocumentListPage` — общий hook списков
- `useDocumentKindModals` — общий hook register/edit modal flow
- `modules/documentKinds/incomingLetter` и `modules/documentKinds/outgoingLetter` — видоспецифичные формы, фильтры, config и helper-логика

Это означает, что добавление нового вида документа теперь требует в первую очередь:

- нового backend handler-а
- нового frontend модуля вида документа
- регистрации вида в системном kind-registry

а не копирования целой параллельной страницы и набора сервисов.

## 8. Регистрация документов

Регистрация документа в UI теперь проходит через единый сценарий:

- плавающая кнопка `Зарегистрировать` в `MainLayout`
- загрузка доступных видов документов через `DocumentKindService.GetAvailableForRegistration()`
- выбор вида документа
- открытие видоспецифичной формы
- сохранение через `DocumentRegistrationService.Register(kindCode, payload)`

Редактирование построено так же:

- вид документа определяет форму
- сохранение выполняется через `DocumentRegistrationService.Update(kindCode, payload)`

Это делает backend единым входом для command-side document-domain.

## 9. Дашборд

Дашборд больше не зависит от ручного переключения роли.

Текущая схема:

- backend вычисляет рабочий профиль пользователя
- поддерживаются профили `admin`, `clerk`, `executor`
- frontend рендерит соответствующий сценарий без `activeRole`

Для document-domain это важно, потому что:

- делопроизводитель видит текущую активность по поручениям и ознакомлениям
- исполнитель видит персональные поручения и свои ознакомления
- пользователь, который управляет ознакомлениями и сам включен в них, может подтвердить ознакомление прямо из дашборда

Статистические показатели вынесены из дашборда в отдельный раздел `Статистика`.

## 10. Системные разделы UI

Левое меню строится по системным правам и document-возможностям:

- `Дашборд` — только для пользователей document-domain
- `Входящие`, `Исходящие`, `Поручения` — для участников документооборота или пользователей с явными document-rights
- `Справочники` — только при `references`
- `Статистика` — при наличии хотя бы одного `stats_*`
- `Настройки` — только при `admin`

Для чистого системного администратора без document-rights:

- document-разделы не показываются
- стартовая страница — `Настройки`

## 11. Системные настройки

Текущие ключевые настройки:

- `organization_name` — полное название организации
- `organization_short_name` — краткое название организации
- `max_file_size_mb` — максимальный размер файла
- `allowed_file_types` — разрешенные типы файлов
- `assignment_completion_attachments_enabled` — можно ли исполнителю прикладывать файлы при завершении поручения

### Первичная настройка

При первом запуске под пользователем с правом `admin`, если не заполнены:

- `organization_name`
- `organization_short_name`

приложение показывает обязательное модальное окно первичной настройки. Закрыть его без сохранения нельзя.

### Значения по умолчанию

Для новой базы:

- `max_file_size_mb = 15`
- `allowed_file_types = .pdf,.doc,.docx,.odt,.xls,.xlsx,.ods`
- `organization_name = ""`
- `organization_short_name = ""`
- `assignment_completion_attachments_enabled = false`

## 12. Текущее состояние платформы

На данный момент в проекте уже зафиксированы следующие архитектурные опоры:

- системные виды документов `incoming_letter` и `outgoing_letter`
- общая корневая сущность `documents`
- kind-aware access layer
- kind-aware query layer
- kind-aware registration/update layer
- общая карточка документа
- общий список документов
- общая регистрация через FAB и выбор вида документа
- `documentKind` как основной транспортный формат вокруг поручений, ознакомлений и связей
- matrix-only модель document-access
- отдельные системные права
- отдельный флаг участника документооборота
- первичная настройка организации при первом запуске администратора

Это означает, что следующий новый вид документа можно добавлять уже как расширение платформы, а не как отдельную ветку логики приложения.

Кодовые опорные точки:

- [auth_service.go](/home/dimas/projects/docs-register-and-track/internal/services/auth_service.go)
- [document_kind.go](/home/dimas/projects/docs-register-and-track/internal/models/document_kind.go)
- [document_kind_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_kind_service.go)
- [document_access_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_access_service.go)
- [document_query_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_query_service.go)
- [document_kind_query_handler.go](/home/dimas/projects/docs-register-and-track/internal/services/document_kind_query_handler.go)
- [document_kind_command_handler.go](/home/dimas/projects/docs-register-and-track/internal/services/document_kind_command_handler.go)
- [journal_service.go](/home/dimas/projects/docs-register-and-track/internal/services/journal_service.go)
- [MainLayout.tsx](/home/dimas/projects/docs-register-and-track/frontend/src/components/MainLayout.tsx)
- [DocumentKindPage.tsx](/home/dimas/projects/docs-register-and-track/frontend/src/components/DocumentKindPage.tsx)
- [DocumentViewModal.tsx](/home/dimas/projects/docs-register-and-track/frontend/src/components/DocumentViewModal.tsx)
- [documentKinds.ts](/home/dimas/projects/docs-register-and-track/frontend/src/constants/documentKinds.ts)
- [useAuthStore.ts](/home/dimas/projects/docs-register-and-track/frontend/src/store/useAuthStore.ts)
